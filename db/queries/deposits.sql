-- db/queries/deposits.sql

-- name: GetDepositByIdempotencyKey :one
SELECT * FROM deposits
WHERE bank_sampah_id = $1 AND idempotency_key = $2;

-- name: CreateDeposit :one
INSERT INTO deposits (
    bank_sampah_id,
    nasabah_id,
    idempotency_key,
    total_weight_grams,
    total_amount_idr,
    notes,
    recorded_by,
    client_recorded_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: CreateDepositLine :one
INSERT INTO deposit_lines (
    deposit_id,
    material_type_id,
    weight_grams,
    price_per_kg_idr,
    subtotal_idr
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetDepositByID :one
SELECT d.*, 
       u_nasabah.full_name AS nasabah_name,
       u_nasabah.username AS nasabah_username,
       u_recorded.full_name AS recorded_by_name,
       COALESCE((SELECT SUM(co2e_grams) FROM carbon_ledger WHERE deposit_id = d.id), 0)::bigint AS total_estimated_co2e_avoided_grams
FROM deposits d
JOIN users u_nasabah ON d.nasabah_id = u_nasabah.id
JOIN users u_recorded ON d.recorded_by = u_recorded.id
WHERE d.id = $1 AND d.bank_sampah_id = $2;

-- name: GetDepositReceiptDetails :one
SELECT d.id AS deposit_id,
       d.bank_sampah_id,
       d.nasabah_id,
       d.idempotency_key,
       d.total_weight_grams,
       d.total_amount_idr,
       d.notes,
       d.client_recorded_at,
       d.created_at,
       COALESCE((SELECT SUM(co2e_grams) FROM carbon_ledger WHERE deposit_id = d.id), 0)::bigint AS total_estimated_co2e_avoided_grams,
       bs.name AS bank_sampah_name,
       bs.code AS bank_sampah_code,
       bs.address AS bank_sampah_address,
       bs.rw AS bank_sampah_rw,
       bs.rt AS bank_sampah_rt,
       bs.kelurahan AS bank_sampah_kelurahan,
       bs.kecamatan AS bank_sampah_kecamatan,
       bs.kabupaten_kota AS bank_sampah_kabupaten_kota,
       bs.contact_phone AS bank_sampah_contact_phone,
       u_nasabah.full_name AS nasabah_name,
       u_nasabah.username AS nasabah_username,
       np.rw AS nasabah_rw,
       np.rt AS nasabah_rt,
       u_nasabah.phone AS nasabah_phone,
       u_recorded.id AS recorded_by_id,
       u_recorded.full_name AS recorded_by_name
FROM deposits d
JOIN bank_sampah bs ON d.bank_sampah_id = bs.id
JOIN users u_nasabah ON d.nasabah_id = u_nasabah.id
JOIN nasabah_profiles np ON d.nasabah_id = np.user_id
JOIN users u_recorded ON d.recorded_by = u_recorded.id
WHERE d.id = $1 AND d.bank_sampah_id = $2;

-- name: GetDepositLinesByDepositID :many
SELECT dl.*,
       mt.name AS material_name,
       mt.category AS material_category,
       mt.unit AS material_unit,
       COALESCE(cl.co2e_grams, 0)::bigint AS estimated_co2e_avoided_grams,
       COALESCE(cl.factor_version, '')::varchar AS factor_version,
       COALESCE(cl.factor_source, '')::text AS factor_source
FROM deposit_lines dl
JOIN material_types mt ON dl.material_type_id = mt.id
LEFT JOIN carbon_ledger cl ON dl.id = cl.deposit_line_id
WHERE dl.deposit_id = $1
ORDER BY dl.created_at ASC;

-- name: ListDepositsByTenant :many
SELECT d.*,
       u_nasabah.full_name AS nasabah_name,
       u_nasabah.username AS nasabah_username,
       u_recorded.full_name AS recorded_by_name,
       COALESCE((SELECT SUM(co2e_grams) FROM carbon_ledger WHERE deposit_id = d.id), 0)::bigint AS total_estimated_co2e_avoided_grams
FROM deposits d
JOIN users u_nasabah ON d.nasabah_id = u_nasabah.id
JOIN users u_recorded ON d.recorded_by = u_recorded.id
WHERE d.bank_sampah_id = $1
  AND ($2::uuid IS NULL OR d.nasabah_id = $2)
  AND ($3::timestamptz IS NULL OR d.client_recorded_at >= $3)
  AND ($4::timestamptz IS NULL OR d.client_recorded_at <= $4)
ORDER BY d.client_recorded_at DESC
LIMIT $5 OFFSET $6;

-- name: CountDepositsByTenant :one
SELECT COUNT(*)
FROM deposits d
WHERE d.bank_sampah_id = $1
  AND ($2::uuid IS NULL OR d.nasabah_id = $2)
  AND ($3::timestamptz IS NULL OR d.client_recorded_at >= $3)
  AND ($4::timestamptz IS NULL OR d.client_recorded_at <= $4);

-- name: GetNasabahBalance :one
SELECT 
    (COALESCE(SUM(d.total_amount_idr), 0) - COALESCE((SELECT SUM(nw.amount_idr) FROM nasabah_withdrawals nw WHERE nw.bank_sampah_id = $1 AND nw.nasabah_id = $2), 0))::bigint AS current_balance_idr,
    COALESCE(SUM(d.total_amount_idr), 0)::bigint AS total_deposited_idr,
    COALESCE((SELECT SUM(nw.amount_idr) FROM nasabah_withdrawals nw WHERE nw.bank_sampah_id = $1 AND nw.nasabah_id = $2), 0)::bigint AS total_withdrawn_idr,
    COALESCE(SUM(d.total_weight_grams), 0)::bigint AS total_weight_grams,
    COUNT(d.id)::bigint AS total_deposits_count,
    MAX(d.client_recorded_at)::timestamptz AS last_deposit_at
FROM deposits d
WHERE d.bank_sampah_id = $1 AND d.nasabah_id = $2;

-- name: GetMaterialPriceAtTime :one
SELECT mt.id AS material_id,
       mt.name AS material_name,
       mt.category AS material_category,
       mt.unit AS material_unit,
       mt.is_active AS material_is_active,
       mt.emission_factor_id,
       ef.material_code AS factor_material_code,
       ef.name AS factor_name,
       ef.factor_version,
       ef.factor_source,
       ef.co2e_factor_ppm,
       mp.price_per_kg_idr
FROM material_types mt
LEFT JOIN material_prices mp ON mt.id = mp.material_type_id
  AND mp.bank_sampah_id = $1
  AND mp.id = (
      COALESCE(
          (SELECT p1.id FROM material_prices p1 
           WHERE p1.material_type_id = mt.id AND p1.bank_sampah_id = $1 AND p1.effective_from <= $2 AND (p1.effective_to IS NULL OR p1.effective_to > $2)
           ORDER BY p1.effective_from DESC LIMIT 1),
          (SELECT p2.id FROM material_prices p2 
           WHERE p2.material_type_id = mt.id AND p2.bank_sampah_id = $1
           ORDER BY p2.effective_from ASC LIMIT 1)
      )
  )
LEFT JOIN emission_factors ef ON mt.emission_factor_id = ef.id
WHERE mt.id = $3 AND mt.bank_sampah_id = $1
LIMIT 1;
