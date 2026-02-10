-- Add 'work' privacy scope
INSERT INTO privacy_scopes (name, description)
VALUES ('work', 'Work-related agents and entities')
ON CONFLICT (name) DO NOTHING;
