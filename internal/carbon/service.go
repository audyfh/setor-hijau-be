package carbon

import (
	"context"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service defines business operations for carbon accounting and ledger auditing.
type Service interface {
	ListFactors(ctx context.Context) ([]EmissionFactor, error)
	GetNasabahCarbonSavings(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*NasabahCarbonSavingsResponse, error)
	GetTenantCarbonSavings(ctx context.Context, bankSampahID uuid.UUID) (*BankSampahCarbonSavingsResponse, error)
	ListLedgerEntries(ctx context.Context, bankSampahID, requesterUserID uuid.UUID, requesterRole string, filter CarbonLedgerFilter) (*CarbonLedgerListResponse, error)
	RecordReversal(ctx context.Context, bankSampahID, pengelolaID uuid.UUID, req CreateReversalRequest) (*CarbonLedgerEntry, error)
}

type service struct {
	repo Repository
}

// NewService creates a new carbon service instance.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// CalculateAvoidedCO2e computes estimated avoided greenhouse gas emissions (in integer grams CO2e)
// using integer half-up division from weight in grams and fixed-point parts-per-million (PPM) factor.
// Formula: floor((weight_grams * factor_ppm + 500_000) / 1_000_000)
func CalculateAvoidedCO2e(weightGrams int64, factorPPM int64) int64 {
	if weightGrams <= 0 || factorPPM <= 0 {
		return 0
	}
	return (weightGrams*factorPPM + 500000) / 1000000
}

func (s *service) ListFactors(ctx context.Context) ([]EmissionFactor, error) {
	rows, err := s.repo.ListActiveEmissionFactors(ctx)
	if err != nil {
		return nil, err
	}

	factors := make([]EmissionFactor, 0, len(rows))
	for _, r := range rows {
		factors = append(factors, EmissionFactor{
			ID:            db.FromPGUUID(r.ID),
			MaterialCode:  r.MaterialCode,
			Name:          r.Name,
			Category:      r.Category,
			FactorVersion: r.FactorVersion,
			FactorSource:  r.FactorSource,
			CO2eFactorPPM: r.Co2eFactorPpm,
			Description:   db.FromPGText(r.Description),
			IsActive:      r.IsActive,
			CreatedAt:     db.FromPGTimestamptz(r.CreatedAt),
		})
	}
	return factors, nil
}

func (s *service) GetNasabahCarbonSavings(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*NasabahCarbonSavingsResponse, error) {
	totalRow, err := s.repo.GetNasabahCarbonSavings(ctx, bankSampahID, nasabahID)
	if err != nil {
		return nil, err
	}

	categoryRows, err := s.repo.GetNasabahCarbonSavingsByCategory(ctx, bankSampahID, nasabahID)
	if err != nil {
		return nil, err
	}

	breakdown := make([]CategoryCarbonSavings, 0, len(categoryRows))
	for _, cr := range categoryRows {
		breakdown = append(breakdown, CategoryCarbonSavings{
			Category:                      cr.Category,
			TotalEstimatedCO2eAvoidedGrams: cr.TotalCo2eGrams,
			TotalWeightGrams:              cr.TotalWeightGrams,
		})
	}

	return &NasabahCarbonSavingsResponse{
		NasabahID:                      nasabahID,
		TotalEstimatedCO2eAvoidedGrams: totalRow.TotalCo2eGrams,
		TotalWeightGrams:              totalRow.TotalWeightGrams,
		TotalEntries:                  totalRow.TotalEntries,
		BreakdownByCategory:           breakdown,
	}, nil
}

func (s *service) GetTenantCarbonSavings(ctx context.Context, bankSampahID uuid.UUID) (*BankSampahCarbonSavingsResponse, error) {
	row, err := s.repo.GetBankSampahCarbonSavings(ctx, bankSampahID)
	if err != nil {
		return nil, err
	}

	return &BankSampahCarbonSavingsResponse{
		BankSampahID:                   bankSampahID,
		TotalEstimatedCO2eAvoidedGrams: row.TotalCo2eGrams,
		TotalWeightGrams:              row.TotalWeightGrams,
		TotalEntries:                  row.TotalEntries,
	}, nil
}

func (s *service) ListLedgerEntries(ctx context.Context, bankSampahID, requesterUserID uuid.UUID, requesterRole string, filter CarbonLedgerFilter) (*CarbonLedgerListResponse, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Hard Rule #1 & Invariant #10: Nasabah is strictly scoped to own records
	if requesterRole == "nasabah" {
		rows, err := s.repo.ListCarbonLedgerByNasabah(ctx, sqlc.ListCarbonLedgerByNasabahParams{
			BankSampahID: db.ToPGUUID(bankSampahID),
			NasabahID:    db.ToPGUUID(requesterUserID),
			Limit:        limit,
			Offset:       offset,
		})
		if err != nil {
			return nil, err
		}

		count, err := s.repo.CountCarbonLedgerByNasabah(ctx, sqlc.CountCarbonLedgerByNasabahParams{
			BankSampahID: db.ToPGUUID(bankSampahID),
			NasabahID:    db.ToPGUUID(requesterUserID),
		})
		if err != nil {
			return nil, err
		}

		entries := make([]CarbonLedgerEntry, 0, len(rows))
		for _, r := range rows {
			entries = append(entries, CarbonLedgerEntry{
				ID:               db.FromPGUUID(r.ID),
				BankSampahID:     db.FromPGUUID(r.BankSampahID),
				NasabahID:        db.FromPGUUID(r.NasabahID),
				DepositID:        db.FromPGUUIDPtr(r.DepositID),
				DepositLineID:    db.FromPGUUIDPtr(r.DepositLineID),
				EmissionFactorID: db.FromPGUUID(r.EmissionFactorID),
				FactorName:       r.FactorName,
				FactorCategory:   r.FactorCategory,
				FactorVersion:    r.FactorVersion,
				FactorSource:     r.FactorSource,
				WeightGrams:      r.WeightGrams,
				CO2eGrams:        r.Co2eGrams,
				EntryType:        r.EntryType,
				ReversalForID:    db.FromPGUUIDPtr(r.ReversalForID),
				Notes:            db.FromPGText(r.Notes),
				CreatedAt:        db.FromPGTimestamptz(r.CreatedAt),
			})
		}

		return &CarbonLedgerListResponse{
			Entries:    entries,
			TotalCount: count,
			Limit:      limit,
			Offset:     offset,
		}, nil
	}

	// Pengelola: can filter by nasabah_id, date ranges
	var nasabahUUID pgtype.UUID
	if filter.NasabahID != nil && *filter.NasabahID != uuid.Nil {
		nasabahUUID = db.ToPGUUID(*filter.NasabahID)
	}

	var startTS, endTS pgtype.Timestamptz
	if filter.StartDate != nil && !filter.StartDate.IsZero() {
		startTS = db.ToPGTimestamptz(*filter.StartDate)
	}
	if filter.EndDate != nil && !filter.EndDate.IsZero() {
		endTS = db.ToPGTimestamptz(*filter.EndDate)
	}

	rows, err := s.repo.ListCarbonLedgerByTenant(ctx, sqlc.ListCarbonLedgerByTenantParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Column2:      nasabahUUID,
		Column3:      startTS,
		Column4:      endTS,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountCarbonLedgerByTenant(ctx, sqlc.CountCarbonLedgerByTenantParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Column2:      nasabahUUID,
		Column3:      startTS,
		Column4:      endTS,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]CarbonLedgerEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, CarbonLedgerEntry{
			ID:               db.FromPGUUID(r.ID),
			BankSampahID:     db.FromPGUUID(r.BankSampahID),
			NasabahID:        db.FromPGUUID(r.NasabahID),
			NasabahName:      r.NasabahName,
			NasabahUsername:  r.NasabahUsername,
			DepositID:        db.FromPGUUIDPtr(r.DepositID),
			DepositLineID:    db.FromPGUUIDPtr(r.DepositLineID),
			EmissionFactorID: db.FromPGUUID(r.EmissionFactorID),
			FactorName:       r.FactorName,
			FactorCategory:   r.FactorCategory,
			FactorVersion:    r.FactorVersion,
			FactorSource:     r.FactorSource,
			WeightGrams:      r.WeightGrams,
			CO2eGrams:        r.Co2eGrams,
			EntryType:        r.EntryType,
			ReversalForID:    db.FromPGUUIDPtr(r.ReversalForID),
			Notes:            db.FromPGText(r.Notes),
			CreatedAt:        db.FromPGTimestamptz(r.CreatedAt),
		})
	}

	return &CarbonLedgerListResponse{
		Entries:    entries,
		TotalCount: count,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func (s *service) RecordReversal(ctx context.Context, bankSampahID, pengelolaID uuid.UUID, req CreateReversalRequest) (*CarbonLedgerEntry, error) {
	// 1. Retrieve original ledger entry
	orig, err := s.repo.GetCarbonLedgerByID(ctx, bankSampahID, req.CarbonLedgerID)
	if err != nil {
		return nil, err
	}
	if orig == nil {
		return nil, ErrCarbonLedgerEntryNotFound
	}

	// 2. Cannot reverse an entry that is already a reversal
	if orig.EntryType == "reversal" {
		return nil, ErrInvalidReversal
	}

	// 3. Create reversal entry with inverted values
	reversalParam := sqlc.CreateCarbonLedgerEntryParams{
		BankSampahID:     orig.BankSampahID,
		NasabahID:        orig.NasabahID,
		DepositID:        orig.DepositID,
		DepositLineID:    orig.DepositLineID,
		EmissionFactorID: orig.EmissionFactorID,
		FactorVersion:    orig.FactorVersion,
		FactorSource:     orig.FactorSource,
		WeightGrams:      -orig.WeightGrams,
		Co2eGrams:        -orig.Co2eGrams,
		EntryType:        "reversal",
		ReversalForID:    orig.ID,
		Notes:            db.ToPGText(req.Notes),
		CreatedAt:        db.ToPGTimestamptz(time.Now().UTC()),
	}

	created, err := s.repo.CreateCarbonLedgerEntry(ctx, reversalParam)
	if err != nil {
		return nil, err
	}

	return &CarbonLedgerEntry{
		ID:               db.FromPGUUID(created.ID),
		BankSampahID:     db.FromPGUUID(created.BankSampahID),
		NasabahID:        db.FromPGUUID(created.NasabahID),
		DepositID:        db.FromPGUUIDPtr(created.DepositID),
		DepositLineID:    db.FromPGUUIDPtr(created.DepositLineID),
		EmissionFactorID: db.FromPGUUID(created.EmissionFactorID),
		FactorName:       orig.FactorName,
		FactorCategory:   orig.FactorCategory,
		FactorVersion:    created.FactorVersion,
		FactorSource:     created.FactorSource,
		WeightGrams:      created.WeightGrams,
		CO2eGrams:        created.Co2eGrams,
		EntryType:        created.EntryType,
		ReversalForID:    db.FromPGUUIDPtr(created.ReversalForID),
		Notes:            db.FromPGText(created.Notes),
		CreatedAt:        db.FromPGTimestamptz(created.CreatedAt),
	}, nil
}
