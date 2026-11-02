package deposit

import (
	"context"
	"errors"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines database operations for waste deposits and balances.
type Repository interface {
	ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error
	GetDepositByIdempotencyKey(ctx context.Context, bankSampahID uuid.UUID, idempotencyKey string) (*sqlc.Deposit, error)
	GetMaterialPriceAtTime(ctx context.Context, bankSampahID uuid.UUID, materialTypeID uuid.UUID, atTime time.Time) (*sqlc.GetMaterialPriceAtTimeRow, error)
	ValidateNasabah(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahDetailByIDRow, error)
	CreateDepositTx(ctx context.Context, q *sqlc.Queries, params sqlc.CreateDepositParams) (*sqlc.Deposit, error)
	CreateDepositLineTx(ctx context.Context, q *sqlc.Queries, params sqlc.CreateDepositLineParams) (*sqlc.DepositLine, error)
	GetDepositByID(ctx context.Context, bankSampahID, depositID uuid.UUID) (*sqlc.GetDepositByIDRow, error)
	GetDepositLinesByDepositID(ctx context.Context, depositID uuid.UUID) ([]sqlc.GetDepositLinesByDepositIDRow, error)
	GetDepositReceiptDetails(ctx context.Context, bankSampahID, depositID uuid.UUID) (*sqlc.GetDepositReceiptDetailsRow, error)
	GetNasabahBalance(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahBalanceRow, error)
	ListDepositsByTenant(ctx context.Context, params sqlc.ListDepositsByTenantParams) ([]sqlc.ListDepositsByTenantRow, error)
	CountDepositsByTenant(ctx context.Context, params sqlc.CountDepositsByTenantParams) (int64, error)
}

type pgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewRepository creates a new PostgreSQL repository for deposits.
func NewRepository(pool *pgxpool.Pool, queries *sqlc.Queries) Repository {
	return &pgRepository{
		pool:    pool,
		queries: queries,
	}
}

func (r *pgRepository) ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
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

func (r *pgRepository) GetDepositByIdempotencyKey(ctx context.Context, bankSampahID uuid.UUID, idempotencyKey string) (*sqlc.Deposit, error) {
	deposit, err := r.queries.GetDepositByIdempotencyKey(ctx, sqlc.GetDepositByIdempotencyKeyParams{
		BankSampahID:   db.ToPGUUID(bankSampahID),
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &deposit, nil
}

func (r *pgRepository) GetMaterialPriceAtTime(ctx context.Context, bankSampahID uuid.UUID, materialTypeID uuid.UUID, atTime time.Time) (*sqlc.GetMaterialPriceAtTimeRow, error) {
	row, err := r.queries.GetMaterialPriceAtTime(ctx, sqlc.GetMaterialPriceAtTimeParams{
		BankSampahID:  db.ToPGUUID(bankSampahID),
		EffectiveFrom: db.ToPGTimestamptz(atTime),
		ID:            db.ToPGUUID(materialTypeID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) ValidateNasabah(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahDetailByIDRow, error) {
	row, err := r.queries.GetNasabahDetailByID(ctx, sqlc.GetNasabahDetailByIDParams{
		ID:           db.ToPGUUID(nasabahID),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNasabahNotFound
		}
		return nil, err
	}
	if !row.IsActive || row.Role != "nasabah" {
		return nil, ErrNasabahNotFound
	}
	return &row, nil
}

func (r *pgRepository) CreateDepositTx(ctx context.Context, q *sqlc.Queries, params sqlc.CreateDepositParams) (*sqlc.Deposit, error) {
	deposit, err := q.CreateDeposit(ctx, params)
	if err != nil {
		return nil, err
	}
	return &deposit, nil
}

func (r *pgRepository) CreateDepositLineTx(ctx context.Context, q *sqlc.Queries, params sqlc.CreateDepositLineParams) (*sqlc.DepositLine, error) {
	line, err := q.CreateDepositLine(ctx, params)
	if err != nil {
		return nil, err
	}
	return &line, nil
}

func (r *pgRepository) GetDepositByID(ctx context.Context, bankSampahID, depositID uuid.UUID) (*sqlc.GetDepositByIDRow, error) {
	row, err := r.queries.GetDepositByID(ctx, sqlc.GetDepositByIDParams{
		ID:           db.ToPGUUID(depositID),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDepositNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetDepositLinesByDepositID(ctx context.Context, depositID uuid.UUID) ([]sqlc.GetDepositLinesByDepositIDRow, error) {
	return r.queries.GetDepositLinesByDepositID(ctx, db.ToPGUUID(depositID))
}

func (r *pgRepository) GetDepositReceiptDetails(ctx context.Context, bankSampahID, depositID uuid.UUID) (*sqlc.GetDepositReceiptDetailsRow, error) {
	row, err := r.queries.GetDepositReceiptDetails(ctx, sqlc.GetDepositReceiptDetailsParams{
		ID:           db.ToPGUUID(depositID),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDepositNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetNasabahBalance(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahBalanceRow, error) {
	row, err := r.queries.GetNasabahBalance(ctx, sqlc.GetNasabahBalanceParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		NasabahID:    db.ToPGUUID(nasabahID),
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) ListDepositsByTenant(ctx context.Context, params sqlc.ListDepositsByTenantParams) ([]sqlc.ListDepositsByTenantRow, error) {
	return r.queries.ListDepositsByTenant(ctx, params)
}

func (r *pgRepository) CountDepositsByTenant(ctx context.Context, params sqlc.CountDepositsByTenantParams) (int64, error) {
	return r.queries.CountDepositsByTenant(ctx, params)
}
