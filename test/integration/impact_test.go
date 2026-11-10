package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/deposit"
	"github.com/audyfh/setor-hijau-be/internal/impact"
	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/audyfh/setor-hijau-be/test/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicMethodsEndpoint(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	resp, err := http.Get(harness.Server.URL + "/api/v1/public/methods")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc impact.PublicMethodsResponse
	err = json.NewDecoder(resp.Body).Decode(&doc)
	require.NoError(t, err)

	assert.NotEmpty(t, doc.AcademicStatement)
	assert.Contains(t, doc.AcademicStatement, "WARM v16")
	assert.Contains(t, doc.AcademicStatement, "SWDM v4")

	// Avoided emissions method
	assert.Contains(t, doc.WARMModel.Version, "WARM Version 16")
	assert.Contains(t, doc.WARMModel.SourceCitation, "US EPA")
	assert.GreaterOrEqual(t, len(doc.WARMModel.TargetMaterials), 7)

	// Digital footprint method
	assert.Equal(t, "SWDM Version 4", doc.SWDMModel.ModelVersion)
	assert.Contains(t, doc.SWDMModel.ModelSource, "Sustainable Web Design Model")
	assert.Equal(t, 494.0, doc.SWDMModel.GridCarbonIntensity_g_per_kWh)
	assert.Equal(t, 0.072, doc.SWDMModel.TotalEnergyPerGB_kWh)
	assert.Equal(t, 35.568, doc.SWDMModel.EmissionsPerGB_gCO2e)
	assert.Equal(t, 15.0, doc.SWDMModel.SegmentsBreakdownPercent.DataCenterPct)
	assert.Equal(t, 14.0, doc.SWDMModel.SegmentsBreakdownPercent.NetworkPct)
	assert.Equal(t, 52.0, doc.SWDMModel.SegmentsBreakdownPercent.UserDevicePct)
	assert.Equal(t, 19.0, doc.SWDMModel.SegmentsBreakdownPercent.EmbodiedHardwarePct)

	// Net impact meter definition
	assert.NotEmpty(t, doc.Formulas.NetImpactFormula)
	assert.NotEmpty(t, doc.Formulas.AvoidedCO2eFormula)
	assert.NotEmpty(t, doc.Formulas.DigitalFootprintFormula)
	assert.NotEmpty(t, doc.Formulas.SustainabilityRatioFormula)
	assert.NotEmpty(t, doc.AssumptionsAndLimitations)
}

