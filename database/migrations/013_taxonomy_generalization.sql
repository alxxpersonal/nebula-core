-- ---
-- TAXONOMY GENERALIZATION
-- ---
-- Purpose:
-- - Keep built-in taxonomy minimal and product-neutral.
-- - Preserve legacy rows as user-manageable (non-builtin).
-- - Enforce case-insensitive uniqueness for taxonomy names.

-- ensure neutral defaults exist (idempotent)
INSERT INTO privacy_scopes (name, description, is_builtin, is_active)
VALUES
    ('public', 'Accessible to all agents', TRUE, TRUE),
    ('personal', 'Private data for a user or small trusted group', TRUE, TRUE),
    ('sensitive', 'High-risk data requiring stricter controls', TRUE, TRUE),
    ('private', 'Private data visible only to explicitly permitted actors', TRUE, TRUE),
    ('admin', 'Administrative and governance operations', TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    is_active = TRUE;

INSERT INTO entity_types (name, description, is_builtin, is_active)
VALUES
    ('person', 'A human individual', TRUE, TRUE),
    ('organization', 'A company, team, or institution', TRUE, TRUE),
    ('project', 'A product or initiative', TRUE, TRUE),
    ('tool', 'Software, model, or utility', TRUE, TRUE),
    ('document', 'A document, note, or specification', TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    is_active = TRUE;

INSERT INTO relationship_types (name, description, is_symmetric, is_builtin, is_active)
VALUES
    ('related-to', 'General relationship between records', TRUE, TRUE, TRUE),
    ('depends-on', 'Source depends on target', FALSE, TRUE, TRUE),
    ('references', 'Source references target', FALSE, TRUE, TRUE),
    ('blocks', 'Source blocks target', FALSE, TRUE, TRUE),
    ('assigned-to', 'Source is assigned to target', FALSE, TRUE, TRUE),
    ('owns', 'Source owns target', FALSE, TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    is_symmetric = EXCLUDED.is_symmetric,
    is_active = TRUE;

INSERT INTO log_types (name, description, value_schema, is_builtin, is_active)
VALUES
    ('event', 'Generic event log', '{"type":"object"}'::jsonb, TRUE, TRUE),
    ('note', 'Generic textual note log', '{"type":"object"}'::jsonb, TRUE, TRUE),
    ('metric', 'Generic metric/value log', '{"type":"object"}'::jsonb, TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    value_schema = EXCLUDED.value_schema,
    is_active = TRUE;

-- legacy rows remain available, but only the neutral set is protected as builtin
UPDATE privacy_scopes SET is_builtin = FALSE;
UPDATE privacy_scopes
SET is_builtin = TRUE
WHERE name IN ('public', 'personal', 'sensitive', 'private', 'admin');

UPDATE entity_types SET is_builtin = FALSE;
UPDATE entity_types
SET is_builtin = TRUE
WHERE name IN ('person', 'organization', 'project', 'tool', 'document');

UPDATE relationship_types SET is_builtin = FALSE;
UPDATE relationship_types
SET is_builtin = TRUE
WHERE name IN ('related-to', 'depends-on', 'references', 'blocks', 'assigned-to', 'owns');

UPDATE log_types SET is_builtin = FALSE;
UPDATE log_types
SET is_builtin = TRUE
WHERE name IN ('event', 'note', 'metric');

-- case-insensitive uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS uq_privacy_scopes_name_ci
    ON privacy_scopes (LOWER(name));
CREATE UNIQUE INDEX IF NOT EXISTS uq_entity_types_name_ci
    ON entity_types (LOWER(name));
CREATE UNIQUE INDEX IF NOT EXISTS uq_relationship_types_name_ci
    ON relationship_types (LOWER(name));
CREATE UNIQUE INDEX IF NOT EXISTS uq_log_types_name_ci
    ON log_types (LOWER(name));
