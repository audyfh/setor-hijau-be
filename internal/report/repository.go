package report

import (
	"context"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetBankSampahProfile(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.BankSampah, error)
	GetMonthlyDepositSummary(ctx context.Context, arg sqlc.GetMonthlyDepositSummaryParams) (*sqlc.GetMonthlyDepositSummaryRow, error)
	GetMonthlyDepositsByCategory(ctx context.Context, arg sqlc.GetMonthlyDepositsByCategoryParams) ([]sqlc.GetMonthlyDepositsByCategoryRow, error)
	GetMonthlySalesSummary(ctx context.Context, arg sqlc.GetMonthlySalesSummaryParams) (*sqlc.GetMonthlySalesSummaryRow, error)
	GetMonthlySalesByCategory(ctx context.Context, arg sqlc.GetMonthlySalesByCategoryParams) ([]sqlc.GetMonthlySalesByCategoryRow, error)
	GetMonthlyCashbookSummary(ctx context.Context, arg sqlc.GetMonthlyCashbookSummaryParams) (*sqlc.GetMonthlyCashbookSummaryRow, error)
	GetMonthlyCashbookByCategory(ctx context.Context, arg sqlc.GetMonthlyCashbookByCategoryParams) ([]sqlc.GetMonthlyCashbookByCategoryRow, error)
	GetCashbookSummary(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetCashbookSummaryRow, error)
	GetMonthlyCarbonSummary(ctx context.Context, arg sqlc.GetMonthlyCarbonSummaryParams) (*sqlc.GetMonthlyCarbonSummaryRow, error)
	GetMonthlyCarbonByCategory(ctx context.Context, arg sqlc.GetMonthlyCarbonByCategoryParams) ([]sqlc.GetMonthlyCarbonByCategoryRow, error)

	GetMonthlyDepositsDetail(ctx context.Context, arg sqlc.GetMonthlyDepositsDetailForReportParams) ([]sqlc.GetMonthlyDepositsDetailForReportRow, error)
	GetMonthlySalesDetail(ctx context.Context, arg sqlc.GetMonthlySalesDetailForReportParams) ([]sqlc.GetMonthlySalesDetailForReportRow, error)
	GetMonthlyCashbookDetail(ctx context.Context, arg sqlc.GetMonthlyCashbookDetailForReportParams) ([]sqlc.GetMonthlyCashbookDetailForReportRow, error)
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

func (r *pgRepository) GetBankSampahProfile(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.BankSampah, error) {
	row, err := r.queries.GetBankSampahByID(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetMonthlyDepositSummary(ctx context.Context, arg sqlc.GetMonthlyDepositSummaryParams) (*sqlc.GetMonthlyDepositSummaryRow, error) {
	row, err := r.queries.GetMonthlyDepositSummary(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetMonthlyDepositsByCategory(ctx context.Context, arg sqlc.GetMonthlyDepositsByCategoryParams) ([]sqlc.GetMonthlyDepositsByCategoryRow, error) {
	return r.queries.GetMonthlyDepositsByCategory(ctx, arg)
}

func (r *pgRepository) GetMonthlySalesSummary(ctx context.Context, arg sqlc.GetMonthlySalesSummaryParams) (*sqlc.GetMonthlySalesSummaryRow, error) {
	row, err := r.queries.GetMonthlySalesSummary(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetMonthlySalesByCategory(ctx context.Context, arg sqlc.GetMonthlySalesByCategoryParams) ([]sqlc.GetMonthlySalesByCategoryRow, error) {
	return r.queries.GetMonthlySalesByCategory(ctx, arg)
}

func (r *pgRepository) GetMonthlyCashbookSummary(ctx context.Context, arg sqlc.GetMonthlyCashbookSummaryParams) (*sqlc.GetMonthlyCashbookSummaryRow, error) {
	row, err := r.queries.GetMonthlyCashbookSummary(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetMonthlyCashbookByCategory(ctx context.Context, arg sqlc.GetMonthlyCashbookByCategoryParams) ([]sqlc.GetMonthlyCashbookByCategoryRow, error) {
	return r.queries.GetMonthlyCashbookByCategory(ctx, arg)
}

func (r *pgRepository) GetCashbookSummary(ctx context.Context, bankSampahID uuid.UUID) (*sqlc.GetCashbookSummaryRow, error) {
	row, err := r.queries.GetCashbookSummary(ctx, db.ToPGUUID(bankSampahID))
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetMonthlyCarbonSummary(ctx context.Context, arg sqlc.GetMonthlyCarbonSummaryParams) (*sqlc.GetMonthlyCarbonSummaryRow, error) {
	row, err := r.queries.GetMonthlyCarbonSummary(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) GetMonthlyCarbonByCategory(ctx context.Context, arg sqlc.GetMonthlyCarbonByCategoryParams) ([]sqlc.GetMonthlyCarbonByCategoryRow, error) {
	return r.queries.GetMonthlyCarbonByCategory(ctx, arg)
}

func (r *pgRepository) GetMonthlyDepositsDetail(ctx context.Context, arg sqlc.GetMonthlyDepositsDetailForReportParams) ([]sqlc.GetMonthlyDepositsDetailForReportRow, error) {
	return r.queries.GetMonthlyDepositsDetailForReport(ctx, arg)
}

func (r *pgRepository) GetMonthlySalesDetail(ctx context.Context, arg sqlc.GetMonthlySalesDetailForReportParams) ([]sqlc.GetMonthlySalesDetailForReportRow, error) {
	return r.queries.GetMonthlySalesDetailForReport(ctx, arg)
}

func (r *pgRepository) GetMonthlyCashbookDetail(ctx context.Context, arg sqlc.GetMonthlyCashbookDetailForReportParams) ([]sqlc.GetMonthlyCashbookDetailForReportRow, error) {
	return r.queries.GetMonthlyCashbookDetailForReport(ctx, arg)
}
