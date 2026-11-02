package nasabah

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

var (
	// ErrNasabahNotFound indicates the nasabah was not found in the tenant.
	ErrNasabahNotFound = errors.New("nasabah tidak ditemukan")
)

// Repository defines database operations for nasabah and their profiles.
type Repository interface {
	ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error
	CreateNasabahUser(ctx context.Context, q *sqlc.Queries, params sqlc.CreateUserParams) (*sqlc.User, error)
	CreateNasabahProfile(ctx context.Context, q *sqlc.Queries, params sqlc.CreateNasabahProfileParams) (*sqlc.NasabahProfile, error)
	GetNasabahDetailByID(ctx context.Context, id, bankSampahID uuid.UUID) (*Nasabah, error)
	GetNasabahByQRTokenHashInBankSampah(ctx context.Context, qrTokenHash string, bankSampahID uuid.UUID) (*Nasabah, error)
	ListNasabahByBankSampah(ctx context.Context, params sqlc.ListNasabahByBankSampahParams) ([]*Nasabah, error)
	CountNasabahByBankSampah(ctx context.Context, params sqlc.CountNasabahByBankSampahParams) (int64, error)
	UpdateNasabahUser(ctx context.Context, params sqlc.UpdateNasabahUserParams) error
	UpdateNasabahProfile(ctx context.Context, params sqlc.UpdateNasabahProfileParams) error
	UpdateNasabahQRTokenHash(ctx context.Context, userID, bankSampahID uuid.UUID, qrTokenHash string) error
	UpdateNasabahPIN(ctx context.Context, userID, bankSampahID uuid.UUID, pinHash string) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

type pgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewRepository creates a new PostgreSQL repository for nasabah.
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

func (r *pgRepository) CreateNasabahUser(ctx context.Context, q *sqlc.Queries, params sqlc.CreateUserParams) (*sqlc.User, error) {
	if q == nil {
		q = r.queries
	}
	u, err := q.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *pgRepository) CreateNasabahProfile(ctx context.Context, q *sqlc.Queries, params sqlc.CreateNasabahProfileParams) (*sqlc.NasabahProfile, error) {
	if q == nil {
		q = r.queries
	}
	np, err := q.CreateNasabahProfile(ctx, params)
	if err != nil {
		return nil, err
	}
	return &np, nil
}

func (r *pgRepository) GetNasabahDetailByID(ctx context.Context, id, bankSampahID uuid.UUID) (*Nasabah, error) {
	row, err := r.queries.GetNasabahDetailByID(ctx, sqlc.GetNasabahDetailByIDParams{
		ID:           db.ToPGUUID(id),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNasabahNotFound
		}
		return nil, err
	}

	var regBy *uuid.UUID
	if row.RegisteredBy.Valid {
		rb := db.FromPGUUID(row.RegisteredBy)
		regBy = &rb
	}

	var lockedUntil *time.Time
	if row.LockedUntil.Valid {
		lu := db.FromPGTimestamptz(row.LockedUntil)
		lockedUntil = &lu
	}

	hasPIN := false
	if b, ok := row.HasPin.(bool); ok {
		hasPIN = b
	}

	return &Nasabah{
		ID:                db.FromPGUUID(row.ID),
		BankSampahID:      db.FromPGUUID(row.BankSampahID),
		Username:          row.Username,
		Email:             db.FromPGText(row.Email),
		FullName:          row.FullName,
		Role:              row.Role,
		Phone:             db.FromPGText(row.Phone),
		IsActive:          row.IsActive,
		CreatedAt:         db.FromPGTimestamptz(row.CreatedAt),
		UpdatedAt:         db.FromPGTimestamptz(row.UpdatedAt),
		QRTokenHash:       row.QrTokenHash,
		RW:                row.Rw,
		RT:                db.FromPGText(row.Rt),
		AddressNotes:      db.FromPGText(row.AddressNotes),
		RegisteredBy:      regBy,
		FailedPinAttempts: row.FailedPinAttempts,
		LockedUntil:       lockedUntil,
		HasPIN:            hasPIN,
	}, nil
}

func (r *pgRepository) GetNasabahByQRTokenHashInBankSampah(ctx context.Context, qrTokenHash string, bankSampahID uuid.UUID) (*Nasabah, error) {
	row, err := r.queries.GetNasabahByQRTokenHashInBankSampah(ctx, sqlc.GetNasabahByQRTokenHashInBankSampahParams{
		QrTokenHash:  qrTokenHash,
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNasabahNotFound
		}
		return nil, err
	}

	hasPIN := false
	if b, ok := row.HasPin.(bool); ok {
		hasPIN = b
	}

	return &Nasabah{
		ID:           db.FromPGUUID(row.ID),
		BankSampahID: db.FromPGUUID(row.BankSampahID),
		Username:     row.Username,
		Email:        db.FromPGText(row.Email),
		FullName:     row.FullName,
		Role:         row.Role,
		Phone:        db.FromPGText(row.Phone),
		IsActive:     row.IsActive,
		CreatedAt:    db.FromPGTimestamptz(row.CreatedAt),
		RW:           row.Rw,
		RT:           db.FromPGText(row.Rt),
		AddressNotes: db.FromPGText(row.AddressNotes),
		HasPIN:       hasPIN,
	}, nil
}

func (r *pgRepository) ListNasabahByBankSampah(ctx context.Context, params sqlc.ListNasabahByBankSampahParams) ([]*Nasabah, error) {
	rows, err := r.queries.ListNasabahByBankSampah(ctx, params)
	if err != nil {
		return nil, err
	}

	results := make([]*Nasabah, len(rows))
	for i, row := range rows {
		hasPIN := false
		if b, ok := row.HasPin.(bool); ok {
			hasPIN = b
		}

		results[i] = &Nasabah{
			ID:           db.FromPGUUID(row.ID),
			BankSampahID: db.FromPGUUID(row.BankSampahID),
			Username:     row.Username,
			Email:        db.FromPGText(row.Email),
			FullName:     row.FullName,
			Phone:        db.FromPGText(row.Phone),
			IsActive:     row.IsActive,
			CreatedAt:    db.FromPGTimestamptz(row.CreatedAt),
			RW:           row.Rw,
			RT:           db.FromPGText(row.Rt),
			HasPIN:       hasPIN,
		}
	}
	return results, nil
}

func (r *pgRepository) CountNasabahByBankSampah(ctx context.Context, params sqlc.CountNasabahByBankSampahParams) (int64, error) {
	return r.queries.CountNasabahByBankSampah(ctx, params)
}

func (r *pgRepository) UpdateNasabahUser(ctx context.Context, params sqlc.UpdateNasabahUserParams) error {
	_, err := r.queries.UpdateNasabahUser(ctx, params)
	return err
}

func (r *pgRepository) UpdateNasabahProfile(ctx context.Context, params sqlc.UpdateNasabahProfileParams) error {
	_, err := r.queries.UpdateNasabahProfile(ctx, params)
	return err
}

func (r *pgRepository) UpdateNasabahQRTokenHash(ctx context.Context, userID, bankSampahID uuid.UUID, qrTokenHash string) error {
	_, err := r.queries.UpdateNasabahQRTokenHash(ctx, sqlc.UpdateNasabahQRTokenHashParams{
		QrTokenHash:  qrTokenHash,
		UserID:       db.ToPGUUID(userID),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	return err
}

func (r *pgRepository) UpdateNasabahPIN(ctx context.Context, userID, bankSampahID uuid.UUID, pinHash string) error {
	_, err := r.queries.UpdateNasabahPIN(ctx, sqlc.UpdateNasabahPINParams{
		PinHash:      db.ToPGText(pinHash),
		UserID:       db.ToPGUUID(userID),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	return err
}

func (r *pgRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		PasswordHash: passwordHash,
		ID:           db.ToPGUUID(userID),
	})
	return err
}
