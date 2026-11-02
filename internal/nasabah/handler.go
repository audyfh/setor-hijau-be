package nasabah

import (
	"errors"
	"net/http"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler processes HTTP requests for Nasabah management.
type Handler struct {
	service Service
}

// NewHandler creates a new Nasabah HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers endpoints for Nasabah under /api/v1.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc) {
	nasabahGroup := rg.Group("/nasabah")
	nasabahGroup.Use(authMid)
	{
		// Pengelola operations
		pengelolaOnly := nasabahGroup.Group("")
		pengelolaOnly.Use(middleware.RequireRole("pengelola"))
		{
			pengelolaOnly.POST("", h.CreateNasabah)
			pengelolaOnly.GET("", h.ListNasabah)
			pengelolaOnly.POST("/lookup-qr", h.LookupQR)
			pengelolaOnly.GET("/:id", h.GetNasabah)
			pengelolaOnly.PUT("/:id", h.UpdateNasabah)
			pengelolaOnly.POST("/:id/regenerate-qr", h.RegenerateQR)
			pengelolaOnly.POST("/:id/reset-password", h.ResetPassword)
			pengelolaOnly.POST("/:id/reset-pin", h.ResetPIN)
		}

		// Nasabah self-service operations
		nasabahOnly := nasabahGroup.Group("")
		nasabahOnly.Use(middleware.RequireRole("nasabah"))
		{
			nasabahOnly.GET("/me", h.GetMe)
			nasabahOnly.PUT("/me/pin", h.SetPINSelf)
		}
	}
}

// CreateNasabah handles nasabah registration by pengelola.
func (h *Handler) CreateNasabah(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	pengelolaID, _ := middleware.GetUserID(c.Request.Context())

	var req CreateNasabahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data input tidak valid: " + err.Error()})
		return
	}

	resp, err := h.service.CreateNasabah(c.Request.Context(), pengelolaID, tenantID, req)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) || errors.Is(err, ErrEmailTaken) || errors.Is(err, ErrInvalidPINFormat) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan nasabah: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// ListNasabah handles listing of nasabah for pengelola.
func (h *Handler) ListNasabah(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	var query ListNasabahQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter tidak valid."})
		return
	}

	resp, err := h.service.ListNasabah(c.Request.Context(), tenantID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar nasabah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetNasabah handles getting nasabah detail by ID.
func (h *Handler) GetNasabah(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid."})
		return
	}

	resp, err := h.service.GetNasabah(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail nasabah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateNasabah handles updating nasabah info by pengelola.
func (h *Handler) UpdateNasabah(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid."})
		return
	}

	var req UpdateNasabahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data input tidak valid: " + err.Error()})
		return
	}

	resp, err := h.service.UpdateNasabah(c.Request.Context(), id, tenantID, req)
	if err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui nasabah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RegenerateQR handles generating a new QR token for a nasabah.
func (h *Handler) RegenerateQR(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid."})
		return
	}

	resp, err := h.service.RegenerateQR(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui token QR."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ResetPassword handles password reset for nasabah by pengelola.
func (h *Handler) ResetPassword(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid."})
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kata sandi baru minimal 8 karakter."})
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), id, tenantID, req.NewPassword); err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset kata sandi nasabah."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kata sandi nasabah berhasil direset."})
}

// ResetPIN handles PIN reset for nasabah by pengelola.
func (h *Handler) ResetPIN(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid."})
		return
	}

	var req ResetPINRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PIN harus berupa 6 digit angka."})
		return
	}

	if err := h.service.ResetPIN(c.Request.Context(), id, tenantID, req.PIN); err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrInvalidPINFormat) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset PIN nasabah."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PIN nasabah berhasil direset."})
}

// LookupQR handles Pengelola scanning a QR token to identify a Nasabah within their own tenant.
func (h *Handler) LookupQR(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	var req LookupQRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token QR wajib disertakan."})
		return
	}

	nasabah, err := h.service.LookupQR(c.Request.Context(), tenantID, req.QRToken)
	if err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah dengan token QR tersebut tidak ditemukan di bank sampah ini."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencari nasabah."})
		return
	}

	c.JSON(http.StatusOK, nasabah)
}

// GetMe handles authenticated nasabah viewing their own profile.
func (h *Handler) GetMe(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	userID, ok := middleware.GetUserID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas pengguna tidak ditemukan."})
		return
	}

	nasabah, err := h.service.GetNasabah(c.Request.Context(), userID, tenantID)
	if err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profil nasabah tidak ditemukan."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil profil nasabah."})
		return
	}

	c.JSON(http.StatusOK, nasabah)
}

// SetPINSelf handles authenticated nasabah setting or updating their PIN.
func (h *Handler) SetPINSelf(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	userID, ok := middleware.GetUserID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas pengguna tidak ditemukan."})
		return
	}

	var req ResetPINRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PIN harus berupa 6 digit angka."})
		return
	}

	if err := h.service.ResetPIN(c.Request.Context(), userID, tenantID, req.PIN); err != nil {
		if errors.Is(err, ErrInvalidPINFormat) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui PIN."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PIN berhasil disimpan."})
}
