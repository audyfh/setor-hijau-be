package docs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/platform/docs"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDocsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	docs.RegisterRoutes(r)

	reqSpec := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	wSpec := httptest.NewRecorder()
	r.ServeHTTP(wSpec, reqSpec)
	assert.Equal(t, http.StatusOK, wSpec.Code)
	assert.Contains(t, wSpec.Header().Get("Content-Type"), "application/yaml")
	assert.Contains(t, wSpec.Body.String(), "openapi: 3.0.3")

	reqUI := httptest.NewRequest(http.MethodGet, "/docs", nil)
	wUI := httptest.NewRecorder()
	r.ServeHTTP(wUI, reqUI)
	assert.Equal(t, http.StatusOK, wUI.Code)
	assert.Contains(t, wUI.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, wUI.Body.String(), "/openapi.yaml")
}
