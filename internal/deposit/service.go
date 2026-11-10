package deposit

import (
	"context"
	"fmt"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/carbon"
	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service defines business operations for waste deposits, balances, and digital receipts.
type Service interface {
	RecordDeposit(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateDepositRequest) (*DepositResponse, bool, error)
	BatchSync(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req BatchSyncRequest) (*BatchSyncResponse, error)
	GetDeposit(ctx context.Context, bankSampahID, depositID, requesterUserID uuid.UUID, requesterRole string) (*DepositResponse, error)
	GetReceipt(ctx context.Context, bankSampahID, depositID, requesterUserID uuid.UUID, requesterRole string) (*DigitalReceiptResponse, error)
	ListDeposits(ctx context.Context, bankSampahID, requesterUserID uuid.UUID, requesterRole string, filter DepositListFilter) (*DepositListResponse, error)
	GetNasabahBalance(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*NasabahBalanceResponse, error)
}

type service struct {
	repo Repository
}

// NewService creates a new deposit service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type lineCalculation struct {
	MaterialTypeID            uuid.UUID
	MaterialName              string
	MaterialCategory          string
	MaterialUnit              string
	WeightGrams               int64
	PricePerKgIDR             int64
	SubtotalIDR               int64
	EmissionFactorID          *uuid.UUID
	FactorVersion             string
	FactorSource              string
	EstimatedCO2eAvoidedGrams int64
}

