// Package auth is going to be used for implementing authentication for this server
package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int64 `json:"-"`
	TenantID int64 `json:"tenant_id"`
}

func (c Claims) GetSubjectID() int64 {
	return c.UserID
}

type jwtClaims struct {
	Claims
	jwt.RegisteredClaims
}

// GenerateToken returns a signed JWT token in string form, signed by a secretKey
func GenerateToken(cc Claims, secretKey []byte) (string, error) {
	claims := jwtClaims{
		Claims:    cc,
		Subject:   strconv.FormatInt(cc.UserID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func ValidateToken(tokenString string, secretKey []byte) (*Claims, error) {
	// 1. Parse into &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	// 2. Type-assert to *jwtClaims
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("invalid token: missing sub claim")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid sub claim: %w", err)
	}
	claims.UserID = userID

	// 3. Return the clean domain Claims!
	return &claims.Claims, nil
}
