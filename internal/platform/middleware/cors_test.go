package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupCORSRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS(origins))
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestCORS_ExplicitOrigin(t *testing.T) {
	r := setupCORSRouter([]string{"https://setor-hijau.vercel.app"})

	// Allowed origin preflight
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://setor-hijau.vercel.app")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://setor-hijau.vercel.app", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))

	// Disallowed origin preflight
	reqBad := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	reqBad.Header.Set("Origin", "https://evil.example.com")
	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, reqBad)

	assert.Equal(t, http.StatusForbidden, wBad.Code)
	assert.Empty(t, wBad.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_WildcardOrigin(t *testing.T) {
	r := setupCORSRouter([]string{"*"})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}
