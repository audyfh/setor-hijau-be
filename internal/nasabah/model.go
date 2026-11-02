package nasabah

import (
	"time"

	"github.com/google/uuid"
)

// Nasabah represents a registered nasabah user and profile.
type Nasabah struct {
	ID                uuid.UUID  `json:"id"`
	BankSampahID      uuid.UUID  `json:"bank_sampah_id"`
	Username          string     `json:"username"`
	Email             string     `json:"email,omitempty"`
	FullName          string     `json:"full_name"`
	Role              string     `json:"role"`
	Phone             string     `json:"phone,omitempty"`
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at,omitempty"`
	QRTokenHash       string     `json:"-"`
	RW                string     `json:"rw"`
	RT                string     `json:"rt,omitempty"`
	AddressNotes      string     `json:"address_notes,omitempty"`
	RegisteredBy      *uuid.UUID `json:"registered_by,omitempty"`
	FailedPinAttempts int32      `json:"failed_pin_attempts,omitempty"`
	LockedUntil       *time.Time `json:"locked_until,omitempty"`
	HasPIN            bool       `json:"has_pin"`
}

// CreateNasabahRequest defines input for Pengelola creating a new Nasabah.
type CreateNasabahRequest struct {
	Username     string `json:"username" binding:"required,min=3,max=64"`
	FullName     string `json:"full_name" binding:"required,min=2,max=255"`
	Password     string `json:"password" binding:"required,min=8"`
	Email        string `json:"email" binding:"omitempty,email"`
	Phone        string `json:"phone" binding:"omitempty,max=32"`
	RW           string `json:"rw" binding:"required,max=10"`
	RT           string `json:"rt" binding:"omitempty,max=10"`
	AddressNotes string `json:"address_notes"`
	PIN          string `json:"pin" binding:"omitempty,len=6,numeric"`
}

// CreateNasabahResponse returns the created nasabah with the plaintext QR token.
type CreateNasabahResponse struct {
	Nasabah *Nasabah `json:"nasabah"`
	QRToken string   `json:"qr_token"`
}

// UpdateNasabahRequest defines input for Pengelola updating a Nasabah.
type UpdateNasabahRequest struct {
	FullName     string `json:"full_name" binding:"required,min=2,max=255"`
	Email        string `json:"email" binding:"omitempty,email"`
	Phone        string `json:"phone" binding:"omitempty,max=32"`
	RW           string `json:"rw" binding:"required,max=10"`
	RT           string `json:"rt" binding:"omitempty,max=10"`
	AddressNotes string `json:"address_notes"`
	IsActive     *bool  `json:"is_active"`
}

// RegenerateQRResponse returns the newly generated plaintext QR token.
type RegenerateQRResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	QRToken string    `json:"qr_token"`
}

// ResetPasswordRequest defines input for Pengelola setting a new password for Nasabah.
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPINRequest defines input for setting a new PIN (Pengelola reset or Nasabah self-service).
type ResetPINRequest struct {
	PIN string `json:"pin" binding:"required,len=6,numeric"`
}

// ChangePasswordRequest defines input for Nasabah self-service password change.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// LookupQRRequest defines input for Pengelola scanning a QR token to identify a Nasabah.
type LookupQRRequest struct {
	QRToken string `json:"qr_token" binding:"required"`
}

// ListNasabahQuery defines query params for listing nasabah.
type ListNasabahQuery struct {
	Search string `form:"search"`
	RW     string `form:"rw"`
	RT     string `form:"rt"`
	Limit  int32  `form:"limit,default=20"`
	Offset int32  `form:"offset,default=0"`
}

// ListNasabahResponse returns paginated nasabah list.
type ListNasabahResponse struct {
	Data   []*Nasabah `json:"data"`
	Total  int64      `json:"total"`
	Limit  int32      `json:"limit"`
	Offset int32      `json:"offset"`
}
