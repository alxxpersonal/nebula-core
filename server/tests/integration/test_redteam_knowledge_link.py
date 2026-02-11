"""Red team tests for knowledge link isolation."""

# Standard Library
import json
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import LinkKnowledgeInput
from nebula_mcp.server import link_knowledge_to_entity


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
    """Insert an agent for knowledge link tests."""

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
    """Insert an entity for knowledge link tests."""

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


async def _make_knowledge(db_pool, enums, title, scopes):
    """Insert a knowledge item for link tests."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]

    row = await db_pool.fetchrow(
        """
        INSERT INTO knowledge_items (title, source_type, content, privacy_scope_ids, status_id, tags, metadata)
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
async def test_link_knowledge_denies_private_entity(db_pool, enums):
    """Public agents should not link knowledge to private entities."""

    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    knowledge = await _make_knowledge(db_pool, enums, "Public Knowledge", ["public"])
    viewer = await _make_agent(db_pool, enums, "knowledge-linker", ["public"], False)
    ctx = _make_context(db_pool, enums, viewer)

    payload = LinkKnowledgeInput(
        knowledge_id=str(knowledge["id"]),
        entity_id=str(private_entity["id"]),
        relationship_type="related-to",
    )

    with pytest.raises(ValueError):
        await link_knowledge_to_entity(payload, ctx)
