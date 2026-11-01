package config_test

import (
	"os"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Load_Success(t *testing.T) {
	// Setup environment
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	t.Setenv("PORT", "9090")
	t.Setenv("ENV", "production")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("JWT_SECRET", "super-secret-jwt-key")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "postgres://user:pass@localhost:5432/testdb?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "super-secret-jwt-key", cfg.JWTSecret)
}

func TestConfig_Load_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("JWT_SECRET")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.NotEmpty(t, cfg.JWTSecret)
}

func TestConfig_Load_MissingJWTSecretInProduction(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	t.Setenv("ENV", "production")
	os.Unsetenv("JWT_SECRET")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "missing required environment variable: JWT_SECRET in production")
}

func TestConfig_Load_MissingDatabaseURL(t *testing.T) {
	os.Unsetenv("DATABASE_URL")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "missing required environment variable: DATABASE_URL")
}
