"""Integration tests for executor functions against a real Postgres database."""

# Standard Library
import json
import re

import pytest

from nebula_mcp.executors import (
    execute_create_entity,
    execute_create_job,
    execute_create_knowledge,
    execute_create_relationship,
    execute_update_entity,
)

pytestmark = pytest.mark.integration


# --- TestCreateEntity ---


class TestCreateEntity:
    """Tests for execute_create_entity."""

    async def test_success(self, db_pool, enums):
        """Creating a valid entity should return a row with an id."""

        result = await execute_create_entity(
            db_pool,
            enums,
            {
                "name": "Alpha Project",
                "type": "project",
                "status": "active",
                "scopes": ["public"],
                "tags": ["test"],
                "metadata": {"description": "A test project"},
            },
        )

        assert "id" in result
        assert result["name"] == "Alpha Project"

    async def test_invalid_status_raises(self, db_pool, enums):
        """An unknown status name should raise ValueError."""

        with pytest.raises(ValueError, match="Unknown status"):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Bad Status",
                    "type": "project",
                    "status": "INVALID_STATUS",
                    "scopes": ["public"],
                },
            )

    async def test_invalid_type_raises(self, db_pool, enums):
        """An unknown entity type should raise ValueError."""

        with pytest.raises(ValueError, match="Unknown entity type"):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Bad Type",
                    "type": "INVALID_TYPE",
                    "status": "active",
                    "scopes": ["public"],
                },
            )

    async def test_invalid_scopes_raises(self, db_pool, enums):
        """An unknown scope name should raise ValueError."""

        with pytest.raises(ValueError, match="Unknown scope"):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Bad Scope",
                    "type": "project",
                    "status": "active",
                    "scopes": ["INVALID_SCOPE"],
                },
            )

    async def test_vault_file_path_dedup_raises(self, db_pool, enums):
        """Inserting two entities with the same vault_file_path should raise."""

        await execute_create_entity(
            db_pool,
            enums,
            {
                "name": "First",
                "type": "project",
                "status": "active",
                "scopes": ["public"],
                "vault_file_path": "00-Vault/unique-path.md",
            },
        )

        with pytest.raises(ValueError, match="Entity already exists for vault file"):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Second",
                    "type": "project",
                    "status": "active",
                    "scopes": ["public"],
                    "vault_file_path": "00-Vault/unique-path.md",
                },
            )

    async def test_name_type_scope_dedup_raises(self, db_pool, enums):
        """Inserting two entities with the same name+type+scopes should raise."""

        await execute_create_entity(
            db_pool,
            enums,
            {
                "name": "Duplicate Test",
                "type": "tool",
                "status": "active",
                "scopes": ["public"],
            },
        )

        with pytest.raises(ValueError, match="already exists"):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Duplicate Test",
                    "type": "tool",
                    "status": "active",
                    "scopes": ["public"],
                },
            )

    async def test_valid_metadata_works(self, db_pool, enums):
        """Valid person metadata with structured fields should succeed."""

        result = await execute_create_entity(
            db_pool,
            enums,
            {
                "name": "Jane Doe",
                "type": "person",
                "status": "active",
                "scopes": ["personal"],
                "metadata": {
                    "first_name": "Jane",
                    "last_name": "Doe",
                    "birth_month": 6,
                    "birth_day": 15,
                },
            },
        )

        assert "id" in result

    async def test_invalid_metadata_month_raises(self, db_pool, enums):
        """Person metadata with birth_month=13 should raise a validation error."""

        with pytest.raises(ValueError, match="Birth month out of range"):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Bad Month Person",
                    "type": "person",
                    "status": "active",
                    "scopes": ["personal"],
                    "metadata": {"birth_month": 13},
                },
            )

    async def test_context_segment_unknown_scope_raises(self, db_pool, enums):
        """Context segment with an unknown scope name should raise."""

        with pytest.raises(ValueError, match="Unknown scope"):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Bad Segment Scope",
                    "type": "project",
                    "status": "active",
                    "scopes": ["public"],
                    "metadata": {
                        "context_segments": [
                            {"text": "secret", "scopes": ["NONEXISTENT_SCOPE"]},
                        ],
                    },
                },
            )

    async def test_context_segment_outside_entity_scopes_raises(self, db_pool, enums):
        """Context segment scope not in entity scopes should raise."""

        with pytest.raises(
            ValueError, match="Context segment scope not in entity scopes"
        ):
            await execute_create_entity(
                db_pool,
                enums,
                {
                    "name": "Scope Mismatch",
                    "type": "project",
                    "status": "active",
                    "scopes": ["public"],
                    "metadata": {
                        "context_segments": [
                            {"text": "private info", "scopes": ["personal"]},
                        ],
                    },
                },
            )

    async def test_json_string_change_details(self, db_pool, enums):
        """Passing change_details as a JSON string should work."""

        payload = json.dumps(
            {
                "name": "JSON String Entity",
                "type": "tool",
                "status": "active",
                "scopes": ["public"],
            }
        )

        result = await execute_create_entity(db_pool, enums, payload)
        assert "id" in result


