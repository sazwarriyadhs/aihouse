-- Enforce that every artifact references one consistent
-- project -> design -> immutable design version chain.

ALTER TABLE core.designs
    ADD CONSTRAINT uq_designs_id_project
    UNIQUE (id, project_id);

ALTER TABLE core.design_versions
    ADD CONSTRAINT uq_design_versions_id_design
    UNIQUE (id, design_id);

CREATE TABLE core.cad_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    job_id UUID NOT NULL,
    project_id UUID NOT NULL,
    design_id UUID NOT NULL,
    version_id UUID NOT NULL,

    dimension VARCHAR(2) NOT NULL
        CHECK (dimension IN ('2d', '3d')),

    format VARCHAR(10) NOT NULL
        CHECK (format IN ('dxf', 'fcstd', 'step', 'stl')),

    storage_key TEXT NOT NULL UNIQUE
        CHECK (
            storage_key <> ''
            AND storage_key !~ '(^/|(^|/)\.\.(/|$)|\\)'
        ),

    checksum_sha256 VARCHAR(64) NOT NULL
        CHECK (checksum_sha256 ~ '^[0-9a-f]{64}$'),

    size_bytes BIGINT NOT NULL
        CHECK (size_bytes > 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cad_artifacts_design_project
        FOREIGN KEY (design_id, project_id)
        REFERENCES core.designs(id, project_id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_cad_artifacts_version_design
        FOREIGN KEY (version_id, design_id)
        REFERENCES core.design_versions(id, design_id)
        ON DELETE RESTRICT,

    CONSTRAINT cad_artifacts_format_dimension_check
        CHECK (
            (dimension = '2d' AND format IN ('dxf', 'fcstd'))
            OR
            (dimension = '3d' AND format IN ('fcstd', 'step', 'stl'))
        ),

    CONSTRAINT cad_artifacts_job_output_unique
        UNIQUE (job_id, dimension, format)
);

CREATE INDEX idx_cad_artifacts_project_id
    ON core.cad_artifacts(project_id);

CREATE INDEX idx_cad_artifacts_design_version
    ON core.cad_artifacts(design_id, version_id);

CREATE INDEX idx_cad_artifacts_job_id
    ON core.cad_artifacts(job_id);

GRANT SELECT, INSERT
    ON core.cad_artifacts
    TO ahg_runtime;