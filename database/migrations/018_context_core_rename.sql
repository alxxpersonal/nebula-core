-- ---
-- CONTEXT CORE RENAME
-- ---
-- Purpose:
-- - Replace knowledge-specific naming with context naming.
-- - Migrate polymorphic node types from knowledge -> context.
-- - Keep migration idempotent for repeated test schema builds.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'knowledge_items'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'context_items'
    ) THEN
        ALTER TABLE knowledge_items RENAME TO context_items;
    END IF;
END $$;

-- Rename well-known triggers if they still use knowledge naming.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_knowledge_items_updated_at') THEN
        ALTER TRIGGER update_knowledge_items_updated_at ON context_items RENAME TO update_context_items_updated_at;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'cascade_knowledge_status_trigger') THEN
        ALTER TRIGGER cascade_knowledge_status_trigger ON context_items RENAME TO cascade_context_status_trigger;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'audit_knowledge_items_trigger') THEN
        ALTER TRIGGER audit_knowledge_items_trigger ON context_items RENAME TO audit_context_items_trigger;
    END IF;
END $$;

-- Rename legacy constraints/indexes when present.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'knowledge_items_tags_limit') THEN
        ALTER TABLE context_items RENAME CONSTRAINT knowledge_items_tags_limit TO context_items_tags_limit;
    END IF;
END $$;

ALTER INDEX IF EXISTS idx_knowledge_status RENAME TO idx_context_status;
ALTER INDEX IF EXISTS idx_knowledge_privacy RENAME TO idx_context_privacy;
ALTER INDEX IF EXISTS idx_knowledge_tags RENAME TO idx_context_tags;
ALTER INDEX IF EXISTS idx_knowledge_metadata RENAME TO idx_context_metadata;
ALTER INDEX IF EXISTS idx_knowledge_source_type RENAME TO idx_context_source_type;
ALTER INDEX IF EXISTS idx_knowledge_search RENAME TO idx_context_search;

-- Update existing data to canonical context naming.
UPDATE relationships
SET source_type = 'context'
WHERE source_type = 'knowledge';

UPDATE relationships
SET target_type = 'context'
WHERE target_type = 'knowledge';

UPDATE semantic_search
SET source_type = 'context'
WHERE source_type = 'knowledge';

UPDATE audit_log
SET table_name = 'context_items'
WHERE table_name = 'knowledge_items';

-- Rewrite pending/processed approval action names and payload keys.
UPDATE approval_requests
SET request_type = 'create_context'
WHERE request_type = 'create_knowledge';

UPDATE approval_requests
SET request_type = 'update_context'
WHERE request_type = 'update_knowledge';

UPDATE approval_requests
SET request_type = 'bulk_import_context'
WHERE request_type = 'bulk_import_knowledge';

UPDATE approval_requests
SET request_type = 'link_context_to_entity'
WHERE request_type = 'link_knowledge_to_entity';

UPDATE approval_requests
SET change_details = jsonb_set(
        change_details - 'knowledge_id',
        '{context_id}',
        change_details->'knowledge_id',
        true
    )
WHERE change_details ? 'knowledge_id';

-- Recreate polymorphic validation function with context node type.
CREATE OR REPLACE FUNCTION validate_relationship_references()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.source_type = 'entity' THEN
        IF NOT EXISTS (SELECT 1 FROM entities WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source entity % does not exist', NEW.source_id;
        END IF;
    ELSIF NEW.source_type = 'context' THEN
        IF NOT EXISTS (SELECT 1 FROM context_items WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source context_item % does not exist', NEW.source_id;
        END IF;
    ELSIF NEW.source_type = 'log' THEN
        IF NOT EXISTS (SELECT 1 FROM logs WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source log % does not exist', NEW.source_id;
        END IF;
    ELSIF NEW.source_type = 'job' THEN
        IF NOT EXISTS (SELECT 1 FROM jobs WHERE id = NEW.source_id) THEN
            RAISE EXCEPTION 'source job % does not exist', NEW.source_id;
        END IF;
    ELSIF NEW.source_type = 'agent' THEN
        IF NOT EXISTS (SELECT 1 FROM agents WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source agent % does not exist', NEW.source_id;
        END IF;
    ELSIF NEW.source_type = 'file' THEN
        IF NOT EXISTS (SELECT 1 FROM files WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source file % does not exist', NEW.source_id;
        END IF;
    ELSIF NEW.source_type = 'protocol' THEN
        IF NOT EXISTS (SELECT 1 FROM protocols WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source protocol % does not exist', NEW.source_id;
        END IF;
    END IF;

    IF NEW.target_type = 'entity' THEN
        IF NOT EXISTS (SELECT 1 FROM entities WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target entity % does not exist', NEW.target_id;
        END IF;
    ELSIF NEW.target_type = 'context' THEN
        IF NOT EXISTS (SELECT 1 FROM context_items WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target context_item % does not exist', NEW.target_id;
        END IF;
    ELSIF NEW.target_type = 'log' THEN
        IF NOT EXISTS (SELECT 1 FROM logs WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target log % does not exist', NEW.target_id;
        END IF;
    ELSIF NEW.target_type = 'job' THEN
        IF NOT EXISTS (SELECT 1 FROM jobs WHERE id = NEW.target_id) THEN
            RAISE EXCEPTION 'target job % does not exist', NEW.target_id;
        END IF;
    ELSIF NEW.target_type = 'agent' THEN
        IF NOT EXISTS (SELECT 1 FROM agents WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target agent % does not exist', NEW.target_id;
        END IF;
    ELSIF NEW.target_type = 'file' THEN
        IF NOT EXISTS (SELECT 1 FROM files WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target file % does not exist', NEW.target_id;
        END IF;
    ELSIF NEW.target_type = 'protocol' THEN
        IF NOT EXISTS (SELECT 1 FROM protocols WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target protocol % does not exist', NEW.target_id;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Recreate cascade mapping function with context table/type naming.
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
        WHEN source_table = 'context_items' THEN 'context'
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

-- Update check constraints to canonical node types.
ALTER TABLE relationships DROP CONSTRAINT IF EXISTS relationships_source_type_check;
ALTER TABLE relationships DROP CONSTRAINT IF EXISTS relationships_target_type_check;
ALTER TABLE relationships
    ADD CONSTRAINT relationships_source_type_check
    CHECK (source_type IN ('entity', 'context', 'log', 'job', 'agent', 'file', 'protocol'));
ALTER TABLE relationships
    ADD CONSTRAINT relationships_target_type_check
    CHECK (target_type IN ('entity', 'context', 'log', 'job', 'agent', 'file', 'protocol'));

ALTER TABLE semantic_search DROP CONSTRAINT IF EXISTS semantic_search_source_type_check;
ALTER TABLE semantic_search
    ADD CONSTRAINT semantic_search_source_type_check
    CHECK (source_type IN ('entity', 'context', 'log', 'job', 'agent', 'file', 'protocol'));
