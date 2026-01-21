package main

import (
	"context"
	"gravity/internal/app"
	"log"
)

func main() {
	// Ensure engines are initialized and HTTP server starts
	ctx := context.Background()
	a, err := app.New(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Gravity: %v", err)
	}

	if err := a.Run(); err != nil {
		log.Fatalf("Gravity runtime error: %v", err)
	}
}
