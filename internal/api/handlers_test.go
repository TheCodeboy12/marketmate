package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TheCodeboy12/marketmate/internal/api"
	"github.com/TheCodeboy12/marketmate/internal/auth"
	"github.com/TheCodeboy12/marketmate/internal/db"
	"github.com/TheCodeboy12/marketmate/internal/store"
)

func setupTestAPI(t *testing.T) (*api.API, *store.Queries, func()) {
	t.Helper()
	dbFile := fmt.Sprintf("test_api_%s.db", t.Name())
	dbPath := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", dbFile)

	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}

	queries := store.New(sqlDB)
	authorizer := auth.NewAuthorizer(queries)
	jwtSecret := []byte("test-secret-key-12345")
	apiInstance := api.New(queries, jwtSecret, authorizer)

	cleanup := func() {
		sqlDB.Close()
		os.Remove(dbFile)
	}

	return apiInstance, queries, cleanup
}

func contextWithUser(ctx context.Context, userID, tenantID int64) context.Context {
	claims := auth.Claims{
		UserID:   userID,
		TenantID: tenantID,
	}
	return api.ContextWithUser(ctx, &claims)
}

func TestListUsersHandler(t *testing.T) {
	apiInstance, _, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/users?limit=2", nil)
	req = req.WithContext(contextWithUser(req.Context(), 2, 1))
	rec := httptest.NewRecorder()

	apiInstance.ListUsersHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var users []store.UserPublic
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestGetUserHandler(t *testing.T) {
	apiInstance, _, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("existing user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/2", nil)
		req.SetPathValue("userID", "2")
		req = req.WithContext(contextWithUser(req.Context(), 2, 1))
		rec := httptest.NewRecorder()

		apiInstance.GetUserHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var user store.GetUserByIdRow
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if user.ID != 2 || user.Email != "alice@example.com" {
			t.Fatalf("unexpected user data: %+v", user)
		}
	})

	t.Run("nonexistent user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/9999", nil)
		req.SetPathValue("userID", "9999")
		req = req.WithContext(contextWithUser(req.Context(), 2, 1))
		rec := httptest.NewRecorder()

		apiInstance.GetUserHandler(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("invalid user ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
		req.SetPathValue("userID", "abc")
		req = req.WithContext(contextWithUser(req.Context(), 2, 1))
		rec := httptest.NewRecorder()

		apiInstance.GetUserHandler(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestCreateUserHandler(t *testing.T) {
	apiInstance, queries, cleanup := setupTestAPI(t)
	defer cleanup()

	body := []byte(`{
		"tenant_id": 1,
		"role_id": 2,
		"name": "David Tester",
		"email": "david@example.com",
		"password": "mypassword123"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req = req.WithContext(contextWithUser(req.Context(), 2, 1))
	rec := httptest.NewRecorder()

	apiInstance.CreateUserHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created store.CreateUserRow
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if created.Name != "David Tester" || created.Email != "david@example.com" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	// Verify password hash in db works
	u, err := queries.GetUserByEmail(context.Background(), "david@example.com")
	if err != nil {
		t.Fatalf("failed to query new user by email: %v", err)
	}
	if u.PasswordHash == nil || !auth.CheckPassword(*u.PasswordHash, "mypassword123") {
		t.Fatalf("password hash verification failed")
	}
}

func TestUpdateUserHandler(t *testing.T) {
	apiInstance, _, cleanup := setupTestAPI(t)
	defer cleanup()

	// Update user 3 (Bob)
	body := []byte(`{
		"name": "Bobby Updated",
		"email": "bob.updated@example.com"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/users/3", bytes.NewReader(body))
	req.SetPathValue("userID", "3")
	req = req.WithContext(contextWithUser(req.Context(), 2, 1)) // caller is Alice (tenant 1)
	rec := httptest.NewRecorder()

	apiInstance.UpdateUserHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var updated store.UpdateUserRow
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode updated user: %v", err)
	}

	if updated.Name != "Bobby Updated" || updated.Email != "bob.updated@example.com" {
		t.Fatalf("unexpected updated fields: %+v", updated)
	}
}

func TestUpdateUserPasswordHandler(t *testing.T) {
	apiInstance, queries, cleanup := setupTestAPI(t)
	defer cleanup()

	// Update password for user 3 (Bob)
	body := []byte(`{
		"password": "newpassword456"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/users/3/password", bytes.NewReader(body))
	req.SetPathValue("userID", "3")
	req = req.WithContext(contextWithUser(req.Context(), 2, 1)) // caller is Alice (tenant 1)
	rec := httptest.NewRecorder()

	apiInstance.UpdateUserPasswordHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify updated password hash works
	u, err := queries.GetUserByEmail(context.Background(), "bob@example.com")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if u.PasswordHash == nil || !auth.CheckPassword(*u.PasswordHash, "newpassword456") {
		t.Fatalf("password was not updated correctly")
	}
}

func TestDeleteUserHandler(t *testing.T) {
	apiInstance, queries, cleanup := setupTestAPI(t)
	defer cleanup()

	t.Run("prevent deleting own account", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/users/2", nil)
		req.SetPathValue("userID", "2")
		req = req.WithContext(contextWithUser(req.Context(), 2, 1)) // Caller is user 2
		rec := httptest.NewRecorder()

		apiInstance.DeleteUserHandler(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for self-deletion, got %d", rec.Code)
		}
	})

	t.Run("successfully delete another user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/users/4", nil)
		req.SetPathValue("userID", "4")
		req = req.WithContext(contextWithUser(req.Context(), 2, 1)) // Caller is user 2 deleting user 4
		rec := httptest.NewRecorder()

		apiInstance.DeleteUserHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify user 4 is gone
		_, err := queries.GetUserById(context.Background(), store.GetUserByIdParams{
			ID:       4,
			TenantID: 1,
		})
		if err == nil {
			t.Fatalf("expected user 4 to be deleted, but found in db")
		}
	})
}

func TestChiRouterIntegration(t *testing.T) {
	apiInstance, _, cleanup := setupTestAPI(t)
	defer cleanup()

	router := apiInstance.Routes()

	// 1. Unauthorized request without token should fail with 401
	unauthReq := httptest.NewRequest(http.MethodGet, "/users", nil)
	unauthRec := httptest.NewRecorder()
	router.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", unauthRec.Code)
	}

	// 2. Login as Alice Admin to get token
	loginBody := []byte(`{"email":"alice@example.com","password":"password123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login failed: %d: %s", loginRec.Code, loginRec.Body.String())
	}

	var loginResp map[string]string
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	token := loginResp["token"]
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	// 3. GET /me with token
	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /me, got %d: %s", meRec.Code, meRec.Body.String())
	}

	// 4. GET /users/2 through Chi URLParam parameter matching
	userReq := httptest.NewRequest(http.MethodGet, "/users/2", nil)
	userReq.Header.Set("Authorization", "Bearer "+token)
	userRec := httptest.NewRecorder()
	router.ServeHTTP(userRec, userReq)
	if userRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /users/2, got %d: %s", userRec.Code, userRec.Body.String())
	}

	// 5. GET /lists with token
	listsReq := httptest.NewRequest(http.MethodGet, "/lists", nil)
	listsReq.Header.Set("Authorization", "Bearer "+token)
	listsRec := httptest.NewRecorder()
	router.ServeHTTP(listsRec, listsReq)
	if listsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /lists, got %d: %s", listsRec.Code, listsRec.Body.String())
	}
}
