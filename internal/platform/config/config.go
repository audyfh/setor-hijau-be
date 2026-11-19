package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all platform configurations loaded from environment variables.
type Config struct {
	Port               string
	Env                string
	LogLevel           string
	DatabaseURL        string
	DBMaxConns         int32
	JWTSecret          string
	CORSAllowedOrigins []string
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

	dbMaxConns := int32(10)
	if rawMaxConns := strings.TrimSpace(os.Getenv("DB_MAX_CONNS")); rawMaxConns != "" {
		parsed, err := strconv.Atoi(rawMaxConns)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("invalid DB_MAX_CONNS value %q: must be a positive integer", rawMaxConns)
		}
		dbMaxConns = int32(parsed)
	}

	var corsOrigins []string
	if rawOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); rawOrigins != "" {
		for _, origin := range strings.Split(rawOrigins, ",") {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				corsOrigins = append(corsOrigins, trimmed)
			}
		}
	} else if env != "production" {
		corsOrigins = []string{"*"}
	}

	return &Config{
		Port:               port,
		Env:                env,
		LogLevel:           logLevel,
		DatabaseURL:        databaseURL,
		DBMaxConns:         dbMaxConns,
		JWTSecret:          jwtSecret,
		CORSAllowedOrigins: corsOrigins,
	}, nil
}
