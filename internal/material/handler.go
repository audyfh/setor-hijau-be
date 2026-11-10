package material

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler processes HTTP requests for Material master data.
type Handler struct {
	service Service
}

// NewHandler creates a new Material HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers endpoints for materials under /api/v1.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc) {
	matGroup := rg.Group("/materials")
	matGroup.Use(authMid)
	{
		// Accessible to both Pengelola and Nasabah
		matGroup.GET("", h.ListMaterials)
		matGroup.GET("/:id", h.GetMaterial)

		// Pengelola only
		pengelolaOnly := matGroup.Group("")
		pengelolaOnly.Use(middleware.RequireRole("pengelola"))
		{
			pengelolaOnly.POST("", h.CreateMaterial)
			pengelolaOnly.PUT("/:id", h.UpdateMaterial)
			pengelolaOnly.POST("/:id/prices", h.UpdatePrice)
			pengelolaOnly.PUT("/:id/price", h.UpdatePrice)
			pengelolaOnly.GET("/:id/prices", h.GetPriceHistory)
		}
	}
}

// CreateMaterial handles creation of a new material type by pengelola.
func (h *Handler) CreateMaterial(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	pengelolaID, _ := middleware.GetUserID(c.Request.Context())

	var req CreateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data input tidak valid: " + err.Error()})
		return
	}

	resp, err := h.service.CreateMaterial(c.Request.Context(), pengelolaID, tenantID, req)
	if err != nil {
		if errors.Is(err, ErrMaterialNameTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrInvalidPrice) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan jenis sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// ListMaterials handles listing of materials (active only or all).
func (h *Handler) ListMaterials(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	activeOnlyStr := c.DefaultQuery("active_only", "false")
	activeOnly, _ := strconv.ParseBool(activeOnlyStr)

	resp, err := h.service.ListMaterials(c.Request.Context(), tenantID, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar jenis sampah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetMaterial handles getting details of a material type by ID.
func (h *Handler) GetMaterial(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID jenis sampah tidak valid."})
		return
	}

	resp, err := h.service.GetMaterial(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, ErrMaterialNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jenis sampah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateMaterial handles updating metadata for a material type.
func (h *Handler) UpdateMaterial(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID jenis sampah tidak valid."})
		return
	}

	var req UpdateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data input tidak valid: " + err.Error()})
		return
	}

	resp, err := h.service.UpdateMaterial(c.Request.Context(), id, tenantID, req)
	if err != nil {
		if errors.Is(err, ErrMaterialNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrMaterialNameTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui jenis sampah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdatePrice handles updating the active price of a material type.
func (h *Handler) UpdatePrice(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	pengelolaID, _ := middleware.GetUserID(c.Request.Context())

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID jenis sampah tidak valid."})
		return
	}

	var req UpdatePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data input tidak valid: " + err.Error()})
		return
	}

	resp, err := h.service.UpdatePrice(c.Request.Context(), pengelolaID, id, tenantID, req)
	if err != nil {
		if errors.Is(err, ErrMaterialNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrInvalidPrice) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui harga jenis sampah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPriceHistory handles fetching the chronological price history of a material.
func (h *Handler) GetPriceHistory(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID jenis sampah tidak valid."})
		return
	}

	resp, err := h.service.GetPriceHistory(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, ErrMaterialNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat harga."})
		return
	}

	c.JSON(http.StatusOK, resp)
}
