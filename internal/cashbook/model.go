package cashbook

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSaleNotFound             = errors.New("transaksi penjualan ke pengepul tidak ditemukan")
	ErrWithdrawalNotFound       = errors.New("transaksi penarikan saldo tidak ditemukan")
	ErrCashbookEntryNotFound    = errors.New("entri buku kas tidak ditemukan")
	ErrInsufficientBalance      = errors.New("saldo nasabah tidak mencukupi untuk melakukan penarikan")
	ErrInvalidAmount            = errors.New("nominal transaksi harus lebih besar dari 0")
	ErrInvalidWeight            = errors.New("berat sampah harus lebih besar dari 0")
	ErrEmptyLines               = errors.New("rincian sampah tidak boleh kosong")
	ErrInvalidEntryType         = errors.New("jenis mutasi kas tidak valid (harus cash_in atau cash_out)")
	ErrUnauthorizedAccess       = errors.New("akses ditolak")
)

// --- Sales to Pengepul Models ---

type CreateSaleLineRequest struct {
	MaterialTypeID uuid.UUID `json:"material_type_id" binding:"required"`
	WeightGrams    int64     `json:"weight_grams" binding:"required,gt=0"`
	PricePerKgIDR  int64     `json:"price_per_kg_idr" binding:"required,gte=0"`
}

type CreateSaleRequest struct {
	BuyerName string                  `json:"buyer_name" binding:"required,max=150"`
	SaleDate  time.Time               `json:"sale_date" binding:"required"`
	Notes     *string                 `json:"notes,omitempty"`
	Lines     []CreateSaleLineRequest `json:"lines" binding:"required,min=1,dive"`
}

type SaleLineResponse struct {
	ID               uuid.UUID `json:"id"`
	SaleID           uuid.UUID `json:"sale_id"`
	MaterialTypeID   uuid.UUID `json:"material_type_id"`
	MaterialName     string    `json:"material_name"`
	MaterialCategory string    `json:"material_category"`
	MaterialUnit     string    `json:"material_unit"`
	WeightGrams      int64     `json:"weight_grams"`
	PricePerKgIDR    int64     `json:"price_per_kg_idr"`
	SubtotalIDR      int64     `json:"subtotal_idr"`
	CreatedAt        time.Time `json:"created_at"`
}

type SaleResponse struct {
	ID               uuid.UUID          `json:"id"`
	BankSampahID     uuid.UUID          `json:"bank_sampah_id"`
	BuyerName        string             `json:"buyer_name"`
	SaleDate         time.Time          `json:"sale_date"`
	TotalWeightGrams int64              `json:"total_weight_grams"`
	TotalAmountIDR   int64              `json:"total_amount_idr"`
	Notes            *string            `json:"notes,omitempty"`
	RecordedBy       uuid.UUID          `json:"recorded_by"`
	RecordedByName   string             `json:"recorded_by_name"`
	CreatedAt        time.Time          `json:"created_at"`
	Lines            []SaleLineResponse `json:"lines,omitempty"`
}

type SaleListFilter struct {
	StartDate *time.Time `form:"start_date" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate   *time.Time `form:"end_date" time_format:"2006-01-02T15:04:05Z07:00"`
	Page      int        `form:"page,default=1"`
	Limit     int        `form:"limit,default=20"`
}

type SaleListResponse struct {
	Data       []SaleResponse `json:"data"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"total_pages"`
}

// --- Nasabah Withdrawals Models ---

type CreateWithdrawalRequest struct {
	NasabahID uuid.UUID `json:"nasabah_id" binding:"required"`
	AmountIDR int64     `json:"amount_idr" binding:"required,gt=0"`
	Notes     *string   `json:"notes,omitempty"`
}

type WithdrawalResponse struct {
	ID                  uuid.UUID `json:"id"`
	BankSampahID        uuid.UUID `json:"bank_sampah_id"`
	NasabahID           uuid.UUID `json:"nasabah_id"`
	NasabahName         string    `json:"nasabah_name"`
	NasabahUsername     string    `json:"nasabah_username"`
	AmountIDR           int64     `json:"amount_idr"`
	RemainingBalanceIDR int64     `json:"remaining_balance_idr"`
	Notes               *string   `json:"notes,omitempty"`
	RecordedBy          uuid.UUID `json:"recorded_by"`
	RecordedByName      string    `json:"recorded_by_name"`
	CreatedAt           time.Time `json:"created_at"`
}

type WithdrawalListFilter struct {
	NasabahID *uuid.UUID `form:"nasabah_id"`
	StartDate *time.Time `form:"start_date" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate   *time.Time `form:"end_date" time_format:"2006-01-02T15:04:05Z07:00"`
	Page      int        `form:"page,default=1"`
	Limit     int        `form:"limit,default=20"`
}

