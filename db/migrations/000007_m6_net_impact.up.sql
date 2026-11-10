-- 000007_m6_net_impact.up.sql
-- Milestone 6: Digital Footprint Measurements (SWDM v4) and Net Impact Meter

CREATE TABLE IF NOT EXISTS digital_footprint_measurements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_sampah_id UUID REFERENCES bank_sampah(id) ON DELETE CASCADE,
    session_id VARCHAR(64),
    page_path VARCHAR(255) NOT NULL,
    data_bytes_transferred BIGINT NOT NULL CHECK (data_bytes_transferred >= 0),
    co2e_micrograms BIGINT NOT NULL CHECK (co2e_micrograms >= 0),
    co2e_grams BIGINT NOT NULL CHECK (co2e_grams >= 0),
    swdm_version VARCHAR(32) NOT NULL DEFAULT 'SWDM v4',
    swdm_model_source TEXT NOT NULL DEFAULT 'Sustainable Web Design Model v4 (SustainableWebDesign.org)',
    visitor_type VARCHAR(16) NOT NULL DEFAULT 'new' CHECK (visitor_type IN ('new', 'returning')),
    client_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_digital_footprint_bank_sampah ON digital_footprint_measurements(bank_sampah_id);
CREATE INDEX IF NOT EXISTS idx_digital_footprint_created_at ON digital_footprint_measurements(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_digital_footprint_path ON digital_footprint_measurements(page_path);
