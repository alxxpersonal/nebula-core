-- List all active API keys with owner info (admin view)
SELECT
    k.id,
    k.entity_id,
    k.agent_id,
    k.key_prefix,
    k.name,
    k.last_used_at,
    k.expires_at,
    k.created_at,
    e.name AS entity_name,
    a.name AS agent_name,
    CASE
        WHEN k.entity_id IS NOT NULL THEN 'user'
        WHEN k.agent_id IS NOT NULL THEN 'agent'
    END AS owner_type
FROM api_keys k
LEFT JOIN entities e ON k.entity_id = e.id
LEFT JOIN agents a ON k.agent_id = a.id
WHERE k.revoked_at IS NULL
ORDER BY k.created_at DESC;
