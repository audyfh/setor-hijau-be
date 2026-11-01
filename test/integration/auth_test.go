package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/auth"
	"github.com/audyfh/setor-hijau-be/internal/tenant"
	"github.com/audyfh/setor-hijau-be/test/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthLifecycle(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// 1. Register a tenant with pengelola
	regReq := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Bersih RW 03",
		Code:              "bs-bersih-rw03",
		Address:           "Jl. Mawar No. 3",
		Rw:                "03",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaUsername: "pengelola_c",
		PengelolaEmail:    "pengelola_c@banksampah.id",
		PengelolaPassword: "password_kuat_c123",
		PengelolaFullName: "Cahyo Utomo",
	}

	body, _ := json.Marshal(regReq)
	res, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusCreated, res.StatusCode)

	// 2. Test Login with wrong password -> 401
	loginWrong := auth.LoginRequest{
		Identifier: "pengelola_c",
		Password:   "password_yang_salah",
	}
	bodyWrong, _ := json.Marshal(loginWrong)
	resWrong, err := http.Post(harness.Server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(bodyWrong))
	require.NoError(t, err)
	defer resWrong.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resWrong.StatusCode)

	// 3. Test Login with valid username -> 200
	loginValid := auth.LoginRequest{
		Identifier: "pengelola_c",
		Password:   "password_kuat_c123",
	}
	bodyValid, _ := json.Marshal(loginValid)
	resValid, err := http.Post(harness.Server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(bodyValid))
	require.NoError(t, err)
	defer resValid.Body.Close()
	assert.Equal(t, http.StatusOK, resValid.StatusCode)

	var loginResp struct {
		Tokens auth.TokenPair    `json:"tokens"`
		User   auth.UserResponse `json:"user"`
	}
	err = json.NewDecoder(resValid.Body).Decode(&loginResp)
	require.NoError(t, err)
	assert.NotEmpty(t, loginResp.Tokens.AccessToken)
	assert.NotEmpty(t, loginResp.Tokens.RefreshToken)
	assert.Equal(t, "pengelola_c", loginResp.User.Username)

	// 4. Access /api/v1/auth/me with access token
	client := &http.Client{}
	reqMe, err := http.NewRequest(http.MethodGet, harness.Server.URL+"/api/v1/auth/me", nil)
	require.NoError(t, err)
	reqMe.Header.Set("Authorization", "Bearer "+loginResp.Tokens.AccessToken)

	resMe, err := client.Do(reqMe)
	require.NoError(t, err)
	defer resMe.Body.Close()
	assert.Equal(t, http.StatusOK, resMe.StatusCode)

	var meResp struct {
		User auth.UserResponse `json:"user"`
	}
	err = json.NewDecoder(resMe.Body).Decode(&meResp)
	require.NoError(t, err)
	assert.Equal(t, "pengelola_c", meResp.User.Username)
	assert.Equal(t, "pengelola_c@banksampah.id", meResp.User.Email)

	// 5. Test Token Refresh (Token Rotation)
	refreshReq := auth.RefreshTokenRequest{
		RefreshToken: loginResp.Tokens.RefreshToken,
	}
	bodyRefresh, _ := json.Marshal(refreshReq)
	resRefresh, err := http.Post(harness.Server.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(bodyRefresh))
	require.NoError(t, err)
	defer resRefresh.Body.Close()
	assert.Equal(t, http.StatusOK, resRefresh.StatusCode)

	var refreshResp struct {
		Tokens auth.TokenPair `json:"tokens"`
	}
	err = json.NewDecoder(resRefresh.Body).Decode(&refreshResp)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshResp.Tokens.AccessToken)
	assert.NotEmpty(t, refreshResp.Tokens.RefreshToken)
	assert.NotEqual(t, loginResp.Tokens.RefreshToken, refreshResp.Tokens.RefreshToken)

	// Old refresh token must now be rejected (revoked due to rotation)
	resReused, err := http.Post(harness.Server.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(bodyRefresh))
	require.NoError(t, err)
	defer resReused.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resReused.StatusCode)

	// 6. Test Logout
	logoutReq := auth.LogoutRequest{
		RefreshToken: refreshResp.Tokens.RefreshToken,
	}
	bodyLogout, _ := json.Marshal(logoutReq)
	resLogout, err := http.Post(harness.Server.URL+"/api/v1/auth/logout", "application/json", bytes.NewReader(bodyLogout))
	require.NoError(t, err)
	defer resLogout.Body.Close()
	assert.Equal(t, http.StatusOK, resLogout.StatusCode)

	// Refreshing after logout should be rejected
	bodyAfterLogout, _ := json.Marshal(auth.RefreshTokenRequest{RefreshToken: refreshResp.Tokens.RefreshToken})
	resAfterLogout, err := http.Post(harness.Server.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(bodyAfterLogout))
	require.NoError(t, err)
	defer resAfterLogout.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resAfterLogout.StatusCode)
}

