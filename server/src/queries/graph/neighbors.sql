-- Graph neighbors within K hops (undirected)
WITH RECURSIVE edges AS (
    SELECT source_type, source_id, target_type, target_id
    FROM relationships
    UNION ALL
    SELECT target_type, target_id, source_type, source_id
    FROM relationships
),
traversal AS (
    SELECT
        $1::text AS node_type,
        $2::text AS node_id,
        0 AS depth,
        ARRAY[$1 || ':' || $2]::text[] AS path
    UNION ALL
    SELECT
        e.target_type,
        e.target_id,
        t.depth + 1,
        t.path || (e.target_type || ':' || e.target_id)
    FROM traversal t
    JOIN edges e
        ON e.source_type = t.node_type
        AND e.source_id = t.node_id
    WHERE t.depth < $3
      AND NOT (e.target_type || ':' || e.target_id = ANY(t.path))
)
SELECT node_type, node_id, depth, path
FROM (
    SELECT DISTINCT ON (node_type, node_id)
        node_type, node_id, depth, path
    FROM traversal
    WHERE depth > 0
    ORDER BY node_type, node_id, depth
) ranked
ORDER BY depth ASC
LIMIT $4;
