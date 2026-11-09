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

func TestSupermarketsTenantSafety(t *testing.T) {
	apiInstance, queries, cleanup := setupTestAPI(t)
	defer cleanup()

	// 1. Create a supermarket in Tenant 1 (Alice Admin)
	body := []byte(`{"name":"Organic Market T1","latitude":32.0853,"longitude":34.7818}`)
	createReq := httptest.NewRequest(http.MethodPost, "/supermarkets", bytes.NewReader(body))
	createReq = createReq.WithContext(contextWithUser(createReq.Context(), 2, 1))
	createRec := httptest.NewRecorder()

	apiInstance.CreateSupermarketHandler(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for supermarket creation, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var created store.Supermarket
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode supermarket: %v", err)
	}
	if created.TenantID != 1 || created.Name != "Organic Market T1" {
		t.Fatalf("unexpected supermarket created: %+v", created)
	}

	// 2. Also create a supermarket in Tenant 2 directly via store
	sm2, err := queries.CreateSupermarket(context.Background(), store.CreateSupermarketParams{
		TenantID:  2,
		Name:      "Supermarket T2",
		Latitude:  40.7128,
		Longitude: -74.0060,
	})
	if err != nil {
		t.Fatalf("failed to create tenant 2 supermarket: %v", err)
	}

	// 3. List Supermarkets as Tenant 1 -> Should ONLY see Tenant 1 supermarket
	listReq := httptest.NewRequest(http.MethodGet, "/supermarkets", nil)
	listReq = listReq.WithContext(contextWithUser(listReq.Context(), 2, 1))
	listRec := httptest.NewRecorder()

	apiInstance.ListSupermarketsHandler(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRec.Code)
	}

	var listT1 []store.Supermarket
	if err := json.Unmarshal(listRec.Body.Bytes(), &listT1); err != nil {
		t.Fatalf("failed to unmarshal list: %v", err)
	}
	if len(listT1) != 1 || listT1[0].ID != created.ID {
		t.Fatalf("expected 1 supermarket for Tenant 1, got %d", len(listT1))
	}

	// 4. Get Supermarket from another tenant -> MUST return 404
	getCrossReq := httptest.NewRequest(http.MethodGet, "/supermarkets/2", nil)
	getCrossReq.SetPathValue("id", "2") // sm2 belongs to Tenant 2
	getCrossReq = getCrossReq.WithContext(contextWithUser(getCrossReq.Context(), 2, 1)) // caller is Tenant 1
	getCrossRec := httptest.NewRecorder()

	apiInstance.GetSupermarketHandler(getCrossRec, getCrossReq)
	if getCrossRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant supermarket get, got %d", getCrossRec.Code)
	}

	// 5. Update Supermarket as owner tenant -> 200
	updateBody := []byte(`{"name":"Organic Market T1 - Updated"}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/supermarkets/1", bytes.NewReader(updateBody))
	updateReq.SetPathValue("id", "1")
	updateReq = updateReq.WithContext(contextWithUser(updateReq.Context(), 2, 1))
	updateRec := httptest.NewRecorder()

	apiInstance.UpdateSupermarketHandler(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for update, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	// 6. Cross-tenant update attempt -> MUST return 404
	updateCrossReq := httptest.NewRequest(http.MethodPut, "/supermarkets/2", bytes.NewReader(updateBody))
	updateCrossReq.SetPathValue("id", "2")
	updateCrossReq = updateCrossReq.WithContext(contextWithUser(updateCrossReq.Context(), 2, 1))
	updateCrossRec := httptest.NewRecorder()

	apiInstance.UpdateSupermarketHandler(updateCrossRec, updateCrossReq)
	if updateCrossRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant update, got %d", updateCrossRec.Code)
	}

	// 7. Cross-tenant delete attempt -> MUST return 404
	deleteCrossReq := httptest.NewRequest(http.MethodDelete, "/supermarkets/2", nil)
	deleteCrossReq.SetPathValue("id", "2")
	deleteCrossReq = deleteCrossReq.WithContext(contextWithUser(deleteCrossReq.Context(), 2, 1))
	deleteCrossRec := httptest.NewRecorder()

	apiInstance.DeleteSupermarketHandler(deleteCrossRec, deleteCrossReq)
	if deleteCrossRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant delete, got %d", deleteCrossRec.Code)
	}

	// 8. Delete own supermarket -> 200
	deleteReq := httptest.NewRequest(http.MethodDelete, "/supermarkets/1", nil)
	deleteReq.SetPathValue("id", "1")
	deleteReq = deleteReq.WithContext(contextWithUser(deleteReq.Context(), 2, 1))
	deleteRec := httptest.NewRecorder()

	apiInstance.DeleteSupermarketHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for delete, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	_ = sm2
}
