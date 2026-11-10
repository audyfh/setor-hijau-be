-- 000006_m5_cashbook_reports.down.sql
-- Rollback Milestone 5 tables

DROP TABLE IF EXISTS cashbook_entries CASCADE;
DROP TABLE IF EXISTS nasabah_withdrawals CASCADE;
DROP TABLE IF EXISTS sales_to_pengepul_lines CASCADE;
DROP TABLE IF EXISTS sales_to_pengepul CASCADE;
