package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all platform configurations loaded from environment variables.
type Config struct {
	Port        string
	Env         string
	LogLevel    string
	DatabaseURL string
	JWTSecret   string
}

// Load loads configuration from environment variables, setting sensible defaults
// and validating required fields.
func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("missing required environment variable: DATABASE_URL")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if env == "production" {
			return nil, fmt.Errorf("missing required environment variable: JWT_SECRET in production")
		}
		jwtSecret = "dev-secret-key-at-least-32-chars-long-setor-hijau"
	}

	return &Config{
		Port:        port,
		Env:         env,
		LogLevel:    logLevel,
		DatabaseURL: databaseURL,
		JWTSecret:   jwtSecret,
	}, nil
}