func TestImpactTelemetryAndNetImpactWorkflow(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Initial platform net impact should be zero
	resp, err := http.Get(harness.Server.URL + "/api/v1/public/impact")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var initialImpact impact.NetImpactSummary
	err = json.NewDecoder(resp.Body).Decode(&initialImpact)
	require.NoError(t, err)
	assert.Equal(t, int64(0), initialImpact.TotalAvoidedCO2eGrams)
	assert.Equal(t, int64(0), initialImpact.TotalDigitalFootprintCO2eGrams)
	assert.Equal(t, int64(0), initialImpact.NetImpactCO2eGrams)

	// 2. Register Tenant
	tenantCode := fmt.Sprintf("bs-impact-%d", time.Now().UnixNano()%1000000)
	regReq := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Net Impact RW 08",
		Code:              tenantCode,
		Address:           "Jl. Lestari No. 8",
		Rw:                "08",
		Kelurahan:         "Sukamaju",
		Kecamatan:         "Klojen",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaUsername: fmt.Sprintf("pengelola_%d", time.Now().UnixNano()%100000),
		PengelolaEmail:    fmt.Sprintf("pengelola_%d@example.com", time.Now().UnixNano()),
		PengelolaPassword: "password123",
		PengelolaFullName: "Petugas Net Impact",
	}

	body, _ := json.Marshal(regReq)
	regResp, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer regResp.Body.Close()
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regResult struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	err = json.NewDecoder(regResp.Body).Decode(&regResult)
	require.NoError(t, err)
	token := regResult.Tokens.AccessToken

	// 3. Post Telemetry for Tenant (e.g., 50 MB = 50,000,000 bytes)
	// Math: (50_000_000 * 35_568 + 500_000) / 1_000_000 = 1,778,400 micrograms.
	// Grams: (1_778_400 + 500_000) / 1_000_000 = 2 grams.
	telemetryReq := impact.RecordTelemetryRequest{
		BankSampahCode:       &tenantCode,
		DataBytesTransferred: 50000000,
		PagePath:             "/dashboard",
		VisitorType:          "new",
	}
	body, _ = json.Marshal(telemetryReq)
	telemResp, err := http.Post(harness.Server.URL+"/api/v1/public/impact/telemetry", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer telemResp.Body.Close()
	require.Equal(t, http.StatusCreated, telemResp.StatusCode)

	var telemResult impact.TelemetryResponse
	err = json.NewDecoder(telemResp.Body).Decode(&telemResult)
	require.NoError(t, err)
	assert.Equal(t, int64(1778400), telemResult.EstimatedCO2eMicrograms)
	assert.Equal(t, int64(2), telemResult.EstimatedCO2eGrams)
	assert.Equal(t, "SWDM v4", telemResult.SWDMVersion)

	// 4. Create Nasabah, Material, Deposit to generate Avoided Emissions
	createNasabahReq := nasabah.CreateNasabahRequest{
		FullName:     "Nasabah Lestari",
		Username:     fmt.Sprintf("nasabah_%d", time.Now().UnixNano()%100000),
		Password:     "password123",
		RW:           "08",
		AddressNotes: "Jl. Lestari No. 10",
	}
	body, _ = json.Marshal(createNasabahReq)
	req, _ := http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/nasabah", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	nasResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer nasResp.Body.Close()
	require.Equal(t, http.StatusCreated, nasResp.StatusCode)

	var nasResult nasabah.CreateNasabahResponse
	err = json.NewDecoder(nasResp.Body).Decode(&nasResult)
	require.NoError(t, err)
	nasabahID := nasResult.Nasabah.ID

	// Fetch emission factors to get Kardus factor ID
	factorsResp, err := http.Get(harness.Server.URL + "/api/v1/carbon/factors")
	require.NoError(t, err)
	defer factorsResp.Body.Close()
	var factorsResult struct {
		Factors []struct {
			ID           uuid.UUID `json:"id"`
			MaterialCode string    `json:"material_code"`
		} `json:"factors"`
	}
	err = json.NewDecoder(factorsResp.Body).Decode(&factorsResult)
	require.NoError(t, err)

	var kardusFactorID uuid.UUID
	for _, f := range factorsResult.Factors {
		if f.MaterialCode == "corrugated_containers" {
			kardusFactorID = f.ID
			break
		}
	}
	require.NotEqual(t, uuid.Nil, kardusFactorID)

	// Create Material
	matReq := material.CreateMaterialRequest{
		Name:             "Kardus Tebal",
		Category:         "kertas",
		Unit:             "kg",
		PricePerKgIDR:    3000,
		EmissionFactorID: &kardusFactorID,
	}
	body, _ = json.Marshal(matReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/materials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	matResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer matResp.Body.Close()
	require.Equal(t, http.StatusCreated, matResp.StatusCode)

	var matResult material.Material
	err = json.NewDecoder(matResp.Body).Decode(&matResult)
	require.NoError(t, err)
	materialID := matResult.ID

	// Create Deposit: 10,000 grams of Kardus
	// Kardus factor is 3,428,188 ppm -> 10,000 * 3,428,188 / 1,000,000 = 34,281 grams avoided CO2e
	depReq := deposit.CreateDepositRequest{
		IdempotencyKey:   fmt.Sprintf("dep-impact-%d", time.Now().UnixNano()),
		NasabahID:        nasabahID,
		ClientRecordedAt: time.Now().UTC().Add(-5 * time.Minute),
		Lines: []deposit.CreateDepositLineRequest{
			{
				MaterialTypeID: materialID,
				WeightGrams:    10000,
			},
		},
	}
	body, _ = json.Marshal(depReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	depResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer depResp.Body.Close()
	require.Equal(t, http.StatusCreated, depResp.StatusCode)

	// 5. Check Public Platform Impact
	// Avoided: 34,281 grams
	// Digital Footprint: 2 grams
	// Net Impact: 34,281 - 2 = 34,279 grams
	resp, err = http.Get(harness.Server.URL + "/api/v1/public/impact")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var platformImpact impact.NetImpactSummary
	err = json.NewDecoder(resp.Body).Decode(&platformImpact)
	require.NoError(t, err)

	assert.Equal(t, int64(34282), platformImpact.TotalAvoidedCO2eGrams)
	assert.Equal(t, int64(2), platformImpact.TotalDigitalFootprintCO2eGrams)
	assert.Equal(t, int64(34280), platformImpact.NetImpactCO2eGrams)
	assert.Greater(t, platformImpact.SustainabilityRatio, float64(1000))
	assert.Equal(t, int64(50000000), platformImpact.TotalDataBytesTransferred)

	// Segments check:
	// Total digital footprint = 2 grams
	// Datacenter 15% -> round(2 * 0.15) = 0
	// Network 14% -> 0
	// User device 52% -> round(2 * 0.52) = 1
	// Embodied 19% -> 0
	assert.GreaterOrEqual(t, platformImpact.DigitalFootprintSegments.UserDeviceGrams, int64(1))

	// 6. Check Public Tenant Impact: GET /api/v1/public/impact/tenants/:code
	resp, err = http.Get(harness.Server.URL + "/api/v1/public/impact/tenants/" + tenantCode)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var tenantImpact impact.NetImpactSummary
	err = json.NewDecoder(resp.Body).Decode(&tenantImpact)
	require.NoError(t, err)
	assert.Equal(t, int64(34282), tenantImpact.TotalAvoidedCO2eGrams)
	assert.Equal(t, int64(2), tenantImpact.TotalDigitalFootprintCO2eGrams)
	assert.Equal(t, int64(34280), tenantImpact.NetImpactCO2eGrams)
	assert.Equal(t, tenantCode, tenantImpact.BankSampahCode)

	// 7. Check Non-existent Tenant Code -> 404
	resp, err = http.Get(harness.Server.URL + "/api/v1/public/impact/tenants/non-existent-code-xyz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// 8. Check Authenticated Tenant Impact: GET /api/v1/impact/tenant
	// Unauthenticated -> 401
	resp, err = http.Get(harness.Server.URL + "/api/v1/impact/tenant")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Authenticated with Pengelola token -> 200
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/impact/tenant", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	authResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer authResp.Body.Close()
	require.Equal(t, http.StatusOK, authResp.StatusCode)

	var authTenantImpact impact.NetImpactSummary
	err = json.NewDecoder(authResp.Body).Decode(&authTenantImpact)
	require.NoError(t, err)
	assert.Equal(t, int64(34282), authTenantImpact.TotalAvoidedCO2eGrams)
	assert.Equal(t, int64(2), authTenantImpact.TotalDigitalFootprintCO2eGrams)
	assert.Equal(t, int64(34280), authTenantImpact.NetImpactCO2eGrams)
}
