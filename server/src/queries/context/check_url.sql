-- Check if context item already exists for URL
SELECT id, title FROM context_items WHERE url = $1
