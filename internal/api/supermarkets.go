package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"rbac-testing/internal/store"
)

type CreateSupermarketRequest struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type UpdateSupermarketRequest struct {
	Name      *string  `json:"name"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// ListSupermarketsHandler lists all supermarkets for the caller's tenant
func (api *API) ListSupermarketsHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	supermarkets, err := api.store.ListSupermarkets(r.Context(), caller.TenantID)
	if err != nil {
		logger.Printf("failed to list supermarkets for tenant %d: %v", caller.TenantID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if supermarkets == nil {
		supermarkets = []store.Supermarket{}
	}

	writeJSON(w, http.StatusOK, supermarkets)
}

// GetSupermarketHandler fetches a single supermarket by ID within the caller's tenant
func (api *API) GetSupermarketHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := pathParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid supermarket ID")
		return
	}

	sm, err := api.store.GetSupermarketByID(r.Context(), store.GetSupermarketByIDParams{
		ID:       id,
		TenantID: caller.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Supermarket %d not found", id))
			return
		}
		logger.Printf("failed to get supermarket %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, sm)
}

// CreateSupermarketHandler creates a new supermarket scoped to caller's tenant
func (api *API) CreateSupermarketHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateSupermarketRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Supermarket name is required")
		return
	}
	if req.Latitude < -90.0 || req.Latitude > 90.0 {
		writeError(w, http.StatusBadRequest, "Latitude must be between -90 and 90")
		return
	}
	if req.Longitude < -180.0 || req.Longitude > 180.0 {
		writeError(w, http.StatusBadRequest, "Longitude must be between -180 and 180")
		return
	}

	created, err := api.store.CreateSupermarket(r.Context(), store.CreateSupermarketParams{
		TenantID:  caller.TenantID,
		Name:      req.Name,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	})
	if err != nil {
		logger.Printf("failed to create supermarket: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create supermarket")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// UpdateSupermarketHandler updates an existing supermarket in the caller's tenant
func (api *API) UpdateSupermarketHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := pathParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid supermarket ID")
		return
	}

	// Verify existence in caller's tenant
	existing, err := api.store.GetSupermarketByID(r.Context(), store.GetSupermarketByIDParams{
		ID:       id,
		TenantID: caller.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Supermarket %d not found", id))
			return
		}
		logger.Printf("failed to get supermarket %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	var req UpdateSupermarketRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	name := existing.Name
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}
	lat := existing.Latitude
	if req.Latitude != nil {
		if *req.Latitude < -90.0 || *req.Latitude > 90.0 {
			writeError(w, http.StatusBadRequest, "Latitude must be between -90 and 90")
			return
		}
		lat = *req.Latitude
	}
	lon := existing.Longitude
	if req.Longitude != nil {
		if *req.Longitude < -180.0 || *req.Longitude > 180.0 {
			writeError(w, http.StatusBadRequest, "Longitude must be between -180 and 180")
			return
		}
		lon = *req.Longitude
	}

	updated, err := api.store.UpdateSupermarket(r.Context(), store.UpdateSupermarketParams{
		ID:        id,
		TenantID:  caller.TenantID,
		Name:      name,
		Latitude:  lat,
		Longitude: lon,
	})
	if err != nil {
		logger.Printf("failed to update supermarket %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Failed to update supermarket")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// DeleteSupermarketHandler deletes a supermarket belonging to the caller's tenant
func (api *API) DeleteSupermarketHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := pathParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid supermarket ID")
		return
	}

	// Verify existence in caller's tenant
	_, err = api.store.GetSupermarketByID(r.Context(), store.GetSupermarketByIDParams{
		ID:       id,
		TenantID: caller.TenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("Supermarket %d not found", id))
			return
		}
		logger.Printf("failed to get supermarket %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if err := api.store.DeleteSupermarket(r.Context(), store.DeleteSupermarketParams{
		ID:       id,
		TenantID: caller.TenantID,
	}); err != nil {
		logger.Printf("failed to delete supermarket %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Failed to delete supermarket")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Supermarket deleted successfully"})
}
