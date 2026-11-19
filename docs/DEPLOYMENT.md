# Panduan Deployment: Google Cloud Run + NeonDB

Panduan ini menjelaskan langkah-langkah setup **NeonDB** (database PostgreSQL utama) dan **Google Cloud Platform (GCP)** melalui **Web Console** maupun **CLI (`gcloud`)**.

Arsitektur produksi:
- **API Backend**: Google Cloud Run (container Go stateless, port `8080`, *scale-to-zero*).
- **Database**: NeonDB (PostgreSQL 16, region Singapore `ap-southeast-1`, *auto-suspend*).
- **Secrets**: Google Cloud Secret Manager (`DATABASE_URL`, `JWT_SECRET`, dll.).
- **Migrations**: Cloud Run Job terpisah menjalankan `/app/migrate up` (tidak pernah saat startup API).

> **Portabilitas:** Image Docker yang sama tetap bisa dijalankan terhadap **Cloud SQL** atau **Supabase Postgres** hanya dengan mengganti nilai `DATABASE_URL`.

---

## Bagian 1: Setup NeonDB di Console

1. **Buat Akun & Project**
   - Buka [console.neon.tech](https://console.neon.tech) dan login.
   - Klik **New Project**.
   - Isi pengaturan berikut:
     - **Project name**: `setor-hijau`
     - **Postgres version**: `16`
     - **Cloud provider**: `AWS`
     - **Region**: `Asia Pacific (Singapore) — ap-southeast-1` (terdekat dengan Indonesia & Cloud Run Singapore/Jakarta).
   - Klik **Create Project**.

2. **Ambil Dua Connection String**
   Di halaman **Dashboard -> Connection Details**:
   - **Connection string 1 — Pooled (untuk Cloud Run API):**
     - Centang / aktifkan toggle **Connection pooling**.
     - Salin URL-nya (hostname mengandung `-pooler`, contoh:
       `postgresql://neondb_owner:PASSWORD@ep-xxxx-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require`).
     - Simpan ini untuk secret **`DATABASE_URL`**.
   - **Connection string 2 — Direct / Unpooled (untuk Migrasi DB):**
     - Matikan toggle **Connection pooling**.
     - Salin URL-nya (hostname **tanpa** `-pooler`, contoh:
       `postgresql://neondb_owner:PASSWORD@ep-xxxx.ap-southeast-1.aws.neon.tech/neondb?sslmode=require`).
     - Simpan ini untuk secret **`DATABASE_URL_DIRECT`** *(karena `golang-migrate` membutuhkan advisory lock yang bekerja paling stabil di koneksi direct)*.

---

## Bagian 2: Setup Google Cloud Platform (GCP) di Console

> Rekomendasi region GCP: **`asia-southeast2` (Jakarta)** atau **`asia-southeast1` (Singapore)** (Singapore paling dekat dengan region NeonDB `ap-southeast-1`). Gunakan region yang sama untuk Artifact Registry dan Cloud Run.

### Langkah 1 — Buat / Pilih Project & Aktifkan Billing
1. Buka [console.cloud.google.com](https://console.cloud.google.com).
2. Pilih atau buat project baru (misal `setor-hijau-prod`).
3. Pastikan **Billing** sudah tertaut ke akun kredit GCP Anda (**Billing -> Link a billing account**).

### Langkah 2 — Aktifkan API yang Dibutuhkan
Buka **APIs & Services -> Enable APIs and Services**, lalu aktifkan 4 API berikut:
1. **Cloud Run Admin API** (`run.googleapis.com`)
2. **Artifact Registry API** (`artifactregistry.googleapis.com`)
3. **Secret Manager API** (`secretmanager.googleapis.com`)
4. **Cloud Build API** (`cloudbuild.googleapis.com`) *(opsional jika build image dari GCP)*

### Langkah 3 — Buat Repository di Artifact Registry
1. Buka menu **Artifact Registry -> Repositories -> Create Repository**.
2. Isi:
   - **Name**: `setor-hijau`
   - **Format**: `Docker`
   - **Location type**: `Region` -> pilih `asia-southeast1` (Singapore) atau `asia-southeast2` (Jakarta).
3. Klik **Create**.

### Langkah 4 — Simpan Secrets di Secret Manager
Buka **Security -> Secret Manager -> Create Secret** dan buat 3 secret berikut:

| Nama Secret | Nilai / Contoh | Keterangan |
|---|---|---|
| `setor-hijau-db-url` | `postgresql://...@ep-xxx-pooler.../neondb?sslmode=require` | Connection string **Pooled** dari NeonDB |
| `setor-hijau-db-url-direct` | `postgresql://...@ep-xxx.../neondb?sslmode=require` | Connection string **Direct** dari NeonDB (khusus migrasi) |
| `setor-hijau-jwt-secret` | *(string acak minimal 32 karakter)* | Generate via `openssl rand -base64 32` |

> **Catatan IAM:** Pastikan Service Account default Cloud Run (`<PROJECT_NUMBER>-compute@developer.gserviceaccount.com`) memiliki role **Secret Manager Secret Accessor** (`roles/secretmanager.secretAccessor`). Di Console, Cloud Run biasanya menawarkan tombol otomatis **"Grant access"** saat Anda menautkan secret.

### Langkah 5 — Build & Push Docker Image
Jalankan dari root repo `setor-hijau-be` menggunakan Google Cloud CLI (`gcloud`) yang sudah login:

```bash
# Set variabel agar praktis
export PROJECT_ID="setor-hijau"
export REGION="asia-southeast2"
export IMAGE_URI="${REGION}-docker.pkg.dev/${PROJECT_ID}/setor-hijau/api:latest"

# Build dan push langsung menggunakan Cloud Build (tidak perlu Docker lokal untuk push)
gcloud builds submit --tag "${IMAGE_URI}" .
```

*(Alternatif jika build di Docker lokal: `gcloud auth configure-docker ${REGION}-docker.pkg.dev`, lalu `docker build -t ${IMAGE_URI} .` dan `docker push ${IMAGE_URI}`.)*

### Langkah 6 — Buat & Jalankan Cloud Run Job untuk Migrasi Database
Migrasi **wajib** dijalankan terpisah sebelum service API baru aktif:

1. Di GCP Console, buka **Cloud Run -> Jobs -> Create Job**.
2. Isi:
   - **Container image URL**: pilih `${REGION}-docker.pkg.dev/${PROJECT_ID}/setor-hijau/api:latest`
   - **Job name**: `setor-hijau-migrate`
   - **Region**: sama dengan Artifact Registry (`asia-southeast2`)
   - Buka tab **Container(s) -> Container command**:
     - **Container command**: `/app/migrate`
     - **Container arguments**: `up`
   - Buka tab **Variables & Secrets -> Secrets -> Reference a Secret**:
     - Secret: `setor-hijau-db-url-direct`
     - Exposed as: **Environment variable** -> Name: `DATABASE_URL`
3. Klik **Create**, lalu klik **Execute** pada job `setor-hijau-migrate`.
4. Cek tab **Logs** dan pastikan muncul pesan: `Migrations applied successfully (up).`

*(Setiap kali ada migration SQL baru di `db/migrations/`, jalankan ulang job `setor-hijau-migrate` setelah push image baru.)*

### Langkah 7 — Deploy Cloud Run Service (API)
1. Di GCP Console, buka **Cloud Run -> Services -> Create Service** (atau **Deploy container**).
2. Isi konfigurasi:
   - **Container image URL**: `${REGION}-docker.pkg.dev/${PROJECT_ID}/setor-hijau/api:latest`
   - **Service name**: `setor-hijau-api`
   - **Region**: `asia-southeast1`
   - **Authentication**: Pilih **Allow unauthenticated invocations** (karena API publik + auth JWT ditangani oleh aplikasi sendiri).
   - **CPU allocation and pricing**: `CPU is only allocated during request processing`
   - **Min instances**: `0` (scale-to-zero) | **Max instances**: `3` (cukup untuk skala RW/skripsi)
   - **Container port**: `8080`
3. Buka bagian **Container(s) -> Variables & Secrets**:
   - **Environment variables**:
     - `ENV` = `production`
     - `LOG_LEVEL` = `info`
     - `DB_MAX_CONNS` = `10`
     - `CORS_ALLOWED_ORIGINS` = `https://domain-nextjs-anda.vercel.app,http://localhost:3000` *(sesuaikan dengan URL frontend Next.js)*
   - **Secrets (Exposed as Environment variables)**:
     - `DATABASE_URL` -> `setor-hijau-db-url` (latest)
     - `JWT_SECRET` -> `setor-hijau-jwt-secret` (latest)
4. *(Opsional tapi disarankan)* **Health checks**:
   - Startup / Liveness probe: HTTP GET `/healthz` port `8080`.
   - Readiness: `/readyz` dapat dipanggil untuk verifikasi koneksi DB.
5. Klik **Create** dan tunggu URL Cloud Run terbit (mis. `https://setor-hijau-api-xxxxx-as.a.run.app`).

---

## Bagian 3: Verifikasi Pasca-Deploy

Uji endpoint berikut menggunakan `curl` atau browser ke URL Cloud Run Anda:

1. **Liveness check:**
   ```bash
   curl https://<CLOUD_RUN_URL>/healthz
   # Harapan: {"status":"ok"}
   ```
2. **Readiness check (menguji koneksi ke NeonDB):**
   ```bash
   curl https://<CLOUD_RUN_URL>/readyz
   # Harapan: {"database":"connected","status":"ready"}
   ```
   *(Catatan: Jika NeonDB baru saja wake-up dari idle >5 menit, hit pertama bisa memakan waktu ~1–3 detik.)*
3. **Endpoint publik Metode & Sumber:**
   ```bash
   curl https://<CLOUD_RUN_URL>/api/v1/public/methods
   ```

---

## Bagian 4: Catatan Fallback (Supabase / Cloud SQL)

Jika suatu saat ingin berpindah dari NeonDB ke **Supabase (Postgres-only)** atau **Cloud SQL**:
1. Tidak perlu mengubah kode atau rebuild Docker image.
2. Jalankan job migrasi menggunakan direct URL DB tujuan.
3. Perbarui nilai secret `setor-hijau-db-url` di Secret Manager dengan `DATABASE_URL` baru (untuk Supabase Transaction Pooler port `6543`, driver `pgx` sudah otomatis memakai `QueryExecModeExec` yang kompatibel).
4. Deploy revisi baru di Cloud Run agar membaca versi secret terbaru.
