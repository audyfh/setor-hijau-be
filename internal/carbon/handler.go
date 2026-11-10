package carbon

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler processes HTTP requests for carbon accounting and ledger auditing.
type Handler struct {
	service Service
}

// NewHandler creates a new Carbon HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers endpoints for emission factors, carbon savings, and ledger records.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc) {
	// Public or authenticated list of emission factors
	rg.GET("/carbon/factors", h.ListFactors)

	carbonGroup := rg.Group("/carbon")
	carbonGroup.Use(authMid)
	{
		// Nasabah self-service carbon balance
		carbonGroup.GET("/balance", middleware.RequireRole("nasabah"), h.GetMyCarbonSavings)

		// Pengelola tenant total carbon savings
		carbonGroup.GET("/summary", middleware.RequireRole("pengelola"), h.GetTenantCarbonSavings)

		// Carbon ledger history (scoped per role)
		carbonGroup.GET("/ledger", h.ListLedger)

		// Pengelola reversal entry
		carbonGroup.POST("/reversals", middleware.RequireRole("pengelola"), h.RecordReversal)
	}

	nasabahGroup := rg.Group("/nasabah")
	nasabahGroup.Use(authMid)
	{
		// Nasabah me carbon alias
		nasabahGroup.GET("/me/carbon", middleware.RequireRole("nasabah"), h.GetMyCarbonSavings)
		// Pengelola check specific nasabah carbon savings
		nasabahGroup.GET("/:id/carbon", middleware.RequireRole("pengelola"), h.GetNasabahCarbonSavings)
	}
}

// ListFactors returns the list of active EPA WARM v16 emission factors.
func (h *Handler) ListFactors(c *gin.Context) {
	factors, err := h.service.ListFactors(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar faktor emisi."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"disclaimer": "Faktor emisi diadaptasi dari US EPA Waste Reduction Model (WARM) Versi 16 (2023) sebagai estimasi ilmiah terbaik pengurangan emisi gas rumah kaca.",
		"factors":    factors,
	})
}

// GetMyCarbonSavings returns accumulated avoided CO2e for the authenticated nasabah.
func (h *Handler) GetMyCarbonSavings(c *gin.Context) {
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

	resp, err := h.service.GetNasabahCarbonSavings(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil tabungan karbon nasabah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetNasabahCarbonSavings allows Pengelola to view a specific nasabah's carbon savings.
func (h *Handler) GetNasabahCarbonSavings(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	idParam := c.Param("id")
	nasabahID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format ID nasabah tidak valid."})
		return
	}

	resp, err := h.service.GetNasabahCarbonSavings(c.Request.Context(), tenantID, nasabahID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data tabungan karbon nasabah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetTenantCarbonSavings allows Pengelola to view the total accumulated carbon savings of the bank sampah.
func (h *Handler) GetTenantCarbonSavings(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}

	resp, err := h.service.GetTenantCarbonSavings(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil ringkasan tabungan karbon bank sampah."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListLedger returns paginated carbon ledger records with multi-tenant and role scoping.
func (h *Handler) ListLedger(c *gin.Context) {
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
	role, ok := middleware.GetUserRole(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Peran pengguna tidak ditemukan."})
		return
	}

	filter := CarbonLedgerFilter{
		Limit:  20,
		Offset: 0,
	}

	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			filter.Limit = int32(val)
		}
	}
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			filter.Offset = int32(val)
		}
	}

	if role == "pengelola" {
		if nid := c.Query("nasabah_id"); nid != "" {
			if parsed, err := uuid.Parse(nid); err == nil {
				filter.NasabahID = &parsed
			}
		}
		if s := c.Query("start_date"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				filter.StartDate = &t
			}
		}
		if e := c.Query("end_date"); e != "" {
			if t, err := time.Parse(time.RFC3339, e); err == nil {
				filter.EndDate = &t
			}
		}
	}

	resp, err := h.service.ListLedgerEntries(c.Request.Context(), tenantID, userID, role, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar jurnal karbon."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RecordReversal handles creation of an append-only reversal entry by Pengelola.
func (h *Handler) RecordReversal(c *gin.Context) {
	tenantID, ok := middleware.GetTenantID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas tenant tidak ditemukan."})
		return
	}
	pengelolaID, ok := middleware.GetUserID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identitas pengelola tidak ditemukan."})
		return
	}

	var req CreateReversalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Input reversal tidak valid atau catatan terlalu pendek."})
		return
	}

	entry, err := h.service.RecordReversal(c.Request.Context(), tenantID, pengelolaID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrCarbonLedgerEntryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Entri jurnal karbon tidak ditemukan."})
		case errors.Is(err, ErrInvalidReversal):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Entri pembalik (reversal) tidak dapat dibalik kembali."})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat entri pembalik karbon."})
		}
		return
	}

	c.JSON(http.StatusCreated, entry)
}
