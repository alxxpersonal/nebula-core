-- Check if knowledge item already exists for URL
SELECT id, title FROM knowledge_items WHERE url = $1
