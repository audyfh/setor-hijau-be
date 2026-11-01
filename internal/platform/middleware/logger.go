package middleware

import (
	"log/slog"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// StructuredLogger logs HTTP request details using structured slog JSON logger.
func StructuredLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		reqID := c.GetString(string(logger.RequestIDKey))

		attrs := []slog.Attr{
			slog.String("request_id", reqID),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Duration("latency", latency),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("client_ip", clientIP),
		}

		if rawQuery != "" {
			attrs = append(attrs, slog.String("query", rawQuery))
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("gin_errors", c.Errors.String()))
		}

		level := slog.LevelInfo
		if statusCode >= 500 {
			level = slog.LevelError
		} else if statusCode >= 400 {
			level = slog.LevelWarn
		}

		log.LogAttrs(c.Request.Context(), level, "http_request", attrs...)
	}
}
