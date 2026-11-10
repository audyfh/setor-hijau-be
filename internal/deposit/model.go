package deposit

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidIdempotencyKey = errors.New("idempotency_key wajib diisi dan maksimal 64 karakter")
	ErrInvalidClientTime     = errors.New("waktu pencatatan client tidak valid atau melebihi batas waktu saat ini")
	ErrEmptyDepositLines     = errors.New("setoran harus memiliki minimal satu baris sampah")
	ErrInvalidWeight         = errors.New("berat sampah harus lebih besar dari 0 gram")
	ErrMaterialNotFound      = errors.New("jenis sampah tidak ditemukan atau tidak aktif")
	ErrMaterialPriceNotSet   = errors.New("harga jenis sampah belum ditentukan untuk waktu transaksi tersebut")
	ErrNasabahNotFound       = errors.New("nasabah tidak ditemukan di bank sampah ini")
	ErrDepositNotFound       = errors.New("data setoran tidak ditemukan")
	ErrUnauthorizedAccess    = errors.New("tidak memiliki hak akses ke data setoran ini")
)

// CalculateSubtotal calculates line subtotal IDR from weight in grams and price per kg.
// It implements standard half-up integer rounding: (weight_grams * price_per_kg_idr + 500) / 1000.
func CalculateSubtotal(weightGrams int64, pricePerKgIDR int64) int64 {
	if weightGrams <= 0 || pricePerKgIDR <= 0 {
		return 0
	}
	return (weightGrams*pricePerKgIDR + 500) / 1000
}

type CreateDepositLineRequest struct {
	MaterialTypeID uuid.UUID `json:"material_type_id" binding:"required"`
	WeightGrams    int64     `json:"weight_grams" binding:"required,gt=0"`
}

type CreateDepositRequest struct {
	NasabahID        uuid.UUID                  `json:"nasabah_id" binding:"required"`
	IdempotencyKey   string                     `json:"idempotency_key" binding:"required,max=64"`
	ClientRecordedAt time.Time                  `json:"client_recorded_at" binding:"required"`
	Notes            *string                    `json:"notes"`
	Lines            []CreateDepositLineRequest `json:"lines" binding:"required,min=1,dive"`
}

type DepositLineResponse struct {
	ID                         uuid.UUID `json:"id"`
	DepositID                  uuid.UUID `json:"deposit_id"`
	MaterialTypeID             uuid.UUID `json:"material_type_id"`
	MaterialName               string    `json:"material_name"`
	MaterialCategory           string    `json:"material_category"`
	MaterialUnit               string    `json:"material_unit"`
	WeightGrams                int64     `json:"weight_grams"`
	PricePerKgIDR              int64     `json:"price_per_kg_idr"`
	SubtotalIDR                int64     `json:"subtotal_idr"`
	EstimatedCO2eAvoidedGrams  int64     `json:"estimated_co2e_avoided_grams"`
	FactorVersion              string    `json:"factor_version,omitempty"`
	FactorSource               string    `json:"factor_source,omitempty"`
	CreatedAt                  time.Time `json:"created_at"`
}

type DepositResponse struct {
	ID                             uuid.UUID             `json:"id"`
	BankSampahID                   uuid.UUID             `json:"bank_sampah_id"`
	NasabahID                      uuid.UUID             `json:"nasabah_id"`
	NasabahName                    string                `json:"nasabah_name"`
	NasabahUsername                string                `json:"nasabah_username"`
	IdempotencyKey                 string                `json:"idempotency_key"`
	TotalWeightGrams               int64                 `json:"total_weight_grams"`
	TotalAmountIDR                 int64                 `json:"total_amount_idr"`
	TotalEstimatedCO2eAvoidedGrams int64                 `json:"total_estimated_co2e_avoided_grams"`
	Notes                          *string               `json:"notes,omitempty"`
	RecordedBy                     uuid.UUID             `json:"recorded_by"`
	RecordedByName                 string                `json:"recorded_by_name"`
	ClientRecordedAt               time.Time             `json:"client_recorded_at"`
	CreatedAt                      time.Time             `json:"created_at"`
	Lines                          []DepositLineResponse `json:"lines"`
}

