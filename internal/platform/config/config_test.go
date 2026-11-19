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
	t.Setenv("DB_MAX_CONNS", "15")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://setor-hijau.vercel.app, http://localhost:3000 ")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "postgres://user:pass@localhost:5432/testdb?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "super-secret-jwt-key", cfg.JWTSecret)
	assert.Equal(t, int32(15), cfg.DBMaxConns)
	assert.Equal(t, []string{"https://setor-hijau.vercel.app", "http://localhost:3000"}, cfg.CORSAllowedOrigins)
}

func TestConfig_Load_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("DB_MAX_CONNS")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, int32(10), cfg.DBMaxConns)
	assert.Equal(t, []string{"*"}, cfg.CORSAllowedOrigins)
	assert.NotEmpty(t, cfg.JWTSecret)
}

func TestConfig_Load_InvalidDBMaxConns(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	t.Setenv("DB_MAX_CONNS", "0")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid DB_MAX_CONNS")
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
