package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/TheCodeboy12/marketmate/internal/store"
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

	// 7. Preferred supermarket cross-tenant validation:
	// Create a supermarket belonging to Tenant 2
	smT2, err := queries.CreateSupermarket(context.Background(), store.CreateSupermarketParams{
		TenantID:  2,
		Name:      "Tenant 2 Supermarket",
		Latitude:  12.34,
		Longitude: 56.78,
	})
	if err != nil {
		t.Fatalf("failed to create supermarket for tenant 2: %v", err)
	}

	// Tenant 1 attempts to add item referencing Tenant 2's supermarket -> MUST BE REJECTED
	badItemBody := []byte(`{"name":"Bananas","amount_to_buy":2,"prefferd_supermarket_id":` + fmt.Sprintf("%d", smT2.ID) + `}`)
	badItemReq := httptest.NewRequest(http.MethodPost, "/lists/1/items", bytes.NewReader(badItemBody))
	badItemReq.SetPathValue("listID", "1")
	badItemReq = badItemReq.WithContext(contextWithUser(badItemReq.Context(), 2, 1))
	badItemRec := httptest.NewRecorder()

	apiInstance.CreateItemHandler(badItemRec, badItemReq)
	if badItemRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for cross-tenant preferred supermarket assignment, got %d", badItemRec.Code)
	}
}

func TestConcurrentToggle(t *testing.T) {
	apiInstance, queries, cleanup := setupTestAPI(t)
	defer cleanup()

	// 1. Create a list and item (starts ticked = false)
	list, err := queries.CreateShoppingList(context.Background(), store.CreateShoppingListParams{
		TenantID:  1,
		Name:      "Groceries",
		CreatedBy: 1,
	})
	if err != nil {
		t.Fatalf("failed to create list: %v", err)
	}

	item, err := queries.CreateItem(context.Background(), store.CreateItemParams{
		ID:          list.ID,
		TenantID:    1,
		Name:        "Milk",
		AmountToBuy: 1,
		CreatedBy:   1,
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}
	if item.Ticked {
		t.Fatalf("item should start unticked (false)")
	}

	// 2. Simulate 2 concurrent users who both press "toggle" at the exact same instant
	const numUsers = 2
	var wg sync.WaitGroup
	startBarrier := make(chan struct{}) // synchronizes start so both hit the API simultaneously
	responses := make([]store.ShoppingListItem, numUsers)
	statusCodes := make([]int, numUsers)

	for i := range numUsers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/items/%d/toggle", item.ID), nil)
			req.SetPathValue("id", fmt.Sprintf("%d", item.ID))
			req = req.WithContext(contextWithUser(req.Context(), int64(idx+2), 1))
			rec := httptest.NewRecorder()

			<-startBarrier // wait until all goroutines are ready to fire

			apiInstance.ToggleItemHandler(rec, req)

			statusCodes[idx] = rec.Code
			if rec.Code == http.StatusOK {
				_ = json.Unmarshal(rec.Body.Bytes(), &responses[idx])
			}
		}(i)
	}

	// Release both requests at the same instant
	close(startBarrier)
	wg.Wait()

	// 3. Verify both requests succeeded with 200 OK
	for i, code := range statusCodes {
		if code != http.StatusOK {
			t.Fatalf("request %d failed with status %d", i, code)
		}
	}

	// 4. Inspect what each user got back:
	// Exactly one user saw 'true', and the other user saw 'false'!
	t.Logf("User 0 received response: ticked = %v", responses[0].Ticked)
	t.Logf("User 1 received response: ticked = %v", responses[1].Ticked)

	if responses[0].Ticked == responses[1].Ticked {
		t.Fatalf("expected one user to get true and one to get false, but both got %v", responses[0].Ticked)
	}

	// 5. Verify database end state:
	// Because it was toggled twice (false -> true -> false), the item ended up UNTICKED (false)!
	finalItems, err := queries.ListItemsByListID(context.Background(), store.ListItemsByListIDParams{
		ShoppingListID: list.ID,
		TenantID:       1,
	})
	if err != nil || len(finalItems) == 0 {
		t.Fatalf("failed to fetch final item from db")
	}

	t.Logf("Final item state in DB: ticked = %v", finalItems[0].Ticked)
	if finalItems[0].Ticked != false {
		t.Fatalf("expected final state to be false (toggled twice), got %v", finalItems[0].Ticked)
	}
}
