package carbon

import (
	"context"
	"errors"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles persistence operations for emission factors and the carbon ledger.
type Repository interface {
	ListActiveEmissionFactors(ctx context.Context) ([]sqlc.EmissionFactor, error)
	GetEmissionFactorByID(ctx context.Context, id uuid.UUID) (*sqlc.EmissionFactor, error)
	CreateCarbonLedgerEntry(ctx context.Context, params sqlc.CreateCarbonLedgerEntryParams) (*sqlc.CarbonLedger, error)
	CreateCarbonLedgerEntryTx(ctx context.Context, q *sqlc.Queries, params sqlc.CreateCarbonLedgerEntryParams) (*sqlc.CarbonLedger, error)
	GetNasabahCarbonSavings(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahCarbonSavingsRow, error)
	GetBankSampahCarbonSavings(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetBankSampahCarbonSavingsRow, error)
	GetNasabahCarbonSavingsByCategory(ctx context.Context, bankSampahID, nasabahID uuid.UUID) ([]sqlc.GetNasabahCarbonSavingsByCategoryRow, error)
	ListCarbonLedgerByNasabah(ctx context.Context, params sqlc.ListCarbonLedgerByNasabahParams) ([]sqlc.ListCarbonLedgerByNasabahRow, error)
	CountCarbonLedgerByNasabah(ctx context.Context, params sqlc.CountCarbonLedgerByNasabahParams) (int64, error)
	ListCarbonLedgerByTenant(ctx context.Context, params sqlc.ListCarbonLedgerByTenantParams) ([]sqlc.ListCarbonLedgerByTenantRow, error)
	CountCarbonLedgerByTenant(ctx context.Context, params sqlc.CountCarbonLedgerByTenantParams) (int64, error)
	GetCarbonLedgerByID(ctx context.Context, bankSampahID, id uuid.UUID) (*sqlc.GetCarbonLedgerByIDRow, error)
	ExecTx(ctx context.Context, fn func(*sqlc.Queries) error) error
}

type pgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewRepository creates a new PostgreSQL repository for carbon accounting.
func NewRepository(pool *pgxpool.Pool, queries *sqlc.Queries) Repository {
	return &pgRepository{
		pool:    pool,
		queries: queries,
	}
}

func (r *pgRepository) ListActiveEmissionFactors(ctx context.Context) ([]sqlc.EmissionFactor, error) {
	return r.queries.ListActiveEmissionFactors(ctx)
}

func (r *pgRepository) GetEmissionFactorByID(ctx context.Context, id uuid.UUID) (*sqlc.EmissionFactor, error) {
	factor, err := r.queries.GetEmissionFactorByID(ctx, db.ToPGUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &factor, nil
}

func (r *pgRepository) CreateCarbonLedgerEntry(ctx context.Context, params sqlc.CreateCarbonLedgerEntryParams) (*sqlc.CarbonLedger, error) {
	entry, err := r.queries.CreateCarbonLedgerEntry(ctx, params)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *pgRepository) CreateCarbonLedgerEntryTx(ctx context.Context, q *sqlc.Queries, params sqlc.CreateCarbonLedgerEntryParams) (*sqlc.CarbonLedger, error) {
	entry, err := q.CreateCarbonLedgerEntry(ctx, params)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *pgRepository) GetNasabahCarbonSavings(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahCarbonSavingsRow, error) {
	row, err := r.queries.GetNasabahCarbonSavings(ctx, sqlc.GetNasabahCarbonSavingsParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		NasabahID:    db.ToPGUUID(nasabahID),
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetBankSampahCarbonSavings(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetBankSampahCarbonSavingsRow, error) {
	row, err := r.queries.GetBankSampahCarbonSavings(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetNasabahCarbonSavingsByCategory(ctx context.Context, bankSampahID, nasabahID uuid.UUID) ([]sqlc.GetNasabahCarbonSavingsByCategoryRow, error) {
	return r.queries.GetNasabahCarbonSavingsByCategory(ctx, sqlc.GetNasabahCarbonSavingsByCategoryParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		NasabahID:    db.ToPGUUID(nasabahID),
	})
}

func (r *pgRepository) ListCarbonLedgerByNasabah(ctx context.Context, params sqlc.ListCarbonLedgerByNasabahParams) ([]sqlc.ListCarbonLedgerByNasabahRow, error) {
	return r.queries.ListCarbonLedgerByNasabah(ctx, params)
}

func (r *pgRepository) CountCarbonLedgerByNasabah(ctx context.Context, params sqlc.CountCarbonLedgerByNasabahParams) (int64, error) {
	return r.queries.CountCarbonLedgerByNasabah(ctx, params)
}

func (r *pgRepository) ListCarbonLedgerByTenant(ctx context.Context, params sqlc.ListCarbonLedgerByTenantParams) ([]sqlc.ListCarbonLedgerByTenantRow, error) {
	return r.queries.ListCarbonLedgerByTenant(ctx, params)
}

func (r *pgRepository) CountCarbonLedgerByTenant(ctx context.Context, params sqlc.CountCarbonLedgerByTenantParams) (int64, error) {
	return r.queries.CountCarbonLedgerByTenant(ctx, params)
}

func (r *pgRepository) GetCarbonLedgerByID(ctx context.Context, bankSampahID, id uuid.UUID) (*sqlc.GetCarbonLedgerByIDRow, error) {
	row, err := r.queries.GetCarbonLedgerByID(ctx, sqlc.GetCarbonLedgerByIDParams{
		ID:           db.ToPGUUID(id),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) ExecTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := r.queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
