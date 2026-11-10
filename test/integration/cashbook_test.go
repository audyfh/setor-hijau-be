package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/cashbook"
	"github.com/audyfh/setor-hijau-be/internal/deposit"
	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCashbookAndReconciliationWorkflow(t *testing.T) {
	h := SetupTestHarness(t)

	// 1. Register Tenant A and login Pengelola
	regPayload := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Lestari RW 04",
		Code:              "bs-lestari-rw04",
		Address:           "Jl. Mawar No. 10",
		Rw:                "04",
		Kelurahan:         "Sukamaju",
		Kecamatan:         "Klojen",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaFullName: "Pengelola Lestari",
		PengelolaUsername: "pengelola_lestari",
		PengelolaEmail:    "lestari@banksampah.id",
		PengelolaPassword: "PasswordKuat123!",
	}
	body, _ := json.Marshal(regPayload)
	resp, err := http.Post(h.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regResp struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	err = json.NewDecoder(resp.Body).Decode(&regResp)
	require.NoError(t, err)
	resp.Body.Close()
	pengelolaToken := regResp.Tokens.AccessToken

	// 2. Setup Material
	matReq := material.CreateMaterialRequest{
		Name:          "Botol PET Bening",
		Category:      "plastik",
		Unit:          "kg",
		PricePerKgIDR: 3000,
	}
	body, _ = json.Marshal(matReq)
	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/materials", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var matResp material.Material
	err = json.NewDecoder(resp.Body).Decode(&matResp)
	require.NoError(t, err)
	resp.Body.Close()

	// 3. Register Nasabah
	nasabahReq := nasabah.CreateNasabahRequest{
		FullName: "Ahmad Dahlan",
		Username: "ahmad04",
		Password: "PasswordNasabah123!",
		RW:       "04",
	}
	body, _ = json.Marshal(nasabahReq)
	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/nasabah", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdNasabah nasabah.CreateNasabahResponse
	err = json.NewDecoder(resp.Body).Decode(&createdNasabah)
	require.NoError(t, err)
	resp.Body.Close()
	nasabahID := createdNasabah.Nasabah.ID

	// Login Nasabah
	nasabahLoginPayload := map[string]string{
		"identifier":       "ahmad04",
		"password":         "PasswordNasabah123!",
		"bank_sampah_code": "bs-lestari-rw04",
	}
	body, _ = json.Marshal(nasabahLoginPayload)
	resp, err = http.Post(h.Server.URL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var nasabahLoginResp struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	err = json.NewDecoder(resp.Body).Decode(&nasabahLoginResp)
	require.NoError(t, err)
	resp.Body.Close()
	nasabahToken := nasabahLoginResp.Tokens.AccessToken

	// 4. Record Deposit: 20 kg (20,000 g) * Rp 3,000/kg = Rp 60,000
	depositReq := deposit.CreateDepositRequest{
		NasabahID:        nasabahID,
		IdempotencyKey:   uuid.New().String(),
		ClientRecordedAt: time.Now().UTC(),
		Lines: []deposit.CreateDepositLineRequest{
			{
				MaterialTypeID: matResp.ID,
				WeightGrams:    20000,
			},
		},
	}
	body, _ = json.Marshal(depositReq)
	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/deposits", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Verify balance = Rp 60,000
	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/nasabah/me/balance", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var balResp deposit.NasabahBalanceResponse
	err = json.NewDecoder(resp.Body).Decode(&balResp)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, int64(60000), balResp.CurrentBalanceIDR)
	assert.Equal(t, int64(60000), balResp.TotalDepositedIDR)
	assert.Equal(t, int64(0), balResp.TotalWithdrawnIDR)

	// 5. Nasabah Withdrawal
	// Attempt overdraft (Rp 100,000 > Rp 60,000) -> Should fail 422
	overdraftReq := cashbook.CreateWithdrawalRequest{
		NasabahID: nasabahID,
		AmountIDR: 100000,
	}
	body, _ = json.Marshal(overdraftReq)
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/nasabah/%s/withdrawals", h.Server.URL, nasabahID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	resp.Body.Close()

	// Successful withdrawal: Rp 25,000
	validWithdrawalReq := cashbook.CreateWithdrawalRequest{
		NasabahID: nasabahID,
		AmountIDR: 25000,
	}
	body, _ = json.Marshal(validWithdrawalReq)
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/nasabah/%s/withdrawals", h.Server.URL, nasabahID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var withdrawalResp cashbook.WithdrawalResponse
	err = json.NewDecoder(resp.Body).Decode(&withdrawalResp)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, int64(25000), withdrawalResp.AmountIDR)
	assert.Equal(t, int64(35000), withdrawalResp.RemainingBalanceIDR)

	// Verify derived balance is now Rp 35,000
	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/nasabah/me/balance", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&balResp)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, int64(35000), balResp.CurrentBalanceIDR)
	assert.Equal(t, int64(60000), balResp.TotalDepositedIDR)
	assert.Equal(t, int64(25000), balResp.TotalWithdrawnIDR)

	// Nasabah views their own withdrawal history
	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/nasabah/me/withdrawals", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var myWithdrawals cashbook.WithdrawalListResponse
	err = json.NewDecoder(resp.Body).Decode(&myWithdrawals)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, int64(1), myWithdrawals.TotalCount)
	assert.Equal(t, int64(25000), myWithdrawals.Data[0].AmountIDR)

	// 6. Record Sale to Pengepul: 30 kg (30,000 g) at Rp 4,500/kg = Rp 135,000
	saleNotes := "Penjualan batch 1"
	saleReq := cashbook.CreateSaleRequest{
		BuyerName: "Lapak Jaya Abadi",
		SaleDate:  time.Now().UTC(),
		Notes:     &saleNotes,
		Lines: []cashbook.CreateSaleLineRequest{
			{
				MaterialTypeID: matResp.ID,
				WeightGrams:    30000,
				PricePerKgIDR:  4500,
			},
		},
	}
	body, _ = json.Marshal(saleReq)
	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/cashbook/sales", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var saleResp cashbook.SaleResponse
	err = json.NewDecoder(resp.Body).Decode(&saleResp)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, int64(135000), saleResp.TotalAmountIDR)
	assert.Equal(t, int64(30000), saleResp.TotalWeightGrams)
	assert.Len(t, saleResp.Lines, 1)

	// Verify GetSaleByID
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/cashbook/sales/%s", h.Server.URL, saleResp.ID), nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 7. Manual Cashbook Entry (Biaya operasional beli tali rafia Rp 10,000)
	manualEntry := cashbook.CreateCashbookEntryRequest{
		EntryType:   "cash_out",
		Category:    "operasional",
		AmountIDR:   10000,
		Description: "Beli tali rafia dan karung",
	}
	body, _ = json.Marshal(manualEntry)
	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/cashbook/entries", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// 8. Verify Cashbook Summary
	// Cash In: Penjualan pengepul = Rp 135,000
	// Cash Out: Penarikan nasabah (Rp 25,000) + Operasional (Rp 10,000) = Rp 35,000
	// Current Balance: 135,000 - 35,000 = Rp 100,000
	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/cashbook/summary", nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var cashSummary cashbook.CashbookSummaryResponse
	err = json.NewDecoder(resp.Body).Decode(&cashSummary)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, int64(135000), cashSummary.TotalCashInIDR)
	assert.Equal(t, int64(35000), cashSummary.TotalCashOutIDR)
	assert.Equal(t, int64(100000), cashSummary.CurrentCashBalanceIDR)

	// 9. Verify Financial Reconciliation
	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/cashbook/reconciliation", nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var recon cashbook.FinancialReconciliationResponse
	err = json.NewDecoder(resp.Body).Decode(&recon)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, int64(135000), recon.TotalSalesToPengepulIDR)
	assert.Equal(t, int64(135000), recon.TotalCashbookSalesIDR)
	assert.True(t, recon.SalesReconciled)

	assert.Equal(t, int64(25000), recon.TotalNasabahWithdrawalsIDR)
	assert.Equal(t, int64(25000), recon.TotalCashbookWithdrawalsIDR)
	assert.True(t, recon.WithdrawalsReconciled)

	assert.Equal(t, int64(100000), recon.CurrentCashOnHandIDR)
	assert.Equal(t, int64(35000), recon.TotalOutstandingNasabahBalanceIDR)
	// Net Surplus = Cash on hand (100,000) - Nasabah liability (35,000) = 65,000
	assert.Equal(t, int64(65000), recon.NetSurplusOrDeficitIDR)
}