type WithdrawalListResponse struct {
	Data       []WithdrawalResponse `json:"data"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

// --- Cashbook Entries Models ---

type CreateCashbookEntryRequest struct {
	EntryType       string     `json:"entry_type" binding:"required,oneof=cash_in cash_out"`
	Category        string     `json:"category" binding:"required,max=64"`
	AmountIDR       int64      `json:"amount_idr" binding:"required,gt=0"`
	Description     string     `json:"description" binding:"required"`
	TransactionDate *time.Time `json:"transaction_date,omitempty"`
}

type CashbookEntryResponse struct {
	ID              uuid.UUID  `json:"id"`
	BankSampahID    uuid.UUID  `json:"bank_sampah_id"`
	EntryType       string     `json:"entry_type"`
	Category        string     `json:"category"`
	AmountIDR       int64      `json:"amount_idr"`
	ReferenceID     *uuid.UUID `json:"reference_id,omitempty"`
	Description     string     `json:"description"`
	RecordedBy      uuid.UUID  `json:"recorded_by"`
	RecordedByName  string     `json:"recorded_by_name"`
	TransactionDate time.Time  `json:"transaction_date"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CashbookListFilter struct {
	EntryType *string    `form:"entry_type"`
	Category  *string    `form:"category"`
	StartDate *time.Time `form:"start_date" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate   *time.Time `form:"end_date" time_format:"2006-01-02T15:04:05Z07:00"`
	Page      int        `form:"page,default=1"`
	Limit     int        `form:"limit,default=20"`
}

type CashbookListResponse struct {
	Data       []CashbookEntryResponse `json:"data"`
	TotalCount int64                   `json:"total_count"`
	Page       int                     `json:"page"`
	Limit      int                     `json:"limit"`
	TotalPages int                     `json:"total_pages"`
}

type CashbookSummaryResponse struct {
	TotalCashInIDR        int64 `json:"total_cash_in_idr"`
	TotalCashOutIDR       int64 `json:"total_cash_out_idr"`
	CurrentCashBalanceIDR int64 `json:"current_cash_balance_idr"`
}

// --- Reconciliation Model ---

type FinancialReconciliationResponse struct {
	TotalSalesToPengepulIDR          int64 `json:"total_sales_to_pengepul_idr"`
	TotalCashbookSalesIDR            int64 `json:"total_cashbook_sales_idr"`
	SalesReconciled                  bool  `json:"sales_reconciled"`
	TotalNasabahWithdrawalsIDR       int64 `json:"total_nasabah_withdrawals_idr"`
	TotalCashbookWithdrawalsIDR      int64 `json:"total_cashbook_withdrawals_idr"`
	WithdrawalsReconciled            bool  `json:"withdrawals_reconciled"`
	TotalDepositsAmountIDR           int64 `json:"total_deposits_amount_idr"`
	CurrentCashOnHandIDR             int64 `json:"current_cash_on_hand_idr"`
	TotalOutstandingNasabahBalanceIDR int64 `json:"total_outstanding_nasabah_balance_idr"`
	NetSurplusOrDeficitIDR           int64 `json:"net_surplus_or_deficit_idr"`
}
