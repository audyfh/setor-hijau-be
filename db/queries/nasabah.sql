-- name: CreateNasabahProfile :one
INSERT INTO nasabah_profiles (
    user_id,
    bank_sampah_id,
    qr_token_hash,
    pin_hash,
    rw,
    rt,
    address_notes,
    registered_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetNasabahProfileByUserID :one
SELECT * FROM nasabah_profiles
WHERE user_id = $1 AND bank_sampah_id = $2
LIMIT 1;

-- name: GetNasabahDetailByID :one
SELECT 
    u.id, 
    u.bank_sampah_id, 
    u.username, 
    u.email, 
    u.full_name, 
    u.role, 
    u.phone, 
    u.is_active, 
    u.created_at, 
    u.updated_at,
    np.qr_token_hash, 
    np.rw, 
    np.rt, 
    np.address_notes, 
    np.registered_by, 
    np.failed_pin_attempts, 
    np.locked_until,
    (np.pin_hash IS NOT NULL) AS has_pin
FROM users u
JOIN nasabah_profiles np ON u.id = np.user_id
WHERE u.id = $1 AND u.bank_sampah_id = $2
LIMIT 1;

-- name: GetNasabahByQRTokenHashInBankSampah :one
SELECT 
    u.id, 
    u.bank_sampah_id, 
    u.username, 
    u.email, 
    u.full_name, 
    u.role, 
    u.phone, 
    u.is_active, 
    u.created_at,
    np.rw, 
    np.rt, 
    np.address_notes, 
    (np.pin_hash IS NOT NULL) AS has_pin
FROM users u
JOIN nasabah_profiles np ON u.id = np.user_id
WHERE np.qr_token_hash = $1 AND u.bank_sampah_id = $2
LIMIT 1;

-- name: GetNasabahForQRLogin :one
SELECT 
    u.id, 
    u.bank_sampah_id, 
    u.username, 
    u.email, 
    u.full_name, 
    u.role, 
    u.phone, 
    u.is_active, 
    u.created_at,
    np.pin_hash, 
    np.failed_pin_attempts, 
    np.locked_until, 
    np.rw, 
    np.rt
FROM users u
JOIN nasabah_profiles np ON u.id = np.user_id
WHERE np.qr_token_hash = $1
LIMIT 1;

-- name: ListNasabahByBankSampah :many
SELECT 
    u.id, 
    u.bank_sampah_id, 
    u.username, 
    u.email, 
    u.full_name, 
    u.phone, 
    u.is_active, 
    u.created_at,
    np.rw, 
    np.rt, 
    (np.pin_hash IS NOT NULL) AS has_pin
FROM users u
JOIN nasabah_profiles np ON u.id = np.user_id
WHERE u.bank_sampah_id = $1 AND u.role = 'nasabah'
  AND ($2::text = '' OR u.full_name ILIKE '%' || $2 || '%' OR u.username ILIKE '%' || $2 || '%' OR (u.phone IS NOT NULL AND u.phone ILIKE '%' || $2 || '%'))
  AND ($3::text = '' OR np.rw = $3)
  AND ($4::text = '' OR np.rt = $4)
ORDER BY u.created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountNasabahByBankSampah :one
SELECT COUNT(*)
FROM users u
JOIN nasabah_profiles np ON u.id = np.user_id
WHERE u.bank_sampah_id = $1 AND u.role = 'nasabah'
  AND ($2::text = '' OR u.full_name ILIKE '%' || $2 || '%' OR u.username ILIKE '%' || $2 || '%' OR (u.phone IS NOT NULL AND u.phone ILIKE '%' || $2 || '%'))
  AND ($3::text = '' OR np.rw = $3)
  AND ($4::text = '' OR np.rt = $4);

-- name: UpdateNasabahUser :one
UPDATE users
SET full_name = $1, phone = $2, email = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $5 AND bank_sampah_id = $6
RETURNING *;

-- name: UpdateNasabahProfile :one
UPDATE nasabah_profiles
SET rw = $1, rt = $2, address_notes = $3, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $4 AND bank_sampah_id = $5
RETURNING *;

-- name: UpdateNasabahQRTokenHash :one
UPDATE nasabah_profiles
SET qr_token_hash = $1, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $2 AND bank_sampah_id = $3
RETURNING *;

-- name: UpdateNasabahPIN :one
UPDATE nasabah_profiles
SET pin_hash = $1, failed_pin_attempts = 0, locked_until = NULL, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $2 AND bank_sampah_id = $3
RETURNING *;

-- name: RecordFailedPINAttempt :one
UPDATE nasabah_profiles
SET failed_pin_attempts = failed_pin_attempts + 1, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1
RETURNING failed_pin_attempts;

-- name: LockNasabahPIN :exec
UPDATE nasabah_profiles
SET locked_until = $1, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $2;

-- name: ResetFailedPINAttempts :exec
UPDATE nasabah_profiles
SET failed_pin_attempts = 0, locked_until = NULL, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1;
