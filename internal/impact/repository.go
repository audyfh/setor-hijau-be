package impact

import (
	"context"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	RecordMeasurement(ctx context.Context, arg sqlc.RecordDigitalFootprintMeasurementParams) (*sqlc.DigitalFootprintMeasurement, error)
	GetPlatformDigitalFootprint(ctx context.Context) (*sqlc.GetPlatformDigitalFootprintSummaryRow, error)
	GetTenantDigitalFootprint(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetTenantDigitalFootprintSummaryRow, error)
	GetPlatformAvoidedCarbon(ctx context.Context) (*sqlc.GetPlatformAvoidedCarbonSummaryRow, error)
	GetTenantAvoidedCarbon(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetTenantAvoidedCarbonSummaryRow, error)
	GetPlatformAvoidedCarbonByCategory(ctx context.Context) ([]sqlc.GetPlatformAvoidedCarbonByCategoryRow, error)
	GetTenantAvoidedCarbonByCategory(ctx context.Context, bankSampahID uuid.UUID) ([]sqlc.GetTenantAvoidedCarbonByCategoryRow, error)
	ListActiveEmissionFactors(ctx context.Context) ([]sqlc.ListActiveEmissionFactorsForMethodsRow, error)
	GetBankSampahByCode(ctx context.Context, code string) (*sqlc.BankSampah, error)
	GetBankSampahByID(ctx context.Context, id uuid.UUID) (*sqlc.BankSampah, error)
}

type pgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *pgRepository) RecordMeasurement(ctx context.Context, arg sqlc.RecordDigitalFootprintMeasurementParams) (*sqlc.DigitalFootprintMeasurement, error) {
	m, err := r.queries.RecordDigitalFootprintMeasurement(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *pgRepository) GetPlatformDigitalFootprint(ctx context.Context) (*sqlc.GetPlatformDigitalFootprintSummaryRow, error) {
	row, err := r.queries.GetPlatformDigitalFootprintSummary(ctx)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetTenantDigitalFootprint(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetTenantDigitalFootprintSummaryRow, error) {
	row, err := r.queries.GetTenantDigitalFootprintSummary(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetPlatformAvoidedCarbon(ctx context.Context) (*sqlc.GetPlatformAvoidedCarbonSummaryRow, error) {
	row, err := r.queries.GetPlatformAvoidedCarbonSummary(ctx)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetTenantAvoidedCarbon(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetTenantAvoidedCarbonSummaryRow, error) {
	row, err := r.queries.GetTenantAvoidedCarbonSummary(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetPlatformAvoidedCarbonByCategory(ctx context.Context) ([]sqlc.GetPlatformAvoidedCarbonByCategoryRow, error) {
	return r.queries.GetPlatformAvoidedCarbonByCategory(ctx)
}

func (r *pgRepository) GetTenantAvoidedCarbonByCategory(ctx context.Context, bankSampahID uuid.UUID) ([]sqlc.GetTenantAvoidedCarbonByCategoryRow, error) {
	return r.queries.GetTenantAvoidedCarbonByCategory(ctx, db.ToPGUUID(bankSampahID))
}

func (r *pgRepository) ListActiveEmissionFactors(ctx context.Context) ([]sqlc.ListActiveEmissionFactorsForMethodsRow, error) {
	return r.queries.ListActiveEmissionFactorsForMethods(ctx)
}

func (r *pgRepository) GetBankSampahByCode(ctx context.Context, code string) (*sqlc.BankSampah, error) {
	b, err := r.queries.GetBankSampahByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *pgRepository) GetBankSampahByID(ctx context.Context, id uuid.UUID) (*sqlc.BankSampah, error) {
	b, err := r.queries.GetBankSampahByID(ctx, db.ToPGUUID(id))
	if err != nil {
		return nil, err
	}
	return &b, nil
}
