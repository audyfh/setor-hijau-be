package tenant

import (
	"context"
	"errors"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrTenantNotFound indicates the bank sampah was not found.
	ErrTenantNotFound = errors.New("bank sampah tidak ditemukan")
)

// Repository defines data operations for bank sampah tenants.
type Repository interface {
	CreateBankSampah(ctx context.Context, q *sqlc.Queries, params sqlc.CreateBankSampahParams) (*BankSampah, error)
	GetBankSampahByID(ctx context.Context, id uuid.UUID) (*BankSampah, error)
	GetBankSampahByCode(ctx context.Context, code string) (*BankSampah, error)
}

type pgRepository struct {
	queries *sqlc.Queries
}

// NewRepository creates a new PostgreSQL repository for tenants.
func NewRepository(queries *sqlc.Queries) Repository {
	return &pgRepository{queries: queries}
}

func toBankSampahModel(b sqlc.BankSampah) *BankSampah {
	return &BankSampah{
		ID:            db.FromPGUUID(b.ID),
		Name:          b.Name,
		Code:          b.Code,
		Address:       b.Address,
		Rw:            b.Rw,
		Rt:            db.FromPGText(b.Rt),
		Kelurahan:     b.Kelurahan,
		Kecamatan:     b.Kecamatan,
		KabupatenKota: b.KabupatenKota,
		Provinsi:      b.Provinsi,
		ContactPhone:  db.FromPGText(b.ContactPhone),
		CreatedAt:     db.FromPGTimestamptz(b.CreatedAt),
		UpdatedAt:     db.FromPGTimestamptz(b.UpdatedAt),
	}
}

func (r *pgRepository) CreateBankSampah(ctx context.Context, q *sqlc.Queries, params sqlc.CreateBankSampahParams) (*BankSampah, error) {
	if q == nil {
		q = r.queries
	}
	b, err := q.CreateBankSampah(ctx, params)
	if err != nil {
		return nil, err
	}
	return toBankSampahModel(b), nil
}

func (r *pgRepository) GetBankSampahByID(ctx context.Context, id uuid.UUID) (*BankSampah, error) {
	b, err := r.queries.GetBankSampahByID(ctx, db.ToPGUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return toBankSampahModel(b), nil
}

func (r *pgRepository) GetBankSampahByCode(ctx context.Context, code string) (*BankSampah, error) {
	b, err := r.queries.GetBankSampahByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return toBankSampahModel(b), nil
}
