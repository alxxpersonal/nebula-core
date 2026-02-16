-- ---
-- SOURCE REFS + FILE URI + JOB SOURCE COLUMN REMOVAL
-- ---
-- Purpose:
-- - Replace vault-specific path naming with source_path.
-- - Introduce neutral external_refs linking model.
-- - Add files.uri as canonical location field (file_path remains temporary fallback).
-- - Backfill and drop legacy jobs.source_platform/source_url columns.

-- Rename vault-specific source columns.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'entities'
          AND column_name = 'vault_file_path'
    )
    AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'entities'
          AND column_name = 'source_path'
    ) THEN
        ALTER TABLE entities RENAME COLUMN vault_file_path TO source_path;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'context_items'
          AND column_name = 'vault_file_path'
    )
    AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'context_items'
          AND column_name = 'source_path'
    ) THEN
        ALTER TABLE context_items RENAME COLUMN vault_file_path TO source_path;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'protocols'
          AND column_name = 'vault_file_path'
    )
    AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'protocols'
          AND column_name = 'source_path'
    ) THEN
        ALTER TABLE protocols RENAME COLUMN vault_file_path TO source_path;
    END IF;
END $$;

-- New neutral external references table.
CREATE TABLE IF NOT EXISTS external_refs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type TEXT NOT NULL,
    node_id TEXT NOT NULL,
    system TEXT NOT NULL,
    external_id TEXT NOT NULL,
    url TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT external_refs_node_type_check
        CHECK (node_type IN ('entity', 'context', 'log', 'job', 'agent', 'file', 'protocol')),
    CONSTRAINT external_refs_metadata_is_object
        CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_external_refs_system_external_id
    ON external_refs (system, external_id);
CREATE INDEX IF NOT EXISTS idx_external_refs_node
    ON external_refs (node_type, node_id);
CREATE INDEX IF NOT EXISTS idx_external_refs_system
    ON external_refs (system);

DROP TRIGGER IF EXISTS update_external_refs_updated_at ON external_refs;
CREATE TRIGGER update_external_refs_updated_at
BEFORE UPDATE ON external_refs
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Backfill source_path-derived refs.
INSERT INTO external_refs (node_type, node_id, system, external_id, metadata)
SELECT 'entity', e.id::text, 'obsidian', e.source_path, '{}'::jsonb
FROM entities e
WHERE e.source_path IS NOT NULL AND btrim(e.source_path) <> ''
ON CONFLICT (system, external_id) DO NOTHING;

INSERT INTO external_refs (node_type, node_id, system, external_id, metadata)
SELECT 'context', c.id::text, 'obsidian', c.source_path, '{}'::jsonb
FROM context_items c
WHERE c.source_path IS NOT NULL AND btrim(c.source_path) <> ''
ON CONFLICT (system, external_id) DO NOTHING;

INSERT INTO external_refs (node_type, node_id, system, external_id, metadata)
SELECT 'protocol', p.id::text, 'obsidian', p.source_path, '{}'::jsonb
FROM protocols p
WHERE p.source_path IS NOT NULL AND btrim(p.source_path) <> ''
ON CONFLICT (system, external_id) DO NOTHING;

-- Backfill jobs legacy source links.
INSERT INTO external_refs (node_type, node_id, system, external_id, url, metadata)
SELECT
    'job',
    j.id,
    COALESCE(NULLIF(btrim(j.source_platform), ''), 'legacy-job-source'),
    j.source_url,
    j.source_url,
    '{}'::jsonb
FROM jobs j
WHERE j.source_url IS NOT NULL AND btrim(j.source_url) <> ''
ON CONFLICT (system, external_id) DO NOTHING;

INSERT INTO external_refs (node_type, node_id, system, external_id, metadata)
SELECT
    'job',
    j.id,
    'legacy-job-source',
    'job:' || j.id,
    jsonb_build_object('source_platform', j.source_platform)
FROM jobs j
WHERE (j.source_url IS NULL OR btrim(j.source_url) = '')
  AND j.source_platform IS NOT NULL
  AND btrim(j.source_platform) <> ''
ON CONFLICT (system, external_id) DO NOTHING;

-- Add canonical file URI and backfill from legacy file_path.
ALTER TABLE files ADD COLUMN IF NOT EXISTS uri TEXT;

UPDATE files
SET uri = CASE
    WHEN file_path IS NULL OR btrim(file_path) = '' THEN uri
    WHEN file_path LIKE '%://%' THEN file_path
    WHEN file_path LIKE '/%' THEN 'file://' || file_path
    ELSE 'path:' || file_path
END
WHERE uri IS NULL;

CREATE INDEX IF NOT EXISTS idx_files_uri ON files(uri);

-- Drop legacy job source columns after backfill.
ALTER TABLE jobs DROP COLUMN IF EXISTS source_platform;
ALTER TABLE jobs DROP COLUMN IF EXISTS source_url;
