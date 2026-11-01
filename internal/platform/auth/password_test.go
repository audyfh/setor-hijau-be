package auth_test

import (
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/platform/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPassword_HashAndVerify(t *testing.T) {
	// Use lighter params for test speed
	testParams := &auth.Argon2Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	password := "RahasiaSuperKuat123"

	encodedHash, err := auth.HashPasswordWithParams(password, testParams)
	require.NoError(t, err)
	assert.NotEmpty(t, encodedHash)
	assert.Contains(t, encodedHash, "$argon2id$")

	// Correct password
	match, err := auth.VerifyPassword(password, encodedHash)
	require.NoError(t, err)
	assert.True(t, match)

	// Wrong password
	match, err = auth.VerifyPassword("SalahPassword", encodedHash)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestPassword_InvalidHash(t *testing.T) {
	match, err := auth.VerifyPassword("password", "invalid_hash_string")
	assert.Error(t, err)
	assert.False(t, match)
}
