package material

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// ErrMaterialNameTaken indicates a material with the same name already exists in this tenant.
	ErrMaterialNameTaken = errors.New("nama jenis sampah sudah digunakan di bank sampah ini")
	// ErrInvalidPrice indicates a negative price.
	ErrInvalidPrice = errors.New("harga tidak boleh bernilai negatif")
)

// Service defines business logic for materials and price history.
type Service interface {
	CreateMaterial(ctx context.Context, pengelolaID, bankSampahID uuid.UUID, req CreateMaterialRequest) (*Material, error)
	GetMaterial(ctx context.Context, id, bankSampahID uuid.UUID) (*Material, error)
	ListMaterials(ctx context.Context, bankSampahID uuid.UUID, activeOnly bool) ([]*Material, error)
	UpdateMaterial(ctx context.Context, id, bankSampahID uuid.UUID, req UpdateMaterialRequest) (*Material, error)
	UpdatePrice(ctx context.Context, pengelolaID, id, bankSampahID uuid.UUID, req UpdatePriceRequest) (*Material, error)
	GetPriceHistory(ctx context.Context, id, bankSampahID uuid.UUID) ([]*PriceHistoryEntry, error)
}

type service struct {
	repo Repository
}

// NewService creates a new Material service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateMaterial(ctx context.Context, pengelolaID, bankSampahID uuid.UUID, req CreateMaterialRequest) (*Material, error) {
	if req.PricePerKgIDR < 0 {
		return nil, ErrInvalidPrice
	}

	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		unit = "kg"
	}

	var emissionFactorUUID pgtype.UUID
	if req.EmissionFactorID != nil && *req.EmissionFactorID != uuid.Nil {
		emissionFactorUUID = db.ToPGUUID(*req.EmissionFactorID)
	}

	now := time.Now().UTC()
	var createdMaterialID uuid.UUID

	err := s.repo.ExecTx(ctx, func(q *sqlc.Queries) error {
		m, err := s.repo.CreateMaterialType(ctx, q, sqlc.CreateMaterialTypeParams{
			BankSampahID:     db.ToPGUUID(bankSampahID),
			Name:             strings.TrimSpace(req.Name),
			Category:         strings.TrimSpace(req.Category),
			Unit:             unit,
			Description:      db.ToPGText(strings.TrimSpace(req.Description)),
			EmissionFactorID: emissionFactorUUID,
			IsActive:         true,
		})
		if err != nil {
			if strings.Contains(err.Error(), "uq_material_types_bank_sampah_name") {
				return ErrMaterialNameTaken
			}
			return fmt.Errorf("failed to create material type: %w", err)
		}
		createdMaterialID = db.FromPGUUID(m.ID)

		_, err = s.repo.CreateMaterialPrice(ctx, q, sqlc.CreateMaterialPriceParams{
			MaterialTypeID: m.ID,
			BankSampahID:   db.ToPGUUID(bankSampahID),
			PricePerKgIdr:  req.PricePerKgIDR,
			EffectiveFrom:  db.ToPGTimestamptz(now),
			EffectiveTo:    db.ToPGTimestamptz(time.Time{}), // NULL
			CreatedBy:      db.ToPGUUID(pengelolaID),
		})
		if err != nil {
			return fmt.Errorf("failed to create initial material price: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.repo.GetMaterialTypeByID(ctx, createdMaterialID, bankSampahID)
}

func (s *service) GetMaterial(ctx context.Context, id, bankSampahID uuid.UUID) (*Material, error) {
	return s.repo.GetMaterialTypeByID(ctx, id, bankSampahID)
}

func (s *service) ListMaterials(ctx context.Context, bankSampahID uuid.UUID, activeOnly bool) ([]*Material, error) {
	return s.repo.ListMaterialTypes(ctx, bankSampahID, activeOnly)
}

func (s *service) UpdateMaterial(ctx context.Context, id, bankSampahID uuid.UUID, req UpdateMaterialRequest) (*Material, error) {
	_, err := s.repo.GetMaterialTypeByID(ctx, id, bankSampahID)
	if err != nil {
		return nil, err
	}

	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		unit = "kg"
	}

	var emissionFactorUUID pgtype.UUID
	if req.EmissionFactorID != nil && *req.EmissionFactorID != uuid.Nil {
		emissionFactorUUID = db.ToPGUUID(*req.EmissionFactorID)
	}

	_, err = s.repo.UpdateMaterialType(ctx, nil, sqlc.UpdateMaterialTypeParams{
		Name:             strings.TrimSpace(req.Name),
		Category:         strings.TrimSpace(req.Category),
		Unit:             unit,
		Description:      db.ToPGText(strings.TrimSpace(req.Description)),
		EmissionFactorID: emissionFactorUUID,
		IsActive:         req.IsActive,
		ID:               db.ToPGUUID(id),
		BankSampahID:     db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if strings.Contains(err.Error(), "uq_material_types_bank_sampah_name") {
			return nil, ErrMaterialNameTaken
		}
		return nil, fmt.Errorf("failed to update material type: %w", err)
	}

	return s.repo.GetMaterialTypeByID(ctx, id, bankSampahID)
}

func (s *service) UpdatePrice(ctx context.Context, pengelolaID, id, bankSampahID uuid.UUID, req UpdatePriceRequest) (*Material, error) {
	if req.PricePerKgIDR < 0 {
		return nil, ErrInvalidPrice
	}

	_, err := s.repo.GetMaterialTypeByID(ctx, id, bankSampahID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	err = s.repo.ExecTx(ctx, func(q *sqlc.Queries) error {
		// Close currently active price
		if err := s.repo.CloseActiveMaterialPrice(ctx, q, now, id, bankSampahID); err != nil {
			return fmt.Errorf("failed to close active price: %w", err)
		}

		// Insert new price point
		_, err := s.repo.CreateMaterialPrice(ctx, q, sqlc.CreateMaterialPriceParams{
			MaterialTypeID: db.ToPGUUID(id),
			BankSampahID:   db.ToPGUUID(bankSampahID),
			PricePerKgIdr:  req.PricePerKgIDR,
			EffectiveFrom:  db.ToPGTimestamptz(now),
			EffectiveTo:    db.ToPGTimestamptz(time.Time{}), // NULL
			CreatedBy:      db.ToPGUUID(pengelolaID),
		})
		if err != nil {
			return fmt.Errorf("failed to insert new price: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.repo.GetMaterialTypeByID(ctx, id, bankSampahID)
}

func (s *service) GetPriceHistory(ctx context.Context, id, bankSampahID uuid.UUID) ([]*PriceHistoryEntry, error) {
	_, err := s.repo.GetMaterialTypeByID(ctx, id, bankSampahID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListMaterialPricesHistory(ctx, id, bankSampahID)
}
