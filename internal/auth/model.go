package auth

import (
	"time"

	"github.com/google/uuid"
)

// Role constants
const (
	RolePengelola = "pengelola"
	RoleNasabah   = "nasabah"
)

// User represents an authenticated account in Setor Hijau.
type User struct {
	ID           uuid.UUID `json:"id"`
	BankSampahID uuid.UUID `json:"bank_sampah_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	Phone        string    `json:"phone,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserResponse represents safe user details returned to clients.
type UserResponse struct {
	ID           uuid.UUID `json:"id"`
	BankSampahID uuid.UUID `json:"bank_sampah_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	Phone        string    `json:"phone,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// TokenPair represents the access and refresh token returned on login or token refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// LoginRequest defines input for identifier + password authentication.
type LoginRequest struct {
	Identifier     string `json:"identifier" binding:"required"`
	Password       string `json:"password" binding:"required"`
	BankSampahCode string `json:"bank_sampah_code,omitempty"`
}

// QRLoginRequest defines input for QR token + PIN authentication.
type QRLoginRequest struct {
	QRToken string `json:"qr_token" binding:"required"`
	PIN     string `json:"pin" binding:"required"`
}

// RefreshTokenRequest defines input for refresh token exchange.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest defines input for revoking a refresh token.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ForgotPasswordRequest defines input for requesting password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest defines input for executing password reset.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
