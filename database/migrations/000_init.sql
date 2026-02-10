-- ---
-- NEBULA GRAPH DATABASE SCHEMA
-- ---

-- --- Extensions ---
CREATE EXTENSION IF NOT EXISTS vector;

-- --- Functions ---

-- ID Generator (YYYYQ#-NNNN format with collision avoidance)
CREATE OR REPLACE FUNCTION public.generate_job_id()
RETURNS TEXT
LANGUAGE plpgsql
AS $$
DECLARE
  alphabet TEXT := 'ABCDEFGHJKMNPQRSTUVWXYZ23456789';
  suffix TEXT := '';
  i INT;
  b INT;
  yr INT := EXTRACT(YEAR FROM NOW())::INT;
  q INT := ((EXTRACT(MONTH FROM NOW())::INT - 1) / 3 + 1)::INT;
  new_id TEXT;
  max_attempts INT := 10;
  attempt INT := 0;
BEGIN
  LOOP
    suffix := '';
    FOR i IN 1..4 LOOP
      b := get_byte(gen_random_bytes(1), 0);
      suffix := suffix || SUBSTR(alphabet, (b % LENGTH(alphabet)) + 1, 1);
    END LOOP;

    new_id := yr::TEXT || 'Q' || q::TEXT || '-' || suffix;

    -- Check if ID already exists
    IF NOT EXISTS (SELECT 1 FROM jobs WHERE id = new_id) THEN
      RETURN new_id;
    END IF;

    attempt := attempt + 1;
    IF attempt >= max_attempts THEN
      RAISE EXCEPTION 'Failed to generate unique job ID after % attempts', max_attempts;
    END IF;
  END LOOP;
END;
$$;

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Validate polymorphic relationship references
CREATE OR REPLACE FUNCTION validate_relationship_references()
RETURNS TRIGGER AS $$
BEGIN
    -- Validate source exists
    IF NEW.source_type = 'entity' THEN
        IF NOT EXISTS (SELECT 1 FROM entities WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source entity % does not exist', NEW.source_id;
        END IF;
    ELSIF NEW.source_type = 'knowledge' THEN
        IF NOT EXISTS (SELECT 1 FROM knowledge_items WHERE id::text = NEW.source_id) THEN
            RAISE EXCEPTION 'source knowledge_item % does not exist', NEW.source_id;
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

    -- Validate target exists
    IF NEW.target_type = 'entity' THEN
        IF NOT EXISTS (SELECT 1 FROM entities WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target entity % does not exist', NEW.target_id;
        END IF;
    ELSIF NEW.target_type = 'knowledge' THEN
        IF NOT EXISTS (SELECT 1 FROM knowledge_items WHERE id::text = NEW.target_id) THEN
            RAISE EXCEPTION 'target knowledge_item % does not exist', NEW.target_id;
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

-- Auto-sync symmetric relationships (polymorphic version)
CREATE OR REPLACE FUNCTION sync_symmetric_relationships()
RETURNS TRIGGER AS $$
DECLARE
    is_sym BOOLEAN;
BEGIN
    -- Check if this relationship type is symmetric
    SELECT is_symmetric INTO is_sym
    FROM relationship_types
    WHERE id = NEW.type_id;

    -- If symmetric, create/update the reverse relationship
    IF is_sym THEN
        IF TG_OP = 'INSERT' THEN
            -- Create reverse relationship (swap source and target)
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
            -- Update reverse relationship
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
            -- Delete reverse relationship
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

-- Cascade status changes to relationships
CREATE OR REPLACE FUNCTION cascade_status_to_relationships()
RETURNS TRIGGER AS $$
DECLARE
    source_table TEXT;
    status_category TEXT;
    type_name TEXT;
BEGIN
    -- Determine which table triggered this
    source_table := TG_TABLE_NAME;

    -- Map table name to type name
    type_name := CASE
        WHEN source_table = 'entities' THEN 'entity'
        WHEN source_table = 'knowledge_items' THEN 'knowledge'
        WHEN source_table = 'logs' THEN 'log'
        WHEN source_table = 'jobs' THEN 'job'
        WHEN source_table = 'agents' THEN 'agent'
        WHEN source_table = 'files' THEN 'file'
        WHEN source_table = 'protocols' THEN 'protocol'
    END;

    -- Get the status category (active or archived)
    SELECT category INTO status_category
    FROM statuses
    WHERE id = NEW.status_id;

    -- If status changed to archived category, update related relationships
    IF status_category = 'archived' THEN
        UPDATE relationships
        SET status_id = NEW.status_id,
            status_changed_at = NOW()
        WHERE (
            (source_type = type_name AND source_id = NEW.id::text)
            OR
            (target_type = type_name AND target_id = NEW.id::text)
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Audit logging function
CREATE OR REPLACE FUNCTION audit_trigger_function()
RETURNS TRIGGER AS $$
DECLARE
    changed_fields TEXT[];
    old_json JSONB;
    new_json JSONB;
    changed_by_type TEXT;
    changed_by_id UUID;
BEGIN
    -- Get current session variables (set by application)
    BEGIN
        changed_by_type := current_setting('app.changed_by_type', TRUE);
        changed_by_id := current_setting('app.changed_by_id', TRUE)::UUID;
    EXCEPTION
        WHEN OTHERS THEN
            changed_by_type := 'system';
            changed_by_id := NULL;
    END;

    -- Convert old and new rows to JSONB
    IF TG_OP = 'DELETE' THEN
        old_json := to_jsonb(OLD);
        new_json := NULL;
    ELSIF TG_OP = 'INSERT' THEN
        old_json := NULL;
        new_json := to_jsonb(NEW);
    ELSIF TG_OP = 'UPDATE' THEN
        old_json := to_jsonb(OLD);
        new_json := to_jsonb(NEW);

        -- Determine which fields changed
        SELECT array_agg(key)
        INTO changed_fields
        FROM jsonb_each(new_json)
        WHERE new_json->key IS DISTINCT FROM old_json->key;
    END IF;

    -- Insert audit record
    INSERT INTO audit_log (
        table_name,
        record_id,
        action,
        changed_by_type,
        changed_by_id,
        old_data,
        new_data,
        changed_fields,
        changed_at
    ) VALUES (
        TG_TABLE_NAME,
        COALESCE(NEW.id::TEXT, OLD.id::TEXT),
        lower(TG_OP),
        changed_by_type,
        changed_by_id,
        old_json,
        new_json,
        changed_fields,
        NOW()
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- --- Tables ---

-- Statuses (unified active + archived states)
CREATE TABLE IF NOT EXISTS statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    category TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (category IN ('active', 'archived'))
);

-- Privacy Scopes
CREATE TABLE IF NOT EXISTS privacy_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Relationship Types
CREATE TABLE IF NOT EXISTS relationship_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_symmetric BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Entities (people, projects, tools, organizations, courses)
CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    privacy_scope_ids UUID[] DEFAULT '{}',
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    status_id UUID REFERENCES statuses(id),
    status_changed_at TIMESTAMPTZ,
    status_reason TEXT,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::jsonb,
    embedding VECTOR(1536),
    vault_file_path TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT metadata_is_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- Knowledge Items (saved content, videos, articles, papers)
CREATE TABLE IF NOT EXISTS knowledge_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    privacy_scope_ids UUID[] DEFAULT '{}',
    title TEXT NOT NULL,
    url TEXT,
    source_type TEXT,
    content TEXT,
    status_id UUID REFERENCES statuses(id),
    status_changed_at TIMESTAMPTZ,
    status_reason TEXT,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::jsonb,
    vault_file_path TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT metadata_is_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- Logs (time-series data: gym, weight, calories, etc)
CREATE TABLE IF NOT EXISTS logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    log_type TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    value JSONB DEFAULT '{}'::jsonb,
    status_id UUID REFERENCES statuses(id),
    status_changed_at TIMESTAMPTZ,
    status_reason TEXT,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT value_is_object CHECK (jsonb_typeof(value) = 'object'),
    CONSTRAINT metadata_is_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- Relationships (polymorphic: links entities, knowledge, logs, jobs, agents, files, protocols)
CREATE TABLE IF NOT EXISTS relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    type_id UUID NOT NULL REFERENCES relationship_types(id) ON DELETE RESTRICT,
    status_id UUID REFERENCES statuses(id),
    status_changed_at TIMESTAMPTZ,
    properties JSONB DEFAULT '{}'::jsonb,
    embedding VECTOR(1536),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (source_type IN ('entity', 'knowledge', 'log', 'job', 'agent', 'file', 'protocol')),
    CHECK (target_type IN ('entity', 'knowledge', 'log', 'job', 'agent', 'file', 'protocol')),
    UNIQUE (source_type, source_id, target_type, target_id, type_id)
);

-- Semantic Search (unified embeddings from all sources)
--
-- IMPLEMENTATION NOTE: This table is populated from application/MCP layer, not triggers.
-- Embedding generation requires external API calls (OpenAI, etc) which can't be done in postgres.
--
-- Recommended implementation (pseudocode):
--
-- ```python
-- async def sync_entity_embeddings(entity_id: UUID):
--     # 1. Get entity and its context segments
--     entity = await db.get_entity(entity_id)
--     segments = entity.metadata.get('context_segments', [])
--
--     # 2. Delete old embeddings for this entity
--     await db.execute(
--         "DELETE FROM semantic_search WHERE source_type = 'entity' AND source_id = $1",
--         entity_id
--     )
--
--     # 3. Generate and insert new embeddings (one per context segment)
--     for idx, segment in enumerate(segments):
--         text = segment['text']
--         scopes = segment['scopes']
--
--         # Call OpenAI API
--         embedding = await openai.embeddings.create(
--             model="text-embedding-3-small",
--             input=text
--         )
--
--         # Insert into semantic_search
--         await db.execute("""
--             INSERT INTO semantic_search (source_type, source_id, segment_index, embedding, scopes)
--             VALUES ('entity', $1, $2, $3, $4)
--         """, entity_id, idx, embedding.data[0].embedding, scopes)
--
-- # Call this function whenever entity/knowledge/log is created or updated
-- ```
--
-- Query example (privacy-aware semantic search):
-- ```sql
-- SELECT
--     ss.source_type,
--     ss.source_id,
--     ss.embedding <-> query_vector as distance
-- FROM semantic_search ss
-- WHERE ss.scopes && ARRAY['public', 'code']  -- agent's scopes
-- ORDER BY ss.embedding <-> query_vector
-- LIMIT 10;
-- ```
CREATE TABLE IF NOT EXISTS semantic_search (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    segment_index INT,
    embedding VECTOR(1536) NOT NULL,
    scopes UUID[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (source_type IN ('entity', 'knowledge', 'log', 'job', 'agent', 'file', 'protocol'))
);

-- Agents (AI agents with capabilities and access scopes)
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    system_prompt TEXT,
    scopes UUID[] DEFAULT '{}',
    capabilities TEXT[] DEFAULT '{}',
    status_id UUID REFERENCES statuses(id),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT metadata_is_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- Jobs (tasks, assignments, todos with YYYYQ#-NNNN IDs)
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY DEFAULT generate_job_id(),
    title TEXT NOT NULL,
    description TEXT,
    job_type TEXT,
    source_platform TEXT,
    source_url TEXT,
    assigned_to UUID REFERENCES entities(id),
    agent_id UUID REFERENCES agents(id),
    status_id UUID REFERENCES statuses(id),
    priority TEXT,
    parent_job_id TEXT REFERENCES jobs(id),
    due_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT metadata_is_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- Approval Requests (PR-like approval system)
CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id TEXT REFERENCES jobs(id),
    request_type TEXT NOT NULL,
    requested_by UUID REFERENCES agents(id),
    change_details JSONB DEFAULT '{}'::jsonb,
    status TEXT DEFAULT 'pending',
    reviewed_by UUID REFERENCES entities(id),
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT change_details_is_object CHECK (jsonb_typeof(change_details) = 'object')
);

-- Files (file metadata and paths)
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename TEXT NOT NULL,
    file_path TEXT NOT NULL,
    mime_type TEXT,
    size_bytes BIGINT,
    checksum TEXT,
    status_id UUID REFERENCES statuses(id),
    status_changed_at TIMESTAMPTZ,
    status_reason TEXT,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT metadata_is_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- Protocols (system protocols and agent interaction guidelines)
CREATE TABLE IF NOT EXISTS protocols (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    version TEXT,
    content TEXT NOT NULL,
    protocol_type TEXT,
    applies_to TEXT[] DEFAULT '{}',
    status_id UUID REFERENCES statuses(id),
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'::jsonb,
    vault_file_path TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT metadata_is_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- Audit Log (tracks all changes to critical tables)
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    action TEXT NOT NULL,
    changed_by_type TEXT,
    changed_by_id UUID,
    old_data JSONB,
    new_data JSONB,
    changed_fields TEXT[],
    change_reason TEXT,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb,
    CHECK (action IN ('insert', 'update', 'delete')),
    CHECK (changed_by_type IN ('agent', 'entity', 'system'))
);

-- --- Triggers ---

-- Update updated_at triggers
CREATE TRIGGER update_entities_updated_at
BEFORE UPDATE ON entities
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_knowledge_items_updated_at
BEFORE UPDATE ON knowledge_items
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_logs_updated_at
BEFORE UPDATE ON logs
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_statuses_updated_at
BEFORE UPDATE ON statuses
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_privacy_scopes_updated_at
BEFORE UPDATE ON privacy_scopes
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_relationship_types_updated_at
BEFORE UPDATE ON relationship_types
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_relationships_updated_at
BEFORE UPDATE ON relationships
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Validate polymorphic relationships
CREATE TRIGGER validate_relationships_trigger
BEFORE INSERT OR UPDATE ON relationships
FOR EACH ROW
EXECUTE FUNCTION validate_relationship_references();

-- Auto-sync symmetric relationships
CREATE TRIGGER sync_symmetric_relationships_trigger
AFTER INSERT OR UPDATE OR DELETE ON relationships
FOR EACH ROW
EXECUTE FUNCTION sync_symmetric_relationships();

-- Cascade status changes to relationships
CREATE TRIGGER cascade_entity_status_trigger
AFTER UPDATE OF status_id ON entities
FOR EACH ROW
EXECUTE FUNCTION cascade_status_to_relationships();

CREATE TRIGGER cascade_knowledge_status_trigger
AFTER UPDATE OF status_id ON knowledge_items
FOR EACH ROW
EXECUTE FUNCTION cascade_status_to_relationships();

CREATE TRIGGER cascade_log_status_trigger
AFTER UPDATE OF status_id ON logs
FOR EACH ROW
EXECUTE FUNCTION cascade_status_to_relationships();

CREATE TRIGGER cascade_job_status_trigger
AFTER UPDATE OF status_id ON jobs
FOR EACH ROW
EXECUTE FUNCTION cascade_status_to_relationships();

CREATE TRIGGER cascade_agent_status_trigger
AFTER UPDATE OF status_id ON agents
FOR EACH ROW
EXECUTE FUNCTION cascade_status_to_relationships();

CREATE TRIGGER cascade_file_status_trigger
AFTER UPDATE OF status_id ON files
FOR EACH ROW
EXECUTE FUNCTION cascade_status_to_relationships();

CREATE TRIGGER cascade_protocol_status_trigger
AFTER UPDATE OF status_id ON protocols
FOR EACH ROW
EXECUTE FUNCTION cascade_status_to_relationships();

CREATE TRIGGER update_agents_updated_at
BEFORE UPDATE ON agents
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jobs_updated_at
BEFORE UPDATE ON jobs
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_files_updated_at
BEFORE UPDATE ON files
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_protocols_updated_at
BEFORE UPDATE ON protocols
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Audit triggers (for critical tables)
CREATE TRIGGER audit_entities_trigger
AFTER INSERT OR UPDATE OR DELETE ON entities
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();

CREATE TRIGGER audit_knowledge_items_trigger
AFTER INSERT OR UPDATE OR DELETE ON knowledge_items
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();

CREATE TRIGGER audit_relationships_trigger
AFTER INSERT OR UPDATE OR DELETE ON relationships
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();

CREATE TRIGGER audit_jobs_trigger
AFTER INSERT OR UPDATE OR DELETE ON jobs
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();

CREATE TRIGGER audit_agents_trigger
AFTER INSERT OR UPDATE OR DELETE ON agents
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();

CREATE TRIGGER audit_approval_requests_trigger
AFTER INSERT OR UPDATE OR DELETE ON approval_requests
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();

CREATE TRIGGER audit_protocols_trigger
AFTER INSERT OR UPDATE OR DELETE ON protocols
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();

-- --- Indexes ---
-- NOTE: Run 001-seed-data.sql after this to populate default data

-- Entities indexes
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(type);
CREATE INDEX IF NOT EXISTS idx_entities_status ON entities(status_id);
CREATE INDEX IF NOT EXISTS idx_entities_privacy ON entities USING gin(privacy_scope_ids);
CREATE INDEX IF NOT EXISTS idx_entities_tags ON entities USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_entities_metadata ON entities USING gin(metadata);
CREATE INDEX IF NOT EXISTS idx_entities_search ON entities
    USING gin(to_tsvector('english', name || ' ' || COALESCE(metadata::text, '')));
CREATE INDEX IF NOT EXISTS idx_entities_embedding ON entities
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- Specific metadata field indexes for entities
CREATE INDEX IF NOT EXISTS idx_entities_uni ON entities((metadata->>'uni'))
    WHERE type = 'person';
CREATE INDEX IF NOT EXISTS idx_entities_role ON entities((metadata->>'role'))
    WHERE type = 'person';
CREATE INDEX IF NOT EXISTS idx_entities_location ON entities((metadata->>'location'))
    WHERE type = 'person';

-- Knowledge items indexes
CREATE INDEX IF NOT EXISTS idx_knowledge_status ON knowledge_items(status_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_privacy ON knowledge_items USING gin(privacy_scope_ids);
CREATE INDEX IF NOT EXISTS idx_knowledge_tags ON knowledge_items USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_knowledge_metadata ON knowledge_items USING gin(metadata);
CREATE INDEX IF NOT EXISTS idx_knowledge_source_type ON knowledge_items(source_type);
CREATE INDEX IF NOT EXISTS idx_knowledge_search ON knowledge_items
    USING gin(to_tsvector('english', title || ' ' || COALESCE(content, '') || ' ' || COALESCE(metadata::text, '')));

-- Logs indexes
CREATE INDEX IF NOT EXISTS idx_logs_type ON logs(log_type);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logs_status ON logs(status_id);
CREATE INDEX IF NOT EXISTS idx_logs_tags ON logs USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_logs_metadata ON logs USING gin(metadata);
CREATE INDEX IF NOT EXISTS idx_logs_type_timestamp ON logs(log_type, timestamp DESC);

-- Relationships indexes
CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON relationships(type_id);
CREATE INDEX IF NOT EXISTS idx_relationships_status ON relationships(status_id);
CREATE INDEX IF NOT EXISTS idx_relationships_full ON relationships(source_type, source_id, target_type, target_id, type_id);

-- Semantic search indexes
CREATE INDEX IF NOT EXISTS idx_semantic_search_vector ON semantic_search
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
CREATE INDEX IF NOT EXISTS idx_semantic_search_scopes ON semantic_search USING gin(scopes);
CREATE INDEX IF NOT EXISTS idx_semantic_search_source ON semantic_search(source_type, source_id);

-- Agents indexes
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status_id);
CREATE INDEX IF NOT EXISTS idx_agents_scopes ON agents USING gin(scopes);
CREATE INDEX IF NOT EXISTS idx_agents_capabilities ON agents USING gin(capabilities);

-- Jobs indexes
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status_id);
CREATE INDEX IF NOT EXISTS idx_jobs_assigned_to ON jobs(assigned_to);
CREATE INDEX IF NOT EXISTS idx_jobs_agent ON jobs(agent_id);
CREATE INDEX IF NOT EXISTS idx_jobs_parent ON jobs(parent_job_id);
CREATE INDEX IF NOT EXISTS idx_jobs_due_at ON jobs(due_at);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(job_type);
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority);
CREATE INDEX IF NOT EXISTS idx_jobs_source_platform ON jobs(source_platform);

-- Approval requests indexes
CREATE INDEX IF NOT EXISTS idx_approval_job ON approval_requests(job_id);
CREATE INDEX IF NOT EXISTS idx_approval_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_requested_by ON approval_requests(requested_by);
CREATE INDEX IF NOT EXISTS idx_approval_reviewed_by ON approval_requests(reviewed_by);

-- Files indexes
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status_id);
CREATE INDEX IF NOT EXISTS idx_files_tags ON files USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_files_mime_type ON files(mime_type);
CREATE INDEX IF NOT EXISTS idx_files_checksum ON files(checksum);

-- Protocols indexes
CREATE INDEX IF NOT EXISTS idx_protocols_type ON protocols(protocol_type);
CREATE INDEX IF NOT EXISTS idx_protocols_status ON protocols(status_id);
CREATE INDEX IF NOT EXISTS idx_protocols_tags ON protocols USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_protocols_applies_to ON protocols USING gin(applies_to);
CREATE INDEX IF NOT EXISTS idx_protocols_name ON protocols(name);
CREATE INDEX IF NOT EXISTS idx_protocols_search ON protocols
    USING gin(to_tsvector('english', title || ' ' || content || ' ' || COALESCE(metadata::text, '')));

-- Audit log indexes
CREATE INDEX IF NOT EXISTS idx_audit_table_record ON audit_log(table_name, record_id);
CREATE INDEX IF NOT EXISTS idx_audit_changed_at ON audit_log(changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_changed_by ON audit_log(changed_by_type, changed_by_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_table_name ON audit_log(table_name);
