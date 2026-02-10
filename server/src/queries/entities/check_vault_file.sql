-- Check if entity exists with given vault_file_path (Layer 1 dedup)
SELECT id, name FROM entities 
WHERE vault_file_path = $1
