-- name: CreateBankSampah :one
INSERT INTO bank_sampah (
    name,
    code,
    address,
    rw,
    rt,
    kelurahan,
    kecamatan,
    kabupaten_kota,
    provinsi,
    contact_phone
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetBankSampahByID :one
SELECT * FROM bank_sampah
WHERE id = $1 LIMIT 1;

-- name: GetBankSampahByCode :one
SELECT * FROM bank_sampah
WHERE code = $1 LIMIT 1;
