package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/auth"
	"github.com/audyfh/setor-hijau-be/internal/carbon"
	"github.com/audyfh/setor-hijau-be/internal/cashbook"
	"github.com/audyfh/setor-hijau-be/internal/deposit"
	"github.com/audyfh/setor-hijau-be/internal/impact"
	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/audyfh/setor-hijau-be/internal/platform/config"
	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/audyfh/setor-hijau-be/internal/platform/logger"
	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/audyfh/setor-hijau-be/internal/platform/router"
	"github.com/audyfh/setor-hijau-be/internal/report"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// TestHarness bundles test infrastructure for integration tests.
type TestHarness struct {
	DB              *db.DB
	Server          *httptest.Server
	Config          *config.Config
	AuthService     auth.Service
	CarbonService   carbon.Service
	CashbookService cashbook.Service
	ReportService   report.Service
	ImpactService   impact.Service
}

// SetupTestHarness initializes a connected database, runs migrations, and starts a test server.
// If the database is unreachable, it skips the calling test.
func SetupTestHarness(t *testing.T) *TestHarness {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/setor_hijau?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping integration test: database unavailable at %s: %v", dbURL, err)
		return nil
	}

	// Apply migrations
	migrationsPath, err := filepath.Abs("../../db/migrations")
	if err != nil {
		t.Fatalf("failed to locate migrations path: %v", err)
	}

	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(migrationsPath))
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		t.Fatalf("failed to initialize migrate: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migration up failed in test setup: %v", err)
	}

	// Clean tables before running test
	cleanTables(t, database)

	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "debug",
		DatabaseURL: dbURL,
		JWTSecret:   "test-secret-at-least-32-chars-length!",
	}

	log := logger.New("debug", io.Discard)
	queries := sqlc.New(database.Pool)

	authRepo := auth.NewRepository(queries)
	emailSender := auth.NewLogEmailSender(log)
	authService := auth.NewService(authRepo, emailSender, []byte(cfg.JWTSecret))
	authHandler := auth.NewHandler(authService)

	tenantRepo := tenant.NewRepository(queries)
	tenantService := tenant.NewService(database.Pool, tenantRepo, authService)
	tenantHandler := tenant.NewHandler(tenantService)

	nasabahRepo := nasabah.NewRepository(database.Pool, queries)
	nasabahService := nasabah.NewService(nasabahRepo)
	nasabahHandler := nasabah.NewHandler(nasabahService)

	materialRepo := material.NewRepository(database.Pool, queries)
	materialService := material.NewService(materialRepo)
	materialHandler := material.NewHandler(materialService)

	depositRepo := deposit.NewRepository(database.Pool, queries)
	depositService := deposit.NewService(depositRepo)
	depositHandler := deposit.NewHandler(depositService)

	carbonRepo := carbon.NewRepository(database.Pool, queries)
	carbonService := carbon.NewService(carbonRepo)
	carbonHandler := carbon.NewHandler(carbonService)

	cashbookRepo := cashbook.NewRepository(database.Pool)
	cashbookService := cashbook.NewService(cashbookRepo)
	cashbookHandler := cashbook.NewHandler(cashbookService)

	reportRepo := report.NewRepository(database.Pool)
	reportService := report.NewService(reportRepo)
	reportHandler := report.NewHandler(reportService)

	impactRepo := impact.NewRepository(database.Pool)
	impactService := impact.NewService(impactRepo)
	impactHandler := impact.NewHandler(impactService)

	authMid := middleware.AuthMiddleware([]byte(cfg.JWTSecret))
	rateLimiter := middleware.NewRateLimiter(100, 1*time.Minute)

	engine := router.New(router.Options{
		Config:          cfg,
		Logger:          log,
		DB:              database,
		AuthHandler:     authHandler,
		TenantHandler:   tenantHandler,
		NasabahHandler:  nasabahHandler,
		MaterialHandler: materialHandler,
		DepositHandler:  depositHandler,
		CarbonHandler:   carbonHandler,
		CashbookHandler: cashbookHandler,
		ReportHandler:   reportHandler,
		ImpactHandler:   impactHandler,
		AuthMid:         authMid,
		RateLimiter:     rateLimiter,
	})

	server := httptest.NewServer(engine)

	t.Cleanup(func() {
		server.Close()
		database.Close()
	})

	return &TestHarness{
		DB:              database,
		Server:          server,
		Config:          cfg,
		AuthService:     authService,
		CarbonService:   carbonService,
		CashbookService: cashbookService,
		ReportService:   reportService,
		ImpactService:   impactService,
	}
}

func cleanTables(t *testing.T, database *db.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Pool.Exec(ctx, "TRUNCATE TABLE digital_footprint_measurements, bank_sampah CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate test tables: %v", err)
	}
}
