-- ---
-- NEBULA INITIAL SEED DATA
-- ---
-- Run this after schema.sql to populate default statuses, privacy scopes,
-- and relationship types.

-- --- Statuses ---

INSERT INTO statuses (name, description, category) VALUES
    -- Active statuses
    ('active', 'Currently active and in use', 'active'),
    ('in-progress', 'Actively being worked on', 'active'),
    ('planning', 'In ideation/planning phase', 'active'),
    ('on-hold', 'Paused temporarily, will resume', 'active'),

    -- Archived statuses
    ('completed', 'Successfully finished', 'archived'),
    ('abandoned', 'Gave up, will not finish', 'archived'),
    ('replaced', 'Superseded by something better', 'archived'),
    ('deleted', 'Soft delete, can be restored', 'archived'),
    ('inactive', 'Not using, undecided if will return', 'archived')
ON CONFLICT (name) DO NOTHING;

-- --- Privacy Scopes ---

INSERT INTO privacy_scopes (name, description) VALUES
    ('public', 'Accessible to all agents'),
    ('personal', 'Private, requires explicit user approval'),
    ('vault-only', 'Only vault management agents'),
    ('uni', 'University agents access'),
    ('code', 'Code/development agents access'),
    ('health', 'Health/biological backend agents access'),
    ('social', 'Social/relationship management'),
    ('sensitive', 'High-security, requires approval for all ops'),
    ('blacklisted', 'Blocked entities, restricted access')
ON CONFLICT (name) DO NOTHING;

-- --- Relationship Types ---

INSERT INTO relationship_types (name, description, is_symmetric) VALUES
    -- Symmetric relationships
    ('friends-with', 'Friendship connection', TRUE),
    ('inner-circle', 'Closest friend tier', TRUE),
    ('dating', 'Romantic relationship', TRUE),
    ('roommates-with', 'Living together', TRUE),
    ('colleagues-with', 'Work together', TRUE),
    ('classmates-with', 'Study together in same class', TRUE),
    ('groupmates-with', 'Same project/study group', TRUE),
    ('partners-with', 'Business partners', TRUE),
    ('gym-partner', 'Work out together', TRUE),
    ('minecraft-friend', 'Gaming buddy (Minecraft)', TRUE),
    ('discord-friend', 'Discord connection', TRUE),
    ('acquaintance', 'Low frequency contact', TRUE),
    ('confidant', 'Deep talk / trusted person', TRUE),
    ('related-to', 'Related knowledge items or entities', TRUE),

    -- Asymmetric relationships (people/projects)
    ('works-on', 'Person working on project', FALSE),
    ('teaches', 'Professor teaching student/course', FALSE),
    ('manages', 'Manager managing person/project', FALSE),
    ('owns', 'Ownership relationship', FALSE),
    ('founded', 'Founder of organization', FALSE),
    ('contributes-to', 'Contributing to project', FALSE),
    ('mentors', 'Mentor-mentee relationship', FALSE),
    ('reports-to', 'Reporting structure', FALSE),
    ('depends-on', 'Dependency relationship (projects/tools)', FALSE),
    ('introduced-by', 'Who brought who into life', FALSE),
    ('former-student', 'Was taught by (professor -> student)', FALSE),
    ('ex-fling', 'Past romantic connection', FALSE),
    ('blacklisted', 'Blocked/cut off person', FALSE),
    ('moderator-of', 'Moderation role in community', FALSE),

    -- Knowledge relationships
    ('about', 'Knowledge item about entity', FALSE),
    ('mentions', 'Knowledge item mentions entity', FALSE),
    ('created-by', 'Knowledge/content created by person', FALSE),

    -- Log relationships
    ('logged-by', 'Log entry created by person', FALSE),
    ('at-location', 'Log entry at location entity', FALSE),
    ('with-person', 'Log entry with person entity', FALSE),

    -- Job relationships
    ('assigned-to', 'Job assigned to person', FALSE),
    ('handled-by', 'Job handled by agent', FALSE),
    ('has-attachment', 'Job has file attachment', FALSE),
    ('blocks', 'Job blocks another job (dependencies)', FALSE),

    -- Agent relationships
    ('manages-agent', 'Person manages agent', FALSE),

    -- File relationships
    ('has-file', 'Entity/knowledge/job has file', FALSE),
    ('profile-pic', 'Entity has profile picture', FALSE),

    -- Protocol relationships
    ('applies-to', 'Protocol applies to entity/system/table', FALSE),
    ('supersedes', 'Protocol supersedes another protocol', FALSE),
    ('references', 'Protocol references another protocol', FALSE)
ON CONFLICT (name) DO NOTHING;
