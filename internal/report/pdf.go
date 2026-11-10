package report

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// GenerateMonthlyPDF produces an official, clean, lightweight A4 PDF report in pure Go.
func GenerateMonthlyPDF(data *MonthlyReportData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// 1. Header (Kop Bank Sampah)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(30, 80, 40) // Forest green
	pdf.CellFormat(180, 6, data.Profile.Name, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(60, 60, 60)
	rtRw := fmt.Sprintf("RW %s", data.Profile.RW)
	if data.Profile.RT != nil && *data.Profile.RT != "" {
		rtRw = fmt.Sprintf("RT %s / RW %s", *data.Profile.RT, data.Profile.RW)
	}
	subHeader := fmt.Sprintf("%s, Kel. %s, Kec. %s, %s", rtRw, data.Profile.Kelurahan, data.Profile.Kecamatan, data.Profile.KabupatenKota)
	pdf.CellFormat(180, 5, subHeader, "", 1, "C", false, 0, "")

	if data.Profile.ContactPhone != nil && *data.Profile.ContactPhone != "" {
		pdf.CellFormat(180, 4, fmt.Sprintf("Kontak / WhatsApp: %s", *data.Profile.ContactPhone), "", 1, "C", false, 0, "")
	}

	// Decorative divider
	pdf.Ln(2)
	pdf.SetDrawColor(46, 139, 87) // Sea green
	pdf.SetLineWidth(0.8)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(4)

	// 2. Title & Period
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(20, 20, 20)
	pdf.CellFormat(180, 6, "LAPORAN BULANAN OPERASIONAL & EMISI", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(180, 4, fmt.Sprintf("Periode: %s  |  Tanggal Cetak: %s", data.Dashboard.Period, time.Now().Format("02-01-2006 15:04")), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// 3. Ringkasan Eksekutif (Card-like summary table)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(30, 80, 40)
	pdf.CellFormat(180, 5, "I. RINGKASAN OPERASIONAL & KEUANGAN", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(240, 248, 240)
	pdf.SetTextColor(40, 40, 40)
	pdf.SetDrawColor(210, 210, 210)
	pdf.SetLineWidth(0.2)

	// Row 1
	pdf.CellFormat(55, 6, "Total Setoran Nasabah", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("%d Transaksi", data.Dashboard.Deposits.DepositsCount), "1", 0, "R", false, 0, "")
	pdf.CellFormat(55, 6, "Total Penjualan ke Pengepul", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("%d Transaksi", data.Dashboard.Sales.SalesCount), "1", 1, "R", false, 0, "")

	// Row 2
	pdf.CellFormat(55, 6, "Nasabah Aktif Menyetor", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("%d Nasabah", data.Dashboard.Deposits.ActiveNasabahCount), "1", 0, "R", false, 0, "")
	pdf.CellFormat(55, 6, "Berat Terjual ke Pengepul", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("%.2f Kg", data.Dashboard.Sales.TotalWeightKg), "1", 1, "R", false, 0, "")

	// Row 3
	pdf.CellFormat(55, 6, "Total Berat Sampah Terkumpul", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("%.2f Kg", data.Dashboard.Deposits.TotalWeightKg), "1", 0, "R", false, 0, "")
	pdf.CellFormat(55, 6, "Total Pendapatan Pengepul", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("Rp %d", data.Dashboard.Sales.TotalAmountIDR), "1", 1, "R", false, 0, "")

	// Row 4
	pdf.CellFormat(55, 6, "Total Saldo Dicatat Nasabah", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("Rp %d", data.Dashboard.Deposits.TotalAmountIDR), "1", 0, "R", false, 0, "")
	pdf.CellFormat(55, 6, "Margin Kotor Operasional", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("Rp %d", data.Dashboard.ProfitSummary.GrossMarginIDR), "1", 1, "R", false, 0, "")

	// Row 5 (Cash summary)
	pdf.CellFormat(55, 6, "Total Kas Masuk (Buku Kas)", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("Rp %d", data.Dashboard.Cashbook.TotalCashInIDR), "1", 0, "R", false, 0, "")
	pdf.CellFormat(55, 6, "Saldo Kas di Tangan", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("Rp %d", data.Dashboard.Cashbook.CurrentCashOnHandIDR), "1", 1, "R", false, 0, "")

	// Row 6
	pdf.CellFormat(55, 6, "Total Kas Keluar (Buku Kas)", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("Rp %d", data.Dashboard.Cashbook.TotalCashOutIDR), "1", 0, "R", false, 0, "")
	pdf.CellFormat(55, 6, "Estimasi Emisi Dicegah", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("%.2f Kg CO2e", data.Dashboard.Carbon.TotalCO2eKg), "1", 1, "R", false, 0, "")

	pdf.Ln(4)

	// 4. Breakdown per Kategori
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(30, 80, 40)
	pdf.CellFormat(180, 5, "II. REKAPITULASI MATERIAL & JEJAK KARBON", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	// Table Header
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(230, 240, 230)
	pdf.SetTextColor(20, 20, 20)
	pdf.CellFormat(45, 6, "Kategori Sampah", "1", 0, "L", true, 0, "")
	pdf.CellFormat(45, 6, "Berat Terkumpul (Kg)", "1", 0, "R", true, 0, "")
	pdf.CellFormat(45, 6, "Nilai Rupiah Nasabah", "1", 0, "R", true, 0, "")
	pdf.CellFormat(45, 6, "Estimasi Emisi Dicegah", "1", 1, "R", true, 0, "")

	carbonMap := make(map[string]float64)
	for _, c := range data.Dashboard.Carbon.Categories {
		carbonMap[c.Category] = c.TotalCO2eKg
	}

	pdf.SetFont("Arial", "", 8)
	if len(data.Dashboard.Deposits.Categories) == 0 {
		pdf.CellFormat(180, 6, "Belum ada transaksi setoran pada periode ini", "1", 1, "C", false, 0, "")
	} else {
		for _, cat := range data.Dashboard.Deposits.Categories {
			co2kg := carbonMap[cat.Category]
			pdf.CellFormat(45, 6, cat.Category, "1", 0, "L", false, 0, "")
			pdf.CellFormat(45, 6, fmt.Sprintf("%.2f Kg", cat.TotalWeightKg), "1", 0, "R", false, 0, "")
			pdf.CellFormat(45, 6, fmt.Sprintf("Rp %d", cat.TotalAmountIDR), "1", 0, "R", false, 0, "")
			pdf.CellFormat(45, 6, fmt.Sprintf("%.3f Kg CO2e", co2kg), "1", 1, "R", false, 0, "")
		}
	}

	pdf.Ln(3)

	// Scientific footnote
	pdf.SetFont("Arial", "I", 7.5)
	pdf.SetTextColor(100, 100, 100)
	pdf.MultiCell(180, 3.5, "* Estimasi emisi gas rumah kaca yang dicegah (CO2e avoided) dihitung menggunakan metodologi US EPA Waste Reduction Model (WARM) v16 untuk sampah anorganik yang dialihkan dari TPA (landfill) ke jalur daur ulang.", "", "L", false)

	pdf.Ln(8)

	// 5. Signature Section
	pdf.SetFont("Arial", "", 8.5)
	pdf.SetTextColor(30, 30, 30)

	curY := pdf.GetY()
	if curY > 240 {
		pdf.AddPage()
		curY = pdf.GetY()
	}

	pdf.SetXY(120, curY)
	pdf.CellFormat(60, 4, fmt.Sprintf("%s, %s", data.Profile.KabupatenKota, time.Now().Format("02 January 2006")), "", 1, "C", false, 0, "")
	pdf.SetX(120)
	pdf.CellFormat(60, 4, "Pengelola Bank Sampah,", "", 1, "C", false, 0, "")
	pdf.Ln(18)
	pdf.SetX(120)
	pdf.SetFont("Arial", "B", 8.5)
	pdf.CellFormat(60, 4, "( ................................................ )", "", 1, "C", false, 0, "")

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
