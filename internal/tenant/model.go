package tenant

import (
	"time"

	"github.com/google/uuid"
)

// BankSampah represents a tenant in Setor Hijau.
type BankSampah struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	Address       string    `json:"address"`
	Rw            string    `json:"rw"`
	Rt            string    `json:"rt,omitempty"`
	Kelurahan     string    `json:"kelurahan"`
	Kecamatan     string    `json:"kecamatan"`
	KabupatenKota string    `json:"kabupaten_kota"`
	Provinsi      string    `json:"provinsi"`
	ContactPhone  string    `json:"contact_phone,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// RegisterBankSampahRequest defines the registration payload for a new Bank Sampah and its initial Pengelola.
type RegisterBankSampahRequest struct {
	// Bank Sampah Info
	Name          string `json:"name" binding:"required"`
	Code          string `json:"code" binding:"required"`
	Address       string `json:"address" binding:"required"`
	Rw            string `json:"rw" binding:"required"`
	Rt            string `json:"rt"`
	Kelurahan     string `json:"kelurahan" binding:"required"`
	Kecamatan     string `json:"kecamatan" binding:"required"`
	KabupatenKota string `json:"kabupaten_kota" binding:"required"`
	Provinsi      string `json:"provinsi" binding:"required"`
	ContactPhone  string `json:"contact_phone"`

	// Initial Pengelola Credentials
	PengelolaUsername string `json:"pengelola_username" binding:"required,min=3"`
	PengelolaEmail    string `json:"pengelola_email" binding:"required,email"`
	PengelolaPassword string `json:"pengelola_password" binding:"required,min=8"`
	PengelolaFullName string `json:"pengelola_full_name" binding:"required"`
	PengelolaPhone    string `json:"pengelola_phone"`
}

// RegisterBankSampahResponse represents the payload returned after registering a new tenant.
type RegisterBankSampahResponse struct {
	BankSampah *BankSampah `json:"bank_sampah"`
	Pengelola  interface{} `json:"pengelola"`
	Tokens     interface{} `json:"tokens"`
}
