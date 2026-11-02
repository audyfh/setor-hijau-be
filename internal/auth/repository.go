package auth

import (
	"context"
	"errors"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrUserNotFound indicates the requested user was not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrTokenNotFound indicates the token was not found or has been revoked.
	ErrTokenNotFound = errors.New("token not found or revoked")
)

// Repository defines data access methods for users and tokens.
type Repository interface {
	CreateUser(ctx context.Context, q *sqlc.Queries, params sqlc.CreateUserParams) (*User, error)
	GetUserByID(ctx context.Context, id, bankSampahID uuid.UUID) (*User, error)
	GetUserByIDGlobal(ctx context.Context, id uuid.UUID) (*User, error)
	GetPengelolaByIdentifier(ctx context.Context, identifier string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByUsernameAndBankSampahCode(ctx context.Context, username, code string) (*User, error)
	GetNasabahForQRLogin(ctx context.Context, qrTokenHash string) (*sqlc.GetNasabahForQRLoginRow, error)
	RecordFailedPINAttempt(ctx context.Context, userID uuid.UUID) (int32, error)
	LockNasabahPIN(ctx context.Context, userID uuid.UUID, lockedUntil time.Time) error
	ResetFailedPINAttempts(ctx context.Context, userID uuid.UUID) error
	UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) (*User, error)
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*sqlc.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*sqlc.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error
}

type pgRepository struct {
	queries *sqlc.Queries
}

// NewRepository creates a new PostgreSQL repository for authentication.
func NewRepository(queries *sqlc.Queries) Repository {
	return &pgRepository{queries: queries}
}

func toUserModel(u sqlc.User) *User {
	return &User{
		ID:           db.FromPGUUID(u.ID),
		BankSampahID: db.FromPGUUID(u.BankSampahID),
		Username:     u.Username,
		Email:        db.FromPGText(u.Email),
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		Role:         u.Role,
		Phone:        db.FromPGText(u.Phone),
		IsActive:     u.IsActive,
		CreatedAt:    db.FromPGTimestamptz(u.CreatedAt),
		UpdatedAt:    db.FromPGTimestamptz(u.UpdatedAt),
	}
}

func (r *pgRepository) CreateUser(ctx context.Context, q *sqlc.Queries, params sqlc.CreateUserParams) (*User, error) {
	if q == nil {
		q = r.queries
	}
	u, err := q.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return toUserModel(u), nil
}

func (r *pgRepository) GetUserByID(ctx context.Context, id, bankSampahID uuid.UUID) (*User, error) {
	u, err := r.queries.GetUserByID(ctx, sqlc.GetUserByIDParams{
		ID:           db.ToPGUUID(id),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUserModel(u), nil
}

func (r *pgRepository) GetUserByIDGlobal(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := r.queries.GetUserByIDGlobal(ctx, db.ToPGUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUserModel(u), nil
}

func (r *pgRepository) GetPengelolaByIdentifier(ctx context.Context, identifier string) (*User, error) {
	u, err := r.queries.GetPengelolaByUsernameOrEmail(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUserModel(u), nil
}

func (r *pgRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u, err := r.queries.GetUserByEmail(ctx, db.ToPGText(email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUserModel(u), nil
}

func (r *pgRepository) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) (*User, error) {
	u, err := r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		PasswordHash: passwordHash,
		ID:           db.ToPGUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUserModel(u), nil
}

func (r *pgRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.queries.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    db.ToPGUUID(userID),
		TokenHash: tokenHash,
		ExpiresAt: db.ToPGTimestamptz(expiresAt),
	})
	return err
}

func (r *pgRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*sqlc.RefreshToken, error) {
	tok, err := r.queries.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &tok, nil
}

func (r *pgRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.queries.RevokeRefreshToken(ctx, db.ToPGUUID(id))
	return err
}

func (r *pgRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return r.queries.RevokeAllUserRefreshTokens(ctx, db.ToPGUUID(userID))
}

func (r *pgRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.queries.CreatePasswordResetToken(ctx, sqlc.CreatePasswordResetTokenParams{
		UserID:    db.ToPGUUID(userID),
		TokenHash: tokenHash,
		ExpiresAt: db.ToPGTimestamptz(expiresAt),
	})
	return err
}

func (r *pgRepository) GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*sqlc.PasswordResetToken, error) {
	tok, err := r.queries.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &tok, nil
}

func (r *pgRepository) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.queries.MarkPasswordResetTokenUsed(ctx, db.ToPGUUID(id))
	return err
}

func (r *pgRepository) GetUserByUsernameAndBankSampahCode(ctx context.Context, username, code string) (*User, error) {
	u, err := r.queries.GetUserByUsernameAndBankSampahCode(ctx, sqlc.GetUserByUsernameAndBankSampahCodeParams{
		Username: username,
		Code:     code,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUserModel(u), nil
}

func (r *pgRepository) GetNasabahForQRLogin(ctx context.Context, qrTokenHash string) (*sqlc.GetNasabahForQRLoginRow, error) {
	row, err := r.queries.GetNasabahForQRLogin(ctx, qrTokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *pgRepository) RecordFailedPINAttempt(ctx context.Context, userID uuid.UUID) (int32, error) {
	return r.queries.RecordFailedPINAttempt(ctx, db.ToPGUUID(userID))
}

func (r *pgRepository) LockNasabahPIN(ctx context.Context, userID uuid.UUID, lockedUntil time.Time) error {
	return r.queries.LockNasabahPIN(ctx, sqlc.LockNasabahPINParams{
		LockedUntil: db.ToPGTimestamptz(lockedUntil),
		UserID:      db.ToPGUUID(userID),
	})
}

func (r *pgRepository) ResetFailedPINAttempts(ctx context.Context, userID uuid.UUID) error {
	return r.queries.ResetFailedPINAttempts(ctx, db.ToPGUUID(userID))
}
