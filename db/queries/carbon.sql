-- db/queries/carbon.sql

-- name: ListActiveEmissionFactors :many
SELECT * FROM emission_factors
WHERE is_active = TRUE
ORDER BY category ASC, name ASC;

-- name: GetEmissionFactorByID :one
SELECT * FROM emission_factors
WHERE id = $1;

-- name: GetEmissionFactorByCodeAndVersion :one
SELECT * FROM emission_factors
WHERE material_code = $1 AND factor_version = $2;

-- name: CreateCarbonLedgerEntry :one
INSERT INTO carbon_ledger (
    bank_sampah_id,
    nasabah_id,
    deposit_id,
    deposit_line_id,
    emission_factor_id,
    factor_version,
    factor_source,
    weight_grams,
    co2e_grams,
    entry_type,
    reversal_for_id,
    notes,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: GetNasabahCarbonSavings :one
SELECT COALESCE(SUM(co2e_grams), 0)::bigint AS total_co2e_grams,
       COALESCE(SUM(CASE WHEN weight_grams > 0 THEN weight_grams ELSE 0 END), 0)::bigint AS total_weight_grams,
       COUNT(*)::bigint AS total_entries
FROM carbon_ledger
WHERE bank_sampah_id = $1 AND nasabah_id = $2;

-- name: GetBankSampahCarbonSavings :one
SELECT COALESCE(SUM(co2e_grams), 0)::bigint AS total_co2e_grams,
       COALESCE(SUM(CASE WHEN weight_grams > 0 THEN weight_grams ELSE 0 END), 0)::bigint AS total_weight_grams,
       COUNT(*)::bigint AS total_entries
FROM carbon_ledger
WHERE bank_sampah_id = $1;

-- name: GetNasabahCarbonSavingsByCategory :many
SELECT ef.category,
       COALESCE(SUM(cl.co2e_grams), 0)::bigint AS total_co2e_grams,
       COALESCE(SUM(cl.weight_grams), 0)::bigint AS total_weight_grams
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
WHERE cl.bank_sampah_id = $1 AND cl.nasabah_id = $2
GROUP BY ef.category
ORDER BY total_co2e_grams DESC;

-- name: ListCarbonLedgerByNasabah :many
SELECT cl.*,
       ef.name AS factor_name,
       ef.category AS factor_category
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
WHERE cl.bank_sampah_id = $1 AND cl.nasabah_id = $2
ORDER BY cl.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountCarbonLedgerByNasabah :one
SELECT COUNT(*)::bigint
FROM carbon_ledger
WHERE bank_sampah_id = $1 AND nasabah_id = $2;

-- name: ListCarbonLedgerByTenant :many
SELECT cl.*,
       ef.name AS factor_name,
       ef.category AS factor_category,
       u.full_name AS nasabah_name,
       u.username AS nasabah_username
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
JOIN users u ON cl.nasabah_id = u.id
WHERE cl.bank_sampah_id = $1
  AND ($2::uuid IS NULL OR cl.nasabah_id = $2)
  AND ($3::timestamptz IS NULL OR cl.created_at >= $3)
  AND ($4::timestamptz IS NULL OR cl.created_at <= $4)
ORDER BY cl.created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountCarbonLedgerByTenant :one
SELECT COUNT(*)::bigint
FROM carbon_ledger cl
WHERE cl.bank_sampah_id = $1
  AND ($2::uuid IS NULL OR cl.nasabah_id = $2)
  AND ($3::timestamptz IS NULL OR cl.created_at >= $3)
  AND ($4::timestamptz IS NULL OR cl.created_at <= $4);

-- name: GetCarbonLedgerByID :one
SELECT cl.*,
       ef.name AS factor_name,
       ef.category AS factor_category
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
WHERE cl.id = $1 AND cl.bank_sampah_id = $2;

-- name: GetCarbonLedgerByDepositID :many
SELECT cl.*,
       ef.name AS factor_name,
       ef.category AS factor_category
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
WHERE cl.deposit_id = $1 AND cl.bank_sampah_id = $2
ORDER BY cl.created_at ASC;
