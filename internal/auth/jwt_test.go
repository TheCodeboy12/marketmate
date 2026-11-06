package auth_test

import (
	"testing"

	"rbac-testing/internal/auth"
)

func TestGenerateToken(t *testing.T) {
	mockedKey := "supersecretkey"
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		cc        auth.Claims
		secretKey []byte
	}{
		// TODO: Add test cases.
		{
			name: "generateing token",
			cc: auth.Claims{
				UserID:   1,
				TenantID: 1,
			},
			secretKey: []byte(mockedKey),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := auth.GenerateToken(tt.cc, tt.secretKey)
			if gotErr != nil {
				t.Error(gotErr)
			}
			val, err := auth.ValidateToken(got, []byte(mockedKey))
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("The jwt that was generated is:\n%s\n%+v", got, val)
		})
	}
}
