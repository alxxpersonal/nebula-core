-- ---
-- ADD REQUIRES_APPROVAL TO AGENTS
-- ---
-- Gates agent actions through approval workflow based on trust level.
-- Trusted agents bypass approval, autonomous agents require review.

-- Add requires_approval column with safe default
ALTER TABLE agents 
ADD COLUMN IF NOT EXISTS requires_approval BOOLEAN NOT NULL DEFAULT true;

-- Index for fast filtering of approval-required agents
CREATE INDEX IF NOT EXISTS idx_agents_requires_approval 
ON agents(requires_approval) 
WHERE requires_approval = true;
