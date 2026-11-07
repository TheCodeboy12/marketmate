package auth_test

import (
	"fmt"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"rbac-testing/internal/auth"
)

func TestGenerateToken(t *testing.T) {
	mockedKey := "supersecretkey"
	tests := []struct {
		name string
		cc   auth.Claims
	}{
		{
			name: "generating RFC compliant token with sub = userID",
			cc: auth.Claims{
				UserID:   42,
				TenantID: 7,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString, err := auth.GenerateToken(tt.cc, []byte(mockedKey))
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}

			// 1. Verify standard ValidateToken recovers UserID and TenantID
			claims, err := auth.ValidateToken(tokenString, []byte(mockedKey))
			if err != nil {
				t.Fatalf("failed to validate token: %v", err)
			}
			if claims.UserID != tt.cc.UserID {
				t.Errorf("expected UserID %d, got %d", tt.cc.UserID, claims.UserID)
			}
			if claims.TenantID != tt.cc.TenantID {
				t.Errorf("expected TenantID %d, got %d", tt.cc.TenantID, claims.TenantID)
			}
			if claims.GetSubjectID() != tt.cc.UserID {
				t.Errorf("expected GetSubjectID %d, got %d", tt.cc.UserID, claims.GetSubjectID())
			}

			// 2. Inspect raw JWT payload to verify RFC 7519 sub compliance
			parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				return []byte(mockedKey), nil
			})
			if err != nil {
				t.Fatalf("failed to parse raw token: %v", err)
			}

			mapClaims, ok := parsedToken.Claims.(jwt.MapClaims)
			if !ok {
				t.Fatalf("expected jwt.MapClaims, got %T", parsedToken.Claims)
			}

			// Verify 'sub' claim exists and matches StringOrURI of UserID
			subVal, hasSub := mapClaims["sub"]
			if !hasSub {
				t.Errorf("expected 'sub' claim in JWT payload, got none")
			}
			expectedSub := fmt.Sprintf("%d", tt.cc.UserID)
			if subVal != expectedSub {
				t.Errorf("expected sub to be %q, got %v", expectedSub, subVal)
			}

			// Verify old non-standard 'user_id' is NOT in payload
			if _, hasUserID := mapClaims["user_id"]; hasUserID {
				t.Errorf("expected 'user_id' to be omitted from payload, but was present")
			}

			t.Logf("JWT payload: %+v", mapClaims)
		})
	}
}

func TestValidateToken_InvalidSub(t *testing.T) {
	mockedKey := []byte("supersecretkey")

	// Token with non-numeric sub
	nonNumericClaims := jwt.MapClaims{
		"sub":       "not-a-number",
		"tenant_id": 1,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, nonNumericClaims)
	signed, err := token.SignedString(mockedKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = auth.ValidateToken(signed, mockedKey)
	if err == nil {
		t.Fatalf("expected error validating token with non-numeric sub, got nil")
	}

	// Token without sub
	missingSubClaims := jwt.MapClaims{
		"tenant_id": 1,
	}
	tokenMissing := jwt.NewWithClaims(jwt.SigningMethodHS256, missingSubClaims)
	signedMissing, err := tokenMissing.SignedString(mockedKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = auth.ValidateToken(signedMissing, mockedKey)
	if err == nil {
		t.Fatalf("expected error validating token with missing sub, got nil")
	}
}
