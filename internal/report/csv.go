package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
)

// GenerateMonthlyCSV generates a clean, well-formatted CSV report with UTF-8 BOM for Microsoft Excel.
func GenerateMonthlyCSV(data *MonthlyReportData) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write UTF-8 BOM so Excel opens accents and symbols without character corruption
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(buf)

	// 1. Profil Bank Sampah
	_ = w.Write([]string{"SETOR HIJAU - SISTEM INFORMASI BANK SAMPAH"})
	_ = w.Write([]string{"Nama Bank Sampah", data.Profile.Name})
	_ = w.Write([]string{"Kode Bank Sampah", data.Profile.Code})
	rtRw := fmt.Sprintf("RW %s", data.Profile.RW)
	if data.Profile.RT != nil && *data.Profile.RT != "" {
		rtRw = fmt.Sprintf("RT %s / RW %s", *data.Profile.RT, data.Profile.RW)
	}
	_ = w.Write([]string{"Wilayah", rtRw})
	_ = w.Write([]string{"Kelurahan / Kecamatan", fmt.Sprintf("%s, %s", data.Profile.Kelurahan, data.Profile.Kecamatan)})
	_ = w.Write([]string{"Kota / Kabupaten", data.Profile.KabupatenKota})
	_ = w.Write([]string{"Periode Laporan", data.Dashboard.Period})
	_ = w.Write([]string{}) // Blank line

	// 2. Ringkasan Eksekutif
	_ = w.Write([]string{"=== RINGKASAN EKSEKUTIF BULANAN ==="})
	_ = w.Write([]string{"Indikator", "Nilai", "Satuan"})
	_ = w.Write([]string{"Total Transaksi Setoran", fmt.Sprintf("%d", data.Dashboard.Deposits.DepositsCount), "Transaksi"})
	_ = w.Write([]string{"Jumlah Nasabah Aktif Menyetor", fmt.Sprintf("%d", data.Dashboard.Deposits.ActiveNasabahCount), "Nasabah"})
	_ = w.Write([]string{"Total Berat Sampah Disetor", fmt.Sprintf("%.2f", data.Dashboard.Deposits.TotalWeightKg), "Kg"})
	_ = w.Write([]string{"Total Saldo Dicatat Nasabah", fmt.Sprintf("%d", data.Dashboard.Deposits.TotalAmountIDR), "Rupiah"})
	_ = w.Write([]string{"Total Transaksi Penjualan Pengepul", fmt.Sprintf("%d", data.Dashboard.Sales.SalesCount), "Transaksi"})
	_ = w.Write([]string{"Total Berat Terjual ke Pengepul", fmt.Sprintf("%.2f", data.Dashboard.Sales.TotalWeightKg), "Kg"})
	_ = w.Write([]string{"Total Pendapatan dari Pengepul", fmt.Sprintf("%d", data.Dashboard.Sales.TotalAmountIDR), "Rupiah"})
	_ = w.Write([]string{"Selisih Kas / Margin Kotor Operasional", fmt.Sprintf("%d", data.Dashboard.ProfitSummary.GrossMarginIDR), "Rupiah"})
	_ = w.Write([]string{"Total Kas Masuk (Buku Kas)", fmt.Sprintf("%d", data.Dashboard.Cashbook.TotalCashInIDR), "Rupiah"})
	_ = w.Write([]string{"Total Kas Keluar (Buku Kas)", fmt.Sprintf("%d", data.Dashboard.Cashbook.TotalCashOutIDR), "Rupiah"})
	_ = w.Write([]string{"Arus Kas Bersih Bulan Ini", fmt.Sprintf("%d", data.Dashboard.Cashbook.NetCashFlowIDR), "Rupiah"})
	_ = w.Write([]string{"Saldo Kas di Tangan (Cash on Hand)", fmt.Sprintf("%d", data.Dashboard.Cashbook.CurrentCashOnHandIDR), "Rupiah"})
	_ = w.Write([]string{"Total Estimasi Emisi Dicegah (US EPA WARM v16)", fmt.Sprintf("%.3f", data.Dashboard.Carbon.TotalCO2eKg), "Kg CO2e"})
	_ = w.Write([]string{}) // Blank line

	// 3. Breakdown Sampah per Kategori
	_ = w.Write([]string{"=== REKAPITULASI SAMPAH & EMISI PER KATEGORI ==="})
	_ = w.Write([]string{"Kategori", "Berat Disetor (Kg)", "Nilai Rupiah Nasabah", "Estimasi CO2e Dicegah (Kg)"})
	carbonMap := make(map[string]float64)
	for _, c := range data.Dashboard.Carbon.Categories {
		carbonMap[c.Category] = c.TotalCO2eKg
	}
	for _, cat := range data.Dashboard.Deposits.Categories {
		co2kg := carbonMap[cat.Category]
		_ = w.Write([]string{
			cat.Category,
			fmt.Sprintf("%.2f", cat.TotalWeightKg),
			fmt.Sprintf("%d", cat.TotalAmountIDR),
			fmt.Sprintf("%.3f", co2kg),
		})
	}
	_ = w.Write([]string{}) // Blank line

	// 4. Rincian Setoran Nasabah
	_ = w.Write([]string{"=== RINCIAN SETORAN SAMPAH NASABAH ==="})
	_ = w.Write([]string{"ID Setoran", "Waktu Pencatatan", "Nama Nasabah", "Username", "Berat (Gram)", "Total (Rupiah)", "Estimasi CO2e (Gram)", "Catatan"})
	for _, d := range data.Deposits {
		notesStr := ""
		if d.Notes != nil {
			notesStr = *d.Notes
		}
		_ = w.Write([]string{
			d.ID.String(),
			d.ClientRecordedAt.Format("2006-01-02 15:04:05"),
			d.NasabahName,
			d.NasabahUsername,
			fmt.Sprintf("%d", d.TotalWeightGrams),
			fmt.Sprintf("%d", d.TotalAmountIDR),
			fmt.Sprintf("%d", d.TotalEstimatedCO2eAvoidedGrams),
			notesStr,
		})
	}
	_ = w.Write([]string{}) // Blank line

	// 5. Rincian Penjualan ke Pengepul
	_ = w.Write([]string{"=== RINCIAN PENJUALAN KE PENGEPUL ==="})
	_ = w.Write([]string{"ID Penjualan", "Tanggal Transaksi", "Nama Pengepul / Pembeli", "Berat (Gram)", "Total Pendapatan (Rupiah)", "Catatan"})
	for _, s := range data.Sales {
		notesStr := ""
		if s.Notes != nil {
			notesStr = *s.Notes
		}
		_ = w.Write([]string{
			s.ID.String(),
			s.SaleDate.Format("2006-01-02 15:04:05"),
			s.BuyerName,
			fmt.Sprintf("%d", s.TotalWeightGrams),
			fmt.Sprintf("%d", s.TotalAmountIDR),
			notesStr,
		})
	}
	_ = w.Write([]string{}) // Blank line

	// 6. Rincian Mutasi Buku Kas
	_ = w.Write([]string{"=== RINCIAN BUKU KAS (CASHBOOK) ==="})
	_ = w.Write([]string{"ID Transaksi", "Tanggal", "Tipe", "Kategori", "Jumlah (Rupiah)", "Keterangan", "Pencatat"})
	for _, c := range data.Cashbook {
		_ = w.Write([]string{
			c.ID.String(),
			c.TransactionDate.Format("2006-01-02 15:04:05"),
			c.EntryType,
			c.Category,
			fmt.Sprintf("%d", c.AmountIDR),
			c.Description,
			c.RecordedByName,
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
