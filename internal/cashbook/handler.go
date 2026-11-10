package cashbook

import (
	"errors"
	"net/http"

	"github.com/audyfh/setor-hijau-be/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMid gin.HandlerFunc) {
	cashbookGroup := rg.Group("/cashbook")
	cashbookGroup.Use(authMid)
	{
		pengelolaOnly := cashbookGroup.Group("")
		pengelolaOnly.Use(middleware.RequireRole("pengelola"))
		{
			// Sales to Pengepul
			pengelolaOnly.POST("/sales", h.RecordSale)
			pengelolaOnly.GET("/sales", h.ListSales)
			pengelolaOnly.GET("/sales/:id", h.GetSaleByID)

			// Cashbook Entries
			pengelolaOnly.POST("/entries", h.CreateManualEntry)
			pengelolaOnly.GET("/entries", h.ListCashbookEntries)
			pengelolaOnly.GET("/summary", h.GetCashbookSummary)
			pengelolaOnly.GET("/reconciliation", h.GetFinancialReconciliation)
		}
	}

	nasabahGroup := rg.Group("/nasabah")
	nasabahGroup.Use(authMid)
	{
		nasabahGroup.GET("/me/withdrawals", middleware.RequireRole("nasabah"), h.ListMyWithdrawals)
		nasabahGroup.POST("/:id/withdrawals", middleware.RequireRole("pengelola"), h.RecordWithdrawal)
		nasabahGroup.GET("/:id/withdrawals", middleware.RequireRole("pengelola"), h.ListNasabahWithdrawals)
	}
}

func (h *Handler) RecordSale(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}
	userID, exists := middleware.GetUserID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing user context"})
		return
	}

	var req CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.RecordSale(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, ErrEmptyLines) || errors.Is(err, ErrInvalidWeight) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) GetSaleByID(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID penjualan tidak valid"})
		return
	}

	resp, err := h.service.GetSaleByID(c.Request.Context(), tenantID, saleID)
	if err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListSales(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	var filter SaleListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ListSales(c.Request.Context(), tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RecordWithdrawal(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}
	recordedBy, exists := middleware.GetUserID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing user context"})
		return
	}

	nasabahID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid"})
		return
	}

	var req CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.NasabahID = nasabahID

	resp, err := h.service.RecordWithdrawal(c.Request.Context(), tenantID, recordedBy, req)
	if err != nil {
		if errors.Is(err, ErrInsufficientBalance) || errors.Is(err, ErrInvalidAmount) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) ListMyWithdrawals(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}
	userID, exists := middleware.GetUserID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing user context"})
		return
	}
	role, _ := middleware.GetUserRole(c.Request.Context())

	var filter WithdrawalListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ListWithdrawals(c.Request.Context(), tenantID, userID, role, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListNasabahWithdrawals(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}
	role, _ := middleware.GetUserRole(c.Request.Context())

	nasabahID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID nasabah tidak valid"})
		return
	}

	var filter WithdrawalListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter.NasabahID = &nasabahID

	resp, err := h.service.ListWithdrawals(c.Request.Context(), tenantID, uuid.Nil, role, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateManualEntry(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}
	recordedBy, exists := middleware.GetUserID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing user context"})
		return
	}

	var req CreateCashbookEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreateManualEntry(c.Request.Context(), tenantID, recordedBy, req)
	if err != nil {
		if errors.Is(err, ErrInvalidAmount) || errors.Is(err, ErrInvalidEntryType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) ListCashbookEntries(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	var filter CashbookListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ListCashbookEntries(c.Request.Context(), tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetCashbookSummary(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	resp, err := h.service.GetCashbookSummary(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetFinancialReconciliation(c *gin.Context) {
	tenantID, exists := middleware.GetTenantID(c.Request.Context())
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing tenant context"})
		return
	}

	resp, err := h.service.GetFinancialReconciliation(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
