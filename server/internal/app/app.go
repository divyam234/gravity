package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gravity/internal/api"
	"gravity/internal/binaries"
	"gravity/internal/config"
	"gravity/internal/engine"
	"gravity/internal/engine/aria2"
	"gravity/internal/engine/hybrid"
	"gravity/internal/engine/native"
	"gravity/internal/engine/rclone"
	"gravity/internal/event"
	"gravity/internal/logger"
	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/provider/alldebrid"
	"gravity/internal/provider/debridlink"
	"gravity/internal/provider/direct"
	"gravity/internal/provider/megadebrid"
	"gravity/internal/provider/premiumize"
	"gravity/internal/provider/realdebrid"
	"gravity/internal/provider/torbox"
	"gravity/internal/provider/ytdlp"
	"gravity/internal/service"
	"gravity/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/rclone/rclone/fs"
	"go.uber.org/zap"
)

type App struct {
	config *config.Config
	store  *store.Store
	bus    *event.Bus
	logger *zap.Logger
	binMgr *binaries.Manager

	DownloadEngine engine.DownloadEngine
	UploadEngine   engine.UploadEngine

	downloadService *service.DownloadService
	uploadService   *service.UploadService
	providerService *service.ProviderService
	statsService    *service.StatsService
	searchService   *service.SearchService

	httpServer *http.Server
	Router     *api.Router
}

func (a *App) Config() *config.Config {
	return a.config
}

func (a *App) SetDownloadEngine(de engine.DownloadEngine) {
	a.DownloadEngine = de
}

func (a *App) SetUploadEngine(ue engine.UploadEngine) {
	a.UploadEngine = ue
}

func New(ctx context.Context, de engine.DownloadEngine, ue engine.UploadEngine) (*App, error) {
	ci := fs.GetConfig(ctx)
	ci.LogLevel = fs.LogLevelError

	cfg := config.Load()

	// Initialize logger
	l := logger.New(cfg.LogLevel, cfg.LogLevel == "debug", cfg.LogFile, cfg.JSONLog)

	// Print Banner
	logger.PrintBanner(logger.StartupInfo{
		Version:  "0.1.0",
		Addr:     fmt.Sprintf(":%d", cfg.Port),
		DataDir:  cfg.DataDir,
		LogLevel: cfg.LogLevel,
	})

	l.Info("starting gravity")

	s, err := store.New(cfg)
	if err != nil {
		return nil, err
	}

	bus := event.NewBus()

	// Initialize Binaries Manager
	binMgr := binaries.NewManager(cfg.DataDir)

	// Repos
	dr := store.NewDownloadRepo(s.GetDB())
	pr := store.NewProviderRepo(s.GetDB())
	sr := store.NewStatsRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	searchRepo := store.NewSearchRepo(s.GetDB())

	// Engines (Initialize both for Hybrid support)
	if de == nil {
		de1 := aria2.NewEngine(cfg.Aria2RPCPort, cfg.DataDir, l)
		de1.GetRunner().SetBinaryPath(binMgr.GetAria2Path())
		de2 := native.NewNativeEngine(cfg.DataDir)
		de = hybrid.NewHybridRouter(de1, de2)
	}
	if ue == nil {
		ue = rclone.NewEngine(ctx, cfg.RcloneConfigPath)
	}

	// Providers
	registry := provider.NewRegistry()
	registry.Register(direct.New())
	registry.Register(ytdlp.New(binMgr.GetYtDlpPath()))
	ad := alldebrid.New()
	registry.Register(ad)
	registry.Register(realdebrid.New())
	registry.Register(premiumize.New())
	registry.Register(debridlink.New())
	registry.Register(torbox.New())
	registry.Register(megadebrid.New())

	// Services
	ps := service.NewProviderService(pr, registry, de)
	ds := service.NewDownloadService(dr, setr, de, ue, bus, ps)
	us := service.NewUploadService(dr, setr, ue, bus)
	ss := service.NewStatsService(sr, setr, dr, de, ue, bus)
	searchService := service.NewSearchService(searchRepo, setr, ue)

	// API
	router := api.NewRouter(cfg.APIKey)

	dh := api.NewDownloadHandler(ds)
	ph := api.NewProviderHandler(ps)
	rh := api.NewRemoteHandler(ue)
	sh := api.NewStatsHandler(ss)
	seth := api.NewSettingsHandler(setr, pr, de, ue, bus)
	sysh := api.NewSystemHandler(ctx, de, ue)
	fh := api.NewFileHandler(ue, ue)
	searchHandler := api.NewSearchHandler(ctx, searchService)
	eh := api.NewEventHandler(bus, de, ss)

	// V1 Router
	v1 := chi.NewRouter()
	v1.Use(router.Auth)
	v1.Use(logger.Middleware(l)) // Use structured request logger
	v1.Mount("/downloads", dh.Routes())
	v1.Mount("/providers", ph.Routes())
	v1.Mount("/remotes", rh.Routes())
	v1.Mount("/stats", sh.Routes())
	v1.Mount("/settings", seth.Routes())
	v1.Mount("/system", sysh.Routes())
	v1.Mount("/files", fh.Routes())
	v1.Mount("/search", searchHandler.Routes())
	v1.Mount("/events", eh.Routes())

	// Mount V1 to root
	router.Mount("/api/v1", v1)
	router.Handle("/*", AssetsHandler())

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router.Handler(),
	}

	return &App{
		config:          cfg,
		store:           s,
		bus:             bus,
		logger:          l,
		binMgr:          binMgr,
		DownloadEngine:  de,
		UploadEngine:    ue,
		downloadService: ds,
		uploadService:   us,
		providerService: ps,
		statsService:    ss,
		searchService:   searchService,
		httpServer:      srv,
		Router:          router,
	}, nil
}

