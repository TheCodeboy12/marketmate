package main

import (
	"log"
	"net/http"

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

	// 5. Setup Routes with standard library net/http (Go 1.22+)
	mux := http.NewServeMux()

	// Public Routes
	mux.HandleFunc("POST /login", server.LoginHandler)

	// Protected Routes:
	// GET /lists -> requires lists:read
	mux.Handle("GET /lists", server.Authenticate(
		server.RequirePermission("lists:read")(http.HandlerFunc(server.ListShoppingLists)),
	))

	// POST /lists -> requires lists:create
	mux.Handle("POST /lists", server.Authenticate(
		server.RequirePermission("lists:create")(http.HandlerFunc(server.CreateShoppingList)),
	))

	// DELETE /lists/{id} -> requires lists:delete
	mux.Handle("DELETE /lists/{id}", server.Authenticate(
		server.RequirePermission("lists:delete")(http.HandlerFunc(server.DeleteShoppingList)),
	))

	log.Println("Server listening on http://localhost:8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
