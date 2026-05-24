package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"invoice-app/internal/auth"
	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/handler"
	"invoice-app/internal/tracing"
)

func main() {
	cfg := config.Load()

	logger := handler.NewLogger(cfg.LogLevel, cfg.LogFormat, cfg.LogIncludeSource)
	shutdownTracing, err := tracing.Init(context.Background(), "invoice-service")
	if err != nil {
		logger.Error("initialize tracing", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(ctx); err != nil {
			logger.Error("shutdown tracing", "error", err)
		}
	}()

	var authDB *db.AuthDB
	var webAuthn *auth.WebAuthnManager
	var sessionManager *auth.SessionManager

	if cfg.AuthEnabled {
		var err error
		authDB, err = db.OpenAuthDB(cfg.AuthDBPath)
		if err != nil {
			logger.Error("open auth database", "error", err)
			os.Exit(1)
		}
		defer func() {
			_ = authDB.Close()
		}()

		if err := authDB.Migrate(cfg.AuthMigrationsDir); err != nil {
			logger.Error("run auth migrations", "error", err)
			os.Exit(1)
		}

		webauthnCfg := auth.AuthConfig{
			RPID:          cfg.WebAuthnRPID,
			RPOrigins:     cfg.WebAuthnRPOrigins,
			RPDisplayName: cfg.WebAuthnRPDisplay,
			SessionTTL:    cfg.SessionTTL,
		}

		webAuthn, err = auth.NewWebAuthnManager(authDB, webauthnCfg)
		if err != nil {
			logger.Error("initialize WebAuthn", "error", err)
			os.Exit(1)
		}

		sessionManager = auth.NewSessionManager(authDB, cfg.SessionTTL, cfg.TrustedProxy)
	}

	multiStore := db.NewMultiStore(cfg.UserDBDir, cfg.MigrationsDir)

	var legacyStore *db.Store
	if !cfg.AuthEnabled {
		var err error
		legacyStore, err = db.Open(cfg.DatabasePath)
		if err != nil {
			logger.Error("open legacy database", "error", err)
			os.Exit(1)
		}
		if err := legacyStore.Migrate(cfg.MigrationsDir); err != nil {
			logger.Error("migrate legacy database", "error", err)
			os.Exit(1)
		}
		multiStore.SetLegacyStore(legacyStore)
	}

	app := handler.New(multiStore, authDB, webAuthn, sessionManager, cfg, logger)

	if !cfg.AuthEnabled {
		app.SetLegacyStore(legacyStore)
	}

	// Start periodic session cleanup (hourly)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	if authDB != nil {
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := authDB.CleanupExpiredSessions(); err != nil {
						logger.Warn("session cleanup failed", "error", err)
					} else {
						logger.Debug("session cleanup complete")
					}
				case <-cleanupCtx.Done():
					return
				}
			}
		}()
	}

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

	if err := multiStore.Close(); err != nil {
		logger.Error("close user databases", "error", err)
	}

	logger.Info("server_stopped")
}
