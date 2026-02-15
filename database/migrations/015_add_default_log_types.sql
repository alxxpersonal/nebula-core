-- ---
-- ADD DEFAULT LOG TYPES (NON-BUILTIN)
-- ---
-- Purpose:
-- - Provide a small set of practical log types used by tooling and local workflows.
-- - Keep them non-builtin so users can edit/archive them if desired.

INSERT INTO log_types (name, description, value_schema, is_builtin, is_active)
VALUES
  ('work', 'Work log entry', '{"type":"object"}'::jsonb, FALSE, TRUE),
  ('system', 'System/internal log entry', '{"type":"object"}'::jsonb, FALSE, TRUE)
ON CONFLICT (name) DO UPDATE
SET is_active = TRUE;

