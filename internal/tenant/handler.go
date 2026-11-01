package tenant

import (
	"errors"
	"net/http"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for tenant management.
type Handler struct {
	service Service
}

// NewHandler creates a new tenant Handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts tenant endpoints onto the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc) {
	tenantGroup := rg.Group("/tenants")
	{
		tenantGroup.POST("/register", h.Register)

		// Protected endpoints
		protected := tenantGroup.Group("")
		protected.Use(authMid)
		{
			protected.GET("/current", h.GetCurrentTenant)
		}
	}
}

// Register godoc
// @Summary Registrasi Bank Sampah Baru
// @Description Mendaftarkan Bank Sampah baru beserta akun Pengelola pertama secara atomic.
func (h *Handler) Register(c *gin.Context) {
	var req RegisterBankSampahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Permintaan tidak valid. Harap lengkapi semua data bank sampah dan pengelola.",
		})
		return
	}

	resp, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCodeAlreadyExists) || errors.Is(err, ErrPengelolaConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan bank sampah."})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetCurrentTenant godoc
// @Summary Data Bank Sampah Aktif
// @Description Mengambil profil bank sampah dari sesi tenant yang terautentikasi.
func (h *Handler) GetCurrentTenant(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tenant tidak valid."})
		return
	}

	bs, err := h.service.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan pada server."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bank_sampah": bs})
}
