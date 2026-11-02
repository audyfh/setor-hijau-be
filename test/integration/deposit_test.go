package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/deposit"
	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/audyfh/setor-hijau-be/test/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepositWorkflowAndIdempotency(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Register a tenant
	regReq := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Lestari RW 03",
		Code:              "bs-lestari-rw03",
		Address:           "Jl. Anggrek No. 3",
		Rw:                "03",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaUsername: "pengelola_lestari",
		PengelolaEmail:    "pengelola.lestari@example.com",
		PengelolaPassword: "password123",
		PengelolaFullName: "Pengelola Lestari",
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

	// 2. Create Material Types: Kardus (Rp 2.000/kg) and Botol Plastik (Rp 3.500/kg)
	createMat1 := material.CreateMaterialRequest{
		Name:          "Kardus Bekas",
		Category:      "kertas",
		Unit:          "kg",
		PricePerKgIDR: 2000,
	}
	body, _ = json.Marshal(createMat1)
	req, _ := http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/materials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var mat1Resp material.Material
	err = json.NewDecoder(resp.Body).Decode(&mat1Resp)
	require.NoError(t, err)
	kardusID := mat1Resp.ID

	createMat2 := material.CreateMaterialRequest{
		Name:          "Botol PET Bening",
		Category:      "plastik",
		Unit:          "kg",
		PricePerKgIDR: 3500,
	}
	body, _ = json.Marshal(createMat2)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/materials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var mat2Resp material.Material
	err = json.NewDecoder(resp.Body).Decode(&mat2Resp)
	require.NoError(t, err)
	botolID := mat2Resp.ID

	// 3. Register Nasabah
	createNasabahReq := nasabah.CreateNasabahRequest{
		Username:     "nasabah_siti",
		FullName:     "Siti Aminah",
		Password:     "password123",
		Email:        "siti@example.com",
		Phone:        "081299998888",
		RW:           "03",
		RT:           "01",
		AddressNotes: "Rumah Pagar Hitam",
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

	// Authenticate as Nasabah to get nasabahToken
	loginReq := map[string]string{
		"identifier":        "nasabah_siti",
		"password":          "password123",
		"bank_sampah_code": "bs-lestari-rw03",
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

	// 4. Create Deposit with 2 lines:
	// - Line 1: 1.500 gram Kardus @ Rp 2.000/kg = Rp 3.000
	// - Line 2: 750 gram Botol PET @ Rp 3.500/kg = (750 * 3500 + 500) / 1000 = Rp 2.625
	// Total weight: 2.250 grams, Total amount: Rp 5.625
	idempotencyKey1 := uuid.New().String()
	recordedAt1 := time.Now().UTC().Add(-10 * time.Minute)
	noteText := "Setoran rutin hari Minggu"

	depReq1 := deposit.CreateDepositRequest{
		NasabahID:        nasabahID,
		IdempotencyKey:   idempotencyKey1,
		ClientRecordedAt: recordedAt1,
		Notes:            &noteText,
		Lines: []deposit.CreateDepositLineRequest{
			{MaterialTypeID: kardusID, WeightGrams: 1500},
			{MaterialTypeID: botolID, WeightGrams: 750},
		},
	}

	body, _ = json.Marshal(depReq1)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdDep deposit.DepositResponse
	err = json.NewDecoder(resp.Body).Decode(&createdDep)
	require.NoError(t, err)

	assert.Equal(t, int64(2250), createdDep.TotalWeightGrams)
	assert.Equal(t, int64(5625), createdDep.TotalAmountIDR)
	assert.Equal(t, 2, len(createdDep.Lines))
	assert.Equal(t, int64(3000), createdDep.Lines[0].SubtotalIDR)
	assert.Equal(t, int64(2625), createdDep.Lines[1].SubtotalIDR)

	// 5. TEST IDEMPOTENCY / REPLAY:
	// Resend the exact same request with same idempotency key
	body, _ = json.Marshal(depReq1)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	// Must return 200 OK (replayed)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var replayedDep deposit.DepositResponse
	err = json.NewDecoder(resp.Body).Decode(&replayedDep)
	require.NoError(t, err)
	assert.Equal(t, createdDep.ID, replayedDep.ID)
	assert.Equal(t, createdDep.TotalAmountIDR, replayedDep.TotalAmountIDR)

	// Check balance after replay: balance must still be exactly 5.625 (no duplicate credit)
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/nasabah/me/balance", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var balanceResp deposit.NasabahBalanceResponse
	err = json.NewDecoder(resp.Body).Decode(&balanceResp)
	require.NoError(t, err)
	assert.Equal(t, int64(5625), balanceResp.CurrentBalanceIDR)
	assert.Equal(t, int64(2250), balanceResp.TotalWeightGrams)
	assert.Equal(t, int64(1), balanceResp.TotalDepositsCount)

	// 6. TEST HISTORICAL PRICE SNAPSHOTTING:
	// Update Kardus price from Rp 2.000 to Rp 3.000
	updatePriceReq := material.UpdatePriceRequest{
		PricePerKgIDR: 3000,
	}
	body, _ = json.Marshal(updatePriceReq)
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/materials/%s/price", harness.Server.URL, kardusID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Deposit 2A: Recorded BEFORE price update (should use old price 2.000)
	// 1.000 gram Kardus @ 2.000 = Rp 2.000
	depReqOldPrice := deposit.CreateDepositRequest{
		NasabahID:        nasabahID,
		IdempotencyKey:   uuid.New().String(),
		ClientRecordedAt: time.Now().UTC().Add(-5 * time.Minute), // recorded before update
		Lines: []deposit.CreateDepositLineRequest{
			{MaterialTypeID: kardusID, WeightGrams: 1000},
		},
	}
	body, _ = json.Marshal(depReqOldPrice)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var depOldPriceResp deposit.DepositResponse
	err = json.NewDecoder(resp.Body).Decode(&depOldPriceResp)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), depOldPriceResp.TotalAmountIDR)
	assert.Equal(t, int64(2000), depOldPriceResp.Lines[0].PricePerKgIDR)

	// Deposit 2B: Recorded AFTER price update (should use new price 3.000)
	// 1.000 gram Kardus @ 3.000 = Rp 3.000
	depReqNewPrice := deposit.CreateDepositRequest{
		NasabahID:        nasabahID,
		IdempotencyKey:   uuid.New().String(),
		ClientRecordedAt: time.Now().UTC(),
		Lines: []deposit.CreateDepositLineRequest{
			{MaterialTypeID: kardusID, WeightGrams: 1000},
		},
	}
	body, _ = json.Marshal(depReqNewPrice)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var depNewPriceResp deposit.DepositResponse
	err = json.NewDecoder(resp.Body).Decode(&depNewPriceResp)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), depNewPriceResp.TotalAmountIDR)
	assert.Equal(t, int64(3000), depNewPriceResp.Lines[0].PricePerKgIDR)

	// 7. TEST DIGITAL RECEIPT (STRUK DIGITAL):
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/deposits/%s/receipt", harness.Server.URL, createdDep.ID), nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var receiptResp deposit.DigitalReceiptResponse
	err = json.NewDecoder(resp.Body).Decode(&receiptResp)
	require.NoError(t, err)

	assert.Equal(t, createdDep.ID, receiptResp.DepositID)
	assert.NotEmpty(t, receiptResp.ReceiptNumber)
	assert.Equal(t, "Bank Sampah Lestari RW 03", receiptResp.BankSampahName)
	assert.Equal(t, "Siti Aminah", receiptResp.NasabahName)
	assert.Equal(t, "Pengelola Lestari", receiptResp.RecordedByName)
	assert.Equal(t, int64(5625), receiptResp.TotalAmountIDR)
	// Balance after 3 deposits = 5625 + 2000 + 3000 = 10625
	assert.Equal(t, int64(10625), receiptResp.NasabahCurrentBalance)

	// 8. TEST OFFLINE BATCH SYNC:
	batchReq := deposit.BatchSyncRequest{
		Deposits: []deposit.CreateDepositRequest{
			{
				NasabahID:        nasabahID,
				IdempotencyKey:   uuid.New().String(),
				ClientRecordedAt: time.Now().UTC().Add(-30 * time.Minute),
				Lines: []deposit.CreateDepositLineRequest{
					{MaterialTypeID: botolID, WeightGrams: 1000},
				},
			},
			{
				// This one is replayed
				NasabahID:        nasabahID,
				IdempotencyKey:   idempotencyKey1,
				ClientRecordedAt: recordedAt1,
				Lines: []deposit.CreateDepositLineRequest{
					{MaterialTypeID: kardusID, WeightGrams: 1500},
				},
			},
		},
	}
	body, _ = json.Marshal(batchReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits/sync", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var batchResp deposit.BatchSyncResponse
	err = json.NewDecoder(resp.Body).Decode(&batchResp)
	require.NoError(t, err)
	assert.Equal(t, 2, batchResp.TotalProcessed)
	assert.Equal(t, 2, batchResp.TotalSuccess)
	assert.Equal(t, 0, batchResp.TotalFailed)
	assert.Equal(t, "created", batchResp.Results[0].Status)
	assert.Equal(t, "replayed", batchResp.Results[1].Status)

	// 9. TEST ROLE-BASED ACCESS CONTROL:
	// Nasabah tries to create a deposit -> must return 403 Forbidden
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/deposits", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}
