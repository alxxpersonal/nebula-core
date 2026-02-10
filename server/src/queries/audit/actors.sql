-- List audit actors with activity counts
SELECT
  audit_log.changed_by_type,
  audit_log.changed_by_id,
  COALESCE(entities.name, agents.name) AS actor_name,
  COUNT(*) AS action_count,
  MAX(audit_log.changed_at) AS last_seen
FROM audit_log
LEFT JOIN entities
  ON audit_log.changed_by_type = 'entity'
  AND audit_log.changed_by_id = entities.id
LEFT JOIN agents
  ON audit_log.changed_by_type = 'agent'
  AND audit_log.changed_by_id = agents.id
WHERE ($1::text IS NULL OR audit_log.changed_by_type = $1)
GROUP BY audit_log.changed_by_type, audit_log.changed_by_id, actor_name
ORDER BY last_seen DESC;
