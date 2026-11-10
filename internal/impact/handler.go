package impact

import (
	"net/http"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc) {
	// Public routes (no authentication required)
	publicGroup := rg.Group("/public")
	{
		publicGroup.GET("/methods", h.GetPublicMethods)
		publicGroup.GET("/impact", h.GetPlatformNetImpact)
		publicGroup.GET("/impact/tenants/:code", h.GetTenantNetImpactByCode)
		publicGroup.POST("/impact/telemetry", h.RecordTelemetry)
	}

	// Authenticated tenant impact route
	impactGroup := rg.Group("/impact")
	impactGroup.Use(authMid)
	{
		impactGroup.GET("/tenant", h.GetTenantNetImpact)
	}
}

func (h *Handler) GetPublicMethods(c *gin.Context) {
	resp, err := h.service.GetPublicMethodsData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data metodologi & sumber."})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPlatformNetImpact(c *gin.Context) {
	resp, err := h.service.GetPlatformNetImpact(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil metrik dampak bersih platform."})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetTenantNetImpactByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode bank sampah harus diisi."})
		return
	}

	resp, err := h.service.GetTenantNetImpactByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan."})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetTenantNetImpact(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	resp, err := h.service.GetTenantNetImpact(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil metrik dampak bersih tenant."})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RecordTelemetry(c *gin.Context) {
	var req RecordTelemetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.RecordTelemetry(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat telemetri jejak digital."})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
