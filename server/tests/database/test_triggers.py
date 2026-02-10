"""Database tests for SQL triggers."""

# Standard Library
import asyncio
import re

import asyncpg
import pytest

pytestmark = pytest.mark.database


# --- Helpers ---


async def _make_entity(pool, enums, name):
    """Insert a minimal entity and return the row."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.entity_types.name_to_id["person"]
    scope_ids = [enums.scopes.name_to_id["public"]]

    row = await pool.fetchrow(
        """
        INSERT INTO entities (privacy_scope_ids, name, type_id, status_id)
        VALUES ($1, $2, $3, $4)
        RETURNING *
        """,
        scope_ids,
        name,
        type_id,
        status_id,
    )
    return row


# --- updated_at Trigger ---


@pytest.mark.asyncio
async def test_updated_at_trigger(db_pool, enums):
    """Updating an entity bumps updated_at via the trigger."""

    row = await _make_entity(db_pool, enums, "trigger-test-entity")
    created_at = row["created_at"]

    await asyncio.sleep(0.01)

    await db_pool.execute(
        "UPDATE entities SET name = $1 WHERE id = $2",
        "trigger-test-entity-renamed",
        row["id"],
    )

    updated = await db_pool.fetchrow("SELECT * FROM entities WHERE id = $1", row["id"])
    assert updated["updated_at"] > created_at


# --- Symmetric Relationship Trigger ---


@pytest.mark.asyncio
async def test_symmetric_relationship_creates_reverse(db_pool, enums):
    """Inserting a symmetric relationship auto-creates the reverse direction."""

    a = await _make_entity(db_pool, enums, "sym-a")
    b = await _make_entity(db_pool, enums, "sym-b")
    type_id = enums.relationship_types.name_to_id["friends-with"]
    status_id = enums.statuses.name_to_id["active"]

    await db_pool.execute(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id)
        VALUES ('entity', $1, 'entity', $2, $3, $4)
        """,
        str(a["id"]),
        str(b["id"]),
        type_id,
        status_id,
    )

    reverse = await db_pool.fetchrow(
        """
        SELECT * FROM relationships
        WHERE source_type = 'entity' AND source_id = $1
          AND target_type = 'entity' AND target_id = $2
          AND type_id = $3
        """,
        str(b["id"]),
        str(a["id"]),
        type_id,
    )
    assert reverse is not None


@pytest.mark.asyncio
async def test_symmetric_delete_cascades_reverse(db_pool, enums):
    """Deleting a symmetric relationship's forward edge also deletes the reverse."""

    a = await _make_entity(db_pool, enums, "sym-del-a")
    b = await _make_entity(db_pool, enums, "sym-del-b")
    type_id = enums.relationship_types.name_to_id["friends-with"]
    status_id = enums.statuses.name_to_id["active"]

    forward = await db_pool.fetchrow(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id)
        VALUES ('entity', $1, 'entity', $2, $3, $4)
        RETURNING id
        """,
        str(a["id"]),
        str(b["id"]),
        type_id,
        status_id,
    )

    await db_pool.execute("DELETE FROM relationships WHERE id = $1", forward["id"])

    reverse_count = await db_pool.fetchval(
        """
        SELECT COUNT(*) FROM relationships
        WHERE source_type = 'entity' AND source_id = $1
          AND target_type = 'entity' AND target_id = $2
          AND type_id = $3
        """,
        str(b["id"]),
        str(a["id"]),
        type_id,
    )
    assert reverse_count == 0


# --- Asymmetric Relationship ---


@pytest.mark.asyncio
async def test_asymmetric_no_reverse(db_pool, enums):
    """Inserting an asymmetric relationship does NOT create a reverse."""

    a = await _make_entity(db_pool, enums, "asym-a")
    b = await _make_entity(db_pool, enums, "asym-b")
    type_id = enums.relationship_types.name_to_id["works-on"]
    status_id = enums.statuses.name_to_id["active"]

    await db_pool.execute(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id)
        VALUES ('entity', $1, 'entity', $2, $3, $4)
        """,
        str(a["id"]),
        str(b["id"]),
        type_id,
        status_id,
    )

    reverse = await db_pool.fetchrow(
        """
        SELECT * FROM relationships
        WHERE source_type = 'entity' AND source_id = $1
          AND target_type = 'entity' AND target_id = $2
          AND type_id = $3
        """,
        str(b["id"]),
        str(a["id"]),
        type_id,
    )
    assert reverse is None


# --- Polymorphic Reference Validation ---


