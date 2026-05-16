package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"invoice-app/internal/config"
	"invoice-app/internal/db"
	"invoice-app/internal/handler"
)

func main() {
	cfg := config.Load()

	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(cfg.MigrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	app := handler.New(store, cfg)
	server := &http.Server{
		Addr:    cfg.Address,
		Handler: app.Routes(),
	}

	fmt.Fprintf(os.Stdout, "invoice-app listening on %s\n", cfg.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
