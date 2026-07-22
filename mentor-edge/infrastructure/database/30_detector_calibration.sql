-- Persistencia de la referencia HSV del detector de vision.
-- Idempotente: puede ejecutarse en instalaciones nuevas y Jetsons existentes.

CREATE TABLE IF NOT EXISTS public.detector_calibration (
    id                       BIGSERIAL PRIMARY KEY,
    line_id                  VARCHAR(64)  NOT NULL,
    device_id                VARCHAR(64)  NOT NULL,
    calibrated_lot_code      VARCHAR(128),
    config_fingerprint       CHAR(64)     NOT NULL,
    algorithm_version        VARCHAR(64)  NOT NULL,
    algorithm_parameters     JSONB        NOT NULL DEFAULT '{}'::JSONB,

    histogram_data           BYTEA        NOT NULL,
    histogram_format         VARCHAR(16)  NOT NULL DEFAULT 'npy-v1',
    histogram_dtype          VARCHAR(32)  NOT NULL,
    histogram_shape          INTEGER[]    NOT NULL,

    samples_used             SMALLINT     NOT NULL,
    quality_score            DOUBLE PRECISION NOT NULL,
    calibrated_at            TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at               TIMESTAMPTZ  NOT NULL,
    active                   BOOLEAN      NOT NULL DEFAULT TRUE,
    deactivated_at           TIMESTAMPTZ,
    deactivation_reason      VARCHAR(64),

    reference_thumbnail_jpeg BYTEA,
    metadata                 JSONB        NOT NULL DEFAULT '{}'::JSONB,

    CONSTRAINT chk_detector_calibration_samples
        CHECK (samples_used > 0),
    CONSTRAINT chk_detector_calibration_quality
        CHECK (quality_score >= 0.0 AND quality_score <= 1.0),
    CONSTRAINT chk_detector_calibration_histogram
        CHECK (
            octet_length(histogram_data) > 0
            AND cardinality(histogram_shape) > 0
        ),
    CONSTRAINT chk_detector_calibration_expiration
        CHECK (expires_at > calibrated_at),
    CONSTRAINT chk_detector_calibration_thumbnail_size
        CHECK (
            reference_thumbnail_jpeg IS NULL
            OR octet_length(reference_thumbnail_jpeg) <= 262144
        ),
    CONSTRAINT chk_detector_calibration_deactivation
        CHECK (
            (active AND deactivated_at IS NULL AND deactivation_reason IS NULL)
            OR
            (NOT active AND deactivated_at IS NOT NULL AND deactivation_reason IS NOT NULL)
        )
);

-- Una linea tiene una sola referencia operativa. device_id se conserva como
-- trazabilidad, pero un reemplazo del Jetson no invalida por si solo la referencia.
CREATE UNIQUE INDEX IF NOT EXISTS uq_detector_calibration_one_active
    ON public.detector_calibration (line_id)
    WHERE active = TRUE;

CREATE INDEX IF NOT EXISTS idx_detector_calibration_lookup
    ON public.detector_calibration (
        line_id,
        config_fingerprint,
        algorithm_version,
        calibrated_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_detector_calibration_expiration
    ON public.detector_calibration (expires_at)
    WHERE active = TRUE;

COMMENT ON TABLE public.detector_calibration IS
    'Referencias HSV persistidas y su historial para el detector de vision edge.';
COMMENT ON COLUMN public.detector_calibration.config_fingerprint IS
    'SHA-256 de ROI, origen de camara y parametros que afectan el histograma.';
COMMENT ON COLUMN public.detector_calibration.reference_thumbnail_jpeg IS
    'Miniatura opcional para auditoria; no participa en la deteccion.';
