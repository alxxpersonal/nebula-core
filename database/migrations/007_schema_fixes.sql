-- ---
-- SCHEMA FIXES
-- ---
-- Bug 1 + 4: fix sync_symmetric_relationships()
-- - COALESCE(NEW.type_id, OLD.type_id) so DELETE branch works
-- - recursion guard for cascade trigger interaction
CREATE OR REPLACE FUNCTION sync_symmetric_relationships()
RETURNS TRIGGER AS $$
DECLARE
    is_sym BOOLEAN;
BEGIN
    SELECT is_symmetric INTO is_sym
    FROM relationship_types
    WHERE id = COALESCE(NEW.type_id, OLD.type_id);

    IF is_sym THEN
        IF TG_OP = 'INSERT' THEN
            INSERT INTO relationships (
                source_type, source_id,
                target_type, target_id,
                type_id, properties, embedding
            )
            VALUES (
                NEW.target_type, NEW.target_id,
                NEW.source_type, NEW.source_id,
                NEW.type_id, NEW.properties, NEW.embedding
            )
            ON CONFLICT (source_type, source_id, target_type, target_id, type_id) DO NOTHING;

        ELSIF TG_OP = 'UPDATE' THEN
            IF current_setting('nebula.cascade_in_progress', true) = 'true' THEN
                RETURN NEW;
            END IF;

            UPDATE relationships
            SET properties = NEW.properties,
                embedding = NEW.embedding,
                updated_at = NOW()
            WHERE source_type = NEW.target_type
              AND source_id = NEW.target_id
              AND target_type = NEW.source_type
              AND target_id = NEW.source_id
              AND type_id = NEW.type_id;

        ELSIF TG_OP = 'DELETE' THEN
            DELETE FROM relationships
            WHERE source_type = OLD.target_type
              AND source_id = OLD.target_id
              AND target_type = OLD.source_type
              AND target_id = OLD.source_id
              AND type_id = OLD.type_id;
            RETURN OLD;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Bug 3: add missing columns to jobs table
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS status_reason TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS status_changed_at TIMESTAMPTZ;

-- Bug 4: fix cascade_status_to_relationships() with recursion guard
CREATE OR REPLACE FUNCTION cascade_status_to_relationships()
RETURNS TRIGGER AS $$
DECLARE
    source_table TEXT;
    status_category TEXT;
    type_name TEXT;
BEGIN
    source_table := TG_TABLE_NAME;

    type_name := CASE
        WHEN source_table = 'entities' THEN 'entity'
        WHEN source_table = 'knowledge_items' THEN 'knowledge'
        WHEN source_table = 'logs' THEN 'log'
        WHEN source_table = 'jobs' THEN 'job'
        WHEN source_table = 'agents' THEN 'agent'
        WHEN source_table = 'files' THEN 'file'
        WHEN source_table = 'protocols' THEN 'protocol'
    END;

    SELECT category INTO status_category
    FROM statuses
    WHERE id = NEW.status_id;

    IF status_category = 'archived' THEN
        PERFORM set_config('nebula.cascade_in_progress', 'true', true);

        UPDATE relationships
        SET status_id = NEW.status_id,
            status_changed_at = NOW()
        WHERE (
            (source_type = type_name AND source_id = NEW.id::text)
            OR
            (target_type = type_name AND target_id = NEW.id::text)
        );

        PERFORM set_config('nebula.cascade_in_progress', 'false', true);
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
