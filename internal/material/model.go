package material

import (
	"time"

	"github.com/google/uuid"
)

// Material represents a recyclable waste type with its current active price and linked emission factor.
type Material struct {
	ID                 uuid.UUID  `json:"id"`
	BankSampahID       uuid.UUID  `json:"bank_sampah_id"`
	Name               string     `json:"name"`
	Category           string     `json:"category"`
	Unit               string     `json:"unit"`
	Description        string     `json:"description,omitempty"`
	EmissionFactorID   *uuid.UUID `json:"emission_factor_id,omitempty"`
	FactorMaterialCode string     `json:"factor_material_code,omitempty"`
	FactorName         string     `json:"factor_name,omitempty"`
	FactorVersion      string     `json:"factor_version,omitempty"`
	FactorSource       string     `json:"factor_source,omitempty"`
	CO2eFactorPPM      int64      `json:"co2e_factor_ppm,omitempty"`
	IsActive           bool       `json:"is_active"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	PriceID            uuid.UUID  `json:"price_id,omitempty"`
	PricePerKgIDR      int64      `json:"price_per_kg_idr"`
	PriceEffectiveFrom *time.Time `json:"price_effective_from,omitempty"`
}

// PriceHistoryEntry represents a historical price point for a material.
type PriceHistoryEntry struct {
	ID             uuid.UUID  `json:"id"`
	MaterialTypeID uuid.UUID  `json:"material_type_id"`
	BankSampahID   uuid.UUID  `json:"bank_sampah_id"`
	PricePerKgIDR  int64      `json:"price_per_kg_idr"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveTo    *time.Time `json:"effective_to,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CreateMaterialRequest defines input for creating a new material type with initial price and emission factor.
type CreateMaterialRequest struct {
	Name             string     `json:"name" binding:"required,min=2,max=100"`
	Category         string     `json:"category" binding:"required,max=50"`
	Unit             string     `json:"unit" binding:"omitempty,max=20"`
	Description      string     `json:"description"`
	EmissionFactorID *uuid.UUID `json:"emission_factor_id"`
	PricePerKgIDR    int64      `json:"price_per_kg_idr" binding:"min=0"`
}

// UpdateMaterialRequest defines input for updating material type metadata and emission factor.
type UpdateMaterialRequest struct {
	Name             string     `json:"name" binding:"required,min=2,max=100"`
	Category         string     `json:"category" binding:"required,max=50"`
	Unit             string     `json:"unit" binding:"omitempty,max=20"`
	Description      string     `json:"description"`
	EmissionFactorID *uuid.UUID `json:"emission_factor_id"`
	IsActive         bool       `json:"is_active"`
}

// UpdatePriceRequest defines input for setting a new price for a material type.
type UpdatePriceRequest struct {
	PricePerKgIDR int64 `json:"price_per_kg_idr" binding:"min=0"`
}
