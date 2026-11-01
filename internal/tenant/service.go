package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/audyfh/setor-hijau-be/internal/auth"
	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrCodeAlreadyExists indicates tenant code is already registered.
	ErrCodeAlreadyExists = errors.New("kode bank sampah sudah terdaftar")
	// ErrPengelolaConflict indicates username or email is already taken.
	ErrPengelolaConflict = errors.New("username atau email pengelola sudah digunakan")
)

// Service defines domain logic for tenant management.
type Service interface {
	Register(ctx context.Context, req RegisterBankSampahRequest) (*RegisterBankSampahResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*BankSampah, error)
}

type service struct {
	pool        *pgxpool.Pool
	repo        Repository
	authService auth.Service
}

// NewService creates a new tenant Service.
func NewService(pool *pgxpool.Pool, repo Repository, authService auth.Service) Service {
	return &service{
		pool:        pool,
		repo:        repo,
		authService: authService,
	}
}

func (s *service) Register(ctx context.Context, req RegisterBankSampahRequest) (*RegisterBankSampahResponse, error) {
	// Execute atomic transaction for bank sampah + pengelola user
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := sqlc.New(tx)

	// 1. Create Bank Sampah
	bankParams := sqlc.CreateBankSampahParams{
		Name:          req.Name,
		Code:          strings.ToLower(strings.TrimSpace(req.Code)),
		Address:       req.Address,
		Rw:            req.Rw,
		Rt:            db.ToPGText(req.Rt),
		Kelurahan:     req.Kelurahan,
		Kecamatan:     req.Kecamatan,
		KabupatenKota: req.KabupatenKota,
		Provinsi:      req.Provinsi,
		ContactPhone:  db.ToPGText(req.ContactPhone),
	}

	bankSampah, err := s.repo.CreateBankSampah(ctx, qtx, bankParams)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrCodeAlreadyExists
		}
		return nil, fmt.Errorf("failed to create bank sampah: %w", err)
	}

	// 2. Create Initial Pengelola User
	pengelola, err := s.authService.CreatePengelola(
		ctx,
		qtx,
		bankSampah.ID,
		req.PengelolaUsername,
		req.PengelolaEmail,
		req.PengelolaPassword,
		req.PengelolaFullName,
		req.PengelolaPhone,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrPengelolaConflict
		}
		return nil, fmt.Errorf("failed to create initial pengelola: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 3. Issue Tokens
	tokens, err := s.authService.GenerateTokenPair(ctx, pengelola)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	return &RegisterBankSampahResponse{
		BankSampah: bankSampah,
		Pengelola: auth.UserResponse{
			ID:           pengelola.ID,
			BankSampahID: pengelola.BankSampahID,
			Username:     pengelola.Username,
			Email:        pengelola.Email,
			FullName:     pengelola.FullName,
			Role:         pengelola.Role,
			Phone:        pengelola.Phone,
			CreatedAt:    pengelola.CreatedAt,
		},
		Tokens: tokens,
	}, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*BankSampah, error) {
	return s.repo.GetBankSampahByID(ctx, id)
}
