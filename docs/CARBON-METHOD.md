# Metode & Sumber Perhitungan Karbon (Carbon Accounting Method) — Setor Hijau

Dokumen ini menjelaskan dasar ilmiah, metodologi perhitungan, faktor emisi, konversi matematis, asumsi, dan batasan dalam mengestimasi emisi gas rumah kaca (GRK) yang berhasil dihindari (*avoided GHG emissions* / $CO_2e$ dihindari) pada sistem Setor Hijau.

Dokumen ini ditulis sebagai prasyarat wajib Milestone M4 sesuai **Hard Rule #4** dan panduan transparansi **Hard Rule #9** pada [`/AGENTS.md`](file:///C:/D_Drive\Kuliah\SEMESTER 7\SIB\setor-hijau-be\AGENTS.md).

---

## 1. Landasan Teori & Model Referensi

Bank Sampah Unit (BSU) di tingkat RW di Indonesia berperan mengalihkan sampah terpilah dari pembuangan akhir ke Tempat Pemrosesan Akhir (TPA / *landfill*). 

Dalam akuntansi karbon berbasis daur hidup (*Life Cycle Assessment* / LCA), pengalihan sampah anorganik yang didaur ulang dibandingkan terhadap skenario dasar (*baseline*) pembuangan konvensional ke TPA menghasilkan pengurangan emisi GRK bersih. Manfaat ini timbul dari:
1. **Pengurangan Penggunaan Bahan Baku Perawan (*Virgin Material Displacement Credit*)**: Memproduksi barang dari material sekunder/daur ulang membutuhkan energi yang jauh lebih sedikit dibandingkan mengekstraksi dan memproses bijih/bahan mentah perawan (misalnya bauksit untuk aluminium atau minyak bumi untuk plastik PET/HDPE).
2. **Penghindaran Emisi Landfill (*Avoided Landfill Emissions*)**: Menghindari degradasi anaerobik sampah berbahan dasar serat tanaman/kertas di TPA yang menghasilkan gas metana ($CH_4$), serta emisi operasional alat berat penimbun sampah.

### Sumber Model: US EPA WARM Version 16 (Desember 2023)
Setor Hijau mengadopsi faktor emisi dari **U.S. Environmental Protection Agency (US EPA) Waste Reduction Model (WARM) Versi 16** (dirilis Desember 2023). WARM merupakan standar metodologi internasional yang telah melalui telaah sejawat (*peer-reviewed*) untuk memperkirakan manfaat iklim dari pengelolaan sampah perkotaan (*Municipal Solid Waste* / MSW).

---

## 2. Ruang Lingkup Material & Pemetaan Kategori Indonesia

