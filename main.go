package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"rbac-testing/internal/api"
	"rbac-testing/internal/auth"
	"rbac-testing/internal/db"
	"rbac-testing/internal/store"
)

func main() {
	jwtSecret := []byte("super-secret-key-1234567890")
	dbPath := "app.db?_pragma=foreign_keys(1)"

	// 1. Initialize SQLite Database (creates app.db and executes schema + seeds)
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize sqlite db: %v", err)
	}
	defer sqlDB.Close()
	log.Println("SQLite database initialized and seeded successfully (app.db)")

	// 2. Instantiate store (implements store.Querier interface)
	queries := store.New(sqlDB)

	// 3. Instantiate Authorizer (implements auth.Authorizer interface)
	authorizer := auth.NewAuthorizer(queries)

	// 4. Instantiate API
	server := api.New(queries, jwtSecret, authorizer)

	// 5. Setup Routes with Chi Router
	r := chi.NewRouter()

	// Standard Chi Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Public Routes
	r.Post("/login", server.LoginHandler)

	// Protected Routes (requires valid JWT)
	r.Group(func(r chi.Router) {
		r.Use(server.Authenticate)

		r.Get("/me", server.MeHandler)

		// Shopping Lists
		r.Route("/lists", func(r chi.Router) {
			r.With(server.RequirePermission("lists:read")).Get("/", server.ListShoppingLists)
			r.With(server.RequirePermission("lists:create")).Post("/", server.CreateShoppingList)
			r.With(server.RequirePermission("lists:delete")).Delete("/{id}", server.DeleteShoppingList)

			// Items within a list
			r.Route("/{listID}/items", func(r chi.Router) {
				r.With(server.RequirePermission("lists:read")).Get("/", server.ListItemsHandler)
				r.With(server.RequirePermission("items:create")).Post("/", server.CreateItemHandler)
			})
		})

		// Items direct actions
		r.Route("/items", func(r chi.Router) {
			r.With(server.RequirePermission("items:check")).Patch("/{id}/toggle", server.ToggleItemHandler)
			r.With(server.RequirePermission("items:create")).Put("/{id}", server.UpdateListItemHandler)
			r.With(server.RequirePermission("items:delete")).Delete("/{id}", server.DeleteItemHandler)
		})

		// Supermarkets
		r.Route("/supermarkets", func(r chi.Router) {
			r.With(server.RequirePermission("lists:read")).Get("/", server.ListSupermarketsHandler)
			r.With(server.RequirePermission("lists:read")).Get("/{id}", server.GetSupermarketHandler)
			r.With(server.RequirePermission("items:create")).Post("/", server.CreateSupermarketHandler)
			r.With(server.RequirePermission("items:create")).Put("/{id}", server.UpdateSupermarketHandler)
			r.With(server.RequirePermission("lists:delete")).Delete("/{id}", server.DeleteSupermarketHandler)
		})

		// Users
		r.Route("/users", func(r chi.Router) {
			r.With(server.RequirePermission("users:read")).Get("/", server.ListUsersHandler)
			r.With(server.RequirePermission("users:read")).Get("/{userID}", server.GetUserHandler)
			r.With(server.RequirePermission("users:create")).Post("/", server.CreateUserHandler)
			r.With(server.RequirePermission("users:modify")).Put("/{userID}", server.UpdateUserHandler)
			r.With(server.RequirePermission("users:modify")).Put("/{userID}/password", server.UpdateUserPasswordHandler)
			r.With(server.RequirePermission("users:delete")).Delete("/{userID}", server.DeleteUserHandler)
		})
	})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	log.Printf("Server listening on http://localhost:%s...", srv.Addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()
	<-ctx.Done()
	// Create shutdown context with 30-second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Trigger graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
