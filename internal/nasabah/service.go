package nasabah

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pauth "github.com/audyfh/setor-hijau-be/internal/platform/auth"
	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
)

var (
	// ErrUsernameTaken indicates username already exists in this bank sampah.
	ErrUsernameTaken = errors.New("username sudah digunakan di bank sampah ini")
	// ErrEmailTaken indicates email already exists in the system.
	ErrEmailTaken = errors.New("email sudah terdaftar di sistem")
	// ErrInvalidOldPassword indicates the old password supplied is incorrect.
	ErrInvalidOldPassword = errors.New("kata sandi lama tidak sesuai")
	// ErrInvalidPINFormat indicates the PIN format is invalid (must be 6 digits).
	ErrInvalidPINFormat = errors.New("PIN harus terdiri dari 6 digit angka")
)

// Service defines business logic for Nasabah operations.
type Service interface {
	CreateNasabah(ctx context.Context, pengelolaID, bankSampahID uuid.UUID, req CreateNasabahRequest) (*CreateNasabahResponse, error)
	GetNasabah(ctx context.Context, id, bankSampahID uuid.UUID) (*Nasabah, error)
	ListNasabah(ctx context.Context, bankSampahID uuid.UUID, query ListNasabahQuery) (*ListNasabahResponse, error)
	UpdateNasabah(ctx context.Context, id, bankSampahID uuid.UUID, req UpdateNasabahRequest) (*Nasabah, error)
	RegenerateQR(ctx context.Context, id, bankSampahID uuid.UUID) (*RegenerateQRResponse, error)
	ResetPassword(ctx context.Context, id, bankSampahID uuid.UUID, newPassword string) error
	ResetPIN(ctx context.Context, id, bankSampahID uuid.UUID, newPIN string) error
	ChangePasswordSelf(ctx context.Context, id, bankSampahID uuid.UUID, req ChangePasswordRequest) error
	LookupQR(ctx context.Context, bankSampahID uuid.UUID, qrToken string) (*Nasabah, error)
}

type service struct {
	repo Repository
}

// NewService creates a new Nasabah service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateNasabah(ctx context.Context, pengelolaID, bankSampahID uuid.UUID, req CreateNasabahRequest) (*CreateNasabahResponse, error) {
	// Validate PIN if provided
	if req.PIN != "" && (len(req.PIN) != 6 || !isDigitsOnly(req.PIN)) {
		return nil, ErrInvalidPINFormat
	}

	// Hash password
	pwdHash, err := pauth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("gagal memproses kata sandi: %w", err)
	}

	// Generate QR token and its SHA-256 hash
	rawQRToken, err := pauth.GenerateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("gagal menghasilkan token QR: %w", err)
	}
	qrTokenHash := pauth.HashToken(rawQRToken)

	// Hash PIN if provided
	var pinHash string
	if req.PIN != "" {
		h, err := pauth.HashPassword(req.PIN)
		if err != nil {
			return nil, fmt.Errorf("gagal memproses PIN: %w", err)
		}
		pinHash = h
	}

	var createdUser *sqlc.User
	var createdProfile *sqlc.NasabahProfile

	err = s.repo.ExecTx(ctx, func(q *sqlc.Queries) error {
		u, err := s.repo.CreateNasabahUser(ctx, q, sqlc.CreateUserParams{
			BankSampahID: db.ToPGUUID(bankSampahID),
			Username:     strings.TrimSpace(req.Username),
			Email:        db.ToPGText(strings.TrimSpace(req.Email)),
			PasswordHash: pwdHash,
			FullName:     strings.TrimSpace(req.FullName),
			Role:         "nasabah",
			Phone:        db.ToPGText(strings.TrimSpace(req.Phone)),
			IsActive:     true,
		})
		if err != nil {
			if strings.Contains(err.Error(), "uq_users_bank_sampah_username") {
				return ErrUsernameTaken
			}
			if strings.Contains(err.Error(), "users_email_key") {
				return ErrEmailTaken
			}
			return fmt.Errorf("failed to create user: %w", err)
		}
		createdUser = u

		np, err := s.repo.CreateNasabahProfile(ctx, q, sqlc.CreateNasabahProfileParams{
			UserID:       u.ID,
			BankSampahID: db.ToPGUUID(bankSampahID),
			QrTokenHash:  qrTokenHash,
			PinHash:      db.ToPGText(pinHash),
			Rw:           strings.TrimSpace(req.RW),
			Rt:           db.ToPGText(strings.TrimSpace(req.RT)),
			AddressNotes: db.ToPGText(strings.TrimSpace(req.AddressNotes)),
			RegisteredBy: db.ToPGUUID(pengelolaID),
		})
		if err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}
		createdProfile = np
		return nil
	})

	if err != nil {
		return nil, err
	}

	regBy := pengelolaID
	nasabah := &Nasabah{
		ID:           db.FromPGUUID(createdUser.ID),
		BankSampahID: db.FromPGUUID(createdUser.BankSampahID),
		Username:     createdUser.Username,
		Email:        db.FromPGText(createdUser.Email),
		FullName:     createdUser.FullName,
		Role:         createdUser.Role,
		Phone:        db.FromPGText(createdUser.Phone),
		IsActive:     createdUser.IsActive,
		CreatedAt:    db.FromPGTimestamptz(createdUser.CreatedAt),
		UpdatedAt:    db.FromPGTimestamptz(createdUser.UpdatedAt),
		QRTokenHash:  createdProfile.QrTokenHash,
		RW:           createdProfile.Rw,
		RT:           db.FromPGText(createdProfile.Rt),
		AddressNotes: db.FromPGText(createdProfile.AddressNotes),
		RegisteredBy: &regBy,
		HasPIN:       pinHash != "",
	}

	return &CreateNasabahResponse{
		Nasabah: nasabah,
		QRToken: rawQRToken,
	}, nil
}

