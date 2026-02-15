-- MCP-native agent enrollment flow
-- Adds enrollment session tracking and review metadata for approvals

-- Approval review metadata for grant overrides (scopes/trust)
ALTER TABLE approval_requests
ADD COLUMN IF NOT EXISTS review_details JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE approval_requests
DROP CONSTRAINT IF EXISTS approval_requests_review_details_is_object;

ALTER TABLE approval_requests
ADD CONSTRAINT approval_requests_review_details_is_object
CHECK (jsonb_typeof(review_details) = 'object');

COMMENT ON COLUMN approval_requests.review_details IS
'Structured reviewer decisions for approvals, including scope/trust grants.';

-- Enrollment sessions used by MCP bootstrap tools
CREATE TABLE IF NOT EXISTS agent_enrollment_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    approval_request_id UUID NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending_approval',
    enrollment_token_hash TEXT NOT NULL,
    requested_scope_ids UUID[] NOT NULL DEFAULT '{}',
    granted_scope_ids UUID[] DEFAULT '{}',
    requested_requires_approval BOOLEAN NOT NULL DEFAULT true,
    granted_requires_approval BOOLEAN,
    rejected_reason TEXT,
    approved_by UUID REFERENCES entities(id),
    approved_at TIMESTAMPTZ,
    redeemed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (status IN ('pending_approval', 'approved', 'rejected', 'redeemed', 'expired')),
    CHECK (char_length(enrollment_token_hash) > 0)
);

CREATE INDEX IF NOT EXISTS idx_agent_enroll_status
ON agent_enrollment_sessions(status);

CREATE INDEX IF NOT EXISTS idx_agent_enroll_expires_at
ON agent_enrollment_sessions(expires_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_enroll_pending_agent
ON agent_enrollment_sessions(agent_id)
WHERE status = 'pending_approval';

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_enroll_approval_request
ON agent_enrollment_sessions(approval_request_id);

DROP TRIGGER IF EXISTS update_agent_enrollment_sessions_updated_at ON agent_enrollment_sessions;
CREATE TRIGGER update_agent_enrollment_sessions_updated_at
BEFORE UPDATE ON agent_enrollment_sessions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS audit_agent_enrollment_sessions_trigger ON agent_enrollment_sessions;
CREATE TRIGGER audit_agent_enrollment_sessions_trigger
AFTER INSERT OR UPDATE OR DELETE ON agent_enrollment_sessions
FOR EACH ROW
EXECUTE FUNCTION audit_trigger_function();
