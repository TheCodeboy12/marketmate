package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/TheCodeboy12/marketmate/internal/store"
)

// ListItemsHandler lists all items in a shopping list belonging to caller's tenant
func (api *API) ListItemsHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	listIDStr := pathParam(r, "listID")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil || listID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid list ID")
		return
	}

	items, err := api.store.ListItemsByListID(r.Context(), store.ListItemsByListIDParams{
		ShoppingListID: listID,
		TenantID:       caller.TenantID,
	})
	if err != nil {
		logger.Printf("failed to list items for list %d: %v", listID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

type CreateItemPayload struct {
	Name                  string  `json:"name"`
	AmountToBuy           int64   `json:"amount_to_buy"`
	Category              *string `json:"category"`
	PrefferdSupermarketID *int64  `json:"prefferd_supermarket_id"`
}

// CreateItemHandler creates a new item in a shopping list belonging to caller's tenant
func (api *API) CreateItemHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	listIDStr := pathParam(r, "listID")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil || listID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid list ID")
		return
	}

	var data CreateItemPayload
	if err := readJSON(r, &data); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if data.Name == "" {
		writeError(w, http.StatusBadRequest, "Item name is required")
		return
	}
	if data.AmountToBuy < 0 {
		writeError(w, http.StatusBadRequest, "amount_to_buy must be >= 0")
		return
	}

	var supermarketID *int64
	if data.PrefferdSupermarketID != nil && *data.PrefferdSupermarketID > 0 {
		supermarketID = data.PrefferdSupermarketID
		// Verify supermarket belongs to caller's tenant
		_, err := api.store.GetSupermarketByID(r.Context(), store.GetSupermarketByIDParams{
			ID:       *supermarketID,
			TenantID: caller.TenantID,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid preferred supermarket ID for tenant")
			return
		}
	}

	item, err := api.store.CreateItem(r.Context(), store.CreateItemParams{
		ID:                    listID, // maps to shopping_list.id check
		TenantID:              caller.TenantID,
		Name:                  data.Name,
		AmountToBuy:           data.AmountToBuy,
		CreatedBy:             caller.UserID,
		Category:              data.Category,
		PrefferdSupermarketID: supermarketID,
	})
	if err != nil {
		logger.Printf("failed to create item: %v", err)
		writeError(w, http.StatusBadRequest, "Unable to create item (list may not exist or belong to another tenant)")
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

type ToggleItemPayload struct {
	Ticked bool `json:"ticked"`
}

// ToggleItemHandler marks an item as ticked or unticked within caller's tenant
func (api *API) ToggleItemHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	itemIDStr := pathParam(r, "id")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil || itemID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var data ToggleItemPayload
	if err := readJSON(r, &data); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = api.store.ToggleItem(r.Context(), store.ToggleItemParams{
		ID:       itemID,
		TenantID: caller.TenantID,
		Ticked:   data.Ticked,
	})
	if err != nil {
		logger.Printf("failed to toggle item %d: %v", itemID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Item updated successfully"})
}

type UpdateListItemPayload struct {
	Name                  *string `json:"name"`
	AmountToBuy           *int64  `json:"amount_to_buy"`
	Category              *string `json:"category"`
	PrefferdSupermarketID *int64  `json:"prefferd_supermarket_id"`
}

// UpdateListItemHandler updates an item's details within caller's tenant
func (api *API) UpdateListItemHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	itemIDStr := pathParam(r, "id")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil || itemID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var data UpdateListItemPayload
	if err := readJSON(r, &data); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var name string
	if data.Name != nil {
		name = *data.Name
	}

	var amount int64
	if data.AmountToBuy != nil {
		amount = *data.AmountToBuy
	}

	var supermarketID *int64
	if data.PrefferdSupermarketID != nil && *data.PrefferdSupermarketID > 0 {
		supermarketID = data.PrefferdSupermarketID
		// Verify supermarket belongs to caller's tenant
		_, err := api.store.GetSupermarketByID(r.Context(), store.GetSupermarketByIDParams{
			ID:       *supermarketID,
			TenantID: caller.TenantID,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid preferred supermarket ID for tenant")
			return
		}
	}

	item, err := api.store.UpdateListItem(r.Context(), store.UpdateListItemParams{
		ID:                    itemID,
		TenantID:              caller.TenantID,
		Name:                  name,
		AmountToBuy:           amount,
		Category:              data.Category,
		PrefferdSupermarketID: supermarketID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Item not found")
			return
		}
		logger.Printf("failed to update item %d: %v", itemID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// DeleteItemHandler deletes an item within caller's tenant
func (api *API) DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	caller, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	itemIDStr := pathParam(r, "id")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil || itemID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	err = api.store.DeleteItem(r.Context(), store.DeleteItemParams{
		ID:       itemID,
		TenantID: caller.TenantID,
	})
	if err != nil {
		logger.Printf("failed to delete item %d: %v", itemID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Item deleted successfully"})
}
