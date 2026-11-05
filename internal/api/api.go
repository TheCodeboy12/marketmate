// Package api defined the http api
package api

import "rbac-testing/internal/store"

type API struct {
	store     *store.Queries
	jwtSecret []byte
}

func New(store *store.Queries, jwtSecret []byte) *API {
	return &API{
		store,
		jwtSecret,
	}
}
