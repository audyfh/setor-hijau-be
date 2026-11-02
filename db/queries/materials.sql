-- name: CreateMaterialType :one
INSERT INTO material_types (
    bank_sampah_id,
    name,
    category,
    unit,
    description,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateMaterialType :one
UPDATE material_types
SET name = $1,
    category = $2,
    unit = $3,
    description = $4,
    is_active = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $6 AND bank_sampah_id = $7
RETURNING *;

-- name: GetMaterialTypeByID :one
SELECT 
    m.id,
    m.bank_sampah_id,
    m.name,
    m.category,
    m.unit,
    m.description,
    m.is_active,
    m.created_at,
    m.updated_at,
    COALESCE(p.id, '00000000-0000-0000-0000-000000000000'::uuid) as price_id,
    COALESCE(p.price_per_kg_idr, 0) as price_per_kg_idr,
    p.effective_from as price_effective_from
FROM material_types m
LEFT JOIN material_prices p ON m.id = p.material_type_id AND p.effective_to IS NULL
WHERE m.id = $1 AND m.bank_sampah_id = $2
LIMIT 1;

-- name: ListMaterialTypes :many
SELECT 
    m.id,
    m.bank_sampah_id,
    m.name,
    m.category,
    m.unit,
    m.description,
    m.is_active,
    m.created_at,
    m.updated_at,
    COALESCE(p.id, '00000000-0000-0000-0000-000000000000'::uuid) as price_id,
    COALESCE(p.price_per_kg_idr, 0) as price_per_kg_idr,
    p.effective_from as price_effective_from
FROM material_types m
LEFT JOIN material_prices p ON m.id = p.material_type_id AND p.effective_to IS NULL
WHERE m.bank_sampah_id = $1
  AND ($2::boolean IS FALSE OR m.is_active = TRUE)
ORDER BY m.name ASC;

-- name: CreateMaterialPrice :one
INSERT INTO material_prices (
    material_type_id,
    bank_sampah_id,
    price_per_kg_idr,
    effective_from,
    effective_to,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: CloseActiveMaterialPrice :exec
UPDATE material_prices
SET effective_to = $1
WHERE material_type_id = $2 AND bank_sampah_id = $3 AND effective_to IS NULL;

-- name: GetActivePriceForMaterial :one
SELECT * FROM material_prices
WHERE material_type_id = $1 AND bank_sampah_id = $2 AND effective_to IS NULL
LIMIT 1;

-- name: ListMaterialPricesHistory :many
SELECT * FROM material_prices
WHERE material_type_id = $1 AND bank_sampah_id = $2
ORDER BY effective_from DESC;
