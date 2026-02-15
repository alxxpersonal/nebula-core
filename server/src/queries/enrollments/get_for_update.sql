-- Lock enrollment row for one-time key redemption
SELECT
  aes.*,
  a.name AS agent_name
FROM agent_enrollment_sessions aes
JOIN agents a ON a.id = aes.agent_id
WHERE aes.id = $1::uuid
FOR UPDATE;