func TestPasswordResetFlow(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// Register tenant
	regReq := tenant.RegisterBankSampahRequest{
		Name:              "Bank Sampah Sejahtera RW 04",
		Code:              "bs-sejahtera-rw04",
		Address:           "Jl. Anggrek No. 4",
		Rw:                "04",
		Kelurahan:         "Ketawanggede",
		Kecamatan:         "Lowokwaru",
		KabupatenKota:     "Kota Malang",
		Provinsi:          "Jawa Timur",
		PengelolaUsername: "pengelola_d",
		PengelolaEmail:    "pengelola_d@banksampah.id",
		PengelolaPassword: "password_lama_123",
		PengelolaFullName: "Dian Permata",
	}
	body, _ := json.Marshal(regReq)
	res, err := http.Post(harness.Server.URL+"/api/v1/tenants/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	res.Body.Close()

	// 1. Forgot password request via API
	forgotReq := auth.ForgotPasswordRequest{
		Email: "pengelola_d@banksampah.id",
	}
	bodyForgot, _ := json.Marshal(forgotReq)
	resForgot, err := http.Post(harness.Server.URL+"/api/v1/auth/forgot-password", "application/json", bytes.NewReader(bodyForgot))
	require.NoError(t, err)
	defer resForgot.Body.Close()
	assert.Equal(t, http.StatusOK, resForgot.StatusCode)

	// Read reset token directly from DB to simulate user clicking link from email
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var tokenHash string
	err = harness.DB.Pool.QueryRow(ctx, `
		SELECT prt.token_hash 
		FROM password_reset_tokens prt
		JOIN users u ON u.id = prt.user_id
		WHERE u.email = 'pengelola_d@banksampah.id'
		ORDER BY prt.created_at DESC
		LIMIT 1
	`).Scan(&tokenHash)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenHash)

	// 2. Direct service reset test with invalid token
	resetReqBad := auth.ResetPasswordRequest{
		Token:       "invalid_token_string",
		NewPassword: "password_baru_123",
	}
	bodyBad, _ := json.Marshal(resetReqBad)
	resBad, err := http.Post(harness.Server.URL+"/api/v1/auth/reset-password", "application/json", bytes.NewReader(bodyBad))
	require.NoError(t, err)
	defer resBad.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resBad.StatusCode)
}

func TestLoginRateLimiting(t *testing.T) {
	harness := integration.SetupTestHarness(t)
	if harness == nil {
		return
	}

	// Make many rapid login attempts to trigger rate limit (test harness limiter allows 100/min, so we test with a tight rate limiter directly or send requests)
	loginBad := auth.LoginRequest{
		Identifier: "non_existent_user",
		Password:   "wrong_password",
	}
	bodyBad, _ := json.Marshal(loginBad)

	hit429 := false
	for i := 0; i < 110; i++ {
		res, err := http.Post(harness.Server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(bodyBad))
		require.NoError(t, err)
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests {
			hit429 = true
			break
		}
	}
	assert.True(t, hit429, "expected to hit 429 Too Many Requests after exceeding rate limit")
}