# --- TestCreateKnowledge ---


class TestCreateKnowledge:
    """Tests for execute_create_knowledge."""

    async def test_success(self, db_pool, enums):
        """Creating a valid knowledge item should return a row with an id."""

        result = await execute_create_knowledge(
            db_pool,
            enums,
            {
                "title": "Test Article",
                "source_type": "article",
                "scopes": ["public"],
                "tags": ["test"],
            },
        )

        assert "id" in result

    async def test_url_dedup_raises(self, db_pool, enums):
        """Inserting two knowledge items with the same URL should raise."""

        await execute_create_knowledge(
            db_pool,
            enums,
            {
                "title": "First Article",
                "url": "https://example.com/unique",
                "source_type": "article",
                "scopes": ["public"],
            },
        )

        with pytest.raises(ValueError, match="Knowledge item already exists for URL"):
            await execute_create_knowledge(
                db_pool,
                enums,
                {
                    "title": "Second Article",
                    "url": "https://example.com/unique",
                    "source_type": "article",
                    "scopes": ["public"],
                },
            )

    async def test_no_url_no_dedup(self, db_pool, enums):
        """Two knowledge items with the same title but no URL should both succeed."""

        r1 = await execute_create_knowledge(
            db_pool,
            enums,
            {
                "title": "Same Title",
                "source_type": "note",
                "scopes": ["public"],
            },
        )

        r2 = await execute_create_knowledge(
            db_pool,
            enums,
            {
                "title": "Same Title",
                "source_type": "note",
                "scopes": ["public"],
            },
        )

        assert r1["id"] != r2["id"]


# --- TestCreateRelationship ---


class TestCreateRelationship:
    """Tests for execute_create_relationship."""

    async def test_success(self, db_pool, enums, test_entity):
        """Creating a relationship between two entities should succeed."""

        # Create a second entity for the target
        status_id = enums.statuses.name_to_id["active"]
        type_id = enums.entity_types.name_to_id["project"]
        scope_ids = [enums.scopes.name_to_id["public"]]

        target = await db_pool.fetchrow(
            """
            INSERT INTO entities (name, type_id, status_id, privacy_scope_ids, tags, metadata)
            VALUES ($1, $2, $3, $4, $5, $6::jsonb)
            RETURNING *
            """,
            "Target Project",
            type_id,
            status_id,
            scope_ids,
            ["test"],
            "{}",
        )

        result = await execute_create_relationship(
            db_pool,
            enums,
            {
                "source_type": "entity",
                "source_id": str(test_entity["id"]),
                "target_type": "entity",
                "target_id": str(target["id"]),
                "relationship_type": "works-on",
            },
        )

        assert "id" in result

    async def test_invalid_type_raises(self, db_pool, enums, test_entity):
        """An unknown relationship type should raise ValueError."""

        with pytest.raises(ValueError, match="Unknown relationship type"):
            await execute_create_relationship(
                db_pool,
                enums,
                {
                    "source_type": "entity",
                    "source_id": str(test_entity["id"]),
                    "target_type": "entity",
                    "target_id": str(test_entity["id"]),
                    "relationship_type": "INVALID_REL_TYPE",
                },
            )


# --- TestCreateJob ---


class TestCreateJob:
    """Tests for execute_create_job."""

    async def test_success(self, db_pool, enums):
        """Creating a valid job should return a row with an id."""

        result = await execute_create_job(
            db_pool,
            enums,
            {
                "title": "Test Job",
                "description": "A test job",
                "priority": "medium",
            },
        )

        assert "id" in result

    async def test_id_format(self, db_pool, enums):
        """Job ID should match the YYYYQ#-XXXX format."""

        result = await execute_create_job(
            db_pool,
            enums,
            {
                "title": "Format Check Job",
                "priority": "high",
            },
        )

        assert re.match(r"^\d{4}Q[1-4]-[A-Z0-9]{4}$", result["id"])


# --- TestUpdateEntity ---


class TestUpdateEntity:
    """Tests for execute_update_entity."""

    async def test_status_change(self, db_pool, enums, test_entity):
        """Updating an entity status should return the updated row."""

        result = await execute_update_entity(
            db_pool,
            enums,
            {
                "entity_id": str(test_entity["id"]),
                "status": "on-hold",
                "status_reason": "Integration test pause",
            },
        )

        assert result["status_id"] == enums.statuses.name_to_id["on-hold"]

    async def test_metadata_change(self, db_pool, enums, test_entity):
        """Updating entity metadata should return the updated row."""

        result = await execute_update_entity(
            db_pool,
            enums,
            {
                "entity_id": str(test_entity["id"]),
                "metadata": {"first_name": "Updated"},
            },
        )

        assert "id" in result

    async def test_nonexistent_raises(self, db_pool, enums):
        """Updating a nonexistent entity should raise ValueError."""

        with pytest.raises(ValueError, match="not found"):
            await execute_update_entity(
                db_pool,
                enums,
                {
                    "entity_id": "00000000-0000-0000-0000-000000000000",
                    "metadata": {"first_name": "Ghost"},
                },
            )