func (a *App) Events() *event.Bus {
	return a.bus
}

func (a *App) Port() int {
	return a.config.Port
}

func (a *App) DownloadService() *service.DownloadService {
	return a.downloadService
}

func (a *App) StatsService() *service.StatsService {
	return a.statsService
}

func (a *App) Start(ctx context.Context) error {
	if err := a.StartEngines(ctx); err != nil {
		return err
	}
	return a.StartServer()
}

func (a *App) StartEngines(ctx context.Context) error {
	// Ensure binaries are present
	a.binMgr.EnsureBinaries(ctx)

	// Load settings from DB
	setr := store.NewSettingsRepo(a.store.GetDB())
	settings, _ := setr.Get(ctx)

	// First run: Initialize defaults if missing
	if settings == nil {
		a.logger.Debug("first run detected: initializing default settings")
		settings = model.DefaultSettings()
		if err := setr.Save(ctx, settings); err != nil {
			a.logger.Warn("failed to save default settings", zap.Error(err))
		}
	}

	// Start engines
	if err := a.DownloadEngine.Start(ctx); err != nil {
		return err
	}

	// Configure engines
	if settings != nil {
		a.DownloadEngine.Configure(ctx, settings)
		a.UploadEngine.Configure(ctx, settings)
	}

	// Start upload engine (Rclone VFS)
	if err := a.UploadEngine.Start(ctx); err != nil {
		a.logger.Warn("upload engine failed to start", zap.Error(err))
	}

	// Init provider configs
	a.providerService.Init(ctx)

	// Sync engine state
	if err := a.downloadService.Sync(ctx); err != nil {
		a.logger.Warn("engine sync failed", zap.Error(err))
	}

	// Start background services
	a.downloadService.Start(ctx)
	a.uploadService.Start(ctx)
	a.statsService.Start(ctx)
	a.searchService.Start(ctx)

	return nil
}

func (a *App) StartServer() error {
	// Start HTTP server
	go func() {
		a.logger.Info("server listening", zap.String("addr", a.httpServer.Addr))
		if err := a.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			a.logger.Error("http server failed", zap.Error(err))
		}
	}()

	return nil
}

func (a *App) Stop() {
	a.logger.Info("shutting down gravity...")

	// Signal services to stop background routines
	a.downloadService.StopGracefully()
	a.statsService.Stop()

	a.DownloadEngine.Stop()
	a.UploadEngine.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("server shutdown failed", zap.Error(err))
	}

	a.store.Close()

	// Sync logger before exit
	_ = a.logger.Sync()
}

func (a *App) Run() error {
	ctx := context.Background()
	if err := a.Start(ctx); err != nil {
		return err
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Stop()
	return nil
}