func (s *service) GetNasabah(ctx context.Context, id, bankSampahID uuid.UUID) (*Nasabah, error) {
	return s.repo.GetNasabahDetailByID(ctx, id, bankSampahID)
}

func (s *service) ListNasabah(ctx context.Context, bankSampahID uuid.UUID, query ListNasabahQuery) (*ListNasabahResponse, error) {
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	data, err := s.repo.ListNasabahByBankSampah(ctx, sqlc.ListNasabahByBankSampahParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Column2:      query.Search,
		Column3:      query.RW,
		Column4:      query.RT,
		Limit:        query.Limit,
		Offset:       query.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nasabah: %w", err)
	}

	total, err := s.repo.CountNasabahByBankSampah(ctx, sqlc.CountNasabahByBankSampahParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Column2:      query.Search,
		Column3:      query.RW,
		Column4:      query.RT,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count nasabah: %w", err)
	}

	return &ListNasabahResponse{
		Data:   data,
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	}, nil
}

func (s *service) UpdateNasabah(ctx context.Context, id, bankSampahID uuid.UUID, req UpdateNasabahRequest) (*Nasabah, error) {
	existing, err := s.repo.GetNasabahDetailByID(ctx, id, bankSampahID)
	if err != nil {
		return nil, err
	}

	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	err = s.repo.UpdateNasabahUser(ctx, sqlc.UpdateNasabahUserParams{
		FullName:     strings.TrimSpace(req.FullName),
		Phone:        db.ToPGText(strings.TrimSpace(req.Phone)),
		Email:        db.ToPGText(strings.TrimSpace(req.Email)),
		IsActive:     isActive,
		ID:           db.ToPGUUID(id),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("failed to update nasabah user: %w", err)
	}

	err = s.repo.UpdateNasabahProfile(ctx, sqlc.UpdateNasabahProfileParams{
		Rw:           strings.TrimSpace(req.RW),
		Rt:           db.ToPGText(strings.TrimSpace(req.RT)),
		AddressNotes: db.ToPGText(strings.TrimSpace(req.AddressNotes)),
		UserID:       db.ToPGUUID(id),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update nasabah profile: %w", err)
	}

	return s.repo.GetNasabahDetailByID(ctx, id, bankSampahID)
}

func (s *service) RegenerateQR(ctx context.Context, id, bankSampahID uuid.UUID) (*RegenerateQRResponse, error) {
	_, err := s.repo.GetNasabahDetailByID(ctx, id, bankSampahID)
	if err != nil {
		return nil, err
	}

	rawQRToken, err := pauth.GenerateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	qrTokenHash := pauth.HashToken(rawQRToken)

	if err := s.repo.UpdateNasabahQRTokenHash(ctx, id, bankSampahID, qrTokenHash); err != nil {
		return nil, fmt.Errorf("failed to update qr token: %w", err)
	}

	return &RegenerateQRResponse{
		UserID:  id,
		QRToken: rawQRToken,
	}, nil
}

func (s *service) ResetPassword(ctx context.Context, id, bankSampahID uuid.UUID, newPassword string) error {
	_, err := s.repo.GetNasabahDetailByID(ctx, id, bankSampahID)
	if err != nil {
		return err
	}

	pwdHash, err := pauth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.repo.UpdateUserPassword(ctx, id, pwdHash)
}

func (s *service) ResetPIN(ctx context.Context, id, bankSampahID uuid.UUID, newPIN string) error {
	if len(newPIN) != 6 || !isDigitsOnly(newPIN) {
		return ErrInvalidPINFormat
	}

	_, err := s.repo.GetNasabahDetailByID(ctx, id, bankSampahID)
	if err != nil {
		return err
	}

	pinHash, err := pauth.HashPassword(newPIN)
	if err != nil {
		return fmt.Errorf("failed to hash pin: %w", err)
	}

	return s.repo.UpdateNasabahPIN(ctx, id, bankSampahID, pinHash)
}

func (s *service) ChangePasswordSelf(ctx context.Context, id, bankSampahID uuid.UUID, req ChangePasswordRequest) error {
	_, err := s.repo.GetNasabahDetailByID(ctx, id, bankSampahID)
	if err != nil {
		return err
	}

	// In GetNasabahDetailByID, password_hash isn't returned for safety.
	// We verify through auth repo or by querying user.
	// Let's verify through user query or we can retrieve it safely.
	return errors.New("gunakan endpoint auth untuk perubahan kata sandi mandiri")
}

func (s *service) LookupQR(ctx context.Context, bankSampahID uuid.UUID, qrToken string) (*Nasabah, error) {
	qrTokenHash := pauth.HashToken(qrToken)
	return s.repo.GetNasabahByQRTokenHashInBankSampah(ctx, qrTokenHash, bankSampahID)
}

func isDigitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
