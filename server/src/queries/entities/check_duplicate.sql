-- Check if entity exists with same name, type, and scopes (Layer 2 dedup)
SELECT id, name FROM entities 
WHERE name = $1 
AND type_id = $2 
AND privacy_scope_ids = $3
