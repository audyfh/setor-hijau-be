-- db/queries/impact.sql

-- name: RecordDigitalFootprintMeasurement :one
INSERT INTO digital_footprint_measurements (
    bank_sampah_id,
    session_id,
    page_path,
    data_bytes_transferred,
    co2e_micrograms,
    co2e_grams,
    swdm_version,
    swdm_model_source,
    visitor_type,
    client_timestamp
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetPlatformDigitalFootprintSummary :one
SELECT 
    COALESCE(SUM(data_bytes_transferred), 0)::bigint AS total_bytes_transferred,
    COALESCE(SUM(co2e_micrograms), 0)::bigint AS total_co2e_micrograms,
    COALESCE(SUM(co2e_grams), 0)::bigint AS total_co2e_grams,
    COUNT(*)::bigint AS measurements_count
FROM digital_footprint_measurements;

-- name: GetTenantDigitalFootprintSummary :one
SELECT 
    COALESCE(SUM(data_bytes_transferred), 0)::bigint AS total_bytes_transferred,
    COALESCE(SUM(co2e_micrograms), 0)::bigint AS total_co2e_micrograms,
    COALESCE(SUM(co2e_grams), 0)::bigint AS total_co2e_grams,
    COUNT(*)::bigint AS measurements_count
FROM digital_footprint_measurements
WHERE bank_sampah_id = $1;

-- name: GetPlatformAvoidedCarbonSummary :one
SELECT 
    COALESCE(SUM(co2e_grams), 0)::bigint AS total_co2e_grams,
    COALESCE(SUM(weight_grams), 0)::bigint AS total_weight_grams,
    COUNT(*)::bigint AS entries_count
FROM carbon_ledger;

-- name: GetTenantAvoidedCarbonSummary :one
SELECT 
    COALESCE(SUM(co2e_grams), 0)::bigint AS total_co2e_grams,
    COALESCE(SUM(weight_grams), 0)::bigint AS total_weight_grams,
    COUNT(*)::bigint AS entries_count
FROM carbon_ledger
WHERE bank_sampah_id = $1;

-- name: GetPlatformAvoidedCarbonByCategory :many
SELECT 
    ef.category,
    COALESCE(SUM(cl.co2e_grams), 0)::bigint AS total_co2e_grams,
    COALESCE(SUM(cl.weight_grams), 0)::bigint AS total_weight_grams
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
GROUP BY ef.category
ORDER BY total_co2e_grams DESC;

-- name: GetTenantAvoidedCarbonByCategory :many
SELECT 
    ef.category,
    COALESCE(SUM(cl.co2e_grams), 0)::bigint AS total_co2e_grams,
    COALESCE(SUM(cl.weight_grams), 0)::bigint AS total_weight_grams
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
WHERE cl.bank_sampah_id = $1
GROUP BY ef.category
ORDER BY total_co2e_grams DESC;

-- name: ListActiveEmissionFactorsForMethods :many
SELECT 
    id,
    material_code,
    name,
    category,
    factor_version,
    factor_source,
    co2e_factor_ppm,
    description
FROM emission_factors
WHERE is_active = TRUE
ORDER BY category ASC, name ASC;
