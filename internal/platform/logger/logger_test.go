package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_Redaction(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("info", &buf)

	log.Info("user action",
		slog.String("username", "budi"),
		slog.String("password", "supersecret123"),
		slog.String("pin", "123456"),
		slog.String("token", "eyJh..."),
	)

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "budi", entry["username"])
	assert.Equal(t, "[REDACTED]", entry["password"])
	assert.Equal(t, "[REDACTED]", entry["pin"])
	assert.Equal(t, "[REDACTED]", entry["token"])
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	baseLog := logger.New("info", &buf)

	ctx := context.WithValue(context.Background(), logger.RequestIDKey, "test-req-123")
	reqLog := logger.WithContext(ctx, baseLog)

	reqLog.Info("request processed")

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "test-req-123", entry["request_id"])
}
