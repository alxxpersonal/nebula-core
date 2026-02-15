-- Get enrollment by linked approval request id
SELECT * FROM agent_enrollment_sessions
WHERE approval_request_id = $1::uuid;
