-- 000005_m4_carbon_ledger.down.sql
-- Rollback Milestone 4: Carbon Ledger and Emission Factors

DROP TABLE IF EXISTS carbon_ledger;

ALTER TABLE material_types
DROP COLUMN IF EXISTS emission_factor_id;

DROP TABLE IF EXISTS emission_factors;
