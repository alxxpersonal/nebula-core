-- ---
-- AGENT API KEY SUPPORT
-- ---
-- Agent API key support: allow api_keys to reference agents directly

ALTER TABLE api_keys ALTER COLUMN entity_id DROP NOT NULL;

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS agent_id UUID REFERENCES agents(id);

CREATE INDEX IF NOT EXISTS idx_api_keys_agent ON api_keys(agent_id);

ALTER TABLE api_keys ADD CONSTRAINT api_keys_owner_check
    CHECK (
        (entity_id IS NOT NULL AND agent_id IS NULL)
        OR (entity_id IS NULL AND agent_id IS NOT NULL)
    );
