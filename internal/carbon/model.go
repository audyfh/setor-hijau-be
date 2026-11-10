package carbon

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmissionFactorNotFound     = errors.New("faktor emisi tidak ditemukan")
	ErrCarbonLedgerEntryNotFound = errors.New("entri jurnal karbon tidak ditemukan")
	ErrInvalidReversal           = errors.New("entri reversal tidak valid")
	ErrAlreadyReversed           = errors.New("entri jurnal karbon sudah dibalik sebelumnya")
	ErrUnauthorizedReversal      = errors.New("hanya pengelola yang dapat membuat entri pembalik (reversal)")
)

// EmissionFactor represents a versioned GHG emission factor from US EPA WARM v16.
type EmissionFactor struct {
	ID            uuid.UUID `json:"id"`
	MaterialCode  string    `json:"material_code"`
	Name          string    `json:"name"`
	Category      string    `json:"category"`
	FactorVersion string    `json:"factor_version"`
	FactorSource  string    `json:"factor_source"`
	CO2eFactorPPM int64     `json:"co2e_factor_ppm"`
	Description   string    `json:"description,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

// CarbonLedgerEntry represents an immutable journal entry in the environmental ledger.
type CarbonLedgerEntry struct {
	ID             uuid.UUID  `json:"id"`
	BankSampahID   uuid.UUID  `json:"bank_sampah_id"`
	NasabahID      uuid.UUID  `json:"nasabah_id"`
	NasabahName    string     `json:"nasabah_name,omitempty"`
	NasabahUsername string    `json:"nasabah_username,omitempty"`
	DepositID      *uuid.UUID `json:"deposit_id,omitempty"`
	DepositLineID  *uuid.UUID `json:"deposit_line_id,omitempty"`
	EmissionFactorID uuid.UUID `json:"emission_factor_id"`
	FactorName     string     `json:"factor_name"`
	FactorCategory string     `json:"factor_category"`
	FactorVersion  string     `json:"factor_version"`
	FactorSource   string     `json:"factor_source"`
	WeightGrams    int64      `json:"weight_grams"`
	CO2eGrams      int64      `json:"estimated_co2e_avoided_grams"`
	EntryType      string     `json:"entry_type"`
	ReversalForID  *uuid.UUID `json:"reversal_for_id,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CategoryCarbonSavings represents aggregated avoided CO2e for a specific material category.
type CategoryCarbonSavings struct {
	Category                      string `json:"category"`
	TotalEstimatedCO2eAvoidedGrams int64  `json:"total_estimated_co2e_avoided_grams"`
	TotalWeightGrams              int64  `json:"total_weight_grams"`
}

// NasabahCarbonSavingsResponse represents the cumulative carbon savings (tabungan karbon) of a nasabah.
type NasabahCarbonSavingsResponse struct {
	NasabahID                      uuid.UUID               `json:"nasabah_id"`
	TotalEstimatedCO2eAvoidedGrams int64                   `json:"total_estimated_co2e_avoided_grams"`
	TotalWeightGrams              int64                   `json:"total_weight_grams"`
	TotalEntries                  int64                   `json:"total_entries"`
	BreakdownByCategory           []CategoryCarbonSavings `json:"breakdown_by_category"`
}

// BankSampahCarbonSavingsResponse represents the cumulative carbon savings of a bank sampah tenant.
type BankSampahCarbonSavingsResponse struct {
	BankSampahID                   uuid.UUID `json:"bank_sampah_id"`
	TotalEstimatedCO2eAvoidedGrams int64     `json:"total_estimated_co2e_avoided_grams"`
	TotalWeightGrams              int64     `json:"total_weight_grams"`
	TotalEntries                  int64     `json:"total_entries"`
}

// CarbonLedgerFilter defines query parameters for filtering ledger records.
type CarbonLedgerFilter struct {
	NasabahID *uuid.UUID
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int32
	Offset    int32
}

// CarbonLedgerListResponse represents a paginated list of carbon ledger records.
type CarbonLedgerListResponse struct {
	Entries    []CarbonLedgerEntry `json:"entries"`
	TotalCount int64               `json:"total_count"`
	Limit      int32               `json:"limit"`
	Offset     int32               `json:"offset"`
}

// CreateReversalRequest defines input for creating a reversal entry for a mistaken or cancelled deposit item.
type CreateReversalRequest struct {
	CarbonLedgerID uuid.UUID `json:"carbon_ledger_id" binding:"required"`
	Notes          string    `json:"notes" binding:"required,min=3,max=500"`
}
