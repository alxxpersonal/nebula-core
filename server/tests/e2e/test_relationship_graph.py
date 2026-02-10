"""E2E test: relationship graph operations."""

# Third-Party
import pytest

pytestmark = pytest.mark.e2e


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


# --- Symmetric Auto-Sync ---


@pytest.mark.asyncio
async def test_symmetric_auto_sync(db_pool, enums):
    """Inserting a symmetric friends-with relationship creates both directions."""

    a = await _make_entity(db_pool, enums, "graph-sym-a")
    b = await _make_entity(db_pool, enums, "graph-sym-b")
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

    forward = await db_pool.fetchrow(
        """
        SELECT * FROM relationships
        WHERE source_type = 'entity' AND source_id = $1
          AND target_type = 'entity' AND target_id = $2
          AND type_id = $3
        """,
        str(a["id"]),
        str(b["id"]),
        type_id,
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

    assert forward is not None
    assert reverse is not None


# --- Archive Cascade ---


@pytest.mark.asyncio
async def test_archive_cascades_to_relationship(db_pool, enums):
    """Archiving an entity cascades the inactive status to its relationships."""

    a = await _make_entity(db_pool, enums, "graph-archive-a")
    b = await _make_entity(db_pool, enums, "graph-archive-b")
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


# --- Asymmetric Direction ---


@pytest.mark.asyncio
async def test_asymmetric_direction(db_pool, enums):
    """Inserting works-on A->B creates only the forward edge, not the reverse."""

    a = await _make_entity(db_pool, enums, "graph-asym-a")
    b = await _make_entity(db_pool, enums, "graph-asym-b")
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

    forward_count = await db_pool.fetchval(
        """
        SELECT COUNT(*) FROM relationships
        WHERE source_type = 'entity' AND source_id = $1
          AND target_type = 'entity' AND target_id = $2
          AND type_id = $3
        """,
        str(a["id"]),
        str(b["id"]),
        type_id,
    )
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

    assert forward_count == 1
    assert reverse_count == 0
