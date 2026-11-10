package cashbook

import (
	"context"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	WithTx(ctx context.Context, fn func(r Repository) error) error

	// Sales to Pengepul
	CreateSale(ctx context.Context, arg sqlc.CreateSaleToPengepulParams) (*sqlc.SalesToPengepul, error)
	CreateSaleLine(ctx context.Context, arg sqlc.CreateSaleToPengepulLineParams) (*sqlc.SalesToPengepulLine, error)
	GetSaleByID(ctx context.Context, bankSampahID, saleID uuid.UUID) (*sqlc.GetSaleToPengepulByIDRow, error)
	GetSaleLines(ctx context.Context, saleID uuid.UUID) ([]sqlc.GetSaleLinesBySaleIDRow, error)
	ListSales(ctx context.Context, arg sqlc.ListSalesToPengepulParams) ([]sqlc.ListSalesToPengepulRow, error)
	CountSales(ctx context.Context, arg sqlc.CountSalesToPengepulParams) (int64, error)

	// Material Validation
	GetMaterialTypeByID(ctx context.Context, bankSampahID, materialTypeID uuid.UUID) (*sqlc.GetMaterialTypeByIDRow, error)

	// Nasabah Withdrawals
	CreateWithdrawal(ctx context.Context, arg sqlc.CreateNasabahWithdrawalParams) (*sqlc.NasabahWithdrawal, error)
	GetWithdrawalByID(ctx context.Context, bankSampahID, id uuid.UUID) (*sqlc.GetWithdrawalByIDRow, error)
	ListWithdrawals(ctx context.Context, arg sqlc.ListWithdrawalsByTenantParams) ([]sqlc.ListWithdrawalsByTenantRow, error)
	CountWithdrawals(ctx context.Context, arg sqlc.CountWithdrawalsByTenantParams) (int64, error)
	GetNasabahBalance(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahBalanceRow, error)

	// Cashbook Entries
	CreateCashbookEntry(ctx context.Context, arg sqlc.CreateCashbookEntryParams) (*sqlc.CashbookEntry, error)
	ListCashbookEntries(ctx context.Context, arg sqlc.ListCashbookEntriesParams) ([]sqlc.ListCashbookEntriesRow, error)
	CountCashbookEntries(ctx context.Context, arg sqlc.CountCashbookEntriesParams) (int64, error)
	GetCashbookSummary(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetCashbookSummaryRow, error)

	// Reconciliation
	GetFinancialReconciliation(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetFinancialReconciliationRow, error)
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

func (r *pgRepository) WithTx(ctx context.Context, fn func(r Repository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := r.queries.WithTx(tx)
	repoTx := &pgRepositoryTx{
		queries: qtx,
	}

	if err := fn(repoTx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *pgRepository) CreateSale(ctx context.Context, arg sqlc.CreateSaleToPengepulParams) (*sqlc.SalesToPengepul, error) {
	sale, err := r.queries.CreateSaleToPengepul(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &sale, nil
}

func (r *pgRepository) CreateSaleLine(ctx context.Context, arg sqlc.CreateSaleToPengepulLineParams) (*sqlc.SalesToPengepulLine, error) {
	line, err := r.queries.CreateSaleToPengepulLine(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &line, nil
}

func (r *pgRepository) GetSaleByID(ctx context.Context, bankSampahID, saleID uuid.UUID) (*sqlc.GetSaleToPengepulByIDRow, error) {
	row, err := r.queries.GetSaleToPengepulByID(ctx, sqlc.GetSaleToPengepulByIDParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		ID:           db.ToPGUUID(saleID),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetSaleLines(ctx context.Context, saleID uuid.UUID) ([]sqlc.GetSaleLinesBySaleIDRow, error) {
	return r.queries.GetSaleLinesBySaleID(ctx, db.ToPGUUID(saleID))
}

func (r *pgRepository) ListSales(ctx context.Context, arg sqlc.ListSalesToPengepulParams) ([]sqlc.ListSalesToPengepulRow, error) {
	return r.queries.ListSalesToPengepul(ctx, arg)
}

func (r *pgRepository) CountSales(ctx context.Context, arg sqlc.CountSalesToPengepulParams) (int64, error) {
	return r.queries.CountSalesToPengepul(ctx, arg)
}

func (r *pgRepository) GetMaterialTypeByID(ctx context.Context, bankSampahID, materialTypeID uuid.UUID) (*sqlc.GetMaterialTypeByIDRow, error) {
	row, err := r.queries.GetMaterialTypeByID(ctx, sqlc.GetMaterialTypeByIDParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		ID:           db.ToPGUUID(materialTypeID),
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) CreateWithdrawal(ctx context.Context, arg sqlc.CreateNasabahWithdrawalParams) (*sqlc.NasabahWithdrawal, error) {
	w, err := r.queries.CreateNasabahWithdrawal(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *pgRepository) GetWithdrawalByID(ctx context.Context, bankSampahID, id uuid.UUID) (*sqlc.GetWithdrawalByIDRow, error) {
	row, err := r.queries.GetWithdrawalByID(ctx, sqlc.GetWithdrawalByIDParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		ID:           db.ToPGUUID(id),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrWithdrawalNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) ListWithdrawals(ctx context.Context, arg sqlc.ListWithdrawalsByTenantParams) ([]sqlc.ListWithdrawalsByTenantRow, error) {
	return r.queries.ListWithdrawalsByTenant(ctx, arg)
}

func (r *pgRepository) CountWithdrawals(ctx context.Context, arg sqlc.CountWithdrawalsByTenantParams) (int64, error) {
	return r.queries.CountWithdrawalsByTenant(ctx, arg)
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

func (r *pgRepository) CreateCashbookEntry(ctx context.Context, arg sqlc.CreateCashbookEntryParams) (*sqlc.CashbookEntry, error) {
	entry, err := r.queries.CreateCashbookEntry(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *pgRepository) ListCashbookEntries(ctx context.Context, arg sqlc.ListCashbookEntriesParams) ([]sqlc.ListCashbookEntriesRow, error) {
	return r.queries.ListCashbookEntries(ctx, arg)
}

func (r *pgRepository) CountCashbookEntries(ctx context.Context, arg sqlc.CountCashbookEntriesParams) (int64, error) {
	return r.queries.CountCashbookEntries(ctx, arg)
}

func (r *pgRepository) GetCashbookSummary(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetCashbookSummaryRow, error) {
	row, err := r.queries.GetCashbookSummary(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetFinancialReconciliation(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetFinancialReconciliationRow, error) {
	row, err := r.queries.GetFinancialReconciliation(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// pgRepositoryTx implements Repository in a transaction
type pgRepositoryTx struct {
	queries *sqlc.Queries
}

func (r *pgRepositoryTx) WithTx(ctx context.Context, fn func(r Repository) error) error {
	return fn(r)
}

func (r *pgRepositoryTx) CreateSale(ctx context.Context, arg sqlc.CreateSaleToPengepulParams) (*sqlc.SalesToPengepul, error) {
	sale, err := r.queries.CreateSaleToPengepul(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &sale, nil
}

func (r *pgRepositoryTx) CreateSaleLine(ctx context.Context, arg sqlc.CreateSaleToPengepulLineParams) (*sqlc.SalesToPengepulLine, error) {
	line, err := r.queries.CreateSaleToPengepulLine(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &line, nil
}

func (r *pgRepositoryTx) GetSaleByID(ctx context.Context, bankSampahID, saleID uuid.UUID) (*sqlc.GetSaleToPengepulByIDRow, error) {
	row, err := r.queries.GetSaleToPengepulByID(ctx, sqlc.GetSaleToPengepulByIDParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		ID:           db.ToPGUUID(saleID),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepositoryTx) GetSaleLines(ctx context.Context, saleID uuid.UUID) ([]sqlc.GetSaleLinesBySaleIDRow, error) {
	return r.queries.GetSaleLinesBySaleID(ctx, db.ToPGUUID(saleID))
}

func (r *pgRepositoryTx) ListSales(ctx context.Context, arg sqlc.ListSalesToPengepulParams) ([]sqlc.ListSalesToPengepulRow, error) {
	return r.queries.ListSalesToPengepul(ctx, arg)
}

func (r *pgRepositoryTx) CountSales(ctx context.Context, arg sqlc.CountSalesToPengepulParams) (int64, error) {
	return r.queries.CountSalesToPengepul(ctx, arg)
}

func (r *pgRepositoryTx) GetMaterialTypeByID(ctx context.Context, bankSampahID, materialTypeID uuid.UUID) (*sqlc.GetMaterialTypeByIDRow, error) {
	row, err := r.queries.GetMaterialTypeByID(ctx, sqlc.GetMaterialTypeByIDParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		ID:           db.ToPGUUID(materialTypeID),
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepositoryTx) CreateWithdrawal(ctx context.Context, arg sqlc.CreateNasabahWithdrawalParams) (*sqlc.NasabahWithdrawal, error) {
	w, err := r.queries.CreateNasabahWithdrawal(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *pgRepositoryTx) GetWithdrawalByID(ctx context.Context, bankSampahID, id uuid.UUID) (*sqlc.GetWithdrawalByIDRow, error) {
	row, err := r.queries.GetWithdrawalByID(ctx, sqlc.GetWithdrawalByIDParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		ID:           db.ToPGUUID(id),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrWithdrawalNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepositoryTx) ListWithdrawals(ctx context.Context, arg sqlc.ListWithdrawalsByTenantParams) ([]sqlc.ListWithdrawalsByTenantRow, error) {
	return r.queries.ListWithdrawalsByTenant(ctx, arg)
}

func (r *pgRepositoryTx) CountWithdrawals(ctx context.Context, arg sqlc.CountWithdrawalsByTenantParams) (int64, error) {
	return r.queries.CountWithdrawalsByTenant(ctx, arg)
}

func (r *pgRepositoryTx) GetNasabahBalance(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*sqlc.GetNasabahBalanceRow, error) {
	row, err := r.queries.GetNasabahBalance(ctx, sqlc.GetNasabahBalanceParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		NasabahID:    db.ToPGUUID(nasabahID),
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepositoryTx) CreateCashbookEntry(ctx context.Context, arg sqlc.CreateCashbookEntryParams) (*sqlc.CashbookEntry, error) {
	entry, err := r.queries.CreateCashbookEntry(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *pgRepositoryTx) ListCashbookEntries(ctx context.Context, arg sqlc.ListCashbookEntriesParams) ([]sqlc.ListCashbookEntriesRow, error) {
	return r.queries.ListCashbookEntries(ctx, arg)
}

func (r *pgRepositoryTx) CountCashbookEntries(ctx context.Context, arg sqlc.CountCashbookEntriesParams) (int64, error) {
	return r.queries.CountCashbookEntries(ctx, arg)
}

func (r *pgRepositoryTx) GetCashbookSummary(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetCashbookSummaryRow, error) {
	row, err := r.queries.GetCashbookSummary(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepositoryTx) GetFinancialReconciliation(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetFinancialReconciliationRow, error) {
	row, err := r.queries.GetFinancialReconciliation(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}
