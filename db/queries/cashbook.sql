-- db/queries/cashbook.sql

-- 1. Sales to Pengepul Queries
-- name: CreateSaleToPengepul :one
INSERT INTO sales_to_pengepul (
    bank_sampah_id,
    buyer_name,
    sale_date,
    total_weight_grams,
    total_amount_idr,
    notes,
    recorded_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: CreateSaleToPengepulLine :one
INSERT INTO sales_to_pengepul_lines (
    sale_id,
    material_type_id,
    weight_grams,
    price_per_kg_idr,
    subtotal_idr
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetSaleToPengepulByID :one
SELECT s.*,
       u.full_name AS recorded_by_name
FROM sales_to_pengepul s
JOIN users u ON s.recorded_by = u.id
WHERE s.bank_sampah_id = $1 AND s.id = $2;

-- name: GetSaleLinesBySaleID :many
SELECT sl.*,
       mt.name AS material_name,
       mt.category AS material_category,
       mt.unit AS material_unit
FROM sales_to_pengepul_lines sl
JOIN material_types mt ON sl.material_type_id = mt.id
WHERE sl.sale_id = $1
ORDER BY sl.created_at ASC;

-- name: ListSalesToPengepul :many
SELECT s.*,
       u.full_name AS recorded_by_name
FROM sales_to_pengepul s
JOIN users u ON s.recorded_by = u.id
WHERE s.bank_sampah_id = $1
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR s.sale_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR s.sale_date <= sqlc.narg('end_date'))
ORDER BY s.sale_date DESC
LIMIT $2 OFFSET $3;

-- name: CountSalesToPengepul :one
SELECT COUNT(*)
FROM sales_to_pengepul s
WHERE s.bank_sampah_id = $1
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR s.sale_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR s.sale_date <= sqlc.narg('end_date'));

