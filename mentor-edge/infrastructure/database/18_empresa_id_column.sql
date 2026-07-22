-- Migration: add empresa_id column to config.line_config
-- Idempotent: IF NOT EXISTS

ALTER TABLE config.line_config
    ADD COLUMN IF NOT EXISTS empresa_id INTEGER NOT NULL DEFAULT 0;
