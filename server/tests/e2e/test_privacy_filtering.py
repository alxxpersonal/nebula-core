"""E2E test: privacy scope filtering."""

# Standard Library
import json
from unittest.mock import MagicMock

import pytest

from nebula_mcp.models import GetEntityInput
from nebula_mcp.server import get_entity

pytestmark = pytest.mark.e2e


# --- Helpers ---


def _mock_ctx(pool, enums, agent):
    """Build a MagicMock MCP context with pool, enums, and agent in lifespan_context."""

    ctx = MagicMock()
    ctx.request_context.lifespan_context = {
        "pool": pool,
        "enums": enums,
        "agent": agent,
    }
    return ctx


async def _make_agent_with_scopes(pool, enums, name, scope_names):
    """Insert an agent with specific scopes and return the row."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scope_names]

    row = await pool.fetchrow(
        """
        INSERT INTO agents (name, status_id, scopes, requires_approval)
        VALUES ($1, $2, $3, false)
        RETURNING *
        """,
        name,
        status_id,
        scope_ids,
    )
    return row


async def _make_entity_with_segments(pool, enums, name, scope_names, segments):
    """Insert an entity with context_segments metadata and return the row."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.entity_types.name_to_id["person"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scope_names]

    metadata = json.dumps({"context_segments": segments})

    row = await pool.fetchrow(
        """
        INSERT INTO entities (privacy_scope_ids, name, type_id, status_id, metadata)
        VALUES ($1, $2, $3, $4, $5::jsonb)
        RETURNING *
        """,
        scope_ids,
        name,
        type_id,
        status_id,
        metadata,
    )
    return row


# --- Privacy Filtering ---


@pytest.mark.asyncio
async def test_agent_denied_when_scope_mismatch(db_pool, enums):
    """Agent with only health scope cannot access a code-scoped entity."""

    agent = await _make_agent_with_scopes(
        db_pool, enums, "health-only-agent", ["health"]
    )

    code_scope_id = enums.scopes.name_to_id["code"]
    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.entity_types.name_to_id["project"]

    entity = await db_pool.fetchrow(
        """
        INSERT INTO entities (privacy_scope_ids, name, type_id, status_id)
        VALUES ($1, $2, $3, $4)
        RETURNING *
        """,
        [code_scope_id],
        "code-only-project",
        type_id,
        status_id,
    )

    ctx = _mock_ctx(db_pool, enums, dict(agent))
    payload = GetEntityInput(
        entity_id=str(entity["id"]),
    )

    with pytest.raises(ValueError, match="Access denied"):
        await get_entity(payload, ctx)


@pytest.mark.asyncio
async def test_context_segments_filtered_by_scope(db_pool, enums):
    """Agent with public scope only sees public segments, not personal ones."""

    agent = await _make_agent_with_scopes(db_pool, enums, "public-agent", ["public"])

    segments = [
        {"text": "Public info about the person", "scopes": ["public"]},
        {"text": "Private details about the person", "scopes": ["personal"]},
    ]

    entity = await _make_entity_with_segments(
        db_pool, enums, "multi-scope-person", ["public", "personal"], segments
    )

    ctx = _mock_ctx(db_pool, enums, dict(agent))
    payload = GetEntityInput(
        entity_id=str(entity["id"]),
    )

    result = await get_entity(payload, ctx)
    meta = result["metadata"]

    if isinstance(meta, str):
        meta = json.loads(meta)

    filtered_segments = meta.get("context_segments", [])
    assert len(filtered_segments) == 1
    assert filtered_segments[0]["text"] == "Public info about the person"


@pytest.mark.asyncio
async def test_agent_with_all_scopes_sees_all_segments(db_pool, enums):
    """Agent with both public and personal scopes sees all segments."""

    agent = await _make_agent_with_scopes(
        db_pool, enums, "all-scope-agent", ["public", "personal"]
    )

    segments = [
        {"text": "Public info", "scopes": ["public"]},
        {"text": "Personal info", "scopes": ["personal"]},
    ]

    entity = await _make_entity_with_segments(
        db_pool, enums, "full-scope-person", ["public", "personal"], segments
    )

    ctx = _mock_ctx(db_pool, enums, dict(agent))
    payload = GetEntityInput(
        entity_id=str(entity["id"]),
    )

    result = await get_entity(payload, ctx)
    meta = result["metadata"]

    if isinstance(meta, str):
        meta = json.loads(meta)

    filtered_segments = meta.get("context_segments", [])
    assert len(filtered_segments) == 2
