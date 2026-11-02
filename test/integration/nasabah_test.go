package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/auth"
	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/audyfh/setor-hijau-be/test/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNasabahManagementAndQRAuth(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Register a tenant with Pengelola
	regReq := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Maju RW 05",
		Code:              "bs-maju-rw05",
		Address:           "Jl. Melati No. 5",
		Rw:                "05",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaUsername: "pengelola_maju",
		PengelolaEmail:    "pengelola.maju@example.com",
		PengelolaPassword: "password123",
		PengelolaFullName: "Pengelola Maju",
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

	// 2. Pengelola creates a Nasabah
	createNasabahReq := nasabah.CreateNasabahRequest{
		Username:     "nasabah_budi",
		FullName:     "Budi Santoso",
		Password:     "nasabahpassword123",
		Email:        "budi@example.com",
		Phone:        "081234567890",
		RW:           "05",
		RT:           "02",
		AddressNotes: "Rumah cat hijau",
		PIN:          "123456",
	}
	body, _ = json.Marshal(createNasabahReq)
	req, _ := http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/nasabah", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createNasabahResp nasabah.CreateNasabahResponse
	err = json.NewDecoder(resp.Body).Decode(&createNasabahResp)
	require.NoError(t, err)
	assert.NotEmpty(t, createNasabahResp.QRToken)
	assert.Equal(t, "nasabah_budi", createNasabahResp.Nasabah.Username)
	assert.True(t, createNasabahResp.Nasabah.HasPIN)
	qrToken := createNasabahResp.QRToken
	nasabahID := createNasabahResp.Nasabah.ID

	// 3. Acceptance Criterion: QR TOKEN ALONE CANNOT AUTHENTICATE
	// 3a. Request without PIN or invalid body
	emptyPINReq := map[string]string{
		"qr_token": qrToken,
		"pin":      "",
	}
	body, _ = json.Marshal(emptyPINReq)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login/qr", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// 3b. Request with wrong PIN -> 401 Unauthorized
	wrongPINReq := auth.QRLoginRequest{
		QRToken: qrToken,
		PIN:     "999999",
	}
	body, _ = json.Marshal(wrongPINReq)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login/qr", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// 3c. Test Lockout after 5 failed attempts (attempt 2, 3, 4, 5)
	for i := 2; i <= 5; i++ {
		body, _ = json.Marshal(wrongPINReq)
		resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login/qr", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		if i < 5 {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
		}
	}

	// 3d. While locked out, even the correct PIN is rejected
	validPINReq := auth.QRLoginRequest{
		QRToken: qrToken,
		PIN:     "123456",
	}
	body, _ = json.Marshal(validPINReq)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login/qr", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	// 4. Pengelola resets Nasabah PIN -> unlocks account and sets new PIN
	resetPINReq := nasabah.ResetPINRequest{
		PIN: "654321",
	}
	body, _ = json.Marshal(resetPINReq)
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/nasabah/%s/reset-pin", harness.Server.URL, nasabahID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 5. Nasabah logs in with QR + newly reset PIN
	newPINReq := auth.QRLoginRequest{
		QRToken: qrToken,
		PIN:     "654321",
	}
	body, _ = json.Marshal(newPINReq)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login/qr", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var qrLoginResp struct {
		Tokens auth.TokenPair    `json:"tokens"`
		User   auth.UserResponse `json:"user"`
	}
	err = json.NewDecoder(resp.Body).Decode(&qrLoginResp)
	require.NoError(t, err)
	nasabahToken := qrLoginResp.Tokens.AccessToken
	assert.Equal(t, "nasabah", qrLoginResp.User.Role)

	// 6. Nasabah accesses /api/v1/nasabah/me with token
	req, _ = http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/nasabah/me", nil)
	req.Header.Set("Authorization", "Bearer "+nasabahToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var meResp nasabah.Nasabah
	err = json.NewDecoder(resp.Body).Decode(&meResp)
	require.NoError(t, err)
	assert.Equal(t, "nasabah_budi", meResp.Username)
	assert.Equal(t, "05", meResp.RW)

	// 7. Nasabah primary login (Username + Password + Bank Sampah Code)
	pwLoginReq := auth.LoginRequest{
		Identifier:     "nasabah_budi",
		Password:       "nasabahpassword123",
		BankSampahCode: "bs-maju-rw05",
	}
	body, _ = json.Marshal(pwLoginReq)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 8. Pengelola scans QR token for Lookup
	lookupReq := nasabah.LookupQRRequest{
		QRToken: qrToken,
	}
	body, _ = json.Marshal(lookupReq)
	req, _ = http.NewRequest(http.MethodPost, harness.Server.URL+"/api/v1/nasabah/lookup-qr", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var lookupResp nasabah.Nasabah
	err = json.NewDecoder(resp.Body).Decode(&lookupResp)
	require.NoError(t, err)
	assert.Equal(t, "nasabah_budi", lookupResp.Username)

	// 9. Pengelola regenerates QR token
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/nasabah/%s/regenerate-qr", harness.Server.URL, nasabahID), nil)
	req.Header.Set("Authorization", "Bearer "+pengelolaToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var regenResp nasabah.RegenerateQRResponse
	err = json.NewDecoder(resp.Body).Decode(&regenResp)
	require.NoError(t, err)
	assert.NotEmpty(t, regenResp.QRToken)
	assert.NotEqual(t, qrToken, regenResp.QRToken)

	// 10. Old QR token is immediately invalidated for login
	oldQRLogin := auth.QRLoginRequest{
		QRToken: qrToken,
		PIN:     "654321",
	}
	body, _ = json.Marshal(oldQRLogin)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login/qr", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// New QR token works!
	newQRLogin := auth.QRLoginRequest{
		QRToken: regenResp.QRToken,
		PIN:     "654321",
	}
	body, _ = json.Marshal(newQRLogin)
	resp, err = http.Post(harness.Server.URL+"/api/v1/auth/login/qr", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
