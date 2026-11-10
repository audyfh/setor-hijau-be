package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/carbon"
	"github.com/audyfh/setor-hijau-be/internal/deposit"
	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/audyfh/setor-hijau-be/test/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCarbonLedgerWorkflow(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Verify Public Emission Factors endpoint: GET /api/v1/carbon/factors
	factorsResp, err := http.Get(harness.Server.URL + "/api/v1/carbon/factors")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, factorsResp.StatusCode)

	var factorsResult struct {
		Disclaimer string                  `json:"disclaimer"`
		Factors    []carbon.EmissionFactor `json:"factors"`
	}
	err = json.NewDecoder(factorsResp.Body).Decode(&factorsResult)
	require.NoError(t, err)
	assert.NotEmpty(t, factorsResult.Disclaimer)
	assert.GreaterOrEqual(t, len(factorsResult.Factors), 7, "semua 7 faktor emisi WARM v16 harus tersedia")

	var kardusFactorID, petFactorID uuid.UUID
	for _, f := range factorsResult.Factors {
		assert.Equal(t, "WARM v16", f.FactorVersion)
		assert.Contains(t, f.FactorSource, "US EPA Waste Reduction Model")
		if f.MaterialCode == "corrugated_containers" {
			kardusFactorID = f.ID
			assert.Equal(t, int64(3428188), f.CO2eFactorPPM)
		}
		if f.MaterialCode == "pet_bottles" {
			petFactorID = f.ID
			assert.Equal(t, int64(1267658), f.CO2eFactorPPM)
		}
	}
	require.NotEqual(t, uuid.Nil, kardusFactorID)
	require.NotEqual(t, uuid.Nil, petFactorID)

	// 2. Register Tenant A
	regReq := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Karbon Hijau RW 05",
		Code:              "bs-karbon-hijau-rw05",
		Address:           "Jl. Anggrek No. 5",
		Rw:                "05",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaUsername: "pengelola_karbon",
		PengelolaEmail:    "pengelola.karbon@example.com",
		PengelolaPassword: "password123",
		PengelolaFullName: "Budi Pengelola Karbon",
	}

	body, _ := json.Marshal(regReq)
	resp, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regResp struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	err = json.NewDecoder(resp.Body).Decode(&regResp)
	require.NoError(t, err)
	pengelolaToken := regResp.Tokens.AccessToken

	// 3. Create Material Types linked to Emission Factors
	createMat1 := material.CreateMaterialRequest{
		Name:             "Kardus Tebal",
		Category:         "kertas",
		Unit:             "kg",
		EmissionFactorID: &kardusFactorID,
		PricePerKgIDR:    2000,
	}
	body, _ = json.Marshal(createMat1)
	req, _ := http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/materials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var mat1 material.Material
	err = json.NewDecoder(resp.Body).Decode(&mat1)
	require.NoError(t, err)
	kardusMatID := mat1.ID

	createMat2 := material.CreateMaterialRequest{
		Name:             "Botol PET Campur",
		Category:         "plastik",
		Unit:             "kg",
		EmissionFactorID: &petFactorID,
		PricePerKgIDR:    3000,
	}
	body, _ = json.Marshal(createMat2)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/materials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var mat2 material.Material
	err = json.NewDecoder(resp.Body).Decode(&mat2)
	require.NoError(t, err)
	petMatID := mat2.ID

	// 4. Register Nasabah
	createNasabahReq := nasabah.CreateNasabahRequest{
		Username:     "nasabah_dewi",
		FullName:     "Dewi Sartika",
		Password:     "password123",
		RW:           "05",
		AddressNotes: "Blok A No. 12",
	}
	body, _ = json.Marshal(createNasabahReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/nasabah", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var nasabahResp nasabah.CreateNasabahResponse
	err = json.NewDecoder(resp.Body).Decode(&nasabahResp)
	require.NoError(t, err)
	nasabahID := nasabahResp.Nasabah.ID

	// Login Nasabah to obtain token
	loginReq := map[string]string{
		"bank_sampah_code": "bs-karbon-hijau-rw05",
		"identifier":       "nasabah_dewi",
		"password":         "password123",
	}
	body, _ = json.Marshal(loginReq)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var nasabahLoginResp struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	err = json.NewDecoder(resp.Body).Decode(&nasabahLoginResp)
	require.NoError(t, err)
	nasabahToken := nasabahLoginResp.Tokens.AccessToken

	// Initial check: Nasabah carbon balance should be 0
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/balance", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var initialCarbon carbon.NasabahCarbonSavingsResponse
	err = json.NewDecoder(resp.Body).Decode(&initialCarbon)
	require.NoError(t, err)
	assert.Equal(t, int64(0), initialCarbon.TotalEstimatedCO2eAvoidedGrams)

	// 5. Record Deposit: 1500g Kardus (expect 5142g CO2e) + 1000g PET (expect 1268g CO2e)
	idempotencyKey := uuid.New().String()
	recordDepositReq := deposit.CreateDepositRequest{
		NasabahID:        nasabahID,
		IdempotencyKey:   idempotencyKey,
		ClientRecordedAt: time.Now().UTC().Add(-10 * time.Minute),
		Lines: []deposit.CreateDepositLineRequest{
			{MaterialTypeID: kardusMatID, WeightGrams: 1500},
			{MaterialTypeID: petMatID, WeightGrams: 1000},
		},
	}
	body, _ = json.Marshal(recordDepositReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var depResp deposit.DepositResponse
	err = json.NewDecoder(resp.Body).Decode(&depResp)
	require.NoError(t, err)
	assert.Equal(t, int64(2500), depResp.TotalWeightGrams)
	// Hand-verified calculation: 5142 + 1268 = 6410 g CO2e
	assert.Equal(t, int64(6410), depResp.TotalEstimatedCO2eAvoidedGrams)
	require.Len(t, depResp.Lines, 2)
	assert.Equal(t, int64(5142), depResp.Lines[0].EstimatedCO2eAvoidedGrams)
	assert.Equal(t, "WARM v16", depResp.Lines[0].FactorVersion)
	assert.Equal(t, int64(1268), depResp.Lines[1].EstimatedCO2eAvoidedGrams)
	assert.Equal(t, "WARM v16", depResp.Lines[1].FactorVersion)

	// 6. Verify Digital Receipt has CO2e figures
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/deposits/%s/receipt", harness.Server.URL, depResp.ID), nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var receiptResp deposit.DigitalReceiptResponse
	err = json.NewDecoder(resp.Body).Decode(&receiptResp)
	require.NoError(t, err)
	assert.Equal(t, int64(6410), receiptResp.TotalEstimatedCO2eAvoidedGrams)

	// 7. Verify Idempotent Replay does not duplicate Carbon Ledger
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var replayResp deposit.DepositResponse
	err = json.NewDecoder(resp.Body).Decode(&replayResp)
	require.NoError(t, err)
	assert.Equal(t, depResp.ID, replayResp.ID)
	assert.Equal(t, int64(6410), replayResp.TotalEstimatedCO2eAvoidedGrams)

	// 8. Verify Nasabah Carbon Balance
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/balance", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var nasabahCarbon carbon.NasabahCarbonSavingsResponse
	err = json.NewDecoder(resp.Body).Decode(&nasabahCarbon)
	require.NoError(t, err)
	assert.Equal(t, int64(6410), nasabahCarbon.TotalEstimatedCO2eAvoidedGrams)
	assert.Equal(t, int64(2500), nasabahCarbon.TotalWeightGrams)
	assert.Equal(t, int64(2), nasabahCarbon.TotalEntries)
	require.Len(t, nasabahCarbon.BreakdownByCategory, 2)

	// 9. Verify Pengelola View: GET /api/v1/nasabah/:id/carbon and GET /api/v1/carbon/summary
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/nasabah/%s/carbon", harness.Server.URL, nasabahID), nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var pengNasabahCarbon carbon.NasabahCarbonSavingsResponse
	err = json.NewDecoder(resp.Body).Decode(&pengNasabahCarbon)
	require.NoError(t, err)
	assert.Equal(t, int64(6410), pengNasabahCarbon.TotalEstimatedCO2eAvoidedGrams)

	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/summary", nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var tenantSummary carbon.BankSampahCarbonSavingsResponse
	err = json.NewDecoder(resp.Body).Decode(&tenantSummary)
	require.NoError(t, err)
	assert.Equal(t, int64(6410), tenantSummary.TotalEstimatedCO2eAvoidedGrams)
	assert.Equal(t, int64(2500), tenantSummary.TotalWeightGrams)

	// 10. Verify Carbon Ledger listing: GET /api/v1/carbon/ledger
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/ledger", nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var ledgerList carbon.CarbonLedgerListResponse
	err = json.NewDecoder(resp.Body).Decode(&ledgerList)
	require.NoError(t, err)
	require.Len(t, ledgerList.Entries, 2)
	assert.Equal(t, int64(2), ledgerList.TotalCount)

	var kardusEntryID uuid.UUID
	for _, entry := range ledgerList.Entries {
		assert.Equal(t, "deposit", entry.EntryType)
		assert.Equal(t, "WARM v16", entry.FactorVersion)
		if entry.FactorName == "Kardus / Karton Gelombang (Corrugated Containers)" {
			kardusEntryID = entry.ID
			assert.Equal(t, int64(1500), entry.WeightGrams)
			assert.Equal(t, int64(5142), entry.CO2eGrams)
		}
	}
	require.NotEqual(t, uuid.Nil, kardusEntryID)

	// 11. Test Reversal Entry: POST /api/v1/carbon/reversals
	revReq := carbon.CreateReversalRequest{
		CarbonLedgerID: kardusEntryID,
		Notes:          "Koreksi kesalahan input kardus",
	}
	body, _ = json.Marshal(revReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/carbon/reversals", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var revResp carbon.CarbonLedgerEntry
	err = json.NewDecoder(resp.Body).Decode(&revResp)
	require.NoError(t, err)
	assert.Equal(t, "reversal", revResp.EntryType)
	assert.Equal(t, int64(-1500), revResp.WeightGrams)
	assert.Equal(t, int64(-5142), revResp.CO2eGrams)
	assert.Equal(t, &kardusEntryID, revResp.ReversalForID)
	assert.Equal(t, "WARM v16", revResp.FactorVersion)

	// 12. Check updated Nasabah balance after reversal: 6410 - 5142 = 1268 g CO2e
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/balance", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updatedCarbon carbon.NasabahCarbonSavingsResponse
	err = json.NewDecoder(resp.Body).Decode(&updatedCarbon)
	require.NoError(t, err)
	assert.Equal(t, int64(1268), updatedCarbon.TotalEstimatedCO2eAvoidedGrams)

	// 13. Verify that trying to reverse the reversal entry fails with HTTP 400 Bad Request
	revReversalReq := carbon.CreateReversalRequest{
		CarbonLedgerID: revResp.ID,
		Notes:          "Mencoba membalik reversal",
	}
	body, _ = json.Marshal(revReversalReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/carbon/reversals", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCarbonLedgerTenantIsolation(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Register Tenant A
	regA := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah A",
		Code:              "bs-a",
		Address:           "Jl. A No. 1",
		Rw:                "01",
		Kelurahan:         "Kel A",
		Kecamatan:         "Kec A",
		KabupatenKota:     "Kota A",
		Provinsi:          "Prov A",
		PengelolaUsername: "pengelola_a",
		PengelolaEmail:    "pengelola.a@example.com",
		PengelolaPassword: "password123",
		PengelolaFullName: "Pengelola A",
	}
	bodyA, _ := json.Marshal(regA)
	respA, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(bodyA))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, respA.StatusCode)

	var regRespA struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	_ = json.NewDecoder(respA.Body).Decode(&regRespA)
	tokenA := regRespA.Tokens.AccessToken

	// 2. Register Tenant B
	regB := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah B",
		Code:              "bs-b",
		Address:           "Jl. B No. 2",
		Rw:                "02",
		Kelurahan:         "Kel B",
		Kecamatan:         "Kec B",
		KabupatenKota:     "Kota B",
		Provinsi:          "Prov B",
		PengelolaUsername: "pengelola_b",
		PengelolaEmail:    "pengelola.b@example.com",
		PengelolaPassword: "password123",
		PengelolaFullName: "Pengelola B",
	}
	bodyB, _ := json.Marshal(regB)
	respB, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(bodyB))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, respB.StatusCode)

	var regRespB struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	_ = json.NewDecoder(respB.Body).Decode(&regRespB)
	tokenB := regRespB.Tokens.AccessToken

	// 3. In Tenant A: Get factor ID for aluminum_cans and create material
	factorsResp, err := http.Get(harness.Server.URL + "/api/v1/carbon/factors")
	require.NoError(t, err)
	var fResult struct {
		Factors []carbon.EmissionFactor `json:"factors"`
	}
	_ = json.NewDecoder(factorsResp.Body).Decode(&fResult)

	var aluFactorID uuid.UUID
	for _, f := range fResult.Factors {
		if f.MaterialCode == "aluminum_cans" {
			aluFactorID = f.ID
			break
		}
	}
	require.NotEqual(t, uuid.Nil, aluFactorID)

	createMat := material.CreateMaterialRequest{
		Name:             "Kaleng Minuman",
		Category:         "logam",
		Unit:             "kg",
		EmissionFactorID: &aluFactorID,
		PricePerKgIDR:    10000,
	}
	bodyMat, _ := json.Marshal(createMat)
	req, _ := http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/materials", bytes.NewReader(bodyMat))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var matA material.Material
	_ = json.NewDecoder(resp.Body).Decode(&matA)

	// Create Nasabah in Tenant A
	createNasReq := nasabah.CreateNasabahRequest{
		Username: "nasabah_a",
		FullName: "Nasabah A",
		Password: "password123",
		RW:       "01",
	}
	bodyNas, _ := json.Marshal(createNasReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/nasabah", bytes.NewReader(bodyNas))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var nasProfile nasabah.CreateNasabahResponse
	_ = json.NewDecoder(resp.Body).Decode(&nasProfile)
	nasabahAID := nasProfile.Nasabah.ID

	// Record Deposit in Tenant A: 500g Aluminum Cans (expect 5021g CO2e)
	depReq := deposit.CreateDepositRequest{
		NasabahID:        nasabahAID,
		IdempotencyKey:   uuid.New().String(),
		ClientRecordedAt: time.Now().UTC().Add(-5 * time.Minute),
		Lines: []deposit.CreateDepositLineRequest{
			{MaterialTypeID: matA.ID, WeightGrams: 500},
		},
	}
	bodyDep, _ := json.Marshal(depReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(bodyDep))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// 4. Verify Tenant A sees the carbon ledger entry
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/ledger", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var ledgerA carbon.CarbonLedgerListResponse
	_ = json.NewDecoder(resp.Body).Decode(&ledgerA)
	require.Len(t, ledgerA.Entries, 1)
	entryAID := ledgerA.Entries[0].ID

	// 5. Tenant Isolation Checks:
	// A) Tenant B cannot see Tenant A's carbon ledger entries
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/ledger", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var ledgerB carbon.CarbonLedgerListResponse
	_ = json.NewDecoder(resp.Body).Decode(&ledgerB)
	assert.Empty(t, ledgerB.Entries, "Tenant B tidak boleh melihat entri jurnal karbon milik Tenant A")

	// B) Tenant B cannot see Tenant A's tenant summary
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/carbon/summary", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var summaryB carbon.BankSampahCarbonSavingsResponse
	_ = json.NewDecoder(resp.Body).Decode(&summaryB)
	assert.Equal(t, int64(0), summaryB.TotalEstimatedCO2eAvoidedGrams, "Tenant B harus memiliki 0 tabungan karbon")

	// C) Tenant B cannot reverse Tenant A's carbon ledger entry
	revReq := carbon.CreateReversalRequest{
		CarbonLedgerID: entryAID,
		Notes:          "Percobaan hacking reversal antar-tenant",
	}
	bodyRev, _ := json.Marshal(revReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/carbon/reversals", bytes.NewReader(bodyRev))
	req.Header.Set("Authorization", "Bearer "+tokenB)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Tenant B harus menerima 404 ketika mencoba membalik entri milik Tenant A")
}
