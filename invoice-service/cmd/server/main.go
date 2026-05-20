package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/handler"
)

func main() {
	cfg := config.Load()

	logger := handler.NewLogger(cfg.LogLevel, cfg.LogFormat)

	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(cfg.MigrationsDir); err != nil {
		logger.Error("run migrations", "error", err)
		os.Exit(1)
	}

	app := handler.New(store, cfg, logger)
	server := &http.Server{
		Addr:    cfg.Address,
		Handler: app.Routes(),
	}

	go func() {
		logger.Info("server_started", "addr", cfg.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("serve", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting_down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server_shutdown", "error", err)
	}

	app.WaitForOCR()
	logger.Info("server_stopped")
}
