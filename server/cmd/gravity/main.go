package main

import (
	"context"
	"gravity/internal/app"
	"gravity/internal/logger"
	"go.uber.org/zap"
)

// @title Gravity API
// @version 1.0
// @description API for Gravity Download Manager
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Ensure engines are initialized and HTTP server starts
	ctx := context.Background()
	a, err := app.New(ctx, nil, nil)
	if err != nil {
		logger.L.Fatal("failed to initialize gravity", zap.Error(err))
	}

	if err := a.Run(); err != nil {
		logger.L.Fatal("gravity runtime error", zap.Error(err))
	}
}
