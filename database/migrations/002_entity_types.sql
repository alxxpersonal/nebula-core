-- ---
-- ENTITY TYPES TABLE + MIGRATION
-- ---
-- Defines allowed entity types and migrates entities.type TEXT to type_id UUID

-- Step 1: Create entity_types table
CREATE TABLE IF NOT EXISTS entity_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Update trigger
CREATE TRIGGER update_entity_types_updated_at
BEFORE UPDATE ON entity_types
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Seed data
INSERT INTO entity_types (name, description) VALUES
    ('person', 'Individual human'),
    ('project', 'Active work or product being built'),
    ('tool', 'Software, service, or utility'),
    ('organization', 'Company, team, or institution'),
    ('course', 'Educational course or program'),
    ('idea', 'Concept or future project idea'),
    ('framework', 'Development framework or library'),
    ('paper', 'Research paper or academic publication'),
    ('university', 'Educational institution')
ON CONFLICT (name) DO NOTHING;

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_entity_types_name ON entity_types(name);

-- Step 2: Add type_id column to entities
ALTER TABLE entities
ADD COLUMN IF NOT EXISTS type_id UUID REFERENCES entity_types(id);

-- Step 3: Migrate existing type values to type_id
UPDATE entities e
SET type_id = et.id
FROM entity_types et
WHERE e.type = et.name
  AND e.type_id IS NULL;

-- Step 4: Check for orphans (types that didn't match entity_types)
-- If this returns rows, add those types to entity_types first
SELECT DISTINCT type FROM entities WHERE type_id IS NULL;

-- Step 5: Drop old type column and make type_id required
ALTER TABLE entities DROP COLUMN IF EXISTS type;
ALTER TABLE entities ALTER COLUMN type_id SET NOT NULL;

-- Step 6: Add index on type_id
CREATE INDEX IF NOT EXISTS idx_entities_type_id ON entities(type_id);
