package impact

import (
	"time"

	"github.com/google/uuid"
)

// --- Telemetry Ingestion DTOs ---

type RecordTelemetryRequest struct {
	BankSampahID         *uuid.UUID `json:"bank_sampah_id,omitempty"`
	BankSampahCode       *string    `json:"bank_sampah_code,omitempty"`
	SessionID            *string    `json:"session_id,omitempty"`
	PagePath             string     `json:"page_path" binding:"required,max=255"`
	DataBytesTransferred int64      `json:"data_bytes_transferred" binding:"required,gte=0"`
	VisitorType          string     `json:"visitor_type" binding:"omitempty,oneof=new returning"`
	ClientTimestamp      *time.Time `json:"client_timestamp,omitempty"`
}

type TelemetryResponse struct {
	ID                       uuid.UUID `json:"id"`
	PagePath                 string    `json:"page_path"`
	DataBytesTransferred     int64     `json:"data_bytes_transferred"`
	EstimatedCO2eMicrograms  int64     `json:"estimated_co2e_micrograms"`
	EstimatedCO2eGrams       int64     `json:"estimated_co2e_grams"`
	SWDMVersion              string    `json:"swdm_version"`
	CreatedAt                time.Time `json:"created_at"`
}

// --- Net Impact DTOs ---

type SystemSegmentsBreakdown struct {
	DataCenterGrams       int64 `json:"data_center_co2e_grams"`
	NetworkGrams          int64 `json:"network_co2e_grams"`
	UserDeviceGrams       int64 `json:"user_device_co2e_grams"`
	EmbodiedHardwareGrams int64 `json:"embodied_hardware_co2e_grams"`
}

type SystemSegmentsBreakdownPct struct {
	DataCenterPct       float64 `json:"data_center_pct"`
	NetworkPct          float64 `json:"network_pct"`
	UserDevicePct       float64 `json:"user_device_pct"`
	EmbodiedHardwarePct float64 `json:"embodied_hardware_pct"`
}

type AvoidedCategoryImpact struct {
	Category         string  `json:"category"`
	AvoidedCO2eGrams int64   `json:"avoided_co2e_grams"`
	AvoidedCO2eKg    float64 `json:"avoided_co2e_kg"`
	WeightGrams      int64   `json:"weight_grams"`
	WeightKg         float64 `json:"weight_kg"`
}

type ModelCitations struct {
	AvoidedEmissionsModel string `json:"avoided_emissions_model"`
	DigitalFootprintModel string `json:"digital_footprint_model"`
}

type NetImpactSummary struct {
	Scope                            string                     `json:"scope"` // "platform" or "tenant"
	BankSampahID                     *uuid.UUID                 `json:"bank_sampah_id,omitempty"`
	BankSampahName                   string                     `json:"bank_sampah_name,omitempty"`
	BankSampahCode                   string                     `json:"bank_sampah_code,omitempty"`
	TotalRecycledWeightGrams         int64                      `json:"total_recycled_weight_grams"`
	TotalRecycledWeightKg            float64                    `json:"total_recycled_weight_kg"`
	TotalAvoidedCO2eGrams            int64                      `json:"total_avoided_co2e_grams"`
	TotalAvoidedCO2eKg               float64                    `json:"total_avoided_co2e_kg"`
	TotalDataBytesTransferred        int64                      `json:"total_data_bytes_transferred"`
	TotalDataMegabytes               float64                    `json:"total_data_megabytes"`
	TotalDigitalFootprintCO2eGrams   int64                      `json:"total_digital_footprint_co2e_grams"`
	TotalDigitalFootprintCO2eKg      float64                    `json:"total_digital_footprint_co2e_kg"`
	NetImpactCO2eGrams               int64                      `json:"net_impact_co2e_grams"`
	NetImpactCO2eKg                  float64                    `json:"net_impact_co2e_kg"`
	SustainabilityRatio              float64                    `json:"sustainability_ratio"`
	AvoidedCategories                []AvoidedCategoryImpact    `json:"avoided_categories"`
	DigitalFootprintSegments         SystemSegmentsBreakdown    `json:"digital_footprint_segments"`
	ModelCitations                   ModelCitations             `json:"model_citations"`
}

// --- Public Method Page DTOs ---

type MethodologyEmissionFactor struct {
	ID                  uuid.UUID `json:"id"`
	MaterialCode        string    `json:"material_code"`
	Name                string    `json:"name"`
	Category            string    `json:"category"`
	FactorVersion       string    `json:"factor_version"`
	FactorSource        string    `json:"factor_source"`
	CO2eFactorPPM       int64     `json:"co2e_factor_ppm"`
	MetricTonCO2ePerTon float64   `json:"metric_ton_co2e_per_metric_ton"`
	Description         *string   `json:"description,omitempty"`
}

type WARMModelDoc struct {
	Version         string                      `json:"version"`
	SourceCitation  string                      `json:"source_citation"`
	BaselineScenario string                     `json:"baseline_scenario"`
	TargetMaterials []MethodologyEmissionFactor `json:"target_materials"`
}

type SWDMParameters struct {
	ModelVersion                  string                     `json:"model_version"`
	ModelSource                   string                     `json:"model_source"`
	OperationalEnergyPerGB_kWh    float64                    `json:"operational_energy_per_gb_kwh"`
	EmbodiedEnergyPerGB_kWh       float64                    `json:"embodied_energy_per_gb_kwh"`
	TotalEnergyPerGB_kWh          float64                    `json:"total_energy_per_gb_kwh"`
	GridCarbonIntensity_g_per_kWh float64                    `json:"grid_carbon_intensity_g_per_kwh"`
	EmissionsPerGB_gCO2e          float64                    `json:"emissions_per_gb_g_co2e"`
	SegmentsBreakdownPercent      SystemSegmentsBreakdownPct `json:"segments_breakdown_percent"`
}

type FormulaExplanations struct {
	AvoidedCO2eFormula        string `json:"avoided_co2e_formula"`
	DigitalFootprintFormula   string `json:"digital_footprint_formula"`
	NetImpactFormula          string `json:"net_impact_formula"`
	SustainabilityRatioFormula string `json:"sustainability_ratio_formula"`
}

type PublicMethodsResponse struct {
	Title                      string              `json:"title"`
	AcademicStatement          string              `json:"academic_statement"`
	WARMModel                  WARMModelDoc        `json:"warm_model"`
	SWDMModel                  SWDMParameters      `json:"swdm_model"`
	Formulas                   FormulaExplanations `json:"formulas"`
	AssumptionsAndLimitations  []string            `json:"assumptions_and_limitations"`
}
