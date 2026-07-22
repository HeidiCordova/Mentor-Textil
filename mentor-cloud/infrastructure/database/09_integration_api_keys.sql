CREATE SCHEMA IF NOT EXISTS integration;

CREATE TABLE IF NOT EXISTS integration.api_keys (
    id          SERIAL PRIMARY KEY,
    empresa_id  INT NOT NULL REFERENCES identity.empresas(id) ON DELETE CASCADE,
    nombre      VARCHAR(100) NOT NULL,
    key_prefix  VARCHAR(12) NOT NULL,
    key_hash    TEXT NOT NULL,
    scopes      TEXT[] NOT NULL DEFAULT ARRAY['oee:read','snapshots:read','paradas:read'],
    activo      BOOLEAN NOT NULL DEFAULT true,
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT now(),
    ultimo_uso  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS api_keys_prefix_idx ON integration.api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS api_keys_empresa_idx ON integration.api_keys(empresa_id);
