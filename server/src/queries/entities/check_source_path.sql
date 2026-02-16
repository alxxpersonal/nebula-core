-- Check if entity exists with given source_path (optional dedup signal)
SELECT id, name FROM entities
WHERE source_path = $1
