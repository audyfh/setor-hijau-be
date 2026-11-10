package report

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMonthPeriod(t *testing.T) {
	t.Run("Valid standard month", func(t *testing.T) {
		start, end, period, err := ParseMonthPeriod("2026-11")
		require.NoError(t, err)
		assert.Equal(t, "2026-11", period)
		assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), start)
		assert.Equal(t, time.Date(2026, 11, 30, 23, 59, 59, 999999999, time.UTC), end)
	})

	t.Run("Leap year February", func(t *testing.T) {
		start, end, period, err := ParseMonthPeriod("2024-02")
		require.NoError(t, err)
		assert.Equal(t, "2024-02", period)
		assert.Equal(t, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), start)
		assert.Equal(t, time.Date(2024, 2, 29, 23, 59, 59, 999999999, time.UTC), end)
	})

	t.Run("Empty string defaults to current month", func(t *testing.T) {
		now := time.Now().UTC()
		start, end, period, err := ParseMonthPeriod("")
		require.NoError(t, err)
		assert.Equal(t, now.Format("2006-01"), period)
		assert.Equal(t, now.Year(), start.Year())
		assert.Equal(t, now.Month(), start.Month())
		assert.Equal(t, 1, start.Day())
		assert.True(t, end.After(start))
	})

	t.Run("Invalid month format", func(t *testing.T) {
		_, _, _, err := ParseMonthPeriod("2026-13")
		assert.Error(t, err)

		_, _, _, err = ParseMonthPeriod("invalid")
		assert.Error(t, err)
	})
}

func mockMonthlyReportData() *MonthlyReportData {
	rt := "03"
	phone := "08123456789"
	notes := "Uji coba catatan"
	saleNotes := "Dibeli kontan"

	return &MonthlyReportData{
		Profile: BankSampahProfile{
			ID:            uuid.New(),
			Name:          "Bank Sampah Hijau Lestari RW 05",
			Code:          "bs-hijau-lestari-rw05",
			Address:       "Jl. Melati No. 12",
			RW:            "05",
			RT:            &rt,
			Kelurahan:     "Sukamaju",
			Kecamatan:     "Klojen",
			KabupatenKota: "Kota Malang",
			ContactPhone:  &phone,
		},
		Dashboard: MonthlyDashboardResponse{
			Period:    "2026-11",
			StartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC),
			Deposits: DepositDashboardMetrics{
				DepositsCount:      15,
				ActiveNasabahCount: 8,
				TotalWeightGrams:   45000,
				TotalWeightKg:      45.0,
				TotalAmountIDR:     135000,
				Categories: []CategoryWeightAmount{
					{Category: "plastik", TotalWeightGrams: 25000, TotalWeightKg: 25.0, TotalAmountIDR: 75000},
					{Category: "kertas", TotalWeightGrams: 20000, TotalWeightKg: 20.0, TotalAmountIDR: 60000},
				},
			},
			Sales: SalesDashboardMetrics{
				SalesCount:       2,
				TotalWeightGrams: 40000,
				TotalWeightKg:    40.0,
				TotalAmountIDR:   180000,
				Categories: []CategoryWeightAmount{
					{Category: "plastik", TotalWeightGrams: 22000, TotalWeightKg: 22.0, TotalAmountIDR: 99000},
					{Category: "kertas", TotalWeightGrams: 18000, TotalWeightKg: 18.0, TotalAmountIDR: 81000},
				},
			},
			Cashbook: CashbookDashboardMetrics{
				TotalCashInIDR:       180000,
				TotalCashOutIDR:      50000,
				NetCashFlowIDR:       130000,
				CurrentCashOnHandIDR: 250000,
				Categories: []CategoryCashFlow{
					{Category: "penjualan_pengepul", EntryType: "cash_in", TotalAmountIDR: 180000, Count: 2},
					{Category: "penarikan_nasabah", EntryType: "cash_out", TotalAmountIDR: 50000, Count: 1},
				},
			},
			Carbon: CarbonDashboardMetrics{
				TotalCO2eGrams:   78000,
				TotalCO2eKg:      78.0,
				TotalWeightGrams: 45000,
				Categories: []CategoryCarbon{
					{Category: "plastik", TotalCO2eGrams: 48000, TotalCO2eKg: 48.0, TotalWeightGrams: 25000},
					{Category: "kertas", TotalCO2eGrams: 30000, TotalCO2eKg: 30.0, TotalWeightGrams: 20000},
				},
			},
			ProfitSummary: ProfitDashboardSummary{
				GrossMarginIDR: 45000, // 180,000 - 135,000
			},
		},
		Deposits: []ReportDepositDetail{
			{
				ID:                             uuid.New(),
				ClientRecordedAt:               time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC),
				NasabahName:                    "Budi Santoso",
				NasabahUsername:                "budi05",
				TotalWeightGrams:               15000,
				TotalAmountIDR:                 45000,
				TotalEstimatedCO2eAvoidedGrams: 26000,
				Notes:                          &notes,
			},
		},
		Sales: []ReportSaleDetail{
			{
				ID:               uuid.New(),
				SaleDate:         time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC),
				BuyerName:        "Lapak Pak Joko",
				TotalWeightGrams: 40000,
				TotalAmountIDR:   180000,
				Notes:            &saleNotes,
			},
		},
		Cashbook: []ReportCashbookDetail{
			{
				ID:              uuid.New(),
				TransactionDate: time.Date(2026, 11, 20, 14, 30, 0, 0, time.UTC),
				EntryType:       "cash_in",
				Category:        "penjualan_pengepul",
				AmountIDR:       180000,
				Description:     "Penjualan sampah plastik & kertas",
				RecordedByName:  "Pengelola Admin",
			},
		},
	}
}

func TestGenerateMonthlyCSV(t *testing.T) {
	data := mockMonthlyReportData()
	csvBytes, err := GenerateMonthlyCSV(data)
	require.NoError(t, err)
	require.NotEmpty(t, csvBytes)

	// Verify UTF-8 BOM prefix
	assert.Equal(t, byte(0xEF), csvBytes[0])
	assert.Equal(t, byte(0xBB), csvBytes[1])
	assert.Equal(t, byte(0xBF), csvBytes[2])

	content := string(csvBytes)
	assert.True(t, strings.Contains(content, "Bank Sampah Hijau Lestari RW 05"))
	assert.True(t, strings.Contains(content, "bs-hijau-lestari-rw05"))
	assert.True(t, strings.Contains(content, "Budi Santoso"))
	assert.True(t, strings.Contains(content, "Lapak Pak Joko"))
	assert.True(t, strings.Contains(content, "180000"))
	assert.True(t, strings.Contains(content, "US EPA WARM v16"))
}

func TestGenerateMonthlyPDF(t *testing.T) {
	data := mockMonthlyReportData()
	pdfBytes, err := GenerateMonthlyPDF(data)
	require.NoError(t, err)
	require.NotEmpty(t, pdfBytes)

	// PDF header check
	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF-"))
}
