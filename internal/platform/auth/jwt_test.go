package auth_test

import (
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_GenerateAndParse(t *testing.T) {
	secret := []byte("secret-key-32-bytes-minimum-value!")
	userID := uuid.New()
	bankSampahID := uuid.New()
	username := "pengelola_rw05"
	role := "pengelola"

	token, err := auth.GenerateAccessToken(secret, userID, bankSampahID, username, role, 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := auth.ParseAccessToken(secret, token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, bankSampahID, claims.BankSampahID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, role, claims.Role)
}

func TestJWT_ExpiredToken(t *testing.T) {
	secret := []byte("secret-key-32-bytes-minimum-value!")
	userID := uuid.New()
	bankSampahID := uuid.New()

	token, err := auth.GenerateAccessToken(secret, userID, bankSampahID, "user", "nasabah", -1*time.Minute)
	require.NoError(t, err)

	claims, err := auth.ParseAccessToken(secret, token)
	assert.ErrorIs(t, err, auth.ErrExpiredToken)
	assert.Nil(t, claims)
}

func TestJWT_InvalidSignature(t *testing.T) {
	secret1 := []byte("secret-key-32-bytes-minimum-value1")
	secret2 := []byte("secret-key-32-bytes-minimum-value2")
	userID := uuid.New()
	bankSampahID := uuid.New()

	token, err := auth.GenerateAccessToken(secret1, userID, bankSampahID, "user", "pengelola", 15*time.Minute)
	require.NoError(t, err)

	claims, err := auth.ParseAccessToken(secret2, token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestToken_RandomAndHash(t *testing.T) {
	tok1, err := auth.GenerateRandomToken()
	require.NoError(t, err)
	assert.NotEmpty(t, tok1)

	tok2, err := auth.GenerateRandomToken()
	require.NoError(t, err)
	assert.NotEqual(t, tok1, tok2)

	h1 := auth.HashToken(tok1)
	h2 := auth.HashToken(tok1)
	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, auth.HashToken(tok2))
}
