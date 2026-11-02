package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/audyfh/setor-hijau-be/test/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterialAndPriceHistory(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Register a tenant
	regReq := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Asri RW 01",
		Code:              "bs-asri-rw01",
		Address:           "Jl. Anggrek No. 1",
		Rw:                "01",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaUsername: "pengelola_asri",
		PengelolaEmail:    "pengelola.asri@example.com",
		PengelolaPassword: "password123",
		PengelolaFullName: "Pengelola Asri",
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

	// 2. Create Material with initial price of Rp 2.500
	createMatReq := material.CreateMaterialRequest{
		Name:          "Kardus Bekas",
		Category:      "kertas",
		Unit:          "kg",
		Description:   "Kardus kering dan tidak basah",
		PricePerKgIDR: 2500,
	}
	body, _ = json.Marshal(createMatReq)
	req, _ := http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/materials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdMat material.Material
	err = json.NewDecoder(resp.Body).Decode(&createdMat)
	require.NoError(t, err)
	assert.Equal(t, "Kardus Bekas", createdMat.Name)
	assert.Equal(t, int64(2500), createdMat.PricePerKgIDR)
	assert.NotNil(t, createdMat.PriceEffectiveFrom)
	materialID := createdMat.ID

	// 3. Update price to Rp 3.000
	updatePriceReq := material.UpdatePriceRequest{
		PricePerKgIDR: 3000,
	}
	body, _ = json.Marshal(updatePriceReq)
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/materials/%s/prices", harness.Server.URL, materialID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updatedMat material.Material
	err = json.NewDecoder(resp.Body).Decode(&updatedMat)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), updatedMat.PricePerKgIDR)

	// 4. Acceptance Criterion: Price changes do not alter past price history
	// Check price history endpoint
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/materials/%s/prices", harness.Server.URL, materialID), nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var history []*material.PriceHistoryEntry
	err = json.NewDecoder(resp.Body).Decode(&history)
	require.NoError(t, err)
	require.Len(t, history, 2)

	// The newest price (Rp 3.000) has EffectiveTo == nil
	assert.Equal(t, int64(3000), history[0].PricePerKgIDR)
	assert.Nil(t, history[0].EffectiveTo)

	// The previous price (Rp 2.500) has EffectiveTo != nil
	assert.Equal(t, int64(2500), history[1].PricePerKgIDR)
	assert.NotNil(t, history[1].EffectiveTo)
}
