# Data Model — Setor Hijau

This document defines the relational database schema, conventions, and invariants for Setor Hijau.

---

## Conventions & Core Invariants

1. **Identifiers**: Primary keys use UUIDv4 via PostgreSQL `gen_random_uuid()`. Foreign keys match the target primary key type.
2. **Numeric Precision (Hard Rule #3)**:
   - **Currency (Money)**: Stored as `BIGINT` representing Indonesian Rupiah (IDR). No fractional rupiah. No floats.
   - **Weight**: Stored as `BIGINT` representing integer **grams** (e.g. 1.5 kg is stored as `1500`). No floats.
   - **Emissions**: Stored as `BIGINT` representing integer **grams of CO2 equivalent** (g CO2e) avoided. No floats.
3. **Timestamps**: All timestamps use `TIMESTAMPTZ` and default to `CURRENT_TIMESTAMP`. Application code always operates in UTC.
4. **Naming**:
   - Table names are `snake_case`, pluralized (e.g., `bank_sampah`, `users`, `refresh_tokens`, `deposits`).
   - Column names are `snake_case`.
   - Foreign keys follow the format `<singular_entity>_id` (e.g., `bank_sampah_id`, `user_id`).
5. **Multi-Tenancy (Hard Rule #1 & ADR 0004)**:
   - Every tenant-scoped table has a non-nullable `bank_sampah_id UUID REFERENCES bank_sampah(id)`.
   - Tenant isolation is enforced in application queries (`sqlc`) using context-injected `bank_sampah_id`.
   - Relational integrity is enforced by composite unique constraints and composite indexes: `(bank_sampah_id, id)` or `(bank_sampah_id, <field>)`.
6. **Immutability (Hard Rule #2)**:
   - Ledgers (`carbon_ledger`, `cashbook_entries`) and deposit records (`deposits`, `deposit_lines`) are append-only. No `UPDATE` or `DELETE` is permitted on committed transactions; corrections are done via explicit reversal records.

---

## Milestone 1 Schema (Tenancy & Authentication)

```mermaid
erDiagram
    BANK_SAMPAH ||--o{ USERS : "has members"
    USERS ||--o{ REFRESH_TOKENS : "has sessions"
    USERS ||--o{ PASSWORD_RESET_TOKENS : "has reset requests"

    BANK_SAMPAH {
        uuid id PK
        varchar name
        varchar code UK
        text address
        varchar rw
        varchar rt
        varchar kelurahan
        varchar kecamatan
        varchar kabupaten_kota
        varchar provinsi
        varchar contact_phone
        timestamptz created_at
        timestamptz updated_at
    }

    USERS {
        uuid id PK
        uuid bank_sampah_id FK
        varchar username
        varchar email UK
        text password_hash
        varchar full_name
        varchar role
        varchar phone
        boolean is_active
        timestamptz created_at
        timestamptz updated_at
    }

    REFRESH_TOKENS {
        uuid id PK
        uuid user_id FK
        text token_hash
        timestamptz expires_at
        timestamptz revoked_at
        timestamptz created_at
    }

    PASSWORD_RESET_TOKENS {
        uuid id PK
        uuid user_id FK
        text token_hash
        timestamptz expires_at
        timestamptz used_at
        timestamptz created_at
    }
```

### 1. `bank_sampah` (Tenants)
Represents the Bank Sampah Unit (BSU) at the RW level.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique tenant identifier |
| `name` | `VARCHAR(255)` | `NOT NULL` | Name of the bank sampah (e.g., "Bank Sampah Hijau Lestari RW 05") |
| `code` | `VARCHAR(64)` | `NOT NULL UNIQUE` | Unique URL-safe identifier / slug (e.g., "bs-hijau-lestari-rw05") |
| `address` | `TEXT` | `NOT NULL` | Street address or base camp location |
| `rw` | `VARCHAR(10)` | `NOT NULL` | RW number |
| `rt` | `VARCHAR(10)` | `NULL` | Optional RT number |
| `kelurahan` | `VARCHAR(100)` | `NOT NULL` | Kelurahan / Desa |
| `kecamatan` | `VARCHAR(100)` | `NOT NULL` | Kecamatan |
| `kabupaten_kota` | `VARCHAR(100)` | `NOT NULL` | Kota / Kabupaten |
| `provinsi` | `VARCHAR(100)` | `NOT NULL` | Provinsi |
| `contact_phone` | `VARCHAR(32)` | `NULL` | Contact phone number (WhatsApp) |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record update timestamp |

### 2. `users`
Accounts for both `pengelola` and `nasabah`.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique user identifier |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `username` | `VARCHAR(64)` | `NOT NULL` | Login identifier |
| `email` | `VARCHAR(255)` | `NULL UNIQUE` | Unique email for Pengelola; optional for Nasabah |
| `password_hash` | `TEXT` | `NOT NULL` | Argon2id hashed password |
| `full_name` | `VARCHAR(255)` | `NOT NULL` | Full name of the user |
| `role` | `VARCHAR(32)` | `NOT NULL` | Role: `pengelola` or `nasabah` |
| `phone` | `VARCHAR(32)` | `NULL` | Contact phone number |
| `is_active` | `BOOLEAN` | `NOT NULL DEFAULT TRUE` | Whether account is active |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Account creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Profile update timestamp |

*Indexes and Constraints:*
- `UNIQUE (bank_sampah_id, username)`: Usernames are unique within each bank sampah for Nasabah.
- `CREATE INDEX idx_users_bank_sampah ON users(bank_sampah_id);`
- `CREATE INDEX idx_users_username ON users(username);`
- `CREATE INDEX idx_users_role ON users(role);`

### 3. `refresh_tokens`
Tracks rotating refresh token sessions.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Session ID |
| `user_id` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE CASCADE` | User owning the session |
| `token_hash` | `TEXT` | `NOT NULL UNIQUE` | SHA-256 hash of the opaque token string |
| `expires_at` | `TIMESTAMPTZ` | `NOT NULL` | Expiry timestamp (typically +7 days) |
| `revoked_at` | `TIMESTAMPTZ` | `NULL` | Revocation timestamp if logged out or rotated |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Issuance timestamp |

*Indexes and Constraints:*
- `CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);`
- `CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);`

### 4. `password_reset_tokens`
Self-service recovery tokens for Pengelola accounts.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Reset request ID |
| `user_id` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE CASCADE` | User requesting reset |
| `token_hash` | `TEXT` | `NOT NULL UNIQUE` | SHA-256 hash of the single-use token |
| `expires_at` | `TIMESTAMPTZ` | `NOT NULL` | Expiration (typically +15 minutes) |
| `used_at` | `TIMESTAMPTZ` | `NULL` | Timestamp when the token was used |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Issuance timestamp |

---

## Milestone 2 Schema (Nasabah & Master Data)

```mermaid
erDiagram
    BANK_SAMPAH ||--o{ NASABAH_PROFILES : "has profiles"
    USERS ||--o| NASABAH_PROFILES : "profile of"
    BANK_SAMPAH ||--o{ MATERIAL_TYPES : "configures"
    MATERIAL_TYPES ||--o{ MATERIAL_PRICES : "has price history"
    BANK_SAMPAH ||--o{ MATERIAL_PRICES : "tenant scoped"

    NASABAH_PROFILES {
        uuid user_id PK, FK
        uuid bank_sampah_id FK
        varchar qr_token_hash UK
        text pin_hash
        int failed_pin_attempts
        timestamptz locked_until
        varchar rw
        varchar rt
        text address_notes
        uuid registered_by FK
        timestamptz created_at
        timestamptz updated_at
    }

    MATERIAL_TYPES {
        uuid id PK
        uuid bank_sampah_id FK
        varchar name
        varchar category
        varchar unit
        text description
        boolean is_active
        timestamptz created_at
        timestamptz updated_at
    }

    MATERIAL_PRICES {
        uuid id PK
        uuid material_type_id FK
        uuid bank_sampah_id FK
        bigint price_per_kg_idr
        timestamptz effective_from
        timestamptz effective_to
        uuid created_by FK
        timestamptz created_at
    }
```

### 5. `nasabah_profiles`
Metadata, QR identification token hash, and PIN security for accounts with role `nasabah`.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `user_id` | `UUID` | `PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE` | User account |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `qr_token_hash` | `VARCHAR(64)` | `NOT NULL UNIQUE` | SHA-256 hash of the 32-byte opaque QR token string |
| `pin_hash` | `TEXT` | `NULL` | Argon2id hash of the optional 6-digit PIN |
| `failed_pin_attempts` | `INT` | `NOT NULL DEFAULT 0` | Consecutive failed PIN attempts |
| `locked_until` | `TIMESTAMPTZ` | `NULL` | Temporary lockout timestamp after 5 failed attempts |
| `rw` | `VARCHAR(10)` | `NOT NULL` | Neighborhood unit (RW) |
| `rt` | `VARCHAR(10)` | `NULL` | Neighborhood unit (RT) |
| `address_notes` | `TEXT` | `NULL` | Optional address or house number notes |
| `registered_by` | `UUID` | `NULL REFERENCES users(id) ON DELETE SET NULL` | Pengelola who created this account |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Profile creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Profile update timestamp |

*Indexes and Constraints:*
- `CREATE INDEX idx_nasabah_profiles_bank_sampah ON nasabah_profiles(bank_sampah_id);`
- `CREATE INDEX idx_nasabah_profiles_qr_token_hash ON nasabah_profiles(qr_token_hash);`

### 6. `material_types`
Master data of recyclable waste categories accepted by a bank sampah.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique material identifier |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `name` | `VARCHAR(100)` | `NOT NULL` | Name of the material (e.g., "Kardus Bekas", "Botol PET") |
| `category` | `VARCHAR(50)` | `NOT NULL` | Category group (e.g. `kertas`, `plastik`, `logam`, `kaca`, `lainnya`) |
| `unit` | `VARCHAR(20)` | `NOT NULL DEFAULT 'kg'` | Display unit (default `kg`) |
| `description` | `TEXT` | `NULL` | Sorting guidelines or notes |
| `emission_factor_id` | `UUID` | `NULL REFERENCES emission_factors(id) ON DELETE SET NULL` | Linked WARM emission factor (added in M4) |
| `is_active` | `BOOLEAN` | `NOT NULL DEFAULT TRUE` | Active for deposit recording |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record update timestamp |

*Indexes and Constraints:*
- `CONSTRAINT uq_material_types_bank_sampah_name UNIQUE (bank_sampah_id, name)`
- `CREATE INDEX idx_material_types_bank_sampah ON material_types(bank_sampah_id);`

### 7. `material_prices`
Append-only historical prices per kilogram in integer Rupiah (Hard Rule #3).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique price entry ID |
| `material_type_id` | `UUID` | `NOT NULL REFERENCES material_types(id) ON DELETE CASCADE` | Material reference |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Tenant reference |
| `price_per_kg_idr` | `BIGINT` | `NOT NULL CHECK (price_per_kg_idr >= 0)` | Price per kilogram in IDR |
| `effective_from` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | When price takes effect |
| `effective_to` | `TIMESTAMPTZ` | `NULL` | When price ceased (`NULL` = currently active) |
| `created_by` | `UUID` | `NULL REFERENCES users(id) ON DELETE SET NULL` | Pengelola who set this price |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Entry creation timestamp |

*Indexes and Constraints:*
- `CREATE INDEX idx_material_prices_material_type ON material_prices(material_type_id);`
- `CREATE INDEX idx_material_prices_bank_sampah ON material_prices(bank_sampah_id);`
- `CREATE INDEX idx_material_prices_effective ON material_prices(material_type_id, effective_to);`

---

## Milestone 3 Schema (Deposits & Balance)

```mermaid
erDiagram
    BANK_SAMPAH ||--o{ DEPOSITS : "tenant scoped"
    USERS ||--o{ DEPOSITS : "recorded for nasabah"
    USERS ||--o{ DEPOSITS : "recorded by pengelola"
    DEPOSITS ||--o{ DEPOSIT_LINES : "contains"
    MATERIAL_TYPES ||--o{ DEPOSIT_LINES : "material type"

    DEPOSITS {
        uuid id PK
        uuid bank_sampah_id FK
        uuid nasabah_id FK
        varchar idempotency_key
        bigint total_weight_grams
        bigint total_amount_idr
        text notes
        uuid recorded_by FK
        timestamptz client_recorded_at
        timestamptz created_at
    }

    DEPOSIT_LINES {
        uuid id PK
        uuid deposit_id FK
        uuid material_type_id FK
        bigint weight_grams
        bigint price_per_kg_idr
        bigint subtotal_idr
        timestamptz created_at
    }
```

### 8. `deposits`
Immutable header record for waste deposits (Hard Rule #2 & Hard Rule #5).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique deposit transaction ID |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `nasabah_id` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Customer who deposited waste |
| `idempotency_key` | `VARCHAR(64)` | `NOT NULL` | Client-generated UUID for offline deduplication |
| `total_weight_grams` | `BIGINT` | `NOT NULL CHECK (total_weight_grams >= 0)` | Aggregated weight in integer grams |
| `total_amount_idr` | `BIGINT` | `NOT NULL CHECK (total_amount_idr >= 0)` | Aggregated value in integer Rupiah |
| `notes` | `TEXT` | `NULL` | Optional transaction notes |
| `recorded_by` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Officer/Pengelola who entered transaction |
| `client_recorded_at` | `TIMESTAMPTZ` | `NOT NULL` | Time when physical deposit happened on device |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Server receipt and persistence timestamp |

*Indexes and Constraints:*
- `CONSTRAINT uq_deposits_bank_sampah_idempotency UNIQUE (bank_sampah_id, idempotency_key)`
- `CREATE INDEX idx_deposits_bank_sampah ON deposits(bank_sampah_id);`
- `CREATE INDEX idx_deposits_nasabah ON deposits(nasabah_id);`
- `CREATE INDEX idx_deposits_bank_sampah_nasabah ON deposits(bank_sampah_id, nasabah_id);`
- `CREATE INDEX idx_deposits_client_recorded ON deposits(bank_sampah_id, client_recorded_at DESC);`

### 9. `deposit_lines`
Immutable itemized waste lines with permanently snapshotted prices (Hard Rule #3 & Invariant #4).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique line item ID |
| `deposit_id` | `UUID` | `NOT NULL REFERENCES deposits(id) ON DELETE CASCADE` | Parent deposit transaction |
| `material_type_id` | `UUID` | `NOT NULL REFERENCES material_types(id) ON DELETE RESTRICT` | Material reference |
| `weight_grams` | `BIGINT` | `NOT NULL CHECK (weight_grams > 0)` | Weight in integer grams |
| `price_per_kg_idr` | `BIGINT` | `NOT NULL CHECK (price_per_kg_idr >= 0)` | Historical price per kg snapshotted at `client_recorded_at` |
| `subtotal_idr` | `BIGINT` | `NOT NULL CHECK (subtotal_idr >= 0)` | Subtotal: `(weight_grams * price_per_kg_idr + 500) / 1000` |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Line creation timestamp |

*Indexes and Constraints:*
- `CREATE INDEX idx_deposit_lines_deposit ON deposit_lines(deposit_id);`
- `CREATE INDEX idx_deposit_lines_material ON deposit_lines(material_type_id);`

---

## Milestone 4 Schema (Carbon Ledger)

```mermaid
erDiagram
    EMISSION_FACTORS ||--o{ MATERIAL_TYPES : "maps to"
    BANK_SAMPAH ||--o{ CARBON_LEDGER : "tenant scoped"
    USERS ||--o{ CARBON_LEDGER : "earned by nasabah"
    DEPOSITS ||--o{ CARBON_LEDGER : "generated by deposit"
    DEPOSIT_LINES ||--o{ CARBON_LEDGER : "line source"
    EMISSION_FACTORS ||--o{ CARBON_LEDGER : "uses factor"
    CARBON_LEDGER ||--o| CARBON_LEDGER : "reverses"

    EMISSION_FACTORS {
        uuid id PK
        varchar material_code
        varchar name
        varchar category
        varchar factor_version
        text factor_source
        bigint co2e_factor_ppm
        text description
        boolean is_active
        timestamptz created_at
    }

    CARBON_LEDGER {
        uuid id PK
        uuid bank_sampah_id FK
        uuid nasabah_id FK
        uuid deposit_id FK
        uuid deposit_line_id FK
        uuid emission_factor_id FK
        varchar factor_version
        text factor_source
        bigint weight_grams
        bigint co2e_grams
        varchar entry_type
        uuid reversal_for_id FK
        text notes
        timestamptz created_at
    }
```

### 10. `emission_factors`
Platform-wide versioned emission factors from US EPA WARM v16 (Hard Rule #4).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique factor identifier |
| `material_code` | `VARCHAR(64)` | `NOT NULL` | Standard material code (e.g., `corrugated_containers`, `pet_bottles`) |
| `name` | `VARCHAR(100)` | `NOT NULL` | Descriptive name in Indonesian / English |
| `category` | `VARCHAR(50)` | `NOT NULL` | Material category group (e.g. `kertas`, `plastik`, `logam`, `kaca`) |
| `factor_version` | `VARCHAR(32)` | `NOT NULL` | Model version (e.g., `'WARM v16'`) |
| `factor_source` | `TEXT` | `NOT NULL` | Official source citation (e.g., `'US EPA Waste Reduction Model Version 16 (2023)'`) |
| `co2e_factor_ppm` | `BIGINT` | `NOT NULL CHECK (co2e_factor_ppm >= 0)` | Fixed-point scale $10^6$ PPM ($\mu\text{g } CO_2e / \text{g waste}$) |
| `description` | `TEXT` | `NULL` | Methodology or sorting details |
| `is_active` | `BOOLEAN` | `NOT NULL DEFAULT TRUE` | Active for calculation |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |

*Indexes and Constraints:*
- `CONSTRAINT uq_emission_factors_code_version UNIQUE (material_code, factor_version)`
- `CREATE INDEX idx_emission_factors_code ON emission_factors(material_code);`
- `CREATE INDEX idx_emission_factors_category ON emission_factors(category);`

### 11. `carbon_ledger`
Append-only environmental journal for estimated avoided CO2e emissions (Hard Rules #1, #2, #3, #4, #9).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique ledger entry ID |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `nasabah_id` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Nasabah receiving carbon savings |
| `deposit_id` | `UUID` | `NULL REFERENCES deposits(id) ON DELETE SET NULL` | Originating deposit transaction |
| `deposit_line_id` | `UUID` | `NULL REFERENCES deposit_lines(id) ON DELETE SET NULL` | Specific deposit item line |
| `emission_factor_id` | `UUID` | `NOT NULL REFERENCES emission_factors(id) ON DELETE RESTRICT` | Emission factor used |
| `factor_version` | `VARCHAR(32)` | `NOT NULL` | Snapshotted factor version at record time |
| `factor_source` | `TEXT` | `NOT NULL` | Snapshotted factor source at record time |
| `weight_grams` | `BIGINT` | `NOT NULL` | Weight in grams (negative for reversals) |
| `co2e_grams` | `BIGINT` | `NOT NULL` | Estimated avoided CO2e in grams (negative for reversals) |
| `entry_type` | `VARCHAR(32)` | `NOT NULL DEFAULT 'deposit'` | Entry type: `'deposit'` or `'reversal'` |
| `reversal_for_id` | `UUID` | `NULL REFERENCES carbon_ledger(id) ON DELETE SET NULL` | Referenced original ledger entry if reversal |
| `notes` | `TEXT` | `NULL` | Optional explanation or reversal justification |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Timestamp of journal entry |

*Indexes and Constraints:*
- `CREATE INDEX idx_carbon_ledger_bank_sampah ON carbon_ledger(bank_sampah_id);`
- `CREATE INDEX idx_carbon_ledger_nasabah ON carbon_ledger(bank_sampah_id, nasabah_id);`
- `CREATE INDEX idx_carbon_ledger_deposit ON carbon_ledger(deposit_id);`
- `CREATE INDEX idx_carbon_ledger_created_at ON carbon_ledger(bank_sampah_id, created_at DESC);`

### 12. `sales_to_pengepul`
Records wholesale sales of collected recyclables to commercial buyers/collectors (ADR 0009).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique sale identifier |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `buyer_name` | `VARCHAR(150)` | `NOT NULL` | Name of collector / lapak buyer |
| `sale_date` | `TIMESTAMPTZ` | `NOT NULL` | Transaction date |
| `total_weight_grams` | `BIGINT` | `NOT NULL CHECK (total_weight_grams >= 0)` | Total weight of sold materials in grams |
| `total_amount_idr` | `BIGINT` | `NOT NULL CHECK (total_amount_idr >= 0)` | Total sale revenue in IDR |
| `notes` | `TEXT` | `NULL` | Optional transaction notes |
| `recorded_by` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Pengelola recording the sale |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |

*Indexes:*
- `CREATE INDEX idx_sales_to_pengepul_bank_sampah ON sales_to_pengepul(bank_sampah_id);`
- `CREATE INDEX idx_sales_to_pengepul_date ON sales_to_pengepul(bank_sampah_id, sale_date DESC);`

### 13. `sales_to_pengepul_lines`
Itemized line items for wholesale waste sales.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique line item ID |
| `sale_id` | `UUID` | `NOT NULL REFERENCES sales_to_pengepul(id) ON DELETE CASCADE` | Parent sale transaction |
| `material_type_id` | `UUID` | `NOT NULL REFERENCES material_types(id) ON DELETE RESTRICT` | Material sold |
| `weight_grams` | `BIGINT` | `NOT NULL CHECK (weight_grams > 0)` | Weight sold in grams |
| `price_per_kg_idr` | `BIGINT` | `NOT NULL CHECK (price_per_kg_idr >= 0)` | Wholesale price negotiated per kg |
| `subtotal_idr` | `BIGINT` | `NOT NULL CHECK (subtotal_idr >= 0)` | Subtotal in IDR |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |

*Indexes:*
- `CREATE INDEX idx_sales_to_pengepul_lines_sale_id ON sales_to_pengepul_lines(sale_id);`

### 14. `nasabah_withdrawals`
Records cash withdrawals (tarik tunai) by nasabah against their accumulated waste balance (Domain Invariant #2, ADR 0009).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique withdrawal identifier |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `nasabah_id` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Nasabah withdrawing funds |
| `amount_idr` | `BIGINT` | `NOT NULL CHECK (amount_idr > 0)` | Withdrawn amount in IDR |
| `notes` | `TEXT` | `NULL` | Optional disbursement notes |
| `recorded_by` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Pengelola dispensing cash |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |

*Indexes:*
- `CREATE INDEX idx_nasabah_withdrawals_bank_sampah ON nasabah_withdrawals(bank_sampah_id);`
- `CREATE INDEX idx_nasabah_withdrawals_nasabah ON nasabah_withdrawals(bank_sampah_id, nasabah_id);`
- `CREATE INDEX idx_nasabah_withdrawals_created_at ON nasabah_withdrawals(bank_sampah_id, created_at DESC);`

### 15. `cashbook_entries`
Append-only physical cash treasury ledger for the bank sampah (Hard Rules #1, #2, #3, ADR 0009).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique entry ID |
| `bank_sampah_id` | `UUID` | `NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Associated tenant |
| `entry_type` | `VARCHAR(16)` | `NOT NULL CHECK (entry_type IN ('cash_in', 'cash_out'))` | Cash movement direction |
| `category` | `VARCHAR(64)` | `NOT NULL` | Category: `penjualan_pengepul`, `penarikan_nasabah`, `operasional`, `modal_awal`, `hibah_donasi`, `lainnya` |
| `amount_idr` | `BIGINT` | `NOT NULL CHECK (amount_idr > 0)` | Magnitude of cash moved in IDR |
| `reference_id` | `UUID` | `NULL` | Foreign reference (e.g. sale ID or withdrawal ID) |
| `description` | `TEXT` | `NOT NULL` | Human-readable explanation |
| `recorded_by` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Pengelola recording the entry |
| `transaction_date` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Date when cash was transferred |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |

*Indexes and Constraints:*
- `CREATE INDEX idx_cashbook_entries_bank_sampah ON cashbook_entries(bank_sampah_id);`
- `CREATE INDEX idx_cashbook_entries_date ON cashbook_entries(bank_sampah_id, transaction_date DESC);`
- `CREATE INDEX idx_cashbook_entries_category ON cashbook_entries(bank_sampah_id, category);`
- Trigger append-only: prevents `UPDATE` and `DELETE`.

### 16. `digital_footprint_measurements`
Digital carbon footprint measurements calculated via Sustainable Web Design Model (SWDM) v4 (ADR 0010).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique measurement ID |
| `bank_sampah_id` | `UUID` | `NULL REFERENCES bank_sampah(id) ON DELETE CASCADE` | Optional tenant attribution (NULL for general platform traffic) |
| `session_id` | `VARCHAR(64)` | `NULL` | Client session identifier |
| `page_path` | `VARCHAR(255)` | `NOT NULL` | Requested web path / route |
| `data_bytes_transferred` | `BIGINT` | `NOT NULL CHECK (data_bytes_transferred >= 0)` | Network payload transferred in bytes |
| `co2e_micrograms` | `BIGINT` | `NOT NULL CHECK (co2e_micrograms >= 0)` | Calculated emissions in integer micrograms ($\mu\text{g } CO_2e$) |
| `co2e_grams` | `BIGINT` | `NOT NULL CHECK (co2e_grams >= 0)` | Calculated emissions in integer grams ($g CO_2e$) |
| `swdm_version` | `VARCHAR(32)` | `NOT NULL DEFAULT 'SWDM v4'` | Standard model version |
| `swdm_model_source` | `TEXT` | `NOT NULL` | Official model citation |
| `visitor_type` | `VARCHAR(16)` | `NOT NULL DEFAULT 'new'` | Visitor type: `'new'` or `'returning'` |
| `client_timestamp` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Time client measured the data |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Ingestion timestamp |

*Indexes:*
- `CREATE INDEX idx_digital_footprint_bank_sampah ON digital_footprint_measurements(bank_sampah_id);`
- `CREATE INDEX idx_digital_footprint_created_at ON digital_footprint_measurements(created_at DESC);`
- `CREATE INDEX idx_digital_footprint_path ON digital_footprint_measurements(page_path);`
