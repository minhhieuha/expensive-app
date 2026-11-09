package main

import (
	"log"
	"net/http"

	"expense-app/api/internal/config"
	"expense-app/api/internal/httpx"
)

func main() {
	cfg := config.Load()

	mux := httpx.SetupRoutes()

	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
