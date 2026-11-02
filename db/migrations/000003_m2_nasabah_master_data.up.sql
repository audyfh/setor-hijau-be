-- 000003_m2_nasabah_master_data.up.sql
-- Milestone 2: Nasabah Profiles, Material Types, and Historical Material Prices

CREATE TABLE IF NOT EXISTS nasabah_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    qr_token_hash VARCHAR(64) NOT NULL UNIQUE,
    pin_hash TEXT,
    failed_pin_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    rw VARCHAR(10) NOT NULL,
    rt VARCHAR(10),
    address_notes TEXT,
    registered_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nasabah_profiles_bank_sampah ON nasabah_profiles(bank_sampah_id);
CREATE INDEX IF NOT EXISTS idx_nasabah_profiles_qr_token_hash ON nasabah_profiles(qr_token_hash);

CREATE TABLE IF NOT EXISTS material_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    unit VARCHAR(20) NOT NULL DEFAULT 'kg',
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_material_types_bank_sampah_name UNIQUE (bank_sampah_id, name)
);

CREATE INDEX IF NOT EXISTS idx_material_types_bank_sampah ON material_types(bank_sampah_id);

CREATE TABLE IF NOT EXISTS material_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_type_id UUID NOT NULL REFERENCES material_types(id) ON DELETE CASCADE,
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    price_per_kg_idr BIGINT NOT NULL CHECK (price_per_kg_idr >= 0),
    effective_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    effective_to TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_material_prices_material_type ON material_prices(material_type_id);
CREATE INDEX IF NOT EXISTS idx_material_prices_bank_sampah ON material_prices(bank_sampah_id);
CREATE INDEX IF NOT EXISTS idx_material_prices_effective ON material_prices(material_type_id, effective_to);
