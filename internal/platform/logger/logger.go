package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// New creates and returns a structured slog.Logger configured with JSON output
// and the specified log level.
func New(levelStr string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}

	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Redact sensitive fields if encountered
			key := strings.ToLower(a.Key)
			if key == "password" || key == "pin" || key == "token" || key == "authorization" || key == "secret" {
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	})

	return slog.New(handler)
}

// WithContext returns a logger that includes contextual attributes like request_id.
func WithContext(ctx context.Context, log *slog.Logger) *slog.Logger {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return log.With(slog.String("request_id", reqID))
	}
	return log
}
