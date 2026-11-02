package nasabah_test

import (
	"context"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/nasabah"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateNasabahValidation(t *testing.T) {
	svc := nasabah.NewService(nil)
	ctx := context.Background()
	pengelolaID := uuid.New()
	tenantID := uuid.New()

	// Invalid PIN (not 6 digits)
	reqInvalidPIN := nasabah.CreateNasabahRequest{
		Username: "nasabah01",
		FullName: "Nasabah Test",
		Password: "password123",
		RW:       "05",
		PIN:      "123", // only 3 digits
	}
	_, err := svc.CreateNasabah(ctx, pengelolaID, tenantID, reqInvalidPIN)
	assert.ErrorIs(t, err, nasabah.ErrInvalidPINFormat)

	// Invalid PIN (non digits)
	reqNonDigitPIN := nasabah.CreateNasabahRequest{
		Username: "nasabah01",
		FullName: "Nasabah Test",
		Password: "password123",
		RW:       "05",
		PIN:      "12345a",
	}
	_, err = svc.CreateNasabah(ctx, pengelolaID, tenantID, reqNonDigitPIN)
	assert.ErrorIs(t, err, nasabah.ErrInvalidPINFormat)
}

func TestResetPINValidation(t *testing.T) {
	svc := nasabah.NewService(nil)
	ctx := context.Background()
	nasabahID := uuid.New()
	tenantID := uuid.New()

	err := svc.ResetPIN(ctx, nasabahID, tenantID, "12345")
	assert.ErrorIs(t, err, nasabah.ErrInvalidPINFormat)

	err = svc.ResetPIN(ctx, nasabahID, tenantID, "1234567")
	assert.ErrorIs(t, err, nasabah.ErrInvalidPINFormat)

	err = svc.ResetPIN(ctx, nasabahID, tenantID, "abcdef")
	assert.ErrorIs(t, err, nasabah.ErrInvalidPINFormat)
}
