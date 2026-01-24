package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gravity/internal/api"
	"gravity/internal/config"
	"gravity/internal/engine/aria2"
	"gravity/internal/engine/rclone"
	"gravity/internal/event"
	"gravity/internal/provider"
	"gravity/internal/provider/alldebrid"
	"gravity/internal/provider/direct"
	"gravity/internal/provider/realdebrid"
	"gravity/internal/service"
	"gravity/internal/store"

	"github.com/go-chi/chi/v5"
)

type App struct {
	config *config.Config
	store  *store.Store
	bus    *event.Bus

	downloadEngine *aria2.Engine
	uploadEngine   *rclone.Engine

	downloadService *service.DownloadService
	uploadService   *service.UploadService
	providerService *service.ProviderService
	magnetService   *service.MagnetService
	statsService    *service.StatsService
	searchService   *service.SearchService

	httpServer *http.Server
	router     *api.Router
}

func New(ctx context.Context) (*App, error) {
	cfg := config.Load()

	s, err := store.New(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	bus := event.NewBus()

	// Repos
	dr := store.NewDownloadRepo(s.GetDB())
	pr := store.NewProviderRepo(s.GetDB())
	sr := store.NewStatsRepo(s.GetDB())

	// Engines
	de := aria2.NewEngine(cfg.Aria2RPCPort, cfg.DataDir)
	ue := rclone.NewEngine(ctx)

	// Providers
	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ad := alldebrid.New()
	registry.Register(ad)
	registry.Register(realdebrid.New())

	// Services
	ps := service.NewProviderService(pr, registry)
	setr := store.NewSettingsRepo(s.GetDB())
	ds := service.NewDownloadService(dr, setr, de, ue, bus, ps)
	us := service.NewUploadService(dr, setr, ue, bus)
	ms := service.NewMagnetService(dr, setr, de, ad, ue, bus)
	ss := service.NewStatsService(sr, setr, dr, de, ue, bus)
	searchService := service.NewSearchService(store.NewSearchRepo(s.GetDB()), ue)

	// API
	router := api.NewRouter(cfg.APIKey)

	dh := api.NewDownloadHandler(ds)
	ph := api.NewProviderHandler(ps)
	rh := api.NewRemoteHandler(ue)
	sh := api.NewStatsHandler(ss)
	seth := api.NewSettingsHandler(setr, pr, de, ue)
	sysh := api.NewSystemHandler(de, ue)
	mh := api.NewMagnetHandler(ms)
	fh := api.NewFileHandler(ue, ue)
	searchHandler := api.NewSearchHandler(searchService)
	eh := api.NewEventHandler(bus, de, ss)

	// V1 Router
	v1 := chi.NewRouter()
	v1.Use(router.Auth)
	v1.Mount("/downloads", dh.Routes())
	v1.Mount("/providers", ph.Routes())
	v1.Mount("/remotes", rh.Routes())
	v1.Mount("/stats", sh.Routes())
	v1.Mount("/settings", seth.Routes())
	v1.Mount("/system", sysh.Routes())
	v1.Mount("/magnets", mh.Routes())
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
		downloadEngine:  de,
		uploadEngine:    ue,
		downloadService: ds,
		uploadService:   us,
		providerService: ps,
		magnetService:   ms,
		statsService:    ss,
		searchService:   searchService,
		httpServer:      srv,
		router:          router,
	}, nil
}

func (a *App) Events() *event.Bus {
	return a.bus
}

func (a *App) Port() int {
	return a.config.Port
}

func (a *App) Start(ctx context.Context) error {
	if err := a.StartEngines(ctx); err != nil {
		return err
	}
	return a.StartServer()
}

func (a *App) StartEngines(ctx context.Context) error {
	// Load settings from DB
	setr := store.NewSettingsRepo(a.store.GetDB())
	settings, _ := setr.Get(ctx)

	// Start engines
	if err := a.downloadEngine.Start(ctx); err != nil {
		return err
	}

	// Configure and start upload engine (Rclone VFS)
	a.uploadEngine.Configure(ctx, settings)
	if err := a.uploadEngine.Start(ctx); err != nil {
		log.Printf("Warning: Upload engine failed to start: %v", err)
	}

	// Init provider configs
	a.providerService.Init(ctx)

	// Sync engine state
	if err := a.downloadService.Sync(ctx); err != nil {
		log.Printf("Warning: Engine sync failed: %v", err)
	}

	// Start background services
	a.downloadService.Start()
	a.uploadService.Start()
	a.statsService.Start()
	a.searchService.Start()

	return nil
}

func (a *App) StartServer() error {
	// Start HTTP server
	go func() {
		log.Printf("Gravity listening on http://localhost%s", a.httpServer.Addr)
		if err := a.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server closed: %v", err)
		}
	}()

	return nil
}

func (a *App) Stop() {
	log.Println("Shutting down Gravity...")

	// Signal services to stop background routines
	a.downloadService.StopGracefully()
	a.statsService.Stop()

	a.downloadEngine.Stop()
	a.uploadEngine.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a.httpServer.Shutdown(shutdownCtx)
	a.store.Close()
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