@pytest.mark.asyncio
async def test_polymorphic_ref_fake_source_raises(db_pool, enums):
    """Inserting a relationship with a nonexistent source entity raises RaiseError."""

    target = await _make_entity(db_pool, enums, "poly-target")
    type_id = enums.relationship_types.name_to_id["works-on"]
    status_id = enums.statuses.name_to_id["active"]
    fake_uuid = "00000000-0000-0000-0000-000000000000"

    with pytest.raises(asyncpg.RaiseError):
        await db_pool.execute(
            """
            INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id)
            VALUES ('entity', $1, 'entity', $2, $3, $4)
            """,
            fake_uuid,
            str(target["id"]),
            type_id,
            status_id,
        )


@pytest.mark.asyncio
async def test_polymorphic_ref_fake_target_raises(db_pool, enums):
    """Inserting a relationship with a nonexistent target entity raises RaiseError."""

    source = await _make_entity(db_pool, enums, "poly-source")
    type_id = enums.relationship_types.name_to_id["works-on"]
    status_id = enums.statuses.name_to_id["active"]
    fake_uuid = "00000000-0000-0000-0000-000000000000"

    with pytest.raises(asyncpg.RaiseError):
        await db_pool.execute(
            """
            INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id)
            VALUES ('entity', $1, 'entity', $2, $3, $4)
            """,
            str(source["id"]),
            fake_uuid,
            type_id,
            status_id,
        )


# --- Status Cascade Trigger ---


@pytest.mark.asyncio
async def test_status_cascade_to_relationships(db_pool, enums):
    """Archiving an entity cascades the status to its relationships."""

    a = await _make_entity(db_pool, enums, "cascade-a")
    b = await _make_entity(db_pool, enums, "cascade-b")
    type_id = enums.relationship_types.name_to_id["friends-with"]
    active_id = enums.statuses.name_to_id["active"]

    await db_pool.execute(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id)
        VALUES ('entity', $1, 'entity', $2, $3, $4)
        """,
        str(a["id"]),
        str(b["id"]),
        type_id,
        active_id,
    )

    inactive_id = enums.statuses.name_to_id["inactive"]
    await db_pool.execute(
        "UPDATE entities SET status_id = $1 WHERE id = $2",
        inactive_id,
        a["id"],
    )

    rel = await db_pool.fetchrow(
        """
        SELECT status_id FROM relationships
        WHERE source_type = 'entity' AND source_id = $1
          AND target_type = 'entity' AND target_id = $2
          AND type_id = $3
        """,
        str(a["id"]),
        str(b["id"]),
        type_id,
    )
    assert rel["status_id"] == inactive_id


# --- Audit Log Trigger ---


@pytest.mark.asyncio
async def test_audit_log_insert(db_pool, enums):
    """Inserting an entity creates an audit_log record with action='insert'."""

    entity = await _make_entity(db_pool, enums, "audit-insert-entity")

    audit = await db_pool.fetchrow(
        """
        SELECT * FROM audit_log
        WHERE table_name = 'entities'
          AND record_id = $1
          AND action = 'insert'
        """,
        str(entity["id"]),
    )
    assert audit is not None


@pytest.mark.asyncio
async def test_audit_log_update(db_pool, enums):
    """Updating an entity creates an audit_log record with changed_fields containing 'name'."""

    entity = await _make_entity(db_pool, enums, "audit-update-entity")

    await db_pool.execute(
        "UPDATE entities SET name = $1 WHERE id = $2",
        "audit-update-renamed",
        entity["id"],
    )

    audit = await db_pool.fetchrow(
        """
        SELECT * FROM audit_log
        WHERE table_name = 'entities'
          AND record_id = $1
          AND action = 'update'
        ORDER BY changed_at DESC
        LIMIT 1
        """,
        str(entity["id"]),
    )
    assert audit is not None
    assert "name" in audit["changed_fields"]


# --- Job ID Generation ---


@pytest.mark.asyncio
async def test_job_id_format(db_pool, enums):
    """Generated job IDs match the YYYYQ#-NNNN pattern."""

    status_id = enums.statuses.name_to_id["active"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO jobs (title, status_id)
        VALUES ('test-job', $1)
        RETURNING id
        """,
        status_id,
    )

    assert re.match(r"^\d{4}Q[1-4]-[A-Z0-9]{4}$", row["id"])


@pytest.mark.asyncio
async def test_job_ids_unique(db_pool, enums):
    """Twenty generated job IDs are all unique."""

    status_id = enums.statuses.name_to_id["active"]
    ids = set()

    for i in range(20):
        row = await db_pool.fetchrow(
            """
            INSERT INTO jobs (title, status_id)
            VALUES ($1, $2)
            RETURNING id
            """,
            f"unique-job-{i}",
            status_id,
        )
        ids.add(row["id"])

    assert len(ids) == 20
