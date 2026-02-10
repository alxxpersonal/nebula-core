-- List privacy scopes with usage counts
SELECT
  ps.id,
  ps.name,
  ps.description,
  (
    SELECT COUNT(*)
    FROM agents a
    WHERE a.scopes && ARRAY[ps.id]
  ) AS agent_count,
  (
    SELECT COUNT(*)
    FROM entities e
    WHERE e.privacy_scope_ids && ARRAY[ps.id]
  ) AS entity_count,
  (
    SELECT COUNT(*)
    FROM knowledge_items k
    WHERE k.privacy_scope_ids && ARRAY[ps.id]
  ) AS knowledge_count
FROM privacy_scopes ps
ORDER BY ps.name;
