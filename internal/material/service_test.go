package material_test

import (
	"context"
	"testing"

	"github.com/audyfh/setor-hijau-be/internal/material"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMaterialValidation(t *testing.T) {
	svc := material.NewService(nil)
	ctx := context.Background()
	pengelolaID := uuid.New()
	tenantID := uuid.New()

	// Negative initial price
	_, err := svc.CreateMaterial(ctx, pengelolaID, tenantID, material.CreateMaterialRequest{
		Name:          "Kardus",
		Category:      "kertas",
		PricePerKgIDR: -500,
	})
	assert.ErrorIs(t, err, material.ErrInvalidPrice)

	// Negative updated price
	_, err = svc.UpdatePrice(ctx, pengelolaID, uuid.New(), tenantID, material.UpdatePriceRequest{
		PricePerKgIDR: -100,
	})
	assert.ErrorIs(t, err, material.ErrInvalidPrice)
}
