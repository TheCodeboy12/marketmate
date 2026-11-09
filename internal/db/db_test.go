package db_test

import (
	"os"
	"testing"

	"github.com/TheCodeboy12/marketmate/internal/db"
)

func TestInitDB(t *testing.T) {
	testDBPath := "test_app.db?_pragma=foreign_keys(1)"
	defer os.Remove("test_app.db")

	database, err := db.InitDB(testDBPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer database.Close()

	// Verify tenant table has seed data
	var tenantCount int
	err = database.QueryRow("SELECT COUNT(*) FROM tenant").Scan(&tenantCount)
	if err != nil {
		t.Fatalf("failed to query tenant count: %v", err)
	}
	if tenantCount == 0 {
		t.Fatalf("expected tenants to be seeded, got %d", tenantCount)
	}

	// Verify users are seeded
	var userCount int
	err = database.QueryRow("SELECT COUNT(*) FROM user").Scan(&userCount)
	if err != nil {
		t.Fatalf("failed to query user count: %v", err)
	}
	if userCount == 0 {
		t.Fatalf("expected users to be seeded, got %d", userCount)
	}
}
