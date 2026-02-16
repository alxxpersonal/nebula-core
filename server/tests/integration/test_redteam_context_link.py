"""Red team tests for context link isolation."""

# Standard Library
import json
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import LinkContextInput
from nebula_mcp.server import link_context_to_entity


def _make_context(pool, enums, agent):
    """Build MCP context with a specific agent."""

    ctx = MagicMock()
    ctx.request_context.lifespan_context = {
        "pool": pool,
        "enums": enums,
        "agent": agent,
    }
    return ctx


async def _make_agent(db_pool, enums, name, scopes, requires_approval):
    """Insert an agent for context link tests."""

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
        requires_approval,
        status_id,
    )
    return dict(row)


async def _make_entity(db_pool, enums, name, scopes):
    """Insert an entity for context link tests."""

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
        json.dumps({"context_segments": [{"text": "secret", "scopes": scopes}]}),
    )
    return dict(row)


async def _make_context_item(db_pool, enums, title, scopes):
    """Insert a context item for link tests."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]

    row = await db_pool.fetchrow(
        """
        INSERT INTO context_items (title, source_type, content, privacy_scope_ids, status_id, tags, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
        RETURNING *
        """,
        title,
        "note",
        "secret",
        scope_ids,
        status_id,
        ["test"],
        json.dumps({"note": "secret"}),
    )
    return dict(row)


@pytest.mark.asyncio
async def test_link_context_denies_private_entity(db_pool, enums):
    """Public agents should not link context to private entities."""

    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    context = await _make_context_item(db_pool, enums, "Public Context", ["public"])
    viewer = await _make_agent(db_pool, enums, "context-linker", ["public"], False)
    ctx = _make_context(db_pool, enums, viewer)

    payload = LinkContextInput(
        context_id=str(context["id"]),
        entity_id=str(private_entity["id"]),
        relationship_type="related-to",
    )

    with pytest.raises(ValueError):
        await link_context_to_entity(payload, ctx)


@pytest.mark.asyncio
async def test_link_context_duplicate_returns_clean_error(db_pool, enums):
    """Duplicate context links should return a domain error, not raw DB internals."""

    owner = await _make_agent(db_pool, enums, "context-link-owner", ["public"], False)
    entity = await _make_entity(db_pool, enums, "Public Entity", ["public"])
    context = await _make_context_item(db_pool, enums, "Public Context", ["public"])
    ctx = _make_context(db_pool, enums, owner)

    payload = LinkContextInput(
        context_id=str(context["id"]),
        entity_id=str(entity["id"]),
        relationship_type="related-to",
    )

    first = await link_context_to_entity(payload, ctx)
    assert first["source_type"] == "context"
    assert first["target_type"] == "entity"

    with pytest.raises(ValueError) as exc_info:
        await link_context_to_entity(payload, ctx)

    message = str(exc_info.value).lower()
    assert "relationship already exists" in message
    assert "constraint" not in message
    assert "duplicate key value" not in message
