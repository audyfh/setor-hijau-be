package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, os.Stdout)
	log.Info("starting setor-hijau backend",
		slog.String("port", cfg.Port),
		slog.String("env", cfg.Env),
	)

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to initialize database connection pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer database.Close()

	// Initialize queries and services
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
	rateLimiter := middleware.NewRateLimiter(30, 1*time.Minute)

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

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Server run context for graceful shutdown
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("server listening", slog.String("addr", srv.Addr))
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error encountered", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-shutdown:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed, forcing exit", slog.String("error", err.Error()))
			_ = srv.Close()
			os.Exit(1)
		}
		log.Info("server stopped gracefully")
	}
}
