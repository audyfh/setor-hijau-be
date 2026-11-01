package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/platform/config"
	"github.com/audyfh/setor-hijau-be/internal/platform/logger"
	"github.com/audyfh/setor-hijau-be/internal/platform/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPinger struct {
	pingErr error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.pingErr
}

func setupTestRouter(pinger *mockPinger) *httptest.Server {
	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "debug",
		DatabaseURL: "postgres://mock",
	}
	log := logger.New("debug", io.Discard)
	engine := router.New(router.Options{
		Config: cfg,
		Logger: log,
		DB:     pinger,
	})
	return httptest.NewServer(engine)
}

func TestHealthz(t *testing.T) {
	ts := setupTestRouter(&mockPinger{})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestReadyz_Success(t *testing.T) {
	ts := setupTestRouter(&mockPinger{pingErr: nil})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ready", body["status"])
	assert.Equal(t, "connected", body["database"])
}

func TestReadyz_DatabaseDown(t *testing.T) {
	ts := setupTestRouter(&mockPinger{pingErr: errors.New("connection refused")})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "unavailable", body["status"])
	assert.Equal(t, "disconnected", body["database"])
	assert.Equal(t, "connection refused", body["error"])
}

func TestRequestID_PreservedIfProvided(t *testing.T) {
	ts := setupTestRouter(&mockPinger{})
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/healthz", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "custom-client-req-id-42")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "custom-client-req-id-42", resp.Header.Get("X-Request-ID"))
}
