package main

import (
	"net/http"
	"os"

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

	logger.Info("server_started", "addr", cfg.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}