-- 2. Nasabah Withdrawals Queries
-- name: CreateNasabahWithdrawal :one
INSERT INTO nasabah_withdrawals (
    bank_sampah_id,
    nasabah_id,
    amount_idr,
    notes,
    recorded_by
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetWithdrawalByID :one
SELECT w.*,
       u_nasabah.full_name AS nasabah_name,
       u_nasabah.username AS nasabah_username,
       u_recorded.full_name AS recorded_by_name
FROM nasabah_withdrawals w
JOIN users u_nasabah ON w.nasabah_id = u_nasabah.id
JOIN users u_recorded ON w.recorded_by = u_recorded.id
WHERE w.bank_sampah_id = $1 AND w.id = $2;

-- name: ListWithdrawalsByTenant :many
SELECT w.*,
       u_nasabah.full_name AS nasabah_name,
       u_nasabah.username AS nasabah_username,
       u_recorded.full_name AS recorded_by_name
FROM nasabah_withdrawals w
JOIN users u_nasabah ON w.nasabah_id = u_nasabah.id
JOIN users u_recorded ON w.recorded_by = u_recorded.id
WHERE w.bank_sampah_id = $1
  AND (sqlc.narg('nasabah_id')::uuid IS NULL OR w.nasabah_id = sqlc.narg('nasabah_id'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR w.created_at >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR w.created_at <= sqlc.narg('end_date'))
ORDER BY w.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountWithdrawalsByTenant :one
SELECT COUNT(*)
FROM nasabah_withdrawals w
WHERE w.bank_sampah_id = $1
  AND (sqlc.narg('nasabah_id')::uuid IS NULL OR w.nasabah_id = sqlc.narg('nasabah_id'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR w.created_at >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR w.created_at <= sqlc.narg('end_date'));

-- 3. Cashbook Entries Queries
-- name: CreateCashbookEntry :one
INSERT INTO cashbook_entries (
    bank_sampah_id,
    entry_type,
    category,
    amount_idr,
    reference_id,
    description,
    recorded_by,
    transaction_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ListCashbookEntries :many
SELECT c.*,
       u.full_name AS recorded_by_name
FROM cashbook_entries c
JOIN users u ON c.recorded_by = u.id
WHERE c.bank_sampah_id = $1
  AND (sqlc.narg('entry_type')::varchar IS NULL OR c.entry_type = sqlc.narg('entry_type'))
  AND (sqlc.narg('category')::varchar IS NULL OR c.category = sqlc.narg('category'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR c.transaction_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR c.transaction_date <= sqlc.narg('end_date'))
ORDER BY c.transaction_date DESC, c.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountCashbookEntries :one
SELECT COUNT(*)
FROM cashbook_entries c
WHERE c.bank_sampah_id = $1
  AND (sqlc.narg('entry_type')::varchar IS NULL OR c.entry_type = sqlc.narg('entry_type'))
  AND (sqlc.narg('category')::varchar IS NULL OR c.category = sqlc.narg('category'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR c.transaction_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR c.transaction_date <= sqlc.narg('end_date'));

-- name: GetCashbookSummary :one
SELECT 
    COALESCE(SUM(CASE WHEN entry_type = 'cash_in' THEN amount_idr ELSE 0 END), 0)::bigint AS total_cash_in_idr,
    COALESCE(SUM(CASE WHEN entry_type = 'cash_out' THEN amount_idr ELSE 0 END), 0)::bigint AS total_cash_out_idr,
    (COALESCE(SUM(CASE WHEN entry_type = 'cash_in' THEN amount_idr ELSE 0 END), 0) -
     COALESCE(SUM(CASE WHEN entry_type = 'cash_out' THEN amount_idr ELSE 0 END), 0))::bigint AS current_cash_balance_idr
FROM cashbook_entries
WHERE bank_sampah_id = $1;

-- 4. Reconciliation Queries
-- name: GetFinancialReconciliation :one
SELECT
    -- Pengepul sales vs cashbook sales
    COALESCE((SELECT SUM(s.total_amount_idr) FROM sales_to_pengepul s WHERE s.bank_sampah_id = bs.id), 0)::bigint AS total_sales_to_pengepul_idr,
    COALESCE((SELECT SUM(c.amount_idr) FROM cashbook_entries c WHERE c.bank_sampah_id = bs.id AND c.category = 'penjualan_pengepul' AND c.entry_type = 'cash_in'), 0)::bigint AS total_cashbook_sales_idr,
    
    -- Nasabah withdrawals vs cashbook withdrawals
    COALESCE((SELECT SUM(w.amount_idr) FROM nasabah_withdrawals w WHERE w.bank_sampah_id = bs.id), 0)::bigint AS total_nasabah_withdrawals_idr,
    COALESCE((SELECT SUM(c.amount_idr) FROM cashbook_entries c WHERE c.bank_sampah_id = bs.id AND c.category = 'penarikan_nasabah' AND c.entry_type = 'cash_out'), 0)::bigint AS total_cashbook_withdrawals_idr,

    -- Total deposits recorded for nasabah
    COALESCE((SELECT SUM(d.total_amount_idr) FROM deposits d WHERE d.bank_sampah_id = bs.id), 0)::bigint AS total_deposits_amount_idr,

    -- Cash on hand
    COALESCE((SELECT SUM(CASE WHEN c.entry_type = 'cash_in' THEN c.amount_idr ELSE -c.amount_idr END) FROM cashbook_entries c WHERE c.bank_sampah_id = bs.id), 0)::bigint AS current_cash_on_hand_idr,

    -- Total outstanding nasabah savings liability = deposits - withdrawals
    (COALESCE((SELECT SUM(d.total_amount_idr) FROM deposits d WHERE d.bank_sampah_id = bs.id), 0) -
     COALESCE((SELECT SUM(w.amount_idr) FROM nasabah_withdrawals w WHERE w.bank_sampah_id = bs.id), 0))::bigint AS total_outstanding_nasabah_balance_idr
FROM bank_sampah bs
WHERE bs.id = $1;
