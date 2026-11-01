package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrExpiredToken indicates the token has expired.
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidToken indicates the token is malformed or signature is invalid.
	ErrInvalidToken = errors.New("invalid token")
)

// Claims represents the JWT claims payload for Setor Hijau.
type Claims struct {
	UserID       uuid.UUID `json:"user_id"`
	BankSampahID uuid.UUID `json:"bank_sampah_id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken signs a new JWT access token using HMAC-SHA256.
func GenerateAccessToken(secret []byte, userID, bankSampahID uuid.UUID, username, role string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:       userID,
		BankSampahID: bankSampahID,
		Username:     username,
		Role:         role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "setor-hijau-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signed, nil
}

// ParseAccessToken parses and validates an HMAC-SHA256 signed JWT string.
func ParseAccessToken(secret []byte, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
