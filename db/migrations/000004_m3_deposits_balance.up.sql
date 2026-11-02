-- 000004_m3_deposits_balance.up.sql

CREATE TABLE deposits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    nasabah_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(64) NOT NULL,
    total_weight_grams BIGINT NOT NULL CHECK (total_weight_grams >= 0),
    total_amount_idr BIGINT NOT NULL CHECK (total_amount_idr >= 0),
    notes TEXT,
    recorded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    client_recorded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_deposits_bank_sampah_idempotency UNIQUE (bank_sampah_id, idempotency_key)
);

CREATE INDEX idx_deposits_bank_sampah ON deposits(bank_sampah_id);
CREATE INDEX idx_deposits_nasabah ON deposits(nasabah_id);
CREATE INDEX idx_deposits_bank_sampah_nasabah ON deposits(bank_sampah_id, nasabah_id);
CREATE INDEX idx_deposits_client_recorded ON deposits(bank_sampah_id, client_recorded_at DESC);

CREATE TABLE deposit_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deposit_id UUID NOT NULL REFERENCES deposits(id) ON DELETE CASCADE,
    material_type_id UUID NOT NULL REFERENCES material_types(id) ON DELETE RESTRICT,
    weight_grams BIGINT NOT NULL CHECK (weight_grams > 0),
    price_per_kg_idr BIGINT NOT NULL CHECK (price_per_kg_idr >= 0),
    subtotal_idr BIGINT NOT NULL CHECK (subtotal_idr >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_deposit_lines_deposit ON deposit_lines(deposit_id);
CREATE INDEX idx_deposit_lines_material ON deposit_lines(material_type_id);
