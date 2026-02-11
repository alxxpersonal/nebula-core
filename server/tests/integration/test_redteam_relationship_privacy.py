"""Red team tests for relationship privacy across entity scopes."""

# Standard Library
import json
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import GetRelationshipsInput, QueryRelationshipsInput
from nebula_mcp.server import get_relationships, query_relationships


def _make_context(pool, enums, agent):
    """Build a mock MCP context for relationship privacy tests."""

    ctx = MagicMock()
    ctx.request_context.lifespan_context = {
        "pool": pool,
        "enums": enums,
        "agent": agent,
    }
    return ctx


async def _make_agent(db_pool, enums, name, scopes):
    """Insert a test agent for relationship privacy scenarios."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]

    row = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        name,
        "redteam agent",
        scope_ids,
        False,
        status_id,
    )
    return dict(row)


async def _make_entity(db_pool, enums, name, scopes):
    """Insert a test entity for relationship privacy scenarios."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.entity_types.name_to_id["person"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]

    row = await db_pool.fetchrow(
        """
        INSERT INTO entities (name, type_id, status_id, privacy_scope_ids, tags, metadata)
        VALUES ($1, $2, $3, $4, $5, $6::jsonb)
        RETURNING *
        """,
        name,
        type_id,
        status_id,
        scope_ids,
        ["test"],
        json.dumps({"note": "scope-test"}),
    )
    return dict(row)


async def _make_relationship(db_pool, enums, source_id, target_id):
    """Insert a relationship linking two entities."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.relationship_types.name_to_id["related-to"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id, properties)
        VALUES ('entity', $1, 'entity', $2, $3, $4, $5::jsonb)
        RETURNING *
        """,
        str(source_id),
        str(target_id),
        type_id,
        status_id,
        json.dumps({"note": "private-link"}),
    )
    return dict(row)


@pytest.mark.asyncio
async def test_get_relationships_hides_private_entities(db_pool, enums):
    """Get relationships should hide links to private entities."""

    public_entity = await _make_entity(db_pool, enums, "Public", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    rel = await _make_relationship(db_pool, enums, public_entity["id"], private_entity["id"])

    viewer = await _make_agent(db_pool, enums, "rel-viewer", ["public"])
    ctx = _make_context(db_pool, enums, viewer)

    payload = GetRelationshipsInput(
        source_type="entity",
        source_id=str(public_entity["id"]),
        direction="both",
        relationship_type=None,
    )
    results = await get_relationships(payload, ctx)
    ids = {row["id"] for row in results}

    assert rel["id"] not in ids


@pytest.mark.asyncio
async def test_query_relationships_hides_private_entities(db_pool, enums):
    """Query relationships should not expose private entity links."""

    public_entity = await _make_entity(db_pool, enums, "Public 2", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private 2", ["sensitive"])
    rel = await _make_relationship(db_pool, enums, public_entity["id"], private_entity["id"])

    viewer = await _make_agent(db_pool, enums, "rel-viewer-2", ["public"])
    ctx = _make_context(db_pool, enums, viewer)

    payload = QueryRelationshipsInput(
        source_type=None,
        target_type=None,
        relationship_types=[],
        status_category="active",
        limit=50,
    )
    results = await query_relationships(payload, ctx)
    ids = {row["id"] for row in results}

    assert rel["id"] not in ids
