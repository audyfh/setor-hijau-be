package cashbook

import (
	"context"
	"fmt"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service interface {
	// Sales to Pengepul
	RecordSale(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateSaleRequest) (*SaleResponse, error)
	GetSaleByID(ctx context.Context, bankSampahID, saleID uuid.UUID) (*SaleResponse, error)
	ListSales(ctx context.Context, bankSampahID uuid.UUID, filter SaleListFilter) (*SaleListResponse, error)

	// Nasabah Withdrawals
	RecordWithdrawal(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateWithdrawalRequest) (*WithdrawalResponse, error)
	ListWithdrawals(ctx context.Context, bankSampahID, requesterUserID uuid.UUID, requesterRole string, filter WithdrawalListFilter) (*WithdrawalListResponse, error)

	// Cashbook Entries
	CreateManualEntry(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateCashbookEntryRequest) (*CashbookEntryResponse, error)
	ListCashbookEntries(ctx context.Context, bankSampahID uuid.UUID, filter CashbookListFilter) (*CashbookListResponse, error)
	GetCashbookSummary(ctx context.Context, bankSampahID uuid.UUID) (*CashbookSummaryResponse, error)

	// Financial Reconciliation
	GetFinancialReconciliation(ctx context.Context, bankSampahID uuid.UUID) (*FinancialReconciliationResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// CalculateSubtotal calculates subtotal with symmetric half-up integer rounding: (weight_grams * price_per_kg_idr + 500) / 1000
func CalculateSubtotal(weightGrams, pricePerKgIDR int64) int64 {
	if weightGrams <= 0 || pricePerKgIDR <= 0 {
		return 0
	}
	return (weightGrams*pricePerKgIDR + 500) / 1000
}

func (s *service) RecordSale(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateSaleRequest) (*SaleResponse, error) {
	if len(req.Lines) == 0 {
		return nil, ErrEmptyLines
	}

	var totalWeight int64
	var totalAmount int64

	type calculatedLine struct {
		MaterialTypeID   uuid.UUID
		MaterialName     string
		MaterialCategory string
		MaterialUnit     string
		WeightGrams      int64
		PricePerKgIDR    int64
		SubtotalIDR      int64
	}

	var linesToCreate []calculatedLine

	for _, l := range req.Lines {
		if l.WeightGrams <= 0 {
			return nil, ErrInvalidWeight
		}
		mat, err := s.repo.GetMaterialTypeByID(ctx, bankSampahID, l.MaterialTypeID)
		if err != nil {
			return nil, fmt.Errorf("material %s tidak valid: %w", l.MaterialTypeID, err)
		}

		subtotal := CalculateSubtotal(l.WeightGrams, l.PricePerKgIDR)
		totalWeight += l.WeightGrams
		totalAmount += subtotal

		linesToCreate = append(linesToCreate, calculatedLine{
			MaterialTypeID:   l.MaterialTypeID,
			MaterialName:     mat.Name,
			MaterialCategory: mat.Category,
			MaterialUnit:     mat.Unit,
			WeightGrams:      l.WeightGrams,
			PricePerKgIDR:    l.PricePerKgIDR,
			SubtotalIDR:      subtotal,
		})
	}

	var createdSale *sqlc.SalesToPengepul
	var createdLineRows []sqlc.SalesToPengepulLine

	err := s.repo.WithTx(ctx, func(txRepo Repository) error {
		var notesPG pgtype.Text
		if req.Notes != nil && *req.Notes != "" {
			notesPG = pgtype.Text{String: *req.Notes, Valid: true}
		}

		sale, err := txRepo.CreateSale(ctx, sqlc.CreateSaleToPengepulParams{
			BankSampahID:     db.ToPGUUID(bankSampahID),
			BuyerName:        req.BuyerName,
			SaleDate:         db.ToPGTimestamptz(req.SaleDate),
			TotalWeightGrams: totalWeight,
			TotalAmountIdr:   totalAmount,
			Notes:            notesPG,
			RecordedBy:       db.ToPGUUID(recordedBy),
		})
		if err != nil {
			return fmt.Errorf("failed to create sale to pengepul: %w", err)
		}
		createdSale = sale

		for _, l := range linesToCreate {
			lineRow, err := txRepo.CreateSaleLine(ctx, sqlc.CreateSaleToPengepulLineParams{
				SaleID:         sale.ID,
				MaterialTypeID: db.ToPGUUID(l.MaterialTypeID),
				WeightGrams:    l.WeightGrams,
				PricePerKgIdr:  l.PricePerKgIDR,
				SubtotalIdr:    l.SubtotalIDR,
			})
			if err != nil {
				return fmt.Errorf("failed to create sale line: %w", err)
			}
			createdLineRows = append(createdLineRows, *lineRow)
		}

		// Automatic cashbook entry for cash_in from pengepul sale
		saleUUID := db.FromPGUUID(sale.ID)
		weightKg := float64(totalWeight) / 1000.0
		desc := fmt.Sprintf("Penjualan sampah ke %s (%.1f kg)", req.BuyerName, weightKg)

		_, err = txRepo.CreateCashbookEntry(ctx, sqlc.CreateCashbookEntryParams{
			BankSampahID:    db.ToPGUUID(bankSampahID),
			EntryType:       "cash_in",
			Category:        "penjualan_pengepul",
			AmountIdr:       totalAmount,
			ReferenceID:     db.ToPGUUID(saleUUID),
			Description:     desc,
			RecordedBy:      db.ToPGUUID(recordedBy),
			TransactionDate: db.ToPGTimestamptz(req.SaleDate),
		})
		if err != nil {
			return fmt.Errorf("failed to create cashbook entry for sale: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var lineResponses []SaleLineResponse
	for i, cl := range linesToCreate {
		lineID := uuid.Nil
		var createdAt time.Time
		if i < len(createdLineRows) {
			lineID = db.FromPGUUID(createdLineRows[i].ID)
			createdAt = db.FromPGTimestamptz(createdLineRows[i].CreatedAt)
		}
		lineResponses = append(lineResponses, SaleLineResponse{
			ID:               lineID,
			SaleID:           db.FromPGUUID(createdSale.ID),
			MaterialTypeID:   cl.MaterialTypeID,
			MaterialName:     cl.MaterialName,
			MaterialCategory: cl.MaterialCategory,
			MaterialUnit:     cl.MaterialUnit,
			WeightGrams:      cl.WeightGrams,
			PricePerKgIDR:    cl.PricePerKgIDR,
			SubtotalIDR:      cl.SubtotalIDR,
			CreatedAt:        createdAt,
		})
	}

	return &SaleResponse{
		ID:               db.FromPGUUID(createdSale.ID),
		BankSampahID:     bankSampahID,
		BuyerName:        createdSale.BuyerName,
		SaleDate:         db.FromPGTimestamptz(createdSale.SaleDate),
		TotalWeightGrams: createdSale.TotalWeightGrams,
		TotalAmountIDR:   createdSale.TotalAmountIdr,
		Notes:            req.Notes,
		RecordedBy:       recordedBy,
		RecordedByName:   "",
		CreatedAt:        db.FromPGTimestamptz(createdSale.CreatedAt),
		Lines:            lineResponses,
	}, nil
}

func (s *service) GetSaleByID(ctx context.Context, bankSampahID, saleID uuid.UUID) (*SaleResponse, error) {
	row, err := s.repo.GetSaleByID(ctx, bankSampahID, saleID)
	if err != nil {
		return nil, err
	}

	lines, err := s.repo.GetSaleLines(ctx, saleID)
	if err != nil {
		return nil, err
	}

	var lineResponses []SaleLineResponse
	for _, l := range lines {
		lineResponses = append(lineResponses, SaleLineResponse{
			ID:               db.FromPGUUID(l.ID),
			SaleID:           db.FromPGUUID(l.SaleID),
			MaterialTypeID:   db.FromPGUUID(l.MaterialTypeID),
			MaterialName:     l.MaterialName,
			MaterialCategory: l.MaterialCategory,
			MaterialUnit:     l.MaterialUnit,
			WeightGrams:      l.WeightGrams,
			PricePerKgIDR:    l.PricePerKgIdr,
			SubtotalIDR:      l.SubtotalIdr,
			CreatedAt:        db.FromPGTimestamptz(l.CreatedAt),
		})
	}

	var notesPtr *string
	if row.Notes.Valid && row.Notes.String != "" {
		notesPtr = &row.Notes.String
	}

	return &SaleResponse{
		ID:               db.FromPGUUID(row.ID),
		BankSampahID:     bankSampahID,
		BuyerName:        row.BuyerName,
		SaleDate:         db.FromPGTimestamptz(row.SaleDate),
		TotalWeightGrams: row.TotalWeightGrams,
		TotalAmountIDR:   row.TotalAmountIdr,
		Notes:            notesPtr,
		RecordedBy:       db.FromPGUUID(row.RecordedBy),
		RecordedByName:   row.RecordedByName,
		CreatedAt:        db.FromPGTimestamptz(row.CreatedAt),
		Lines:            lineResponses,
	}, nil
}

func (s *service) ListSales(ctx context.Context, bankSampahID uuid.UUID, filter SaleListFilter) (*SaleListResponse, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	offset := (filter.Page - 1) * filter.Limit

	var startDatePG pgtype.Timestamptz
	if filter.StartDate != nil {
		startDatePG = db.ToPGTimestamptz(*filter.StartDate)
	}

	var endDatePG pgtype.Timestamptz
	if filter.EndDate != nil {
		endDatePG = db.ToPGTimestamptz(*filter.EndDate)
	}

	rows, err := s.repo.ListSales(ctx, sqlc.ListSalesToPengepulParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Limit:        int32(filter.Limit),
		Offset:       int32(offset),
		StartDate:    startDatePG,
		EndDate:      endDatePG,
	})
	if err != nil {
		return nil, err
	}

	totalCount, err := s.repo.CountSales(ctx, sqlc.CountSalesToPengepulParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		StartDate:    startDatePG,
		EndDate:      endDatePG,
	})
	if err != nil {
		return nil, err
	}

	data := make([]SaleResponse, 0, len(rows))
	for _, r := range rows {
		var notesPtr *string
		if r.Notes.Valid && r.Notes.String != "" {
			notesPtr = &r.Notes.String
		}
		data = append(data, SaleResponse{
			ID:               db.FromPGUUID(r.ID),
			BankSampahID:     bankSampahID,
			BuyerName:        r.BuyerName,
			SaleDate:         db.FromPGTimestamptz(r.SaleDate),
			TotalWeightGrams: r.TotalWeightGrams,
			TotalAmountIDR:   r.TotalAmountIdr,
			Notes:            notesPtr,
			RecordedBy:       db.FromPGUUID(r.RecordedBy),
			RecordedByName:   r.RecordedByName,
			CreatedAt:        db.FromPGTimestamptz(r.CreatedAt),
			Lines:            nil,
		})
	}

	totalPages := int((totalCount + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &SaleListResponse{
		Data:       data,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *service) RecordWithdrawal(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateWithdrawalRequest) (*WithdrawalResponse, error) {
	if req.AmountIDR <= 0 {
		return nil, ErrInvalidAmount
	}

	balanceRow, err := s.repo.GetNasabahBalance(ctx, bankSampahID, req.NasabahID)
	if err != nil {
		return nil, err
	}

	if balanceRow.CurrentBalanceIdr < req.AmountIDR {
		return nil, ErrInsufficientBalance
	}

	var createdWithdrawal *sqlc.NasabahWithdrawal

	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		var notesPG pgtype.Text
		if req.Notes != nil && *req.Notes != "" {
			notesPG = pgtype.Text{String: *req.Notes, Valid: true}
		}

		w, err := txRepo.CreateWithdrawal(ctx, sqlc.CreateNasabahWithdrawalParams{
			BankSampahID: db.ToPGUUID(bankSampahID),
			NasabahID:    db.ToPGUUID(req.NasabahID),
			AmountIdr:    req.AmountIDR,
			Notes:        notesPG,
			RecordedBy:   db.ToPGUUID(recordedBy),
		})
		if err != nil {
			return fmt.Errorf("failed to create withdrawal: %w", err)
		}
		createdWithdrawal = w

		// Automatic cashbook entry for cash_out
		wUUID := db.FromPGUUID(w.ID)
		desc := fmt.Sprintf("Penarikan saldo tunai nasabah (Rp %d)", req.AmountIDR)

		_, err = txRepo.CreateCashbookEntry(ctx, sqlc.CreateCashbookEntryParams{
			BankSampahID:    db.ToPGUUID(bankSampahID),
			EntryType:       "cash_out",
			Category:        "penarikan_nasabah",
			AmountIdr:       req.AmountIDR,
			ReferenceID:     db.ToPGUUID(wUUID),
			Description:     desc,
			RecordedBy:      db.ToPGUUID(recordedBy),
			TransactionDate: db.ToPGTimestamptz(time.Now().UTC()),
		})
		if err != nil {
			return fmt.Errorf("failed to create cashbook entry for withdrawal: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	detail, err := s.repo.GetWithdrawalByID(ctx, bankSampahID, db.FromPGUUID(createdWithdrawal.ID))
	if err != nil {
		return nil, err
	}

	var notesPtr *string
	if detail.Notes.Valid && detail.Notes.String != "" {
		notesPtr = &detail.Notes.String
	}

	return &WithdrawalResponse{
		ID:                  db.FromPGUUID(detail.ID),
		BankSampahID:        bankSampahID,
		NasabahID:           db.FromPGUUID(detail.NasabahID),
		NasabahName:         detail.NasabahName,
		NasabahUsername:     detail.NasabahUsername,
		AmountIDR:           detail.AmountIdr,
		RemainingBalanceIDR: balanceRow.CurrentBalanceIdr - req.AmountIDR,
		Notes:               notesPtr,
		RecordedBy:          db.FromPGUUID(detail.RecordedBy),
		RecordedByName:      detail.RecordedByName,
		CreatedAt:           db.FromPGTimestamptz(detail.CreatedAt),
	}, nil
}

func (s *service) ListWithdrawals(ctx context.Context, bankSampahID, requesterUserID uuid.UUID, requesterRole string, filter WithdrawalListFilter) (*WithdrawalListResponse, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	offset := (filter.Page - 1) * filter.Limit

	targetNasabahID := filter.NasabahID
	if requesterRole == "nasabah" {
		targetNasabahID = &requesterUserID
	}

	var nasabahUUID pgtype.UUID
	if targetNasabahID != nil && *targetNasabahID != uuid.Nil {
		nasabahUUID = db.ToPGUUID(*targetNasabahID)
	}

	var startDatePG pgtype.Timestamptz
	if filter.StartDate != nil {
		startDatePG = db.ToPGTimestamptz(*filter.StartDate)
	}

	var endDatePG pgtype.Timestamptz
	if filter.EndDate != nil {
		endDatePG = db.ToPGTimestamptz(*filter.EndDate)
	}

	rows, err := s.repo.ListWithdrawals(ctx, sqlc.ListWithdrawalsByTenantParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Limit:        int32(filter.Limit),
		Offset:       int32(offset),
		NasabahID:    nasabahUUID,
		StartDate:    startDatePG,
		EndDate:      endDatePG,
	})
	if err != nil {
		return nil, err
	}

	totalCount, err := s.repo.CountWithdrawals(ctx, sqlc.CountWithdrawalsByTenantParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		NasabahID:    nasabahUUID,
		StartDate:    startDatePG,
		EndDate:      endDatePG,
	})
	if err != nil {
		return nil, err
	}

	data := make([]WithdrawalResponse, 0, len(rows))
	for _, r := range rows {
		var notesPtr *string
		if r.Notes.Valid && r.Notes.String != "" {
			notesPtr = &r.Notes.String
		}
		data = append(data, WithdrawalResponse{
			ID:                  db.FromPGUUID(r.ID),
			BankSampahID:        bankSampahID,
			NasabahID:           db.FromPGUUID(r.NasabahID),
			NasabahName:         r.NasabahName,
			NasabahUsername:     r.NasabahUsername,
			AmountIDR:           r.AmountIdr,
			RemainingBalanceIDR: 0,
			Notes:               notesPtr,
			RecordedBy:          db.FromPGUUID(r.RecordedBy),
			RecordedByName:      r.RecordedByName,
			CreatedAt:           db.FromPGTimestamptz(r.CreatedAt),
		})
	}

	totalPages := int((totalCount + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &WithdrawalListResponse{
		Data:       data,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *service) CreateManualEntry(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateCashbookEntryRequest) (*CashbookEntryResponse, error) {
	if req.AmountIDR <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.EntryType != "cash_in" && req.EntryType != "cash_out" {
		return nil, ErrInvalidEntryType
	}

	txDate := time.Now().UTC()
	if req.TransactionDate != nil && !req.TransactionDate.IsZero() {
		txDate = *req.TransactionDate
	}

	entry, err := s.repo.CreateCashbookEntry(ctx, sqlc.CreateCashbookEntryParams{
		BankSampahID:    db.ToPGUUID(bankSampahID),
		EntryType:       req.EntryType,
		Category:        req.Category,
		AmountIdr:       req.AmountIDR,
		ReferenceID:     pgtype.UUID{}, // Null
		Description:     req.Description,
		RecordedBy:      db.ToPGUUID(recordedBy),
		TransactionDate: db.ToPGTimestamptz(txDate),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create manual cashbook entry: %w", err)
	}

	return &CashbookEntryResponse{
		ID:              db.FromPGUUID(entry.ID),
		BankSampahID:    bankSampahID,
		EntryType:       entry.EntryType,
		Category:        entry.Category,
		AmountIDR:       entry.AmountIdr,
		ReferenceID:     nil,
		Description:     entry.Description,
		RecordedBy:      recordedBy,
		RecordedByName:  "",
		TransactionDate: db.FromPGTimestamptz(entry.TransactionDate),
		CreatedAt:       db.FromPGTimestamptz(entry.CreatedAt),
	}, nil
}

func (s *service) ListCashbookEntries(ctx context.Context, bankSampahID uuid.UUID, filter CashbookListFilter) (*CashbookListResponse, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	offset := (filter.Page - 1) * filter.Limit

	var entryTypePG pgtype.Text
	if filter.EntryType != nil && *filter.EntryType != "" {
		entryTypePG = pgtype.Text{String: *filter.EntryType, Valid: true}
	}

	var categoryPG pgtype.Text
	if filter.Category != nil && *filter.Category != "" {
		categoryPG = pgtype.Text{String: *filter.Category, Valid: true}
	}

	var startDatePG pgtype.Timestamptz
	if filter.StartDate != nil {
		startDatePG = db.ToPGTimestamptz(*filter.StartDate)
	}

	var endDatePG pgtype.Timestamptz
	if filter.EndDate != nil {
		endDatePG = db.ToPGTimestamptz(*filter.EndDate)
	}

	rows, err := s.repo.ListCashbookEntries(ctx, sqlc.ListCashbookEntriesParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Limit:        int32(filter.Limit),
		Offset:       int32(offset),
		EntryType:    entryTypePG,
		Category:     categoryPG,
		StartDate:    startDatePG,
		EndDate:      endDatePG,
	})
	if err != nil {
		return nil, err
	}

	totalCount, err := s.repo.CountCashbookEntries(ctx, sqlc.CountCashbookEntriesParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		EntryType:    entryTypePG,
		Category:     categoryPG,
		StartDate:    startDatePG,
		EndDate:      endDatePG,
	})
	if err != nil {
		return nil, err
	}

	data := make([]CashbookEntryResponse, 0, len(rows))
	for _, r := range rows {
		refID := db.FromPGUUIDPtr(r.ReferenceID)
		data = append(data, CashbookEntryResponse{
			ID:              db.FromPGUUID(r.ID),
			BankSampahID:    bankSampahID,
			EntryType:       r.EntryType,
			Category:        r.Category,
			AmountIDR:       r.AmountIdr,
			ReferenceID:     refID,
			Description:     r.Description,
			RecordedBy:      db.FromPGUUID(r.RecordedBy),
			RecordedByName:  r.RecordedByName,
			TransactionDate: db.FromPGTimestamptz(r.TransactionDate),
			CreatedAt:       db.FromPGTimestamptz(r.CreatedAt),
		})
	}

	totalPages := int((totalCount + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &CashbookListResponse{
		Data:       data,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *service) GetCashbookSummary(ctx context.Context, bankSampahID uuid.UUID) (*CashbookSummaryResponse, error) {
	row, err := s.repo.GetCashbookSummary(ctx, bankSampahID)
	if err != nil {
		return nil, err
	}

	return &CashbookSummaryResponse{
		TotalCashInIDR:        row.TotalCashInIdr,
		TotalCashOutIDR:       row.TotalCashOutIdr,
		CurrentCashBalanceIDR: row.CurrentCashBalanceIdr,
	}, nil
}

func (s *service) GetFinancialReconciliation(ctx context.Context, bankSampahID uuid.UUID) (*FinancialReconciliationResponse, error) {
	row, err := s.repo.GetFinancialReconciliation(ctx, bankSampahID)
	if err != nil {
		return nil, err
	}

	salesReconciled := (row.TotalSalesToPengepulIdr == row.TotalCashbookSalesIdr)
	withdrawalsReconciled := (row.TotalNasabahWithdrawalsIdr == row.TotalCashbookWithdrawalsIdr)
	netSurplus := row.CurrentCashOnHandIdr - row.TotalOutstandingNasabahBalanceIdr

	return &FinancialReconciliationResponse{
		TotalSalesToPengepulIDR:           row.TotalSalesToPengepulIdr,
		TotalCashbookSalesIDR:             row.TotalCashbookSalesIdr,
		SalesReconciled:                   salesReconciled,
		TotalNasabahWithdrawalsIDR:        row.TotalNasabahWithdrawalsIdr,
		TotalCashbookWithdrawalsIDR:       row.TotalCashbookWithdrawalsIdr,
		WithdrawalsReconciled:             withdrawalsReconciled,
		TotalDepositsAmountIDR:            row.TotalDepositsAmountIdr,
		CurrentCashOnHandIDR:              row.CurrentCashOnHandIdr,
		TotalOutstandingNasabahBalanceIDR: row.TotalOutstandingNasabahBalanceIdr,
		NetSurplusOrDeficitIDR:            netSurplus,
	}, nil
}
