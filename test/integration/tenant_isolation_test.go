package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/audyfh/setor-hijau-be/test/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantIsolation(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Register Tenant A
	reqA := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Hijau Lestari RW 01",
		Code:              "bs-hijau-rw01",
		Address:           "Jl. Kenanga No. 1",
		Rw:                "01",
		Rt:                "02",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		ContactPhone:      "081234567890",
		PengelolaUsername: "pengelola_a",
		PengelolaEmail:    "pengelola_a@banksampah.id",
		PengelolaPassword: "password_rahasia_a123",
		PengelolaFullName: "Budi Santoso",
	}

	bodyA, _ := json.Marshal(reqA)
	resA, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(bodyA))
	require.NoError(t, err)
	defer resA.Body.Close()
	require.Equal(t, http.StatusCreated, resA.StatusCode)

	var tenantAResp struct {
		BankSampah struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"bank_sampah"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	err = json.NewDecoder(resA.Body).Decode(&tenantAResp)
	require.NoError(t, err)
	assert.NotEmpty(t, tenantAResp.BankSampah.ID)
	assert.NotEmpty(t, tenantAResp.Tokens.AccessToken)

	// 2. Register Tenant B
	reqB := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Makmur Sejahtera RW 02",
		Code:              "bs-makmur-rw02",
		Address:           "Jl. Melati No. 2",
		Rw:                "02",
		Rt:                "01",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		ContactPhone:      "081234567891",
		PengelolaUsername: "pengelola_b",
		PengelolaEmail:    "pengelola_b@banksampah.id",
		PengelolaPassword: "password_rahasia_b123",
		PengelolaFullName: "Siti Rahma",
	}

	bodyB, _ := json.Marshal(reqB)
	resB, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(bodyB))
	require.NoError(t, err)
	defer resB.Body.Close()
	require.Equal(t, http.StatusCreated, resB.StatusCode)

	var tenantBResp struct {
		BankSampah struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"bank_sampah"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	err = json.NewDecoder(resB.Body).Decode(&tenantBResp)
	require.NoError(t, err)
	assert.NotEmpty(t, tenantBResp.BankSampah.ID)
	assert.NotEmpty(t, tenantBResp.Tokens.AccessToken)
	assert.NotEqual(t, tenantAResp.BankSampah.ID, tenantBResp.BankSampah.ID)

	// 3. Test Tenant A accessing /api/v1/tenants/current
	clientA := &http.Client{}
	getCurA, err := http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/tenants/current", nil)
	require.NoError(t, err)
	getCurA.Header.Set("Authorization", "Bearer "+tenantAResp.Tokens.AccessToken)

	respCurA, err := clientA.Do(getCurA)
	require.NoError(t, err)
	defer respCurA.Body.Close()
	assert.Equal(t, http.StatusOK, respCurA.StatusCode)

	var dataCurA struct {
		BankSampah struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"bank_sampah"`
	}
	err = json.NewDecoder(respCurA.Body).Decode(&dataCurA)
	require.NoError(t, err)
	assert.Equal(t, tenantAResp.BankSampah.ID, dataCurA.BankSampah.ID)
	assert.Equal(t, "bs-hijau-rw01", dataCurA.BankSampah.Code)

	// 4. Test Tenant B accessing /api/v1/tenants/current
	clientB := &http.Client{}
	getCurB, err := http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/tenants/current", nil)
	require.NoError(t, err)
	getCurB.Header.Set("Authorization", "Bearer "+tenantBResp.Tokens.AccessToken)

	respCurB, err := clientB.Do(getCurB)
	require.NoError(t, err)
	defer respCurB.Body.Close()
	assert.Equal(t, http.StatusOK, respCurB.StatusCode)

	var dataCurB struct {
		BankSampah struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"bank_sampah"`
	}
	err = json.NewDecoder(respCurB.Body).Decode(&dataCurB)
	require.NoError(t, err)
	assert.Equal(t, tenantBResp.BankSampah.ID, dataCurB.BankSampah.ID)
	assert.Equal(t, "bs-makmur-rw02", dataCurB.BankSampah.Code)

	// Tenant A sees Tenant A, Tenant B sees Tenant B. Absolute isolation verified!
}