Sesuai ruang lingkup Setor Hijau ([`docs/PRODUCT.md`](file:///C:/D_Drive\Kuliah\SEMESTER 7\SIB\setor-hijau-be\docs\PRODUCT.md)), pencatatan emisi dibatasi secara ketat pada **sampah anorganik daur ulang bernilai ekonomi** yang umum diterima di Bank Sampah tingkat RW di Indonesia.

Berikut pemetaan 7 kategori sampah anorganik ke komoditas referensi US EPA WARM v16:

| No | Kode Material (`material_code`) | Nama Material di Bank Sampah | Komoditas WARM v16 | Kategori Umum |
|---|---|---|---|---|
| 1 | `corrugated_containers` | Kardus / Karton Gelombang | *Corrugated Containers* | Kertas |
| 2 | `mixed_paper` | Kertas Campur / HVS / Koran / Majalah / Duplex | *Mixed Paper* | Kertas |
| 3 | `pet_bottles` | Botol Plastik PET Transparan / Warna | *PET (Polyethylene Terephthalate)* | Plastik |
| 4 | `hdpe_containers` | Plastik HDPE Keras / Jeriken / Botol Sampo / Tutup Botol | *HDPE (High-Density Polyethylene)* | Plastik |
| 5 | `aluminum_cans` | Kaleng Minuman Aluminium | *Aluminum Cans* | Logam |
| 6 | `steel_cans` | Kaleng Besi / Seng / Kaleng Biskuit / Kaleng Susu | *Steel Cans* | Logam |
| 7 | `glass_containers` | Botol Kaca / Wadah Beling | *Glass* | Kaca |

---

## 3. Konversi Unit & Representasi Integer Tanpa Float

### 3.1. Konversi dari Satuan Imperial ke Metrik
Model US EPA WARM secara standar mempublikasikan faktor emisi dalam satuan **Metric Tons of Carbon Dioxide Equivalent per Short Ton** ($MTCO_2e / \text{short ton}$).

Konversi standar satuan massa internasional:
$$1 \text{ Short Ton} = 2.000 \text{ pon} = 907,18474 \text{ kg} = 0,90718474 \text{ Metric Ton}$$

Maka faktor emisi per Metrik Ton ($E_{\text{metric ton}}$):
$$E_{\text{metric ton}} = \frac{E_{\text{short ton}}}{0,90718474} \quad (\text{satuan: } MTCO_2e / \text{Metric Ton})$$

Karena $1 \text{ Metric Ton } CO_2e = 1.000.000 \text{ gram } CO_2e$ dan $1 \text{ Metric Ton sampah} = 1.000.000 \text{ gram sampah}$, maka secara proporsional:
$$1 \text{ } MTCO_2e / \text{Metric Ton} \equiv 1 \text{ gram } CO_2e / \text{gram sampah}$$

### 3.2. Skala Fixed-Point PPM (Hard Rule #3: No Floats)
Untuk mematuhi **Hard Rule #3** (tidak boleh ada floating-point pada uang, bobot, maupun emisi), faktor emisi disimpan dalam database sebagai integer `BIGINT` berpresisi **Parts Per Million (PPM)** atau $\mu g \text{ } CO_2e / \text{gram sampah}$:
$$\text{co2e\_factor\_ppm} = \text{round}\left( \frac{E_{\text{short ton}}}{0,90718474} \times 1.000.000 \right)$$

Berikut tabel lengkap nilai faktor US EPA WARM v16 dan nilai PPM integer yang tersimpan dalam tabel `emission_factors`:

| Kode Material | WARM v16 ($MTCO_2e / \text{short ton}$) | Faktor Metrik ($MTCO_2e / \text{ton}$) | `co2e_factor_ppm` (Integer DB) | Sumber Resmi |
|---|---|---|---|---|
| `corrugated_containers` | 3,11 | 3,42818817... | **3.428.188** | US EPA WARM v16 (2023) |
| `mixed_paper` | 3,53 | 3,89115886... | **3.891.159** | US EPA WARM v16 (2023) |
| `pet_bottles` | 1,15 | 1,26765803... | **1.267.658** | US EPA WARM v16 (2023) |
| `hdpe_containers` | 0,85 | 0,93696459... | **936.965** | US EPA WARM v16 (2023) |
| `aluminum_cans` | 9,11 | 10,04205658... | **10.042.057** | US EPA WARM v16 (2023) |
| `steel_cans` | 1,81 | 1,99518337... | **1.995.183** | US EPA WARM v16 (2023) |
| `glass_containers` | 0,28 | 0,30864716... | **308.647** | US EPA WARM v16 (2023) |

---

## 4. Formula Perhitungan Setoran

Untuk setiap baris setoran sampah (*deposit line*) dengan berat timbangan $w$ dalam satuan **integer gram**:
$$\text{co2e\_grams} = \left\lfloor \frac{w \times \text{co2e\_factor\_ppm} + 500.000}{1.000.000} \right\rfloor$$

Formula di atas menerapkan pembulatan setengah ke atas (*half-up rounding*) integer division:
- Tambahan $+500.000$ merepresentasikan $0,5 \times 1.000.000$ sebagai titik tengah pembulatan ke gram $CO_2e$ terdekat.
- Operasi dilakukan seluruhnya dengan tipe data 64-bit integer (`int64` / PostgreSQL `BIGINT`).

---

## 5. Kasus Uji Perhitungan Manual (Hand-Verified Test Cases)

Untuk memenuhi kriteria keberhasilan Milestone M4 (*Accept: hand-verified calculation test cases*), berikut adalah hasil perhitungan manual yang diverifikasi sebagai acuan pengujian unit test dan integrasi:

| Skenario Uji | Material | Bobot Gram ($w$) | `co2e_factor_ppm` | Operasi Matematika | Hasil Integer ($\text{g } CO_2e$) |
|---|---|---|---|---|---|
| TC-01 | Kardus / Karton | 1.500 g | 3.428.188 | $(1.500 \times 3.428.188 + 500.000) / 1.000.000 = 5.142.782.000 / 1.000.000$ | **5.142** |
| TC-02 | Kardus / Karton | 2.000 g | 3.428.188 | $(2.000 \times 3.428.188 + 500.000) / 1.000.000 = 6.856.876.000 / 1.000.000$ | **6.856** |
| TC-03 | Kaleng Aluminium | 500 g | 10.042.057 | $(500 \times 10.042.057 + 500.000) / 1.000.000 = 5.021.528.500 / 1.000.000$ | **5.021** |
| TC-04 | Botol Plastik PET | 1.000 g | 1.267.658 | $(1.000 \times 1.267.658 + 500.000) / 1.000.000 = 1.268.158.000 / 1.000.000$ | **1.268** |
| TC-05 | Plastik HDPE | 2.500 g | 936.965 | $(2.500 \times 936.965 + 500.000) / 1.000.000 = 2.342.912.500 / 1.000.000$ | **2.342** |
| TC-06 | Botol Kaca | 5.000 g | 308.647 | $(5.000 \times 308.647 + 500.000) / 1.000.000 = 1.543.735.000 / 1.000.000$ | **1.543** |
| TC-07 | Kaleng Besi/Seng | 1.000 g | 1.995.183 | $(1.000 \times 1.995.183 + 500.000) / 1.000.000 = 1.995.683.000 / 1.000.000$ | **1.995** |
| TC-08 | Kertas Campur | 3.000 g | 3.891.159 | $(3.000 \times 3.891.159 + 500.000) / 1.000.000 = 11.673.977.000 / 1.000.000$ | **11.673** |
| TC-09 | Batas Bawah | 1 g (PET) | 1.267.658 | $(1 \times 1.267.658 + 500.000) / 1.000.000 = 1.767.658 / 1.000.000$ | **1** |
| TC-10 | Titik Tengah Half-Up | 500 g (Kaca) | 308.647 | $(500 \times 308.647 + 500.000) / 1.000.000 = 154.823.500 / 1.000.000$ | **154** |

---

## 6. Auditabilitas & Keterlacakan (Auditability & Traceability)

Untuk memastikan prinsip *Green Software Engineering* dan integritas audit akademik:
1. **Pencatatan Versi dan Sumber pada Setiap Baris**: Setiap baris di `carbon_ledger` menyalin dan menyimpan nilai `factor_version` (misal `"WARM v16"`) dan `factor_source` (misal `"US EPA Waste Reduction Model Version 16 (2023)"`). Perubahan tabel faktor emisi di masa depan tidak akan pernah merevisi atau mengubah riwayat data lampau (Hard Rule #4 & Invariant #5).
2. **Kekekalan Jurnal Karbon (Hard Rule #2)**: Tabel `carbon_ledger` bersifat *append-only*. Tidak ada perintah `UPDATE` atau `DELETE`. Jika terjadi pembatalan setoran atau koreksi administrasi, baris baru bertipe `entry_type = 'reversal'` ditambahkan dengan tanda emisi negatif ($- \text{co2e\_grams}$) serta tautan referensi `reversal_for_id`.
3. **Penyajian Terderivasi**: Saldo tabungan karbon nasabah (`tabungan_karbon`) selalu dihitung secara matematis menggunakan `SUM(co2e_grams)` dari seluruh baris jurnal yang bersangkutan.

---

## 7. Asumsi, Keterbatasan, & Kejujuran Metodologis (Hard Rule #9)

1. **Faktor Berbasis Amerika Serikat**: Faktor WARM v16 dikembangkan oleh US EPA berdasarkan bauran energi listrik dan efisiensi industri manufaktur di AS. Karena data Life Cycle Assessment (LCA) spesifik pabrik daur ulang di Indonesia belum tersedia secara komprehensif di tingkat nasional, angka ini digunakan sebagai **estimasi ilmiah terbaik (*best available scientific estimate*)**.
2. **Aturan Penamaan Estimasi**: Sesuai **Hard Rule #9**, angka karbon tidak boleh disajikan sebagai pengukuran absolut/pasti. Seluruh DTO API, parameter, dan antarmuka pengguna wajib menggunakan kata kunci estimasi (misalnya `estimated_co2e_avoided_grams`).
3. **Eksklusi Sampah Organik**: Penanganan sampah organik (pengomposan/maggot) tidak dicakup dalam kalkulasi ini karena dinamika dekomposisi organik memerlukan parameter kelembapan dan rasio C/N lokal yang berbeda.
