-- ---
-- ENTERPRISE DEFAULTS (MINIMAL ACTIVE TAXONOMY)
-- ---
-- Purpose:
-- - Ship a minimal, product-neutral taxonomy for fresh installs.
-- - Deactivate legacy/non-neutral seeded taxonomy (without deleting rows).
-- - Mark the minimal enterprise set as builtin.
--
-- Notes:
-- - Users can later activate additional scopes/types via taxonomy management.
-- - This migration is idempotent.

-- --- Privacy Scopes ---
INSERT INTO privacy_scopes (name, description, is_builtin, is_active)
VALUES
    ('public', 'Accessible to all agents', TRUE, TRUE),
    ('private', 'Private data visible only to explicitly permitted actors', TRUE, TRUE),
    ('sensitive', 'High-risk data requiring stricter controls', TRUE, TRUE),
    ('admin', 'Administrative and governance operations', TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    is_builtin = TRUE,
    is_active = TRUE;

UPDATE privacy_scopes
SET is_active = FALSE
WHERE name NOT IN ('public', 'private', 'sensitive', 'admin');

UPDATE privacy_scopes
SET is_builtin = FALSE
WHERE name NOT IN ('public', 'private', 'sensitive', 'admin');

-- --- Entity Types ---
INSERT INTO entity_types (name, description, is_builtin, is_active)
VALUES
    ('person', 'A human individual', TRUE, TRUE),
    ('organization', 'A company, team, or institution', TRUE, TRUE),
    ('project', 'A product or initiative', TRUE, TRUE),
    ('tool', 'Software, model, or utility', TRUE, TRUE),
    ('document', 'A document, note, or specification', TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    is_builtin = TRUE,
    is_active = TRUE;

UPDATE entity_types
SET is_active = FALSE
WHERE name NOT IN ('person', 'organization', 'project', 'tool', 'document');

UPDATE entity_types
SET is_builtin = FALSE
WHERE name NOT IN ('person', 'organization', 'project', 'tool', 'document');

-- --- Relationship Types ---
INSERT INTO relationship_types (name, description, is_symmetric, is_builtin, is_active)
VALUES
    ('related-to', 'General relationship between records', TRUE, TRUE, TRUE),
    ('depends-on', 'Source depends on target', FALSE, TRUE, TRUE),
    ('references', 'Source references target', FALSE, TRUE, TRUE),
    ('blocks', 'Source blocks target', FALSE, TRUE, TRUE),
    ('assigned-to', 'Source is assigned to target', FALSE, TRUE, TRUE),
    ('owns', 'Source owns target', FALSE, TRUE, TRUE),
    ('about', 'Source is about target', FALSE, TRUE, TRUE),
    ('mentions', 'Source mentions target', FALSE, TRUE, TRUE),
    ('created-by', 'Source created by target', FALSE, TRUE, TRUE),
    ('has-file', 'Source has file attachment target', FALSE, TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    is_symmetric = EXCLUDED.is_symmetric,
    is_builtin = TRUE,
    is_active = TRUE;

UPDATE relationship_types
SET is_active = FALSE
WHERE name NOT IN (
    'related-to',
    'depends-on',
    'references',
    'blocks',
    'assigned-to',
    'owns',
    'about',
    'mentions',
    'created-by',
    'has-file'
);

UPDATE relationship_types
SET is_builtin = FALSE
WHERE name NOT IN (
    'related-to',
    'depends-on',
    'references',
    'blocks',
    'assigned-to',
    'owns',
    'about',
    'mentions',
    'created-by',
    'has-file'
);

-- --- Log Types ---
INSERT INTO log_types (name, description, value_schema, is_builtin, is_active)
VALUES
    ('event', 'Generic event log', '{"type":"object"}'::jsonb, TRUE, TRUE),
    ('note', 'Generic textual note log', '{"type":"object"}'::jsonb, TRUE, TRUE),
    ('metric', 'Generic metric/value log', '{"type":"object"}'::jsonb, TRUE, TRUE)
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    value_schema = EXCLUDED.value_schema,
    is_builtin = TRUE,
    is_active = TRUE;

UPDATE log_types
SET is_active = FALSE
WHERE name NOT IN ('event', 'note', 'metric');

UPDATE log_types
SET is_builtin = FALSE
WHERE name NOT IN ('event', 'note', 'metric');
