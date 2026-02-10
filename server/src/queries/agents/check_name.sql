-- Check if agent name already exists
SELECT id FROM agents WHERE name = $1
