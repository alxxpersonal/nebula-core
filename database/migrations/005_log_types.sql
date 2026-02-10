-- ---
-- LOG TYPES TABLE + MIGRATION
-- ---
-- Defines allowed log types and migrates logs.log_type TEXT to log_type_id UUID
-- Similar to entity_types pattern - ensures consistency and validation

-- Step 1: Create log_types table
CREATE TABLE IF NOT EXISTS log_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    value_schema JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Update trigger
CREATE TRIGGER update_log_types_updated_at
BEFORE UPDATE ON log_types
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Step 2: Seed common log types
INSERT INTO log_types (name, description, value_schema) VALUES
    ('gym-session', 'Workout session', '{"type": "object", "properties": {"duration_minutes": {"type": "number"}, "exercises": {"type": "array"}}}'::jsonb),
    ('weight', 'Body weight measurement', '{"type": "object", "properties": {"kg": {"type": "number"}}, "required": ["kg"]}'::jsonb),
    ('mood', 'Mood tracking', '{"type": "object", "properties": {"score": {"type": "integer", "minimum": 1, "maximum": 10}, "note": {"type": "string"}}, "required": ["score"]}'::jsonb),
    ('sleep', 'Sleep tracking', '{"type": "object", "properties": {"hours": {"type": "number"}, "quality": {"type": "integer", "minimum": 1, "maximum": 10}}, "required": ["hours"]}'::jsonb),
    ('calories', 'Calorie intake', '{"type": "object", "properties": {"kcal": {"type": "number"}}, "required": ["kcal"]}'::jsonb),
    ('workout', 'General workout entry', '{"type": "object", "properties": {"type": {"type": "string"}, "duration_minutes": {"type": "number"}}}'::jsonb),
    ('meditation', 'Meditation session', '{"type": "object", "properties": {"duration_minutes": {"type": "number"}, "technique": {"type": "string"}}}'::jsonb),
    ('reading', 'Reading session', '{"type": "object", "properties": {"pages": {"type": "integer"}, "book": {"type": "string"}}}'::jsonb),
    ('water-intake', 'Water consumption', '{"type": "object", "properties": {"liters": {"type": "number"}}, "required": ["liters"]}'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_log_types_name ON log_types(name);

-- Step 3: Add log_type_id column to logs
ALTER TABLE logs
ADD COLUMN IF NOT EXISTS log_type_id UUID REFERENCES log_types(id);

-- Step 4: Migrate existing log_type values to log_type_id
-- First, insert any log types that exist in logs but not in log_types
INSERT INTO log_types (name, description)
SELECT DISTINCT log_type, 'Auto-migrated log type'
FROM logs
WHERE log_type NOT IN (SELECT name FROM log_types)
ON CONFLICT (name) DO NOTHING;

-- Now migrate all existing logs
UPDATE logs l
SET log_type_id = lt.id
FROM log_types lt
WHERE l.log_type = lt.name
  AND l.log_type_id IS NULL;

-- Step 5: Check for orphans (logs that didn't match any log_type)
-- If this returns rows, something went wrong
DO $$
DECLARE
    orphan_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO orphan_count FROM logs WHERE log_type_id IS NULL;
    IF orphan_count > 0 THEN
        RAISE WARNING 'Found % logs without matching log_type_id. Check data before proceeding.', orphan_count;
    END IF;
END $$;

-- Step 6: Drop old log_type column and make log_type_id required
-- Only proceed if no orphans exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM logs WHERE log_type_id IS NULL) THEN
        ALTER TABLE logs DROP COLUMN IF EXISTS log_type;
        ALTER TABLE logs ALTER COLUMN log_type_id SET NOT NULL;
    ELSE
        RAISE EXCEPTION 'Cannot drop log_type column - orphaned rows exist. Fix data first.';
    END IF;
END $$;

-- Step 7: Add index on log_type_id
CREATE INDEX IF NOT EXISTS idx_logs_type_id ON logs(log_type_id);

-- Step 8: Update composite index to use log_type_id instead of log_type
DROP INDEX IF EXISTS idx_logs_type_timestamp;
CREATE INDEX IF NOT EXISTS idx_logs_type_id_timestamp ON logs(log_type_id, timestamp DESC);
