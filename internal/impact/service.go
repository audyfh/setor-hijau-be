package impact

import (
	"context"
	"math"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service interface {
	RecordTelemetry(ctx context.Context, req RecordTelemetryRequest) (*TelemetryResponse, error)
	GetPlatformNetImpact(ctx context.Context) (*NetImpactSummary, error)
	GetTenantNetImpact(ctx context.Context, bankSampahID uuid.UUID) (*NetImpactSummary, error)
	GetTenantNetImpactByCode(ctx context.Context, code string) (*NetImpactSummary, error)
	GetPublicMethodsData(ctx context.Context) (*PublicMethodsResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// CalculateDigitalEmissions computes integer micrograms and grams CO2e for transferred bytes under SWDM v4:
// 1 GB (1,000,000,000 bytes) = 0.072 kWh/GB * 494 g/kWh = 35.568 g CO2e = 35,568,000 micrograms CO2e.
// Micrograms = (bytes * 35568 + 500,000) / 1,000,000
// Grams = (micrograms + 500,000) / 1,000,000
func CalculateDigitalEmissions(bytesTransferred int64) (micrograms int64, grams int64) {
	if bytesTransferred <= 0 {
		return 0, 0
	}
	micrograms = (bytesTransferred*35568 + 500000) / 1000000
	grams = (micrograms + 500000) / 1000000
	return micrograms, grams
}

func (s *service) RecordTelemetry(ctx context.Context, req RecordTelemetryRequest) (*TelemetryResponse, error) {
	co2Micrograms, co2Grams := CalculateDigitalEmissions(req.DataBytesTransferred)

	visitorType := "new"
	if req.VisitorType != "" {
		visitorType = req.VisitorType
	}

	clientTime := time.Now().UTC()
	if req.ClientTimestamp != nil && !req.ClientTimestamp.IsZero() {
		clientTime = *req.ClientTimestamp
	}

	var bsUUID pgtype.UUID
	if req.BankSampahID != nil && *req.BankSampahID != uuid.Nil {
		bsUUID = db.ToPGUUID(*req.BankSampahID)
	} else if req.BankSampahCode != nil && *req.BankSampahCode != "" {
		bs, err := s.repo.GetBankSampahByCode(ctx, *req.BankSampahCode)
		if err == nil && bs != nil {
			bsUUID = bs.ID
		}
	}

	var sessionText pgtype.Text
	if req.SessionID != nil && *req.SessionID != "" {
		sessionText = pgtype.Text{String: *req.SessionID, Valid: true}
	}

	m, err := s.repo.RecordMeasurement(ctx, sqlc.RecordDigitalFootprintMeasurementParams{
		BankSampahID:         bsUUID,
		SessionID:            sessionText,
		PagePath:             req.PagePath,
		DataBytesTransferred: req.DataBytesTransferred,
		Co2eMicrograms:       co2Micrograms,
		Co2eGrams:            co2Grams,
		SwdmVersion:          "SWDM v4",
		SwdmModelSource:      "Sustainable Web Design Model v4 (SustainableWebDesign.org / CO2.js)",
		VisitorType:          visitorType,
		ClientTimestamp:      db.ToPGTimestamptz(clientTime),
	})
	if err != nil {
		return nil, err
	}

	return &TelemetryResponse{
		ID:                      db.FromPGUUID(m.ID),
		PagePath:                m.PagePath,
		DataBytesTransferred:    m.DataBytesTransferred,
		EstimatedCO2eMicrograms: m.Co2eMicrograms,
		EstimatedCO2eGrams:      m.Co2eGrams,
		SWDMVersion:             m.SwdmVersion,
		CreatedAt:               db.FromPGTimestamptz(m.CreatedAt),
	}, nil
}

func (s *service) GetPlatformNetImpact(ctx context.Context) (*NetImpactSummary, error) {
	avoided, err := s.repo.GetPlatformAvoidedCarbon(ctx)
	if err != nil {
		return nil, err
	}

	footprint, err := s.repo.GetPlatformDigitalFootprint(ctx)
	if err != nil {
		return nil, err
	}

	categories, err := s.repo.GetPlatformAvoidedCarbonByCategory(ctx)
	if err != nil {
		return nil, err
	}

	return s.buildNetImpactSummary("platform", nil, "", "", avoided.TotalWeightGrams, avoided.TotalCo2eGrams, footprint.TotalBytesTransferred, footprint.TotalCo2eGrams, categories), nil
}

func (s *service) GetTenantNetImpact(ctx context.Context, bankSampahID uuid.UUID) (*NetImpactSummary, error) {
	bs, err := s.repo.GetBankSampahByID(ctx, bankSampahID)
	if err != nil {
		return nil, err
	}

	avoided, err := s.repo.GetTenantAvoidedCarbon(ctx, bankSampahID)
	if err != nil {
		return nil, err
	}

	footprint, err := s.repo.GetTenantDigitalFootprint(ctx, bankSampahID)
	if err != nil {
		return nil, err
	}

	categories, err := s.repo.GetTenantAvoidedCarbonByCategory(ctx, bankSampahID)
	if err != nil {
		return nil, err
	}

	bsID := db.FromPGUUID(bs.ID)
	return s.buildNetImpactSummary("tenant", &bsID, bs.Name, bs.Code, avoided.TotalWeightGrams, avoided.TotalCo2eGrams, footprint.TotalBytesTransferred, footprint.TotalCo2eGrams, categories), nil
}

func (s *service) GetTenantNetImpactByCode(ctx context.Context, code string) (*NetImpactSummary, error) {
	bs, err := s.repo.GetBankSampahByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	bsID := db.FromPGUUID(bs.ID)
	return s.GetTenantNetImpact(ctx, bsID)
}

func (s *service) buildNetImpactSummary(
	scope string,
	bsID *uuid.UUID,
	bsName string,
	bsCode string,
	recycledGrams int64,
	avoidedGrams int64,
	bytesTransferred int64,
	footprintGrams int64,
	catRows interface{},
) *NetImpactSummary {
	netGrams := avoidedGrams - footprintGrams

	var ratio float64
	if footprintGrams > 0 {
		ratio = math.Round((float64(avoidedGrams)/float64(footprintGrams))*100) / 100
	} else if avoidedGrams > 0 {
		ratio = float64(avoidedGrams)
	}

	// SWDM v4 Segment Breakdown (Data center 15%, Network 14%, Device 52%, Embodied 19%)
	segments := SystemSegmentsBreakdown{
		DataCenterGrams:       (footprintGrams*15 + 50) / 100,
		NetworkGrams:          (footprintGrams*14 + 50) / 100,
		UserDeviceGrams:       (footprintGrams*52 + 50) / 100,
		EmbodiedHardwareGrams: (footprintGrams*19 + 50) / 100,
	}

	var catImpacts []AvoidedCategoryImpact
	switch rows := catRows.(type) {
	case []sqlc.GetPlatformAvoidedCarbonByCategoryRow:
		for _, r := range rows {
			catImpacts = append(catImpacts, AvoidedCategoryImpact{
				Category:         r.Category,
				AvoidedCO2eGrams: r.TotalCo2eGrams,
				AvoidedCO2eKg:    float64(r.TotalCo2eGrams) / 1000.0,
				WeightGrams:      r.TotalWeightGrams,
				WeightKg:         float64(r.TotalWeightGrams) / 1000.0,
			})
		}
	case []sqlc.GetTenantAvoidedCarbonByCategoryRow:
		for _, r := range rows {
			catImpacts = append(catImpacts, AvoidedCategoryImpact{
				Category:         r.Category,
				AvoidedCO2eGrams: r.TotalCo2eGrams,
				AvoidedCO2eKg:    float64(r.TotalCo2eGrams) / 1000.0,
				WeightGrams:      r.TotalWeightGrams,
				WeightKg:         float64(r.TotalWeightGrams) / 1000.0,
			})
		}
	}

	return &NetImpactSummary{
		Scope:                          scope,
		BankSampahID:                   bsID,
		BankSampahName:                 bsName,
		BankSampahCode:                 bsCode,
		TotalRecycledWeightGrams:       recycledGrams,
		TotalRecycledWeightKg:          float64(recycledGrams) / 1000.0,
		TotalAvoidedCO2eGrams:          avoidedGrams,
		TotalAvoidedCO2eKg:             float64(avoidedGrams) / 1000.0,
		TotalDataBytesTransferred:      bytesTransferred,
		TotalDataMegabytes:             math.Round((float64(bytesTransferred)/(1024*1024))*100) / 100,
		TotalDigitalFootprintCO2eGrams: footprintGrams,
		TotalDigitalFootprintCO2eKg:    float64(footprintGrams) / 1000.0,
		NetImpactCO2eGrams:             netGrams,
		NetImpactCO2eKg:                float64(netGrams) / 1000.0,
		SustainabilityRatio:            ratio,
		AvoidedCategories:              catImpacts,
		DigitalFootprintSegments:       segments,
		ModelCitations: ModelCitations{
			AvoidedEmissionsModel: "US EPA Waste Reduction Model (WARM) Version 16 (December 2023)",
			DigitalFootprintModel: "Sustainable Web Design Model (SWDM) Version 4 (SustainableWebDesign.org / CO2.js)",
		},
	}
}

func (s *service) GetPublicMethodsData(ctx context.Context) (*PublicMethodsResponse, error) {
	factors, err := s.repo.ListActiveEmissionFactors(ctx)
	if err != nil {
		return nil, err
	}

	var targetMaterials []MethodologyEmissionFactor
	for _, f := range factors {
		var descPtr *string
		if f.Description.Valid && f.Description.String != "" {
			descPtr = &f.Description.String
		}
		metricTonPerTon := float64(f.Co2eFactorPpm) / 1000000.0

		targetMaterials = append(targetMaterials, MethodologyEmissionFactor{
			ID:                  db.FromPGUUID(f.ID),
			MaterialCode:        f.MaterialCode,
			Name:                f.Name,
			Category:            f.Category,
			FactorVersion:       f.FactorVersion,
			FactorSource:        f.FactorSource,
			CO2eFactorPPM:       f.Co2eFactorPpm,
			MetricTonCO2ePerTon: math.Round(metricTonPerTon*1000000) / 1000000,
			Description:         descPtr,
		})
	}

	return &PublicMethodsResponse{
		Title:             "Metodologi & Sumber Akuntansi Karbon Setor Hijau",
		AcademicStatement: "Setor Hijau mengintegrasikan akuntansi emisi gas rumah kaca berbasis daur hidup (Life Cycle Assessment / LCA mengacu WARM v16) untuk sampah daur ulang dan komputasi hijau (Green Software Engineering mengacu SWDM v4) untuk aplikasi web. Seluruh perhitungan dapat diverifikasi dan direproduksi secara terbuka dari data mentah yang tersimpan.",
		WARMModel: WARMModelDoc{
			Version:          "WARM Version 16 (Desember 2023)",
			SourceCitation:   "U.S. Environmental Protection Agency (US EPA) Waste Reduction Model (WARM) Version 16",
			BaselineScenario: "Pengalihan sampah anorganik dari pembuangan konvensional ke Tempat Pemrosesan Akhir (TPA / Landfill) menuju fasilitas daur ulang (Recycling).",
			TargetMaterials:  targetMaterials,
		},
		SWDMModel: SWDMParameters{
			ModelVersion:                  "SWDM Version 4",
			ModelSource:                   "Sustainable Web Design Model v4 (Wholegrain Digital, Mightybytes, Medina Works, CO2.js >= 0.18)",
			OperationalEnergyPerGB_kWh:    0.057,
			EmbodiedEnergyPerGB_kWh:       0.015,
			TotalEnergyPerGB_kWh:          0.072,
			GridCarbonIntensity_g_per_kWh: 494.0,
			EmissionsPerGB_gCO2e:          35.568,
			SegmentsBreakdownPercent: SystemSegmentsBreakdownPct{
				DataCenterPct:       15.0,
				NetworkPct:          14.0,
				UserDevicePct:       52.0,
				EmbodiedHardwarePct: 19.0,
			},
		},
		Formulas: FormulaExplanations{
			AvoidedCO2eFormula:         "co2e_grams = floor((weight_grams * co2e_factor_ppm + 500,000) / 1,000,000)",
			DigitalFootprintFormula:    "co2e_micrograms = floor((bytes_transferred * 35,568 + 500,000) / 1,000,000); co2e_grams = floor((co2e_micrograms + 500,000) / 1,000,000)",
			NetImpactFormula:           "net_impact_co2e_grams = sum(avoided_co2e_grams) - sum(digital_footprint_co2e_grams)",
			SustainabilityRatioFormula: "sustainability_ratio = total_avoided_co2e_grams / total_digital_footprint_co2e_grams",
		},
		AssumptionsAndLimitations: []string{
			"Faktor emisi daur ulang US EPA WARM v16 mengacu pada bauran energi dan efisiensi industri manufaktur di AS sebagai estimasi ilmiah terbaik yang tersedia (best available scientific estimate).",
			"Seluruh angka karbon disajikan sebagai estimasi terhitung (estimated) sesuai Hard Rule #9 dan tidak dimaksudkan sebagai pengukuran sensor gas fisik di lokasi.",
			"Jejak karbon digital web diestimasi menggunakan volume data transfer (data transfer proxy) melalui standar SWDM v4, bukan pengukuran langsung watt-meter pada perangkat fisik pengguna.",
			"Perhitungan emisi dibatasi secara ketat pada sampah anorganik bernilai ekonomi (kertas, plastik, logam, kaca) dan tidak mencakup sampah organik.",
		},
	}, nil
}
