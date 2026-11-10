package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/cashbook"
	"github.com/audyfh/setor-hijau-be/internal/deposit"
	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/audyfh/setor-hijau-be/internal/report"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonthlyDashboardAndReportExports(t *testing.T) {
	h := SetupTestHarness(t)

	// 1. Register Tenant and login Pengelola
	regPayload := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Makmur RW 06",
		Code:              "bs-makmur-rw06",
		Address:           "Jl. Kenanga No. 8",
		Rw:                "06",
		Kelurahan:         "Oro-Oro Dowo",
		Kecamatan:         "Klojen",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaFullName: "Pengelola Makmur",
		PengelolaUsername: "pengelola_makmur",
		PengelolaEmail:    "makmur@banksampah.id",
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
		Name:          "Kardus Tebal",
		Category:      "kertas",
		Unit:          "kg",
		PricePerKgIDR: 2000,
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
		FullName: "Siti Aminah",
		Username: "siti06",
		Password: "PasswordSiti123!",
		RW:       "06",
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

	// 4. Record Deposit: 10 kg (10,000 g) * Rp 2,000 = Rp 20,000
	depositReq := deposit.CreateDepositRequest{
		NasabahID:        createdNasabah.Nasabah.ID,
		IdempotencyKey:   uuid.New().String(),
		ClientRecordedAt: time.Now().UTC(),
		Lines: []deposit.CreateDepositLineRequest{
			{
				MaterialTypeID: matResp.ID,
				WeightGrams:    10000,
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

	// 5. Record Sale to Pengepul: 10 kg at Rp 3,000 = Rp 30,000
	saleReq := cashbook.CreateSaleRequest{
		BuyerName: "Lapak Barokah",
		SaleDate:  time.Now().UTC(),
		Lines: []cashbook.CreateSaleLineRequest{
			{
				MaterialTypeID: matResp.ID,
				WeightGrams:    10000,
				PricePerKgIDR:  3000,
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
	resp.Body.Close()

	// 6. Test Monthly Dashboard Endpoint
	currentMonth := time.Now().UTC().Format("2006-01")
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/dashboard/monthly?month=%s", h.Server.URL, currentMonth), nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var dashboard report.MonthlyDashboardResponse
	err = json.NewDecoder(resp.Body).Decode(&dashboard)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, currentMonth, dashboard.Period)
	assert.Equal(t, int64(1), dashboard.Deposits.DepositsCount)
	assert.Equal(t, int64(1), dashboard.Deposits.ActiveNasabahCount)
	assert.Equal(t, int64(10000), dashboard.Deposits.TotalWeightGrams)
	assert.Equal(t, int64(20000), dashboard.Deposits.TotalAmountIDR)

	assert.Equal(t, int64(1), dashboard.Sales.SalesCount)
	assert.Equal(t, int64(10000), dashboard.Sales.TotalWeightGrams)
	assert.Equal(t, int64(30000), dashboard.Sales.TotalAmountIDR)

	// Gross margin: 30,000 - 20,000 = 10,000
	assert.Equal(t, int64(10000), dashboard.ProfitSummary.GrossMarginIDR)

	// Cash on hand: 30,000 (from sale)
	assert.Equal(t, int64(30000), dashboard.Cashbook.TotalCashInIDR)
	assert.Equal(t, int64(30000), dashboard.Cashbook.CurrentCashOnHandIDR)

	// 7. Test Monthly CSV Export Endpoint
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/reports/monthly/csv?month=%s", h.Server.URL, currentMonth), nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.True(t, strings.Contains(resp.Header.Get("Content-Disposition"), "attachment; filename=\"laporan-bulanan-"))

	csvBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// Verify UTF-8 BOM
	assert.Equal(t, byte(0xEF), csvBytes[0])
	assert.Equal(t, byte(0xBB), csvBytes[1])
	assert.Equal(t, byte(0xBF), csvBytes[2])

	csvContent := string(csvBytes)
	assert.True(t, strings.Contains(csvContent, "Bank Sampah Makmur RW 06"))
	assert.True(t, strings.Contains(csvContent, "Lapak Barokah"))
	assert.True(t, strings.Contains(csvContent, "Siti Aminah"))

	// 8. Test Monthly PDF Export Endpoint
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/reports/monthly/pdf?month=%s", h.Server.URL, currentMonth), nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	assert.True(t, strings.Contains(resp.Header.Get("Content-Disposition"), "attachment; filename=\"laporan-bulanan-"))

	pdfBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF-"))
}