func TestCashbookTenantIsolation(t *testing.T) {
	h := SetupTestHarness(t)

	// Register Tenant A
	regA := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah A",
		Code:              "bs-a-rw01",
		Address:           "Jl. A",
		Rw:                "01",
		Kelurahan:         "Kel A",
		Kecamatan:         "Kec A",
		KabupatenKota:     "Kota A",
		Provinsi:          "Prov A",
		PengelolaFullName: "Pengelola A",
		PengelolaUsername: "pengelola_a",
		PengelolaEmail:    "a@banksampah.id",
		PengelolaPassword: "Password123!",
	}
	body, _ := json.Marshal(regA)
	respA, _ := http.Post(h.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewBuffer(body))
	var regRespA struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	_ = json.NewDecoder(respA.Body).Decode(&regRespA)
	respA.Body.Close()

	// Register Tenant B
	regB := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah B",
		Code:              "bs-b-rw02",
		Address:           "Jl. B",
		Rw:                "02",
		Kelurahan:         "Kel B",
		Kecamatan:         "Kec B",
		KabupatenKota:     "Kota B",
		Provinsi:          "Prov B",
		PengelolaFullName: "Pengelola B",
		PengelolaUsername: "pengelola_b",
		PengelolaEmail:    "b@banksampah.id",
		PengelolaPassword: "Password123!",
	}
	body, _ = json.Marshal(regB)
	respB, _ := http.Post(h.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewBuffer(body))
	var regRespB struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	_ = json.NewDecoder(respB.Body).Decode(&regRespB)
	respB.Body.Close()

	tokenA := regRespA.Tokens.AccessToken
	tokenB := regRespB.Tokens.AccessToken

	// Tenant A creates manual entry: Rp 500,000 cash in
	manualA := cashbook.CreateCashbookEntryRequest{
		EntryType:   "cash_in",
		Category:    "modal_awal",
		AmountIDR:   500000,
		Description: "Modal kas awal Bank Sampah A",
	}
	body, _ = json.Marshal(manualA)
	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/cashbook/entries", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Tenant B checks summary -> should be 0
	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/cashbook/summary", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	resp, _ = http.DefaultClient.Do(req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var summaryB cashbook.CashbookSummaryResponse
	_ = json.NewDecoder(resp.Body).Decode(&summaryB)
	resp.Body.Close()
	assert.Equal(t, int64(0), summaryB.TotalCashInIDR)
	assert.Equal(t, int64(0), summaryB.CurrentCashBalanceIDR)
}
