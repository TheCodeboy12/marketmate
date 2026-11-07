package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rbac-testing/internal/store"
)

func TestItemHandlersTenantSafety(t *testing.T) {
	apiInstance, queries, cleanup := setupTestAPI(t)
	defer cleanup()

	// 1. Create a list in Tenant 1 (belonging to user 2 Alice)
	list1, err := queries.CreateShoppingList(context.Background(), store.CreateShoppingListParams{
		TenantID:  1,
		Name:      "Groceries Tenant 1",
		CreatedBy: 2,
	})
	if err != nil {
		t.Fatalf("failed to create test list: %v", err)
	}

	// 2. Caller in Tenant 1 creates an item in List 1 -> should succeed
	itemBody := []byte(`{"name":"Organic Apples","amount_to_buy":5,"category":"fruit"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/lists/1/items", bytes.NewReader(itemBody))
	createReq.SetPathValue("listID", "1")
	createReq = createReq.WithContext(contextWithUser(createReq.Context(), 2, 1)) // Alice Tenant 1
	createRec := httptest.NewRecorder()

	apiInstance.CreateItemHandler(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid item creation, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var createdItem store.ShoppingListItem
	if err := json.Unmarshal(createRec.Body.Bytes(), &createdItem); err != nil {
		t.Fatalf("failed to decode created item: %v", err)
	}
	if createdItem.Name != "Organic Apples" || createdItem.ShoppingListID != list1.ID {
		t.Fatalf("unexpected created item: %+v", createdItem)
	}

	// 3. Caller from Tenant 2 tries to add item to List 1 -> MUST BE REJECTED
	attackReq := httptest.NewRequest(http.MethodPost, "/lists/1/items", bytes.NewReader(itemBody))
	attackReq.SetPathValue("listID", "1")
	attackReq = attackReq.WithContext(contextWithUser(attackReq.Context(), 99, 2)) // Tenant 2
	attackRec := httptest.NewRecorder()

	apiInstance.CreateItemHandler(attackRec, attackReq)

	if attackRec.Code != http.StatusBadRequest && attackRec.Code != http.StatusNotFound {
		t.Fatalf("expected 400 or 404 for cross-tenant item creation, got %d", attackRec.Code)
	}

	// 4. Caller from Tenant 2 tries to list items of List 1 -> should return empty slice (0 items)
	listReq := httptest.NewRequest(http.MethodGet, "/lists/1/items", nil)
	listReq.SetPathValue("listID", "1")
	listReq = listReq.WithContext(contextWithUser(listReq.Context(), 99, 2)) // Tenant 2
	listRec := httptest.NewRecorder()

	apiInstance.ListItemsHandler(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var tenant2Items []store.ShoppingListItem
	if err := json.Unmarshal(listRec.Body.Bytes(), &tenant2Items); err != nil {
		t.Fatalf("failed to parse items: %v", err)
	}
	if len(tenant2Items) != 0 {
		t.Fatalf("expected 0 items for cross-tenant list query, got %d", len(tenant2Items))
	}

	// 5. Caller from Tenant 1 updates the item -> succeeds
	updateBody := []byte(`{"name":"Honeycrisp Apples","amount_to_buy":10}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/items/1", bytes.NewReader(updateBody))
	updateReq.SetPathValue("id", "1")
	updateReq = updateReq.WithContext(contextWithUser(updateReq.Context(), 2, 1)) // Tenant 1
	updateRec := httptest.NewRecorder()

	apiInstance.UpdateListItemHandler(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for item update, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	// 6. Caller from Tenant 2 tries to delete item 1 -> MUST FAIL
	deleteReq := httptest.NewRequest(http.MethodDelete, "/items/1", nil)
	deleteReq.SetPathValue("id", "1")
	deleteReq = deleteReq.WithContext(contextWithUser(deleteReq.Context(), 99, 2)) // Tenant 2
	deleteRec := httptest.NewRecorder()

	apiInstance.DeleteItemHandler(deleteRec, deleteReq)

	// Since tenant 2 has no access, 0 rows are deleted. Verify item 1 still exists:
	checkItems, err := queries.ListItemsByListID(context.Background(), store.ListItemsByListIDParams{
		ShoppingListID: list1.ID,
		TenantID:       1,
	})
	if err != nil || len(checkItems) == 0 {
		t.Fatalf("expected item 1 to remain undeleted after cross-tenant delete attempt")
	}
}
