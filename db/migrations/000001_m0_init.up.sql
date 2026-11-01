-- 000001_m0_init.up.sql
-- Baseline schema initialization for Setor Hijau

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS _system_metadata (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO _system_metadata (key, value)
VALUES ('m0_scaffolding', 'initialized')
ON CONFLICT (key) DO NOTHING;
