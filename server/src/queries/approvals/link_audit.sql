-- Link audit log entry to approval request via metadata
UPDATE audit_log 
SET metadata = metadata || jsonb_build_object('approval_id', $1::text)
WHERE id = (
    SELECT id FROM audit_log
    WHERE table_name = 'entities' 
    AND record_id = $2
    AND action = 'insert'
    ORDER BY changed_at DESC 
    LIMIT 1
)
