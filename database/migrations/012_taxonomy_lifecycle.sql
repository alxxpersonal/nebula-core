-- ---
-- TAXONOMY LIFECYCLE + NEUTRAL DEFAULTS
-- ---
-- Adds lifecycle controls for taxonomy tables so defaults remain flexible:
-- - is_builtin: seeded system value
-- - is_active: archive without hard delete
-- - metadata: optional extension field

-- privacy_scopes
ALTER TABLE privacy_scopes
    ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'privacy_scopes_metadata_is_object'
    ) THEN
        ALTER TABLE privacy_scopes
            ADD CONSTRAINT privacy_scopes_metadata_is_object
            CHECK (jsonb_typeof(metadata) = 'object');
    END IF;
END $$;

-- entity_types
ALTER TABLE entity_types
    ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'entity_types_metadata_is_object'
    ) THEN
        ALTER TABLE entity_types
            ADD CONSTRAINT entity_types_metadata_is_object
            CHECK (jsonb_typeof(metadata) = 'object');
    END IF;
END $$;

-- relationship_types
ALTER TABLE relationship_types
    ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'relationship_types_metadata_is_object'
    ) THEN
        ALTER TABLE relationship_types
            ADD CONSTRAINT relationship_types_metadata_is_object
            CHECK (jsonb_typeof(metadata) = 'object');
    END IF;
END $$;

-- log_types
ALTER TABLE log_types
    ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- mark seeded values as builtin
UPDATE privacy_scopes
SET is_builtin = TRUE
WHERE name IN (
    'public', 'personal', 'vault-only', 'uni', 'code', 'health',
    'social', 'sensitive', 'blacklisted', 'work'
);

UPDATE entity_types
SET is_builtin = TRUE
WHERE name IN (
    'person', 'project', 'tool', 'organization', 'course',
    'idea', 'framework', 'paper', 'university'
);

UPDATE relationship_types
SET is_builtin = TRUE
WHERE name IN (
    'friends-with', 'inner-circle', 'dating', 'roommates-with', 'colleagues-with',
    'classmates-with', 'groupmates-with', 'partners-with', 'gym-partner',
    'minecraft-friend', 'discord-friend', 'acquaintance', 'confidant', 'related-to',
    'works-on', 'teaches', 'manages', 'owns', 'founded', 'contributes-to', 'mentors',
    'reports-to', 'depends-on', 'introduced-by', 'former-student', 'ex-fling',
    'blacklisted', 'moderator-of', 'about', 'mentions', 'created-by', 'logged-by',
    'at-location', 'with-person', 'assigned-to', 'handled-by', 'has-attachment',
    'blocks', 'manages-agent', 'has-file', 'profile-pic', 'applies-to', 'supersedes',
    'references'
);

UPDATE log_types
SET is_builtin = TRUE
WHERE name IN (
    'gym-session', 'weight', 'mood', 'sleep', 'calories',
    'workout', 'meditation', 'reading', 'water-intake'
);

-- add neutral defaults if missing
INSERT INTO privacy_scopes (name, description, is_builtin, is_active)
VALUES
    ('private', 'Private data visible only to explicitly permitted actors', TRUE, TRUE),
    ('admin', 'Administrative and governance operations', TRUE, TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO entity_types (name, description, is_builtin, is_active)
VALUES
    ('document', 'Generic document or note entity', TRUE, TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO relationship_types (name, description, is_symmetric, is_builtin, is_active)
VALUES
    ('references', 'Source references target', FALSE, TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET is_builtin = TRUE,
    is_active = TRUE;

INSERT INTO log_types (name, description, value_schema, is_builtin, is_active)
VALUES
    ('event', 'Generic event log', '{"type":"object"}'::jsonb, TRUE, TRUE),
    ('note', 'Generic textual note log', '{"type":"object"}'::jsonb, TRUE, TRUE),
    ('metric', 'Generic metric/value log', '{"type":"object"}'::jsonb, TRUE, TRUE)
ON CONFLICT (name) DO NOTHING;

-- indexes for active filters
CREATE INDEX IF NOT EXISTS idx_privacy_scopes_active_name
    ON privacy_scopes (is_active, name);
CREATE INDEX IF NOT EXISTS idx_entity_types_active_name
    ON entity_types (is_active, name);
CREATE INDEX IF NOT EXISTS idx_relationship_types_active_name
    ON relationship_types (is_active, name);
CREATE INDEX IF NOT EXISTS idx_log_types_active_name
    ON log_types (is_active, name);
