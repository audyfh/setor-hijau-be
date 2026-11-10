# ADR 0008: Carbon Ledger Architecture, Versioned Emission Factors, and Reversal Semantics

## Status
Accepted

## Context
Setor Hijau membedakan diri dari bank sampah konvensional dengan menghadirkan **Carbon Ledger** (pencatatan tabungan karbon berupa estimasi emisi GRK yang berhasil dihindari per transaksi setoran).

Terkait perancangan modul karbon, terdapat sejumlah aturan non-negosiasi dan batasan arsitektural:
1. **Hard Rule #1 & Invariant #1**: Seluruh kueri pada tabel tenant-scoped wajib difilter berdasarkan `bank_sampah_id` dari konteks otentikasi.
2. **Hard Rule #2 & Invariant #3**: `carbon_ledger` bersifat *append-only*. Perbaikan atau koreksi transaksi dilakukan melalui entri pembalik (*reversing entries*), dilarang menggunakan `UPDATE` atau `DELETE`.
3. **Hard Rule #3 & Invariant #6**: Tidak boleh ada tipe `float` untuk uang, bobot, maupun emisi. Emisi harus disimpan sebagai integer gram $CO_2e$ (`BIGINT`).
4. **Hard Rule #4 & Invariant #5**: Faktor emisi tidak boleh di-hardcode dalam kode Go; faktor harus hidup di tabel database berversi dengan mencantumkan sumber dan versi resminya. Setiap baris jurnal karbon wajib mencatat versi dan sumber faktor yang digunakan saat transaksi dicatat.
5. **Hard Rule #9**: Angka karbon merupakan estimasi berbasis WARM. Seluruh penamaan field dan dokumentasi wajib menggunakan label estimasi (`estimated_...`).

## Decisions

### 1. Platform-Wide `emission_factors` vs Tenant-Scoped `carbon_ledger`
- **Tabel `emission_factors`** bersifat platform-wide (global), karena faktor emisi berasal dari standar sains lingkungan internasional (US EPA WARM v16) yang berlaku seragam untuk semua Bank Sampah. Tabel ini memiliki constraint unik `(material_code, factor_version)`.
- **Tabel `carbon_ledger`** bersifat tenant-scoped dengan kolom non-nullable `bank_sampah_id UUID REFERENCES bank_sampah(id)`. Setiap kueri penyajian saldo maupun riwayat jurnal wajib memvalidasi dan memfilter `bank_sampah_id`.

### 2. Representasi Integer PPM untuk Faktor Emisi
Faktor emisi metrik ($MTCO_2e / \text{Metric Ton}$) diskalakan dengan faktor pengali $1.000.000$ (PPM / $\mu g \text{ } CO_2e / \text{g sampah}$) dan disimpan sebagai `BIGINT` di kolom `co2e_factor_ppm`.

Kalkulasi emisi untuk setoran seberat $w$ gram dilakukan dengan pembulatan integer half-up tanpa melibatkan float:
$$\text{co2e\_grams} = \left\lfloor \frac{w \times \text{co2e\_factor\_ppm} + 500.000}{1.000.000} \right\rfloor$$

### 3. Snapshotting Versi & Sumber pada Setiap Baris Jurnal
Setiap baris di `carbon_ledger` menyalin nilai:
- `factor_version VARCHAR(32)` (contoh: `"WARM v16"`)
- `factor_source TEXT` (contoh: `"US EPA Waste Reduction Model Version 16 (2023)"`)
Hal ini menjamin bahwa setiap entri historis bersifat swasembada (*self-contained*) dan tetap dapat diaudit secara ilmiah meskipun di masa mendatang faktor emisi diperbarui ke versi baru.

### 4. Semantik Reversal Entry (Append-Only)
Jika pengelola membatalkan atau mengoreksi setoran:
- Baris jurnal lama tetap utuh di database.
- Baris pembalik baru dibuat dengan:
  - `entry_type = 'reversal'`
  - `reversal_for_id` merujuk ke ID entri `carbon_ledger` yang dibatalkan.
  - `co2e_grams` bernilai negatif ($- \text{original.co2e\_grams}$).
  - `weight_grams` bernilai negatif ($- \text{original.weight\_grams}$).
  - `notes` memuat alasan koreksi dari Pengelola.
- Dengan mekanisme ini, saldo tabungan karbon nasabah selalu dapat dihitung secara instan dan tepat menggunakan:
  ```sql
  SELECT COALESCE(SUM(co2e_grams), 0)
  FROM carbon_ledger
  WHERE bank_sampah_id = $1 AND nasabah_id = $2;
  ```

### 5. Transaksi Atomik dengan Pencatatan Setoran
Pencatatan setoran (`internal/deposit`) dan penulisan baris ke `carbon_ledger` dieksekusi dalam satu transaksi database PostgreSQL yang sama (`ExecTx`). Jika salah satu gagal, seluruh transaksi dibatalkan (*rollback*).

Pada pengiriman ulang idempoten (*idempotent replay*) dengan `idempotency_key` yang sama, sistem mengembalikan data setoran sebelumnya tanpa menduplikasi entri di `carbon_ledger`.

## Consequences
- **Positive**:
  - Memenuhi seluruh Aturan Keras (#1, #2, #3, #4, #9).
  - Jejak audit karbon transparan, terverifikasi, dan tidak dapat dimanipulasi.
  - Bebas dari galat pembulatan floating point (IEEE 754).
- **Negative**:
  - Kolom snapshot versi dan sumber menambah sedikit ukuran storage per baris jurnal, namun dapat dijustifikasi demi integritas audit ilmiah dan kepatuhan regulasi.
