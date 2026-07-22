-- Migración: añadir apellido y password_hash a shared.usuarios
-- Ejecutar en Jetsons existentes para sincronizar credenciales desde cloud
ALTER TABLE shared.usuarios
    ADD COLUMN IF NOT EXISTS apellido      VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS password_hash TEXT         NOT NULL DEFAULT '';
