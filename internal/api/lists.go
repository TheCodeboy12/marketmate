package api

import (
	"net/http"
	"strconv"

	"rbac-testing/internal/store"
)

type CreateListPayload struct {
	Name string `json:"name"`
}

func (api *API) ListShoppingLists(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	lists, err := api.store.ListShoppingListsByTenant(r.Context(), user.TenantID)
	if err != nil {
		logger.Printf("could not list shopping lists: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, lists)
}

func (api *API) CreateShoppingList(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var data CreateListPayload
	if err := readJSON(r, &data); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	list, err := api.store.CreateShoppingList(r.Context(), store.CreateShoppingListParams{
		TenantID:  user.TenantID,
		Name:      data.Name,
		CreatedBy: user.UserID,
	})
	if err != nil {
		logger.Printf("could not create shopping list: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, list)
}

func (api *API) DeleteShoppingList(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := pathParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid list ID")
		return
	}

	err = api.store.DeleteShoppingList(r.Context(), store.DeleteShoppingListParams{
		ID:       id,
		TenantID: user.TenantID,
	})
	if err != nil {
		logger.Printf("could not delete shopping list: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Shopping list deleted successfully"})
}
