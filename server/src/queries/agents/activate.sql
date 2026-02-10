-- Activate agent by setting status
UPDATE agents SET status_id = $1 WHERE id = $2::uuid RETURNING *
