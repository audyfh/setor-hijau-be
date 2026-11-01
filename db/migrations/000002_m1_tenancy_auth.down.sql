-- 000002_m1_tenancy_auth.down.sql
-- Rollback Milestone 1: Tenancy and Authentication schema

DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS bank_sampah;
