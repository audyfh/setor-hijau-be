-- 000005_m4_carbon_ledger.up.sql
-- Milestone 4: Versioned Emission Factors and Append-Only Carbon Ledger

CREATE TABLE IF NOT EXISTS emission_factors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_code VARCHAR(64) NOT NULL,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    factor_version VARCHAR(32) NOT NULL,
    factor_source TEXT NOT NULL,
    co2e_factor_ppm BIGINT NOT NULL CHECK (co2e_factor_ppm >= 0),
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_emission_factors_code_version UNIQUE (material_code, factor_version)
);

CREATE INDEX IF NOT EXISTS idx_emission_factors_code ON emission_factors(material_code);
CREATE INDEX IF NOT EXISTS idx_emission_factors_category ON emission_factors(category);

-- Add emission_factor_id reference to material_types
ALTER TABLE material_types 
ADD COLUMN IF NOT EXISTS emission_factor_id UUID REFERENCES emission_factors(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_material_types_emission_factor ON material_types(emission_factor_id);

-- Append-only carbon ledger
CREATE TABLE IF NOT EXISTS carbon_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_sampah_id UUID NOT NULL REFERENCES bank_sampah(id) ON DELETE CASCADE,
    nasabah_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deposit_id UUID REFERENCES deposits(id) ON DELETE SET NULL,
    deposit_line_id UUID REFERENCES deposit_lines(id) ON DELETE SET NULL,
    emission_factor_id UUID NOT NULL REFERENCES emission_factors(id) ON DELETE RESTRICT,
    factor_version VARCHAR(32) NOT NULL,
    factor_source TEXT NOT NULL,
    weight_grams BIGINT NOT NULL,
    co2e_grams BIGINT NOT NULL,
    entry_type VARCHAR(32) NOT NULL DEFAULT 'deposit',
    reversal_for_id UUID REFERENCES carbon_ledger(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_carbon_ledger_bank_sampah ON carbon_ledger(bank_sampah_id);
CREATE INDEX IF NOT EXISTS idx_carbon_ledger_nasabah ON carbon_ledger(bank_sampah_id, nasabah_id);
CREATE INDEX IF NOT EXISTS idx_carbon_ledger_deposit ON carbon_ledger(deposit_id);
CREATE INDEX IF NOT EXISTS idx_carbon_ledger_created_at ON carbon_ledger(bank_sampah_id, created_at DESC);

-- Seed EPA WARM v16 emission factors
INSERT INTO emission_factors (material_code, name, category, factor_version, factor_source, co2e_factor_ppm, description)
VALUES 
('corrugated_containers', 'Kardus / Karton Gelombang (Corrugated Containers)', 'kertas', 'WARM v16', 'US EPA Waste Reduction Model Version 16 (2023)', 3428188, 'Faktor emisi daur ulang kardus/karton gelombang vs TPA'),
('mixed_paper', 'Kertas Campur / HVS / Koran / Majalah (Mixed Paper)', 'kertas', 'WARM v16', 'US EPA Waste Reduction Model Version 16 (2023)', 3891159, 'Faktor emisi daur ulang kertas kantor/HVS/koran campur vs TPA'),
('pet_bottles', 'Botol Plastik PET Transparan / Warna (PET Bottles)', 'plastik', 'WARM v16', 'US EPA Waste Reduction Model Version 16 (2023)', 1267658, 'Faktor emisi daur ulang botol plastik PET vs TPA'),
('hdpe_containers', 'Plastik HDPE Keras / Jeriken / Botol Detergen (HDPE Containers)', 'plastik', 'WARM v16', 'US EPA Waste Reduction Model Version 16 (2023)', 936965, 'Faktor emisi daur ulang plastik HDPE vs TPA'),
('aluminum_cans', 'Kaleng Minuman Aluminium (Aluminum Cans)', 'logam', 'WARM v16', 'US EPA Waste Reduction Model Version 16 (2023)', 10042057, 'Faktor emisi daur ulang kaleng aluminium vs TPA'),
('steel_cans', 'Kaleng Besi / Seng / Kaleng Makanan (Steel Cans)', 'logam', 'WARM v16', 'US EPA Waste Reduction Model Version 16 (2023)', 1995183, 'Faktor emisi daur ulang kaleng besi/baja vs TPA'),
('glass_containers', 'Botol Kaca / Wadah Gelas (Glass Containers)', 'kaca', 'WARM v16', 'US EPA Waste Reduction Model Version 16 (2023)', 308647, 'Faktor emisi daur ulang wadah kaca vs TPA')
ON CONFLICT (material_code, factor_version) DO NOTHING;

-- Map existing default materials if any by matching category
UPDATE material_types mt
SET emission_factor_id = ef.id
FROM emission_factors ef
WHERE mt.emission_factor_id IS NULL
  AND ((mt.category = 'kertas' AND ef.material_code = 'corrugated_containers')
    OR (mt.category = 'plastik' AND ef.material_code = 'pet_bottles')
    OR (mt.category = 'logam' AND ef.material_code = 'aluminum_cans')
    OR (mt.category = 'kaca' AND ef.material_code = 'glass_containers'));
