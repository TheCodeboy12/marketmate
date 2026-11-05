package auth_test

import (
	"testing"

	"rbac-testing/internal/auth"
)

func TestPasswordHash(t *testing.T) {
	password := "password123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if !auth.CheckPassword(hash, password) {
		t.Fatal("expected password to match")
	}

	t.Logf("Hash for '%s': %s", password, hash)
}
