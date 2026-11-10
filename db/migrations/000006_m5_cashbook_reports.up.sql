-- 000006_m5_cashbook_reports.up.sql
-- Milestone 5: Sales to Pengepul, Nasabah Withdrawals, and Cashbook

-- 1. Wholesale sales to collectors / lapak pengepul
CREATE TABLE IF NOT EXISTS sales_to_pengepul (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    buyer_name VARCHAR(150) NOT NULL,
    sale_date TIMESTAMPTZ NOT NULL,
    total_weight_grams BIGINT NOT NULL CHECK (total_weight_grams >= 0),
    total_amount_idr BIGINT NOT NULL CHECK (total_amount_idr >= 0),
    notes TEXT,
    recorded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sales_to_pengepul_bank_sampah ON sales_to_pengepul(bank_sampah_id);
CREATE INDEX IF NOT EXISTS idx_sales_to_pengepul_date ON sales_to_pengepul(bank_sampah_id, sale_date DESC);

-- 2. Line items for sales to pengepul
CREATE TABLE IF NOT EXISTS sales_to_pengepul_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales_to_pengepul(id) ON DELETE CASCADE,
    material_type_id UUID NOT NULL REFERENCES material_types(id) ON DELETE RESTRICT,
    weight_grams BIGINT NOT NULL CHECK (weight_grams > 0),
    price_per_kg_idr BIGINT NOT NULL CHECK (price_per_kg_idr >= 0),
    subtotal_idr BIGINT NOT NULL CHECK (subtotal_idr >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sales_to_pengepul_lines_sale_id ON sales_to_pengepul_lines(sale_id);

-- 3. Nasabah cash balance withdrawals (tarik tunai)
CREATE TABLE IF NOT EXISTS nasabah_withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    nasabah_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_idr BIGINT NOT NULL CHECK (amount_idr > 0),
    notes TEXT,
    recorded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nasabah_withdrawals_bank_sampah ON nasabah_withdrawals(bank_sampah_id);
CREATE INDEX IF NOT EXISTS idx_nasabah_withdrawals_nasabah ON nasabah_withdrawals(bank_sampah_id, nasabah_id);
CREATE INDEX IF NOT EXISTS idx_nasabah_withdrawals_created_at ON nasabah_withdrawals(bank_sampah_id, created_at DESC);

-- 4. Central cashbook journal (append-only physical treasury ledger)
CREATE TABLE IF NOT EXISTS cashbook_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    entry_type VARCHAR(16) NOT NULL CHECK (entry_type IN ('cash_in', 'cash_out')),
    category VARCHAR(64) NOT NULL,
    amount_idr BIGINT NOT NULL CHECK (amount_idr > 0),
    reference_id UUID,
    description TEXT NOT NULL,
    recorded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    transaction_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cashbook_entries_bank_sampah ON cashbook_entries(bank_sampah_id);
CREATE INDEX IF NOT EXISTS idx_cashbook_entries_date ON cashbook_entries(bank_sampah_id, transaction_date DESC);
CREATE INDEX IF NOT EXISTS idx_cashbook_entries_category ON cashbook_entries(bank_sampah_id, category);
CREATE INDEX IF NOT EXISTS idx_cashbook_entries_reference ON cashbook_entries(reference_id);
