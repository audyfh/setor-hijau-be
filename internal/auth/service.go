package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	pauth "github.com/audyfh/setor-hijau-be/internal/platform/auth"
	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials indicates wrong username/email or password.
	ErrInvalidCredentials = errors.New("identifikasi atau kata sandi tidak valid")
	// ErrAccountInactive indicates the account is deactivated.
	ErrAccountInactive = errors.New("akun tidak aktif")
	// ErrInvalidResetToken indicates the reset token is invalid, expired, or already used.
	ErrInvalidResetToken = errors.New("token reset kata sandi tidak valid atau telah kedaluwarsa")
)

// Service defines domain logic for authentication and credential management.
type Service interface {
	Login(ctx context.Context, req LoginRequest) (*TokenPair, *UserResponse, error)
	RefreshToken(ctx context.Context, req RefreshTokenRequest) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	GetMe(ctx context.Context, userID, bankSampahID uuid.UUID) (*UserResponse, error)
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
	CreatePengelola(ctx context.Context, q *sqlc.Queries, bankSampahID uuid.UUID, username, email, password, fullName, phone string) (*User, error)
	GenerateTokenPair(ctx context.Context, user *User) (*TokenPair, error)
}

type service struct {
	repo            Repository
	emailSender     EmailSender
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewService creates a new authentication service.
func NewService(repo Repository, emailSender EmailSender, jwtSecret []byte) Service {
	return &service{
		repo:            repo,
		emailSender:     emailSender,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*TokenPair, *UserResponse, error) {
	user, err := s.repo.GetPengelolaByIdentifier(ctx, req.Identifier)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, ErrAccountInactive
	}

	match, err := pauth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, nil, ErrInvalidCredentials
	}

	tokenPair, err := s.GenerateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	userResp := &UserResponse{
		ID:           user.ID,
		BankSampahID: user.BankSampahID,
		Username:     user.Username,
		Email:        user.Email,
		FullName:     user.FullName,
		Role:         user.Role,
		Phone:        user.Phone,
		CreatedAt:    user.CreatedAt,
	}

	return tokenPair, userResp, nil
}

func (s *service) GenerateTokenPair(ctx context.Context, user *User) (*TokenPair, error) {
	accessToken, err := pauth.GenerateAccessToken(s.jwtSecret, user.ID, user.BankSampahID, user.Username, user.Role, s.accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	rawRefreshToken, err := pauth.GenerateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenHash := pauth.HashToken(rawRefreshToken)
	expiresAt := time.Now().UTC().Add(s.refreshTokenTTL)

	if err := s.repo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to persist refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*TokenPair, error) {
	tokenHash := pauth.HashToken(req.RefreshToken)
	storedToken, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrTokenNotFound
	}

	// Check if already revoked
	if storedToken.RevokedAt.Valid {
		return nil, ErrTokenNotFound
	}

	// Check expiration
	if time.Now().UTC().After(storedToken.ExpiresAt.Time) {
		return nil, ErrTokenNotFound
	}

	// Rotate token: revoke old one immediately
	tokenID := db.FromPGUUID(storedToken.ID)
	if err := s.repo.RevokeRefreshToken(ctx, tokenID); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	userID := db.FromPGUUID(storedToken.UserID)
	user, err := s.repo.GetUserByIDGlobal(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	return s.GenerateTokenPair(ctx, user)
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := pauth.HashToken(refreshToken)
	storedToken, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		// Idempotent: return nil even if not found
		return nil
	}

	if !storedToken.RevokedAt.Valid {
		tokenID := db.FromPGUUID(storedToken.ID)
		return s.repo.RevokeRefreshToken(ctx, tokenID)
	}

	return nil
}

func (s *service) GetMe(ctx context.Context, userID, bankSampahID uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID, bankSampahID)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:           user.ID,
		BankSampahID: user.BankSampahID,
		Username:     user.Username,
		Email:        user.Email,
		FullName:     user.FullName,
		Role:         user.Role,
		Phone:        user.Phone,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Silent success to prevent account enumeration
		return nil
	}

	if !user.IsActive || user.Role != RolePengelola {
		return nil
	}

	rawToken, err := pauth.GenerateRandomToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	tokenHash := pauth.HashToken(rawToken)
	expiresAt := time.Now().UTC().Add(15 * time.Minute)

	if err := s.repo.CreatePasswordResetToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	if err := s.emailSender.SendPasswordReset(ctx, user.Email, rawToken); err != nil {
		return fmt.Errorf("failed to deliver password reset notification: %w", err)
	}

	return nil
}

func (s *service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	tokenHash := pauth.HashToken(req.Token)
	resetToken, err := s.repo.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		return ErrInvalidResetToken
	}

	if resetToken.UsedAt.Valid || time.Now().UTC().After(resetToken.ExpiresAt.Time) {
		return ErrInvalidResetToken
	}

	newHash, err := pauth.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	userID := db.FromPGUUID(resetToken.UserID)
	if _, err := s.repo.UpdateUserPassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	tokenID := db.FromPGUUID(resetToken.ID)
	if err := s.repo.MarkPasswordResetTokenUsed(ctx, tokenID); err != nil {
		return fmt.Errorf("failed to mark reset token as used: %w", err)
	}

	// Revoke all existing sessions for security
	_ = s.repo.RevokeAllUserRefreshTokens(ctx, userID)

	return nil
}

func (s *service) CreatePengelola(ctx context.Context, q *sqlc.Queries, bankSampahID uuid.UUID, username, email, password, fullName, phone string) (*User, error) {
	hash, err := pauth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	params := sqlc.CreateUserParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Username:     username,
		Email:        db.ToPGText(email),
		PasswordHash: hash,
		FullName:     fullName,
		Role:         RolePengelola,
		Phone:        db.ToPGText(phone),
		IsActive:     true,
	}

	return s.repo.CreateUser(ctx, q, params)
}
