package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/auth"
	"github.com/audyfh/setor-hijau-be/internal/platform/config"
	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/gin-gonic/gin"
)

// Options holds dependencies required to build the HTTP router.
type Options struct {
	Config        *config.Config
	Logger        *slog.Logger
	DB            db.Pinger
	AuthHandler   *auth.Handler
	TenantHandler *tenant.Handler
	AuthMid       gin.HandlerFunc
	RateLimiter   *middleware.RateLimiter
}

// New constructs and configures the Gin engine.
func New(opts Options) *gin.Engine {
	if opts.Config != nil && opts.Config.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()

	// Middlewares
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	if opts.Logger != nil {
		r.Use(middleware.StructuredLogger(opts.Logger))
	}

	// Liveness probe (process is running)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Readiness probe (database is reachable)
	r.GET("/readyz", func(c *gin.Context) {
		if opts.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unavailable",
				"database": "disconnected",
				"error":    "database client not configured",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := opts.DB.Ping(ctx); err != nil {
			if opts.Logger != nil {
				opts.Logger.ErrorContext(c.Request.Context(), "readiness probe failed", slog.String("error", err.Error()))
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unavailable",
				"database": "disconnected",
				"error":    err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ready",
			"database": "connected",
		})
	})

	// API v1 routes
	if opts.AuthHandler != nil || opts.TenantHandler != nil {
		v1 := r.Group("/api/v1")
		if opts.TenantHandler != nil {
			opts.TenantHandler.RegisterRoutes(v1, opts.AuthMid)
		}
		if opts.AuthHandler != nil {
			opts.AuthHandler.RegisterRoutes(v1, opts.AuthMid, opts.RateLimiter)
		}
	}

	return r
}
