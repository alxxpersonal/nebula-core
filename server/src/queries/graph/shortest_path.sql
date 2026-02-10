-- Shortest path between two nodes (undirected)
WITH RECURSIVE edges AS (
    SELECT source_type, source_id, target_type, target_id
    FROM relationships
    UNION ALL
    SELECT target_type, target_id, source_type, source_id
    FROM relationships
),
search AS (
    SELECT
        $1::text AS node_type,
        $2::text AS node_id,
        0 AS depth,
        ARRAY[$1 || ':' || $2]::text[] AS path
    UNION ALL
    SELECT
        e.target_type,
        e.target_id,
        s.depth + 1,
        s.path || (e.target_type || ':' || e.target_id)
    FROM search s
    JOIN edges e
        ON e.source_type = s.node_type
        AND e.source_id = s.node_id
    WHERE s.depth < $5
      AND NOT (e.target_type || ':' || e.target_id = ANY(s.path))
)
SELECT path, depth
FROM search
WHERE node_type = $3
  AND node_id = $4
ORDER BY depth ASC
LIMIT 1;
