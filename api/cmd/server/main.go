package main

import (
	"log"
	"net/http"

	"expense-app/api/internal/config"
	"expense-app/api/internal/db"
	"expense-app/api/internal/httpx"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database at %s: %v", cfg.DBPath, err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("Database opened and migrations applied successfully")

	mux := httpx.SetupRoutes()

	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