type DigitalReceiptResponse struct {
	DepositID                      uuid.UUID             `json:"deposit_id"`
	ReceiptNumber                  string                `json:"receipt_number"`
	BankSampahID                   uuid.UUID             `json:"bank_sampah_id"`
	BankSampahName                 string                `json:"bank_sampah_name"`
	BankSampahCode                 string                `json:"bank_sampah_code"`
	BankSampahAddress              string                `json:"bank_sampah_address"`
	BankSampahRW                   string                `json:"bank_sampah_rw"`
	BankSampahRT                   *string               `json:"bank_sampah_rt,omitempty"`
	BankSampahKelurahan            string                `json:"bank_sampah_kelurahan"`
	BankSampahKecamatan            string                `json:"bank_sampah_kecamatan"`
	BankSampahKabupatenKota        string                `json:"bank_sampah_kabupaten_kota"`
	BankSampahContactPhone         *string               `json:"bank_sampah_contact_phone,omitempty"`
	NasabahID                      uuid.UUID             `json:"nasabah_id"`
	NasabahName                    string                `json:"nasabah_name"`
	NasabahUsername                string                `json:"nasabah_username"`
	NasabahRW                      string                `json:"nasabah_rw"`
	NasabahRT                      *string               `json:"nasabah_rt,omitempty"`
	NasabahPhone                   *string               `json:"nasabah_phone,omitempty"`
	RecordedByID                   uuid.UUID             `json:"recorded_by_id"`
	RecordedByName                 string                `json:"recorded_by_name"`
	TotalWeightGrams               int64                 `json:"total_weight_grams"`
	TotalAmountIDR                 int64                 `json:"total_amount_idr"`
	TotalEstimatedCO2eAvoidedGrams int64                 `json:"total_estimated_co2e_avoided_grams"`
	NasabahCurrentBalance          int64                 `json:"nasabah_current_balance_idr"`
	Notes                          *string               `json:"notes,omitempty"`
	ClientRecordedAt               time.Time             `json:"client_recorded_at"`
	CreatedAt                      time.Time             `json:"created_at"`
	Lines                   []DepositLineResponse `json:"lines"`
}

type NasabahBalanceResponse struct {
	NasabahID          uuid.UUID  `json:"nasabah_id"`
	CurrentBalanceIDR  int64      `json:"current_balance_idr"`
	TotalWeightGrams   int64      `json:"total_weight_grams"`
	TotalDepositsCount int64      `json:"total_deposits_count"`
	LastDepositAt      *time.Time `json:"last_deposit_at,omitempty"`
}

type DepositListFilter struct {
	NasabahID *uuid.UUID `form:"nasabah_id"`
	StartDate *time.Time `form:"start_date" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate   *time.Time `form:"end_date" time_format:"2006-01-02T15:04:05Z07:00"`
	Page      int        `form:"page,default=1"`
	Limit     int        `form:"limit,default=20"`
}

type DepositListResponse struct {
	Data       []DepositResponse `json:"data"`
	TotalCount int64             `json:"total_count"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

type BatchSyncRequest struct {
	Deposits []CreateDepositRequest `json:"deposits" binding:"required,min=1,dive"`
}

type BatchSyncItemResult struct {
	IdempotencyKey string           `json:"idempotency_key"`
	Status         string           `json:"status"` // "created", "replayed", "failed"
	Deposit        *DepositResponse `json:"deposit,omitempty"`
	Error          *string          `json:"error,omitempty"`
}

type BatchSyncResponse struct {
	TotalProcessed int                   `json:"total_processed"`
	TotalSuccess   int                   `json:"total_success"`
	TotalFailed    int                   `json:"total_failed"`
	Results        []BatchSyncItemResult `json:"results"`
}
