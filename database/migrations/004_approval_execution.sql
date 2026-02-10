-- ---
-- APPROVAL EXECUTION SUPPORT
-- ---
-- Migration 004: Approval Execution Support
--
-- Adds execution tracking for approved requests:
-- - execution_error: stores error message if execution fails
-- - Updated status constraint to include 'approved-failed'
-- - GIN index on audit_log metadata for fast approval lookups

-- 1. Add execution error tracking column
ALTER TABLE approval_requests 
ADD COLUMN IF NOT EXISTS execution_error TEXT;

-- 2. Drop old status constraint and add new one with 'approved-failed'
ALTER TABLE approval_requests 
DROP CONSTRAINT IF EXISTS approval_requests_status_check;

ALTER TABLE approval_requests 
ADD CONSTRAINT approval_requests_status_check 
CHECK (status IN ('pending', 'approved', 'rejected', 'approved-failed'));

-- 3. Add GIN index for querying approvals by audit metadata
-- This enables fast queries like: WHERE metadata->>'approval_id' = 'uuid'
CREATE INDEX IF NOT EXISTS idx_audit_log_metadata_approval 
ON audit_log USING GIN (metadata);

-- 4. Add documentation comment
COMMENT ON COLUMN approval_requests.execution_error IS 
'Error message if execution failed after approval. Used for agent feedback when status is approved-failed.';
