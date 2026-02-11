"""Red team tests for invalid UUID handling across MCP tools."""

# Standard Library
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import GetRelationshipsInput, UpdateEntityInput, UpdateRelationshipInput
from nebula_mcp.server import get_relationships, update_entity, update_relationship


def _make_context(pool, enums, agent):
    """Build MCP context with a specific agent."""

    ctx = MagicMock()
    ctx.request_context.lifespan_context = {
        "pool": pool,
        "enums": enums,
        "agent": agent,
    }
    return ctx


async def _make_agent(db_pool, enums, name, scopes):
    """Insert an agent for invalid UUID MCP tests."""

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


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_update_entity_rejects_invalid_uuid(db_pool, enums):
    """update_entity should reject malformed UUIDs cleanly."""

    agent = await _make_agent(db_pool, enums, "update-uuid-agent", ["public"])
    ctx = _make_context(db_pool, enums, agent)

    payload = UpdateEntityInput(
        entity_id="not-a-uuid",
        metadata={"note": "bad"},
    )

    with pytest.raises(ValueError):
        await update_entity(payload, ctx)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_update_relationship_rejects_invalid_uuid(db_pool, enums):
    """update_relationship should reject malformed UUIDs cleanly."""

    agent = await _make_agent(db_pool, enums, "rel-uuid-agent", ["public"])
    ctx = _make_context(db_pool, enums, agent)

    payload = UpdateRelationshipInput(
        relationship_id="not-a-uuid",
        properties={"note": "bad"},
    )

    with pytest.raises(ValueError):
        await update_relationship(payload, ctx)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs are accepted without validation")
async def test_get_relationships_rejects_invalid_uuid(db_pool, enums):
    """get_relationships should reject malformed UUIDs cleanly."""

    agent = await _make_agent(db_pool, enums, "get-rel-uuid-agent", ["public"])
    ctx = _make_context(db_pool, enums, agent)
    payload = GetRelationshipsInput(
        source_type="entity",
        source_id="not-a-uuid",
        relationship_type=None,
        direction="both",
    )

    with pytest.raises(ValueError):
        await get_relationships(payload, ctx)
