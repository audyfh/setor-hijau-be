package auth

import (
	"errors"
	"net/http"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	service Service
}

// NewHandler creates a new auth Handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts auth endpoints onto the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc, rateLimiter *middleware.RateLimiter) {
	authGroup := rg.Group("/auth")
	{
		// Public endpoints protected by rate limiting
		if rateLimiter != nil {
			authGroup.POST("/login", middleware.RateLimitMiddleware(rateLimiter), h.Login)
			authGroup.POST("/forgot-password", middleware.RateLimitMiddleware(rateLimiter), h.ForgotPassword)
		} else {
			authGroup.POST("/login", h.Login)
			authGroup.POST("/forgot-password", h.ForgotPassword)
		}

		authGroup.POST("/refresh", h.RefreshToken)
		authGroup.POST("/logout", h.Logout)
		authGroup.POST("/reset-password", h.ResetPassword)

		// Protected endpoints
		protected := authGroup.Group("")
		protected.Use(authMid)
		{
			protected.GET("/me", h.GetMe)
		}
	}
}

// Login godoc
// @Summary Login Pengelola
// @Description Autentikasi Pengelola menggunakan username/email dan password.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Permintaan tidak valid. Harap periksa identifier dan password.",
		})
		return
	}

	tokenPair, user, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrAccountInactive) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan pada server."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokenPair,
		"user":   user,
	})
}

// RefreshToken godoc
// @Summary Refresh Token Akses
// @Description Menukar refresh token yang valid dengan pasangan token baru (token rotation).
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Permintaan tidak valid. refresh_token diperlukan.",
		})
		return
	}

	tokenPair, err := h.service.RefreshToken(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) || errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Refresh token tidak valid atau telah kedaluwarsa.",
			})
			return
		}
		if errors.Is(err, ErrAccountInactive) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan pada server."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokenPair,
	})
}

// Logout godoc
// @Summary Logout
// @Description Mencabut refresh token pengguna.
func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Permintaan tidak valid. refresh_token diperlukan.",
		})
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan saat logout."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil keluar.",
	})
}

// GetMe godoc
// @Summary Profil Pengguna Aktif
// @Description Mengambil informasi pengguna dan tenant dari sesi aktif.
func (h *Handler) GetMe(c *gin.Context) {
	userID, okUser := middleware.GetUserID(c.Request.Context())
	tenantID, okTenant := middleware.GetTenantID(c.Request.Context())

	if !okUser || !okTenant {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Sesi tidak valid.",
		})
		return
	}

	user, err := h.service.GetMe(c.Request.Context(), userID, tenantID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pengguna tidak ditemukan."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan pada server."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// ForgotPassword godoc
// @Summary Lupa Kata Sandi
// @Description Meminta tautan reset kata sandi untuk akun Pengelola via email.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Permintaan tidak valid. Harap masukkan format email yang benar.",
		})
		return
	}

	_ = h.service.ForgotPassword(c.Request.Context(), req)

	// Always return 200 to prevent user enumeration
	c.JSON(http.StatusOK, gin.H{
		"message": "Jika email terdaftar, tautan pemulihan kata sandi telah dikirimkan.",
	})
}

// ResetPassword godoc
// @Summary Atur Ulang Kata Sandi
// @Description Mengatur ulang kata sandi menggunakan token reset yang valid.
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Permintaan tidak valid. Kata sandi baru minimal 8 karakter.",
		})
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req); err != nil {
		if errors.Is(err, ErrInvalidResetToken) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan pada server."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kata sandi berhasil diperbarui. Silakan login kembali dengan kata sandi baru.",
	})
}
