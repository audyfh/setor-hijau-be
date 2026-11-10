-- db/queries/reports.sql

-- name: GetMonthlyDepositSummary :one
SELECT 
    COUNT(*)::bigint AS deposits_count,
    COUNT(DISTINCT nasabah_id)::bigint AS active_nasabah_count,
    COALESCE(SUM(total_weight_grams), 0)::bigint AS total_weight_grams,
    COALESCE(SUM(total_amount_idr), 0)::bigint AS total_amount_idr
FROM deposits
WHERE bank_sampah_id = $1
  AND client_recorded_at >= $2
  AND client_recorded_at <= $3;

-- name: GetMonthlyDepositsByCategory :many
SELECT 
    mt.category,
    COALESCE(SUM(dl.weight_grams), 0)::bigint AS total_weight_grams,
    COALESCE(SUM(dl.subtotal_idr), 0)::bigint AS total_subtotal_idr
FROM deposit_lines dl
JOIN deposits d ON dl.deposit_id = d.id
JOIN material_types mt ON dl.material_type_id = mt.id
WHERE d.bank_sampah_id = $1
  AND d.client_recorded_at >= $2
  AND d.client_recorded_at <= $3
GROUP BY mt.category
ORDER BY total_weight_grams DESC;

-- name: GetMonthlySalesSummary :one
SELECT 
    COUNT(*)::bigint AS sales_count,
    COALESCE(SUM(total_weight_grams), 0)::bigint AS total_weight_grams,
    COALESCE(SUM(total_amount_idr), 0)::bigint AS total_amount_idr
FROM sales_to_pengepul
WHERE bank_sampah_id = $1
  AND sale_date >= $2
  AND sale_date <= $3;

-- name: GetMonthlySalesByCategory :many
SELECT 
    mt.category,
    COALESCE(SUM(sl.weight_grams), 0)::bigint AS total_weight_grams,
    COALESCE(SUM(sl.subtotal_idr), 0)::bigint AS total_subtotal_idr
FROM sales_to_pengepul_lines sl
JOIN sales_to_pengepul s ON sl.sale_id = s.id
JOIN material_types mt ON sl.material_type_id = mt.id
WHERE s.bank_sampah_id = $1
  AND s.sale_date >= $2
  AND s.sale_date <= $3
GROUP BY mt.category
ORDER BY total_weight_grams DESC;

-- name: GetMonthlyCashbookSummary :one
SELECT 
    COALESCE(SUM(CASE WHEN entry_type = 'cash_in' THEN amount_idr ELSE 0 END), 0)::bigint AS total_cash_in_idr,
    COALESCE(SUM(CASE WHEN entry_type = 'cash_out' THEN amount_idr ELSE 0 END), 0)::bigint AS total_cash_out_idr
FROM cashbook_entries
WHERE bank_sampah_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3;

-- name: GetMonthlyCashbookByCategory :many
SELECT 
    category,
    entry_type,
    COALESCE(SUM(amount_idr), 0)::bigint AS total_amount_idr,
    COUNT(*)::bigint AS count
FROM cashbook_entries
WHERE bank_sampah_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3
GROUP BY category, entry_type
ORDER BY total_amount_idr DESC;

-- name: GetMonthlyCarbonSummary :one
SELECT 
    COALESCE(SUM(co2e_grams), 0)::bigint AS total_co2e_grams,
    COALESCE(SUM(weight_grams), 0)::bigint AS total_weight_grams
FROM carbon_ledger
WHERE bank_sampah_id = $1
  AND created_at >= $2
  AND created_at <= $3;

-- name: GetMonthlyCarbonByCategory :many
SELECT 
    ef.category,
    COALESCE(SUM(cl.co2e_grams), 0)::bigint AS total_co2e_grams,
    COALESCE(SUM(cl.weight_grams), 0)::bigint AS total_weight_grams
FROM carbon_ledger cl
JOIN emission_factors ef ON cl.emission_factor_id = ef.id
WHERE cl.bank_sampah_id = $1
  AND cl.created_at >= $2
  AND cl.created_at <= $3
GROUP BY ef.category
ORDER BY total_co2e_grams DESC;

-- name: GetMonthlyDepositsDetailForReport :many
SELECT 
    d.id,
    d.client_recorded_at,
    u.full_name AS nasabah_name,
    u.username AS nasabah_username,
    d.total_weight_grams,
    d.total_amount_idr,
    COALESCE((SELECT SUM(co2e_grams) FROM carbon_ledger WHERE deposit_id = d.id), 0)::bigint AS total_estimated_co2e_avoided_grams,
    d.notes
FROM deposits d
JOIN users u ON d.nasabah_id = u.id
WHERE d.bank_sampah_id = $1
  AND d.client_recorded_at >= $2
  AND d.client_recorded_at <= $3
ORDER BY d.client_recorded_at ASC;

-- name: GetMonthlySalesDetailForReport :many
SELECT 
    s.id,
    s.sale_date,
    s.buyer_name,
    s.total_weight_grams,
    s.total_amount_idr,
    s.notes
FROM sales_to_pengepul s
WHERE s.bank_sampah_id = $1
  AND s.sale_date >= $2
  AND s.sale_date <= $3
ORDER BY s.sale_date ASC;

-- name: GetMonthlyCashbookDetailForReport :many
SELECT 
    c.id,
    c.transaction_date,
    c.entry_type,
    c.category,
    c.amount_idr,
    c.description,
    u.full_name AS recorded_by_name
FROM cashbook_entries c
JOIN users u ON c.recorded_by = u.id
WHERE c.bank_sampah_id = $1
  AND c.transaction_date >= $2
  AND c.transaction_date <= $3
ORDER BY c.transaction_date ASC, c.created_at ASC;
