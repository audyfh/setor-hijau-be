-- name: GetSystemMetadata :one
SELECT key, value, updated_at
FROM _system_metadata
WHERE key = $1
LIMIT 1;

-- name: SetSystemMetadata :exec
INSERT INTO _system_metadata (key, value, updated_at)
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = CURRENT_TIMESTAMP;
