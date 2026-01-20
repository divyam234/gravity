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
}

func New() (*App, error) {
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
	cr := store.NewCacheRepo(s.GetDB())

	// Engines
	de := aria2.NewEngine(cfg.Aria2RPCPort, cfg.DataDir)
	ue := rclone.NewEngine(cfg.RcloneRPCPort, cr)

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
	us := service.NewUploadService(dr, ue, bus)
	ms := service.NewMagnetService(dr, setr, de, ad, ue)
	ss := service.NewStatsService(sr, dr, de, ue, bus)
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
	fh := api.NewFileHandler(ue)
	searchHandler := api.NewSearchHandler(searchService)

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

	// Mount V1 to root
	router.Mount("/api/v1", v1)

	// Frontend
	fs := http.FileServer(http.Dir("./dist"))
	router.Handle("/*", fs)

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
	}, nil
}

func (a *App) Run() error {
	ctx := context.Background()

	// Start engines
	if err := a.downloadEngine.Start(ctx); err != nil {
		return err
	}
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

	// Start HTTP server
	go func() {
		log.Printf("Gravity listening on http://localhost%s", a.httpServer.Addr)
		if err := a.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Gravity...")

	a.downloadEngine.Stop()
	a.uploadEngine.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a.httpServer.Shutdown(shutdownCtx)
	a.store.Close()

	return nil
}
