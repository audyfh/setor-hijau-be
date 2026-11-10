package report

import (
	"time"

	"github.com/google/uuid"
)

type CategoryWeightAmount struct {
	Category         string  `json:"category"`
	TotalWeightGrams int64   `json:"total_weight_grams"`
	TotalWeightKg    float64 `json:"total_weight_kg"`
	TotalAmountIDR   int64   `json:"total_amount_idr"`
}

type CategoryCashFlow struct {
	Category       string `json:"category"`
	EntryType      string `json:"entry_type"`
	TotalAmountIDR int64  `json:"total_amount_idr"`
	Count          int64  `json:"count"`
}

type CategoryCarbon struct {
	Category         string  `json:"category"`
	TotalCO2eGrams   int64   `json:"total_co2e_grams"`
	TotalCO2eKg      float64 `json:"total_co2e_kg"`
	TotalWeightGrams int64   `json:"total_weight_grams"`
}

type DepositDashboardMetrics struct {
	DepositsCount      int64                  `json:"deposits_count"`
	ActiveNasabahCount int64                  `json:"active_nasabah_count"`
	TotalWeightGrams   int64                  `json:"total_weight_grams"`
	TotalWeightKg      float64                `json:"total_weight_kg"`
	TotalAmountIDR     int64                  `json:"total_amount_idr"`
	Categories         []CategoryWeightAmount `json:"categories"`
}

type SalesDashboardMetrics struct {
	SalesCount       int64                  `json:"sales_count"`
	TotalWeightGrams int64                  `json:"total_weight_grams"`
	TotalWeightKg    float64                `json:"total_weight_kg"`
	TotalAmountIDR   int64                  `json:"total_amount_idr"`
	Categories       []CategoryWeightAmount `json:"categories"`
}

type CashbookDashboardMetrics struct {
	TotalCashInIDR       int64              `json:"total_cash_in_idr"`
	TotalCashOutIDR      int64              `json:"total_cash_out_idr"`
	NetCashFlowIDR       int64              `json:"net_cash_flow_idr"`
	CurrentCashOnHandIDR int64              `json:"current_cash_on_hand_idr"`
	Categories           []CategoryCashFlow `json:"categories"`
}

type CarbonDashboardMetrics struct {
	TotalCO2eGrams   int64            `json:"total_co2e_grams"`
	TotalCO2eKg      float64          `json:"total_co2e_kg"`
	TotalWeightGrams int64            `json:"total_weight_grams"`
	Categories       []CategoryCarbon `json:"categories"`
}

type ProfitDashboardSummary struct {
	GrossMarginIDR int64 `json:"gross_margin_idr"`
}

type MonthlyDashboardResponse struct {
	Period        string                   `json:"period"` // e.g. "2026-11"
	StartDate     time.Time                `json:"start_date"`
	EndDate       time.Time                `json:"end_date"`
	Deposits      DepositDashboardMetrics  `json:"deposits"`
	Sales         SalesDashboardMetrics    `json:"sales"`
	Cashbook      CashbookDashboardMetrics `json:"cashbook"`
	Carbon        CarbonDashboardMetrics   `json:"carbon"`
	ProfitSummary ProfitDashboardSummary   `json:"profit_summary"`
}

// --- Report Export Data Models ---

type BankSampahProfile struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	Address       string    `json:"address"`
	RW            string    `json:"rw"`
	RT            *string   `json:"rt,omitempty"`
	Kelurahan     string    `json:"kelurahan"`
	Kecamatan     string    `json:"kecamatan"`
	KabupatenKota string    `json:"kabupaten_kota"`
	ContactPhone  *string   `json:"contact_phone,omitempty"`
}

type ReportDepositDetail struct {
	ID                             uuid.UUID
	ClientRecordedAt               time.Time
	NasabahName                    string
	NasabahUsername                string
	TotalWeightGrams               int64
	TotalAmountIDR                 int64
	TotalEstimatedCO2eAvoidedGrams int64
	Notes                          *string
}

type ReportSaleDetail struct {
	ID               uuid.UUID
	SaleDate         time.Time
	BuyerName        string
	TotalWeightGrams int64
	TotalAmountIDR   int64
	Notes            *string
}

type ReportCashbookDetail struct {
	ID              uuid.UUID
	TransactionDate time.Time
	EntryType       string
	Category        string
	AmountIDR       int64
	Description     string
	RecordedByName  string
}

type MonthlyReportData struct {
	Profile   BankSampahProfile
	Dashboard MonthlyDashboardResponse
	Deposits  []ReportDepositDetail
	Sales     []ReportSaleDetail
	Cashbook  []ReportCashbookDetail
}