func (s *service) RecordDeposit(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req CreateDepositRequest) (*DepositResponse, bool, error) {
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 64 {
		return nil, false, ErrInvalidIdempotencyKey
	}
	if req.ClientRecordedAt.IsZero() || req.ClientRecordedAt.After(time.Now().Add(5*time.Minute)) {
		return nil, false, ErrInvalidClientTime
	}
	if len(req.Lines) == 0 {
		return nil, false, ErrEmptyDepositLines
	}
	for _, l := range req.Lines {
		if l.WeightGrams <= 0 {
			return nil, false, ErrInvalidWeight
		}
	}

	// 1. Check idempotency: if already exists, return existing record (isReplay = true)
	existing, err := s.repo.GetDepositByIdempotencyKey(ctx, bankSampahID, req.IdempotencyKey)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		resp, err := s.GetDeposit(ctx, bankSampahID, db.FromPGUUID(existing.ID), uuid.Nil, "pengelola")
		if err != nil {
			return nil, false, err
		}
		return resp, true, nil
	}

	// 2. Validate Nasabah
	nasabah, err := s.repo.ValidateNasabah(ctx, bankSampahID, req.NasabahID)
	if err != nil {
		return nil, false, err
	}

	// 3. Resolve prices at client_recorded_at and calculate line subtotals & avoided emissions
	var calculatedLines []lineCalculation
	var totalWeight int64
	var totalAmount int64
	var totalAvoidedCO2e int64

	for _, lineReq := range req.Lines {
		priceRow, err := s.repo.GetMaterialPriceAtTime(ctx, bankSampahID, lineReq.MaterialTypeID, req.ClientRecordedAt)
		if err != nil {
			return nil, false, err
		}
		if priceRow == nil || !priceRow.MaterialIsActive {
			return nil, false, ErrMaterialNotFound
		}
		if !priceRow.PricePerKgIdr.Valid {
			return nil, false, ErrMaterialPriceNotSet
		}

		pricePerKg := priceRow.PricePerKgIdr.Int64
		subtotal := CalculateSubtotal(lineReq.WeightGrams, pricePerKg)

		var factorID *uuid.UUID
		var factorVersion, factorSource string
		var avoidedCO2e int64
		if priceRow.EmissionFactorID.Valid && priceRow.Co2eFactorPpm.Valid && priceRow.Co2eFactorPpm.Int64 > 0 {
			fid := db.FromPGUUID(priceRow.EmissionFactorID)
			factorID = &fid
			factorVersion = db.FromPGText(priceRow.FactorVersion)
			factorSource = db.FromPGText(priceRow.FactorSource)
			avoidedCO2e = carbon.CalculateAvoidedCO2e(lineReq.WeightGrams, priceRow.Co2eFactorPpm.Int64)
		}

		calculatedLines = append(calculatedLines, lineCalculation{
			MaterialTypeID:            lineReq.MaterialTypeID,
			MaterialName:              priceRow.MaterialName,
			MaterialCategory:          priceRow.MaterialCategory,
			MaterialUnit:              priceRow.MaterialUnit,
			WeightGrams:               lineReq.WeightGrams,
			PricePerKgIDR:             pricePerKg,
			SubtotalIDR:               subtotal,
			EmissionFactorID:          factorID,
			FactorVersion:             factorVersion,
			FactorSource:              factorSource,
			EstimatedCO2eAvoidedGrams: avoidedCO2e,
		})

		totalWeight += lineReq.WeightGrams
		totalAmount += subtotal
		totalAvoidedCO2e += avoidedCO2e
	}

	// 4. Persist in database transaction
	var createdDeposit *sqlc.Deposit
	var createdLineRows []sqlc.DepositLine

	var notesText pgtype.Text
	if req.Notes != nil && *req.Notes != "" {
		notesText = db.ToPGText(*req.Notes)
	}

	err = s.repo.ExecTx(ctx, func(q *sqlc.Queries) error {
		depositRow, err := s.repo.CreateDepositTx(ctx, q, sqlc.CreateDepositParams{
			BankSampahID:     db.ToPGUUID(bankSampahID),
			NasabahID:        db.ToPGUUID(req.NasabahID),
			IdempotencyKey:   req.IdempotencyKey,
			TotalWeightGrams: totalWeight,
			TotalAmountIdr:   totalAmount,
			Notes:            notesText,
			RecordedBy:       db.ToPGUUID(recordedBy),
			ClientRecordedAt: db.ToPGTimestamptz(req.ClientRecordedAt),
		})
		if err != nil {
			return err
		}
		createdDeposit = depositRow

		for _, cl := range calculatedLines {
			lineRow, err := s.repo.CreateDepositLineTx(ctx, q, sqlc.CreateDepositLineParams{
				DepositID:      depositRow.ID,
				MaterialTypeID: db.ToPGUUID(cl.MaterialTypeID),
				WeightGrams:    cl.WeightGrams,
				PricePerKgIdr:  cl.PricePerKgIDR,
				SubtotalIdr:    cl.SubtotalIDR,
			})
			if err != nil {
				return err
			}
			createdLineRows = append(createdLineRows, *lineRow)

			// Hard Rule #2 & Invariant #5: Record to carbon ledger atomically if material has emission factor
			if cl.EmissionFactorID != nil && *cl.EmissionFactorID != uuid.Nil {
				_, err = q.CreateCarbonLedgerEntry(ctx, sqlc.CreateCarbonLedgerEntryParams{
					BankSampahID:     db.ToPGUUID(bankSampahID),
					NasabahID:        db.ToPGUUID(req.NasabahID),
					DepositID:        depositRow.ID,
					DepositLineID:    lineRow.ID,
					EmissionFactorID: db.ToPGUUID(*cl.EmissionFactorID),
					FactorVersion:    cl.FactorVersion,
					FactorSource:     cl.FactorSource,
					WeightGrams:      cl.WeightGrams,
					Co2eGrams:        cl.EstimatedCO2eAvoidedGrams,
					EntryType:        "deposit",
					ReversalForID:    pgtype.UUID{}, // NULL
					Notes:            pgtype.Text{}, // NULL
					CreatedAt:        db.ToPGTimestamptz(req.ClientRecordedAt),
				})
				if err != nil {
					return fmt.Errorf("failed to create carbon ledger entry: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	// 5. Build response
	var lineResponses []DepositLineResponse
	for i, cl := range calculatedLines {
		lineID := uuid.Nil
		var createdAt time.Time
		if i < len(createdLineRows) {
			lineID = db.FromPGUUID(createdLineRows[i].ID)
			createdAt = db.FromPGTimestamptz(createdLineRows[i].CreatedAt)
		}
		lineResponses = append(lineResponses, DepositLineResponse{
			ID:                        lineID,
			DepositID:                 db.FromPGUUID(createdDeposit.ID),
			MaterialTypeID:            cl.MaterialTypeID,
			MaterialName:              cl.MaterialName,
			MaterialCategory:          cl.MaterialCategory,
			MaterialUnit:              cl.MaterialUnit,
			WeightGrams:               cl.WeightGrams,
			PricePerKgIDR:             cl.PricePerKgIDR,
			SubtotalIDR:               cl.SubtotalIDR,
			EstimatedCO2eAvoidedGrams: cl.EstimatedCO2eAvoidedGrams,
			FactorVersion:             cl.FactorVersion,
			FactorSource:              cl.FactorSource,
			CreatedAt:                 createdAt,
		})
	}

	return &DepositResponse{
		ID:                             db.FromPGUUID(createdDeposit.ID),
		BankSampahID:                   bankSampahID,
		NasabahID:                      req.NasabahID,
		NasabahName:                    nasabah.FullName,
		NasabahUsername:                nasabah.Username,
		IdempotencyKey:                 req.IdempotencyKey,
		TotalWeightGrams:               totalWeight,
		TotalAmountIDR:                 totalAmount,
		TotalEstimatedCO2eAvoidedGrams: totalAvoidedCO2e,
		Notes:                          req.Notes,
		RecordedBy:                     recordedBy,
		RecordedByName:                 "",
		ClientRecordedAt:               req.ClientRecordedAt,
		CreatedAt:                      db.FromPGTimestamptz(createdDeposit.CreatedAt),
		Lines:                          lineResponses,
	}, false, nil
}

func (s *service) BatchSync(ctx context.Context, bankSampahID, recordedBy uuid.UUID, req BatchSyncRequest) (*BatchSyncResponse, error) {
	resp := &BatchSyncResponse{
		TotalProcessed: len(req.Deposits),
		Results:        make([]BatchSyncItemResult, 0, len(req.Deposits)),
	}

	for _, d := range req.Deposits {
		depResp, isReplay, err := s.RecordDeposit(ctx, bankSampahID, recordedBy, d)
		if err != nil {
			errStr := err.Error()
			resp.TotalFailed++
			resp.Results = append(resp.Results, BatchSyncItemResult{
				IdempotencyKey: d.IdempotencyKey,
				Status:         "failed",
				Error:          &errStr,
			})
			continue
		}

		status := "created"
		if isReplay {
			status = "replayed"
		}
		resp.TotalSuccess++
		resp.Results = append(resp.Results, BatchSyncItemResult{
			IdempotencyKey: d.IdempotencyKey,
			Status:         status,
			Deposit:        depResp,
		})
	}

	return resp, nil
}

func (s *service) GetDeposit(ctx context.Context, bankSampahID, depositID, requesterUserID uuid.UUID, requesterRole string) (*DepositResponse, error) {
	row, err := s.repo.GetDepositByID(ctx, bankSampahID, depositID)
	if err != nil {
		return nil, err
	}

	nasabahID := db.FromPGUUID(row.NasabahID)
	if requesterRole == "nasabah" && requesterUserID != uuid.Nil && nasabahID != requesterUserID {
		return nil, ErrUnauthorizedAccess
	}

	lines, err := s.repo.GetDepositLinesByDepositID(ctx, depositID)
	if err != nil {
		return nil, err
	}

	var lineResponses []DepositLineResponse
	for _, l := range lines {
		lineResponses = append(lineResponses, DepositLineResponse{
			ID:                        db.FromPGUUID(l.ID),
			DepositID:                 db.FromPGUUID(l.DepositID),
			MaterialTypeID:            db.FromPGUUID(l.MaterialTypeID),
			MaterialName:              l.MaterialName,
			MaterialCategory:          l.MaterialCategory,
			MaterialUnit:              l.MaterialUnit,
			WeightGrams:               l.WeightGrams,
			PricePerKgIDR:             l.PricePerKgIdr,
			SubtotalIDR:               l.SubtotalIdr,
			EstimatedCO2eAvoidedGrams: l.EstimatedCo2eAvoidedGrams,
			FactorVersion:             l.FactorVersion,
			FactorSource:              l.FactorSource,
			CreatedAt:                 db.FromPGTimestamptz(l.CreatedAt),
		})
	}

	var notesPtr *string
	if row.Notes.Valid && row.Notes.String != "" {
		notesPtr = &row.Notes.String
	}

	return &DepositResponse{
		ID:                             db.FromPGUUID(row.ID),
		BankSampahID:                   bankSampahID,
		NasabahID:                      nasabahID,
		NasabahName:                    row.NasabahName,
		NasabahUsername:                row.NasabahUsername,
		IdempotencyKey:                 row.IdempotencyKey,
		TotalWeightGrams:               row.TotalWeightGrams,
		TotalAmountIDR:                 row.TotalAmountIdr,
		TotalEstimatedCO2eAvoidedGrams: row.TotalEstimatedCo2eAvoidedGrams,
		Notes:                          notesPtr,
		RecordedBy:                     db.FromPGUUID(row.RecordedBy),
		RecordedByName:                 row.RecordedByName,
		ClientRecordedAt:               db.FromPGTimestamptz(row.ClientRecordedAt),
		CreatedAt:                      db.FromPGTimestamptz(row.CreatedAt),
		Lines:                          lineResponses,
	}, nil
}

func (s *service) GetReceipt(ctx context.Context, bankSampahID, depositID, requesterUserID uuid.UUID, requesterRole string) (*DigitalReceiptResponse, error) {
	row, err := s.repo.GetDepositReceiptDetails(ctx, bankSampahID, depositID)
	if err != nil {
		return nil, err
	}

	nasabahID := db.FromPGUUID(row.NasabahID)
	if requesterRole == "nasabah" && requesterUserID != uuid.Nil && nasabahID != requesterUserID {
		return nil, ErrUnauthorizedAccess
	}

	lines, err := s.repo.GetDepositLinesByDepositID(ctx, depositID)
	if err != nil {
		return nil, err
	}

	var lineResponses []DepositLineResponse
	for _, l := range lines {
		lineResponses = append(lineResponses, DepositLineResponse{
			ID:                        db.FromPGUUID(l.ID),
			DepositID:                 db.FromPGUUID(l.DepositID),
			MaterialTypeID:            db.FromPGUUID(l.MaterialTypeID),
			MaterialName:              l.MaterialName,
			MaterialCategory:          l.MaterialCategory,
			MaterialUnit:              l.MaterialUnit,
			WeightGrams:               l.WeightGrams,
			PricePerKgIDR:             l.PricePerKgIdr,
			SubtotalIDR:               l.SubtotalIdr,
			EstimatedCO2eAvoidedGrams: l.EstimatedCo2eAvoidedGrams,
			FactorVersion:             l.FactorVersion,
			FactorSource:              l.FactorSource,
			CreatedAt:                 db.FromPGTimestamptz(l.CreatedAt),
		})
	}

	balanceRow, err := s.repo.GetNasabahBalance(ctx, bankSampahID, nasabahID)
	if err != nil {
		return nil, err
	}

	var notesPtr *string
	if row.Notes.Valid && row.Notes.String != "" {
		notesPtr = &row.Notes.String
	}

	var bsRtPtr *string
	if row.BankSampahRt.Valid && row.BankSampahRt.String != "" {
		bsRtPtr = &row.BankSampahRt.String
	}

	var bsPhonePtr *string
	if row.BankSampahContactPhone.Valid && row.BankSampahContactPhone.String != "" {
		bsPhonePtr = &row.BankSampahContactPhone.String
	}

	var nasabahRtPtr *string
	if row.NasabahRt.Valid && row.NasabahRt.String != "" {
		nasabahRtPtr = &row.NasabahRt.String
	}

	var nasabahPhonePtr *string
	if row.NasabahPhone.Valid && row.NasabahPhone.String != "" {
		nasabahPhonePtr = &row.NasabahPhone.String
	}

	clientTime := db.FromPGTimestamptz(row.ClientRecordedAt)
	receiptNumber := fmt.Sprintf("STR-%s-%s-%s",
		row.BankSampahCode,
		clientTime.Format("0601021504"),
		depositID.String()[:8],
	)

	return &DigitalReceiptResponse{
		DepositID:                      depositID,
		ReceiptNumber:                  receiptNumber,
		BankSampahID:                   bankSampahID,
		BankSampahName:                 row.BankSampahName,
		BankSampahCode:                 row.BankSampahCode,
		BankSampahAddress:              row.BankSampahAddress,
		BankSampahRW:                   row.BankSampahRw,
		BankSampahRT:                   bsRtPtr,
		BankSampahKelurahan:            row.BankSampahKelurahan,
		BankSampahKecamatan:            row.BankSampahKecamatan,
		BankSampahKabupatenKota:        row.BankSampahKabupatenKota,
		BankSampahContactPhone:         bsPhonePtr,
		NasabahID:                      nasabahID,
		NasabahName:                    row.NasabahName,
		NasabahUsername:                row.NasabahUsername,
		NasabahRW:                      row.NasabahRw,
		NasabahRT:                      nasabahRtPtr,
		NasabahPhone:                   nasabahPhonePtr,
		RecordedByID:                   db.FromPGUUID(row.RecordedByID),
		RecordedByName:                 row.RecordedByName,
		TotalWeightGrams:               row.TotalWeightGrams,
		TotalAmountIDR:                 row.TotalAmountIdr,
		TotalEstimatedCO2eAvoidedGrams: row.TotalEstimatedCo2eAvoidedGrams,
		NasabahCurrentBalance:          balanceRow.CurrentBalanceIdr,
		Notes:                          notesPtr,
		ClientRecordedAt:               clientTime,
		CreatedAt:                      db.FromPGTimestamptz(row.CreatedAt),
		Lines:                          lineResponses,
	}, nil
}

func (s *service) ListDeposits(ctx context.Context, bankSampahID, requesterUserID uuid.UUID, requesterRole string, filter DepositListFilter) (*DepositListResponse, error) {
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

	// Nasabah can only query their own deposits
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

	rows, err := s.repo.ListDepositsByTenant(ctx, sqlc.ListDepositsByTenantParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Column2:      nasabahUUID,
		Column3:      startDatePG,
		Column4:      endDatePG,
		Limit:        int32(filter.Limit),
		Offset:       int32(offset),
	})
	if err != nil {
		return nil, err
	}

	totalCount, err := s.repo.CountDepositsByTenant(ctx, sqlc.CountDepositsByTenantParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Column2:      nasabahUUID,
		Column3:      startDatePG,
		Column4:      endDatePG,
	})
	if err != nil {
		return nil, err
	}

	data := make([]DepositResponse, 0, len(rows))
	for _, r := range rows {
		var notesPtr *string
		if r.Notes.Valid && r.Notes.String != "" {
			notesPtr = &r.Notes.String
		}
		data = append(data, DepositResponse{
			ID:                             db.FromPGUUID(r.ID),
			BankSampahID:                   bankSampahID,
			NasabahID:                      db.FromPGUUID(r.NasabahID),
			NasabahName:                    r.NasabahName,
			NasabahUsername:                r.NasabahUsername,
			IdempotencyKey:                 r.IdempotencyKey,
			TotalWeightGrams:               r.TotalWeightGrams,
			TotalAmountIDR:                 r.TotalAmountIdr,
			TotalEstimatedCO2eAvoidedGrams: r.TotalEstimatedCo2eAvoidedGrams,
			Notes:                          notesPtr,
			RecordedBy:                     db.FromPGUUID(r.RecordedBy),
			RecordedByName:                 r.RecordedByName,
			ClientRecordedAt:               db.FromPGTimestamptz(r.ClientRecordedAt),
			CreatedAt:                      db.FromPGTimestamptz(r.CreatedAt),
			Lines:                          nil, // Lines omitted in list view for Green Software performance
		})
	}

	totalPages := int((totalCount + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 && totalCount == 0 {
		totalPages = 1
	}

	return &DepositListResponse{
		Data:       data,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *service) GetNasabahBalance(ctx context.Context, bankSampahID, nasabahID uuid.UUID) (*NasabahBalanceResponse, error) {
	_, err := s.repo.ValidateNasabah(ctx, bankSampahID, nasabahID)
	if err != nil {
		return nil, err
	}

	row, err := s.repo.GetNasabahBalance(ctx, bankSampahID, nasabahID)
	if err != nil {
		return nil, err
	}

	var lastDepositTime *time.Time
	if row.LastDepositAt.Valid {
		t := db.FromPGTimestamptz(row.LastDepositAt)
		lastDepositTime = &t
	}

	return &NasabahBalanceResponse{
		NasabahID:          nasabahID,
		CurrentBalanceIDR:  row.CurrentBalanceIdr,
		TotalWeightGrams:   row.TotalWeightGrams,
		TotalDepositsCount: row.TotalDepositsCount,
		LastDepositAt:      lastDepositTime,
	}, nil
}
