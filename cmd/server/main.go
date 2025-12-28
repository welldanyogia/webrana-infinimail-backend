package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Infinimail Backend Server...")

	// TODO: Load configuration
	// TODO: Initialize database connection
	// TODO: Initialize repositories
	// TODO: Initialize file storage
	// TODO: Initialize WebSocket hub
	// TODO: Initialize HTTP server
	// TODO: Initialize SMTP server
	// TODO: Start servers

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("Shutting down server...")

	_ = ctx // Will be used for graceful shutdown
	slog.Info("Server stopped")
}
