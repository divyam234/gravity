package main

import (
	"context"
	"fmt"
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

	// Start engines AND the HTTP server
	if err := a.Start(appCtx); err != nil {
		log.Fatal(err)
	}

	url := fmt.Sprintf("http://localhost:%d", a.Port())
	wailsApp := application.New(application.Options{
		Name:        "Gravity",
		Description: "A modern download manager",
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Gravity",
		URL:    url,
		Width:  1280,
		Height: 800,
	})

	wailsApp.OnShutdown(func() {
		a.Stop()
	})

	err = wailsApp.Run()
	if err != nil {
		log.Fatal(err)
	}
}
