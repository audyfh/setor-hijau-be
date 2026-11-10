package report

import (
	"fmt"
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
	dashboardGroup := rg.Group("/dashboard")
	dashboardGroup.Use(authMid, middleware.RequireRole("pengelola"))
	{
		dashboardGroup.GET("/monthly", h.GetMonthlyDashboard)
	}

	reportsGroup := rg.Group("/reports")
	reportsGroup.Use(authMid, middleware.RequireRole("pengelola"))
	{
		reportsGroup.GET("/monthly/csv", h.ExportMonthlyCSV)
		reportsGroup.GET("/monthly/pdf", h.ExportMonthlyPDF)
	}
}

func (h *Handler) GetMonthlyDashboard(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	monthStr := c.Query("month")
	resp, err := h.service.GetMonthlyDashboard(c.Request.Context(), tenantID, monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ExportMonthlyCSV(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	monthStr := c.Query("month")
	fileBytes, fileName, err := h.service.ExportMonthlyCSV(c.Request.Context(), tenantID, monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", fileBytes)
}

func (h *Handler) ExportMonthlyPDF(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	monthStr := c.Query("month")
	fileBytes, fileName, err := h.service.ExportMonthlyPDF(c.Request.Context(), tenantID, monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Data(http.StatusOK, "application/pdf", fileBytes)
}
