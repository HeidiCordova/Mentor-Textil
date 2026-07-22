-- Migration: añade empresa_nombre a sync_plantas
-- Necesario para mostrar el selector de Empresa en la tablet en modo Edge
ALTER TABLE sync_plantas
    ADD COLUMN IF NOT EXISTS empresa_nombre VARCHAR(200) NOT NULL DEFAULT '';
