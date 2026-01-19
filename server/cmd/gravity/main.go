package main

import (
	"log"

	"gravity/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize Gravity: %v", err)
	}

	if err := a.Run(); err != nil {
		log.Fatalf("Gravity runtime error: %v", err)
	}
}
