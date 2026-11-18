package main

import (
	"context"
	"log"
	"net/http"

	"expense-app/api/internal/auth"
	"expense-app/api/internal/category"
	"expense-app/api/internal/config"
	"expense-app/api/internal/db"
	"expense-app/api/internal/httpx"
	"expense-app/api/internal/transaction"
)

type catValidatorAdapter struct {
	repo *category.Repo
}

func (a *catValidatorAdapter) GetCategory(ctx context.Context, userID int64, categoryID int64) (*transaction.CategoryInfo, error) {
	c, err := a.repo.GetByID(ctx, userID, categoryID)
	if err != nil {
		return nil, err
	}
	return &transaction.CategoryInfo{ID: c.ID, Kind: c.Kind}, nil
}

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

	authRepo := auth.NewRepo(database)
	authService := auth.NewService(authRepo, cfg.SessionTTLHours)
	authHandler := auth.NewHandler(authService, cfg.CookieSecure)
	authHandler.RegisterRoutes(mux)

	authValidator := func(ctx context.Context, token string) (int64, error) {
		user, err := authService.GetCurrentUser(ctx, token)
		if err != nil {
			return 0, err
		}
		return user.ID, nil
	}
	requireAuthMid := httpx.RequireAuth(authValidator)

	categoryRepo := category.NewRepo(database)
	categoryService := category.NewService(categoryRepo)
	categoryHandler := category.NewHandler(categoryService, requireAuthMid)
	categoryHandler.RegisterRoutes(mux)

	txRepo := transaction.NewRepo(database)
	txService := transaction.NewService(txRepo, &catValidatorAdapter{repo: categoryRepo})
	txHandler := transaction.NewHandler(txService, requireAuthMid)
	txHandler.RegisterRoutes(mux)

	handler := httpx.Chain(
		mux,
		httpx.Recovery,
		httpx.SecurityHeaders,
		httpx.RequireCSRF,
	)

	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
