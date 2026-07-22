-- Tabla de medidores dinamicos — sin limite, sin rebuild de Docker.
-- energy.snapshots ya discrimina por meter_id: no se necesita una tabla por medidor.

CREATE TABLE IF NOT EXISTS energy.meters (
    id         SERIAL PRIMARY KEY,
    meter_id   VARCHAR(100) UNIQUE NOT NULL,
    unit_id    INT NOT NULL CHECK (unit_id BETWEEN 1 AND 247),
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Migrar medidores existentes desde energy.config (patron meter_id_N / meter_unit_id_N)
DO $$
BEGIN
    INSERT INTO energy.meters (meter_id, unit_id)
    SELECT c1.value,
           COALESCE(NULLIF(c2.value, ''), '1')::INT
    FROM   energy.config c1
    JOIN   energy.config c2
             ON c2.key = replace(c1.key, 'meter_id_', 'meter_unit_id_')
    WHERE  c1.key ~ '^meter_id_\d+$'
      AND  c1.value != ''
    ON CONFLICT (meter_id) DO NOTHING;
EXCEPTION
    WHEN OTHERS THEN NULL;
END;
$$;
