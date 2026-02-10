-- Mark approval request as failed with error message
UPDATE approval_requests 
SET status = 'approved-failed', execution_error = $1 
WHERE id = $2
