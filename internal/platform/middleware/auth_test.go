package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/auth"
	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddleware_Success(t *testing.T) {
	secret := []byte("secret-key-32-bytes-minimum-value!")
	userID := uuid.New()
	bankSampahID := uuid.New()

	token, err := auth.GenerateAccessToken(secret, userID, bankSampahID, "audy_pengelola", "pengelola", 15*time.Minute)
	require.NoError(t, err)

	r := gin.New()
	r.Use(middleware.AuthMiddleware(secret))
	r.GET("/protected", func(c *gin.Context) {
		tID, okT := middleware.GetTenantID(c.Request.Context())
		uID, okU := middleware.GetUserID(c.Request.Context())
		role, okR := middleware.GetUserRole(c.Request.Context())
		username, okN := middleware.GetUsername(c.Request.Context())

		assert.True(t, okT)
		assert.True(t, okU)
		assert.True(t, okR)
		assert.True(t, okN)
		assert.Equal(t, bankSampahID, tID)
		assert.Equal(t, userID, uID)
		assert.Equal(t, "pengelola", role)
		assert.Equal(t, "audy_pengelola", username)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingOrInvalidToken(t *testing.T) {
	secret := []byte("secret-key-32-bytes-minimum-value!")
	r := gin.New()
	r.Use(middleware.AuthMiddleware(secret))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Missing header
	req1 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// Malformed header
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "InvalidHeaderFormat")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// Bad token
	req3 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req3.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusUnauthorized, w3.Code)
}

func TestRequireRole(t *testing.T) {
	secret := []byte("secret-key-32-bytes-minimum-value!")
	userID := uuid.New()
	bankSampahID := uuid.New()

	pengelolaToken, err := auth.GenerateAccessToken(secret, userID, bankSampahID, "admin", "pengelola", 15*time.Minute)
	require.NoError(t, err)

	nasabahToken, err := auth.GenerateAccessToken(secret, userID, bankSampahID, "resident", "nasabah", 15*time.Minute)
	require.NoError(t, err)

	r := gin.New()
	r.Use(middleware.AuthMiddleware(secret))
	r.GET("/admin-only", middleware.RequireRole("pengelola"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "admin_granted"})
	})

	// Pengelola accesses admin-only -> allowed
	req1 := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req1.Header.Set("Authorization", "Bearer "+pengelolaToken)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Nasabah accesses admin-only -> forbidden (403)
	req2 := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req2.Header.Set("Authorization", "Bearer "+nasabahToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)
}

func TestRateLimiter(t *testing.T) {
	limiter := middleware.NewRateLimiter(2, 500*time.Millisecond)

	r := gin.New()
	r.Use(middleware.RateLimitMiddleware(limiter))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "pong"})
	})

	// 1st request -> ok
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/ping", nil))
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2nd request -> ok
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/ping", nil))
	assert.Equal(t, http.StatusOK, w2.Code)

	// 3rd request -> 429 Too Many Requests
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/ping", nil))
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)

	var resp map[string]string
	err := json.Unmarshal(w3.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "Terlalu banyak permintaan")
}
