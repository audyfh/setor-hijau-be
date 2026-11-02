-- name: CreateUser :one
INSERT INTO users (
    bank_sampah_id,
    username,
    email,
    password_hash,
    full_name,
    role,
    phone,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND bank_sampah_id = $2 LIMIT 1;

-- name: GetUserByIDGlobal :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetPengelolaByUsernameOrEmail :one
SELECT * FROM users
WHERE (username = $1 OR (email IS NOT NULL AND email = $1))
  AND role = 'pengelola'
LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 AND bank_sampah_id = $2 LIMIT 1;

-- name: GetUserByUsernameAndBankSampahCode :one
SELECT u.* FROM users u
JOIN bank_sampah b ON u.bank_sampah_id = b.id
WHERE u.username = $1 AND b.code = $2 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1 LIMIT 1;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetPasswordResetTokenByHash :one
SELECT * FROM password_reset_tokens
WHERE token_hash = $1 LIMIT 1;

-- name: MarkPasswordResetTokenUsed :one
UPDATE password_reset_tokens
SET used_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
