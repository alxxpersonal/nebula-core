-- Security hardening updates

ALTER TABLE protocols
    ADD COLUMN IF NOT EXISTS trusted BOOLEAN DEFAULT FALSE;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'entities_tags_limit'
    ) THEN
        ALTER TABLE entities
            ADD CONSTRAINT entities_tags_limit
            CHECK (COALESCE(array_length(tags, 1), 0) <= 50);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'knowledge_items_tags_limit'
    ) THEN
        ALTER TABLE knowledge_items
            ADD CONSTRAINT knowledge_items_tags_limit
            CHECK (COALESCE(array_length(tags, 1), 0) <= 50);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'logs_tags_limit'
    ) THEN
        ALTER TABLE logs
            ADD CONSTRAINT logs_tags_limit
            CHECK (COALESCE(array_length(tags, 1), 0) <= 50);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'files_tags_limit'
    ) THEN
        ALTER TABLE files
            ADD CONSTRAINT files_tags_limit
            CHECK (COALESCE(array_length(tags, 1), 0) <= 50);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'protocols_tags_limit'
    ) THEN
        ALTER TABLE protocols
            ADD CONSTRAINT protocols_tags_limit
            CHECK (COALESCE(array_length(tags, 1), 0) <= 50);
    END IF;
END $$;
