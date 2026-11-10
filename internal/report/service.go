package report

import (
	"context"
	"fmt"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
)

type Service interface {
	GetMonthlyDashboard(ctx context.Context, bankSampahID uuid.UUID, monthStr string) (*MonthlyDashboardResponse, error)
	ExportMonthlyCSV(ctx context.Context, bankSampahID uuid.UUID, monthStr string) ([]byte, string, error)
	ExportMonthlyPDF(ctx context.Context, bankSampahID uuid.UUID, monthStr string) ([]byte, string, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// ParseMonthPeriod parses "YYYY-MM" into start and end of that month in UTC.
func ParseMonthPeriod(monthStr string) (time.Time, time.Time, string, error) {
	if monthStr == "" {
		monthStr = time.Now().UTC().Format("2006-01")
	}

	parsed, err := time.Parse("2006-01", monthStr)
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("format bulan tidak valid (gunakan YYYY-MM): %w", err)
	}

	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return start, end, monthStr, nil
}

func (s *service) GetMonthlyDashboard(ctx context.Context, bankSampahID uuid.UUID, monthStr string) (*MonthlyDashboardResponse, error) {
	start, end, period, err := ParseMonthPeriod(monthStr)
	if err != nil {
		return nil, err
	}

	startPG := db.ToPGTimestamptz(start)
	endPG := db.ToPGTimestamptz(end)
	bsUUID := db.ToPGUUID(bankSampahID)

	// 1. Deposits summary & categories
	depSummary, err := s.repo.GetMonthlyDepositSummary(ctx, sqlc.GetMonthlyDepositSummaryParams{
		BankSampahID:     bsUUID,
		ClientRecordedAt: startPG,
		ClientRecordedAt_2: endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deposit summary: %w", err)
	}

	depCategories, err := s.repo.GetMonthlyDepositsByCategory(ctx, sqlc.GetMonthlyDepositsByCategoryParams{
		BankSampahID:     bsUUID,
		ClientRecordedAt: startPG,
		ClientRecordedAt_2: endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deposit categories: %w", err)
	}

	var depCatResponses []CategoryWeightAmount
	for _, c := range depCategories {
		depCatResponses = append(depCatResponses, CategoryWeightAmount{
			Category:         c.Category,
			TotalWeightGrams: c.TotalWeightGrams,
			TotalWeightKg:    float64(c.TotalWeightGrams) / 1000.0,
			TotalAmountIDR:   c.TotalSubtotalIdr,
		})
	}

	// 2. Sales to Pengepul summary & categories
	saleSummary, err := s.repo.GetMonthlySalesSummary(ctx, sqlc.GetMonthlySalesSummaryParams{
		BankSampahID: bsUUID,
		SaleDate:     startPG,
		SaleDate_2:   endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sales summary: %w", err)
	}

	saleCategories, err := s.repo.GetMonthlySalesByCategory(ctx, sqlc.GetMonthlySalesByCategoryParams{
		BankSampahID: bsUUID,
		SaleDate:     startPG,
		SaleDate_2:   endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sales categories: %w", err)
	}

	var saleCatResponses []CategoryWeightAmount
	for _, c := range saleCategories {
		saleCatResponses = append(saleCatResponses, CategoryWeightAmount{
			Category:         c.Category,
			TotalWeightGrams: c.TotalWeightGrams,
			TotalWeightKg:    float64(c.TotalWeightGrams) / 1000.0,
			TotalAmountIDR:   c.TotalSubtotalIdr,
		})
	}

	// 3. Cashbook summary & categories
	cashSummary, err := s.repo.GetMonthlyCashbookSummary(ctx, sqlc.GetMonthlyCashbookSummaryParams{
		BankSampahID:    bsUUID,
		TransactionDate: startPG,
		TransactionDate_2: endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cashbook summary: %w", err)
	}

	cashCategories, err := s.repo.GetMonthlyCashbookByCategory(ctx, sqlc.GetMonthlyCashbookByCategoryParams{
		BankSampahID:    bsUUID,
		TransactionDate: startPG,
		TransactionDate_2: endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cashbook categories: %w", err)
	}

	var cashCatResponses []CategoryCashFlow
	for _, c := range cashCategories {
		cashCatResponses = append(cashCatResponses, CategoryCashFlow{
			Category:       c.Category,
			EntryType:      c.EntryType,
			TotalAmountIDR: c.TotalAmountIdr,
			Count:          c.Count,
		})
	}

	// Current cash on hand all-time
	allTimeCash, err := s.repo.GetCashbookSummary(ctx, bankSampahID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all-time cash on hand: %w", err)
	}

	// 4. Carbon savings summary & categories
	carbonSummary, err := s.repo.GetMonthlyCarbonSummary(ctx, sqlc.GetMonthlyCarbonSummaryParams{
		BankSampahID: bsUUID,
		CreatedAt:    startPG,
		CreatedAt_2:  endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch carbon summary: %w", err)
	}

	carbonCategories, err := s.repo.GetMonthlyCarbonByCategory(ctx, sqlc.GetMonthlyCarbonByCategoryParams{
		BankSampahID: bsUUID,
		CreatedAt:    startPG,
		CreatedAt_2:  endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch carbon categories: %w", err)
	}

	var carbonCatResponses []CategoryCarbon
	for _, c := range carbonCategories {
		carbonCatResponses = append(carbonCatResponses, CategoryCarbon{
			Category:         c.Category,
			TotalCO2eGrams:   c.TotalCo2eGrams,
			TotalCO2eKg:      float64(c.TotalCo2eGrams) / 1000.0,
			TotalWeightGrams: c.TotalWeightGrams,
		})
	}

	grossMargin := saleSummary.TotalAmountIdr - depSummary.TotalAmountIdr
	netCashFlow := cashSummary.TotalCashInIdr - cashSummary.TotalCashOutIdr

	return &MonthlyDashboardResponse{
		Period:    period,
		StartDate: start,
		EndDate:   end,
		Deposits: DepositDashboardMetrics{
			DepositsCount:      depSummary.DepositsCount,
			ActiveNasabahCount: depSummary.ActiveNasabahCount,
			TotalWeightGrams:   depSummary.TotalWeightGrams,
			TotalWeightKg:      float64(depSummary.TotalWeightGrams) / 1000.0,
			TotalAmountIDR:     depSummary.TotalAmountIdr,
			Categories:         depCatResponses,
		},
		Sales: SalesDashboardMetrics{
			SalesCount:       saleSummary.SalesCount,
			TotalWeightGrams: saleSummary.TotalWeightGrams,
			TotalWeightKg:    float64(saleSummary.TotalWeightGrams) / 1000.0,
			TotalAmountIDR:   saleSummary.TotalAmountIdr,
			Categories:       saleCatResponses,
		},
		Cashbook: CashbookDashboardMetrics{
			TotalCashInIDR:       cashSummary.TotalCashInIdr,
			TotalCashOutIDR:      cashSummary.TotalCashOutIdr,
			NetCashFlowIDR:       netCashFlow,
			CurrentCashOnHandIDR: allTimeCash.CurrentCashBalanceIdr,
			Categories:           cashCatResponses,
		},
		Carbon: CarbonDashboardMetrics{
			TotalCO2eGrams:   carbonSummary.TotalCo2eGrams,
			TotalCO2eKg:      float64(carbonSummary.TotalCo2eGrams) / 1000.0,
			TotalWeightGrams: carbonSummary.TotalWeightGrams,
			Categories:       carbonCatResponses,
		},
		ProfitSummary: ProfitDashboardSummary{
			GrossMarginIDR: grossMargin,
		},
	}, nil
}

func (s *service) getMonthlyReportData(ctx context.Context, bankSampahID uuid.UUID, monthStr string) (*MonthlyReportData, error) {
	dashboard, err := s.GetMonthlyDashboard(ctx, bankSampahID, monthStr)
	if err != nil {
		return nil, err
	}

	profileRow, err := s.repo.GetBankSampahProfile(ctx, bankSampahID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bank sampah profile: %w", err)
	}

	var rtPtr *string
	if profileRow.Rt.Valid && profileRow.Rt.String != "" {
		rtPtr = &profileRow.Rt.String
	}
	var phonePtr *string
	if profileRow.ContactPhone.Valid && profileRow.ContactPhone.String != "" {
		phonePtr = &profileRow.ContactPhone.String
	}

	profile := BankSampahProfile{
		ID:            db.FromPGUUID(profileRow.ID),
		Name:          profileRow.Name,
		Code:          profileRow.Code,
		Address:       profileRow.Address,
		RW:            profileRow.Rw,
		RT:            rtPtr,
		Kelurahan:     profileRow.Kelurahan,
		Kecamatan:     profileRow.Kecamatan,
		KabupatenKota: profileRow.KabupatenKota,
		ContactPhone:  phonePtr,
	}

	startPG := db.ToPGTimestamptz(dashboard.StartDate)
	endPG := db.ToPGTimestamptz(dashboard.EndDate)
	bsUUID := db.ToPGUUID(bankSampahID)

	// Detail rows
	depRows, err := s.repo.GetMonthlyDepositsDetail(ctx, sqlc.GetMonthlyDepositsDetailForReportParams{
		BankSampahID:     bsUUID,
		ClientRecordedAt: startPG,
		ClientRecordedAt_2: endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deposit details: %w", err)
	}

	var deposits []ReportDepositDetail
	for _, d := range depRows {
		var notesPtr *string
		if d.Notes.Valid && d.Notes.String != "" {
			notesPtr = &d.Notes.String
		}
		deposits = append(deposits, ReportDepositDetail{
			ID:                             db.FromPGUUID(d.ID),
			ClientRecordedAt:               db.FromPGTimestamptz(d.ClientRecordedAt),
			NasabahName:                    d.NasabahName,
			NasabahUsername:                d.NasabahUsername,
			TotalWeightGrams:               d.TotalWeightGrams,
			TotalAmountIDR:                 d.TotalAmountIdr,
			TotalEstimatedCO2eAvoidedGrams: d.TotalEstimatedCo2eAvoidedGrams,
			Notes:                          notesPtr,
		})
	}

	saleRows, err := s.repo.GetMonthlySalesDetail(ctx, sqlc.GetMonthlySalesDetailForReportParams{
		BankSampahID: bsUUID,
		SaleDate:     startPG,
		SaleDate_2:   endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sale details: %w", err)
	}

	var sales []ReportSaleDetail
	for _, r := range saleRows {
		var notesPtr *string
		if r.Notes.Valid && r.Notes.String != "" {
			notesPtr = &r.Notes.String
		}
		sales = append(sales, ReportSaleDetail{
			ID:               db.FromPGUUID(r.ID),
			SaleDate:         db.FromPGTimestamptz(r.SaleDate),
			BuyerName:        r.BuyerName,
			TotalWeightGrams: r.TotalWeightGrams,
			TotalAmountIDR:   r.TotalAmountIdr,
			Notes:            notesPtr,
		})
	}

	cashRows, err := s.repo.GetMonthlyCashbookDetail(ctx, sqlc.GetMonthlyCashbookDetailForReportParams{
		BankSampahID:    bsUUID,
		TransactionDate: startPG,
		TransactionDate_2: endPG,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cashbook details: %w", err)
	}

	var cashbook []ReportCashbookDetail
	for _, c := range cashRows {
		cashbook = append(cashbook, ReportCashbookDetail{
			ID:              db.FromPGUUID(c.ID),
			TransactionDate: db.FromPGTimestamptz(c.TransactionDate),
			EntryType:       c.EntryType,
			Category:        c.Category,
			AmountIDR:       c.AmountIdr,
			Description:     c.Description,
			RecordedByName:  c.RecordedByName,
		})
	}

	return &MonthlyReportData{
		Profile:   profile,
		Dashboard: *dashboard,
		Deposits:  deposits,
		Sales:     sales,
		Cashbook:  cashbook,
	}, nil
}

func (s *service) ExportMonthlyCSV(ctx context.Context, bankSampahID uuid.UUID, monthStr string) ([]byte, string, error) {
	data, err := s.getMonthlyReportData(ctx, bankSampahID, monthStr)
	if err != nil {
		return nil, "", err
	}

	bytes, err := GenerateMonthlyCSV(data)
	if err != nil {
		return nil, "", err
	}

	fileName := fmt.Sprintf("laporan-bulanan-%s-%s.csv", data.Profile.Code, data.Dashboard.Period)
	return bytes, fileName, nil
}

func (s *service) ExportMonthlyPDF(ctx context.Context, bankSampahID uuid.UUID, monthStr string) ([]byte, string, error) {
	data, err := s.getMonthlyReportData(ctx, bankSampahID, monthStr)
	if err != nil {
		return nil, "", err
	}

	bytes, err := GenerateMonthlyPDF(data)
	if err != nil {
		return nil, "", err
	}

	fileName := fmt.Sprintf("laporan-bulanan-%s-%s.pdf", data.Profile.Code, data.Dashboard.Period)
	return bytes, fileName, nil
}
