package main

import (
	"context"
	"log"

	"gravity/internal/app"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	appCtx := context.Background()
	a, err := app.New(appCtx)
	if err != nil {
		log.Fatalf("Failed to initialize Gravity: %v", err)
	}

	// Start engines but NOT the HTTP server
	if err := a.StartEngines(appCtx); err != nil {
		log.Fatal(err)
	}

	wailsApp := application.New(application.Options{
		Name:        "Gravity",
		Description: "A modern download manager",
		Services: []application.Service{
			application.NewService(app.NewBridge(a)),
		},
		Assets: application.AssetOptions{
			Handler: a.Handler(),
		},
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Gravity",
		URL:    "/",
		Width:  1280,
		Height: 800,
	})

	wailsApp.OnShutdown(func() {
		a.Stop()
	})

	// Forward events from Bus to Wails
	go func() {
		ch := a.Events().Subscribe()
		for ev := range ch {
			wailsApp.Event.Emit("gravity-event", ev)
		}
	}()

	err = wailsApp.Run()
	if err != nil {
		log.Fatal(err)
	}
}
