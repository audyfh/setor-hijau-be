package deposit

import (
	"errors"
	"net/http"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler processes HTTP requests for deposit recording and balances.
type Handler struct {
	service Service
}

// NewHandler creates a new Deposit HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers endpoints for Deposits and Nasabah Balances.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc) {
	depositsGroup := rg.Group("/deposits")
	depositsGroup.Use(authMid)
	{
		// Pengelola operations
		pengelolaOnly := depositsGroup.Group("")
		pengelolaOnly.Use(middleware.RequireRole("pengelola"))
		{
			pengelolaOnly.POST("", h.CreateDeposit)
			pengelolaOnly.POST("/sync", h.BatchSync)
		}

		// Operations accessible to both Pengelola and Nasabah (access scoped in service)
		depositsGroup.GET("", h.ListDeposits)
		depositsGroup.GET("/:id", h.GetDeposit)
		depositsGroup.GET("/:id/receipt", h.GetReceipt)
	}

	nasabahGroup := rg.Group("/nasabah")
	nasabahGroup.Use(authMid)
	{
		// Nasabah self-service balance
		nasabahGroup.GET("/me/balance", middleware.RequireRole("nasabah"), h.GetMyBalance)
		// Pengelola check nasabah balance
		nasabahGroup.GET("/:id/balance", middleware.RequireRole("pengelola"), h.GetNasabahBalance)
	}
}

// CreateDeposit handles single deposit creation by Pengelola.
func (h *Handler) CreateDeposit(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	recordedBy, ok := middleware.GetUserID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas pengelola tidak ditemukan."})
		return
	}

	var req CreateDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data input setoran tidak valid: " + err.Error()})
		return
	}

	resp, isReplay, err := h.service.RecordDeposit(c.Request.Context(), tenantID, recordedBy, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIdempotencyKey), errors.Is(err, ErrInvalidClientTime),
			errors.Is(err, ErrEmptyDepositLines), errors.Is(err, ErrInvalidWeight):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, ErrNasabahNotFound), errors.Is(err, ErrMaterialNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ErrMaterialPriceNotSet):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat setoran: " + err.Error()})
		}
		return
	}

	if isReplay {
		c.JSON(http.StatusOK, resp)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// BatchSync handles offline batch sync submissions by Pengelola.
func (h *Handler) BatchSync(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	recordedBy, ok := middleware.GetUserID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas pengelola tidak ditemukan."})
		return
	}

	var req BatchSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data batch sync tidak valid: " + err.Error()})
		return
	}

	resp, err := h.service.BatchSync(c.Request.Context(), tenantID, recordedBy, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses batch sync."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetDeposit retrieves a deposit by ID.
func (h *Handler) GetDeposit(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	requesterID, _ := middleware.GetUserID(c.Request.Context())
	requesterRole, _ := middleware.GetUserRole(c.Request.Context())

	idParam := c.Param("id")
	depositID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID setoran tidak valid."})
		return
	}

	resp, err := h.service.GetDeposit(c.Request.Context(), tenantID, depositID, requesterID, requesterRole)
	if err != nil {
		if errors.Is(err, ErrDepositNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data setoran tidak ditemukan."})
			return
		}
		if errors.Is(err, ErrUnauthorizedAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke data setoran ini."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data setoran."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetReceipt retrieves the digital receipt for a deposit.
func (h *Handler) GetReceipt(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	requesterID, _ := middleware.GetUserID(c.Request.Context())
	requesterRole, _ := middleware.GetUserRole(c.Request.Context())

	idParam := c.Param("id")
	depositID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID setoran tidak valid."})
		return
	}

	resp, err := h.service.GetReceipt(c.Request.Context(), tenantID, depositID, requesterID, requesterRole)
	if err != nil {
		if errors.Is(err, ErrDepositNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data setoran tidak ditemukan."})
			return
		}
		if errors.Is(err, ErrUnauthorizedAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke struk setoran ini."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil struk setoran."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListDeposits lists deposits for Pengelola (tenant-wide) or Nasabah (own only).
func (h *Handler) ListDeposits(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	requesterID, _ := middleware.GetUserID(c.Request.Context())
	requesterRole, _ := middleware.GetUserRole(c.Request.Context())

	var filter DepositListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter filter tidak valid: " + err.Error()})
		return
	}

	resp, err := h.service.ListDeposits(c.Request.Context(), tenantID, requesterID, requesterRole, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar setoran."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetMyBalance allows an authenticated nasabah to view their current balance and deposit stats.
func (h *Handler) GetMyBalance(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	userID, ok := middleware.GetUserID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas nasabah tidak ditemukan."})
		return
	}

	resp, err := h.service.GetNasabahBalance(c.Request.Context(), tenantID, userID)
	if err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data nasabah tidak ditemukan."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data saldo nasabah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetNasabahBalance allows Pengelola to view balance of a specific nasabah in the tenant.
func (h *Handler) GetNasabahBalance(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	nasabahID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid."})
		return
	}

	resp, err := h.service.GetNasabahBalance(c.Request.Context(), tenantID, nasabahID)
	if err != nil {
		if errors.Is(err, ErrNasabahNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data nasabah tidak ditemukan di bank sampah ini."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data saldo nasabah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}
