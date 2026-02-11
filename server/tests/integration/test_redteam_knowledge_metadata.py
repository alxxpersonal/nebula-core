"""Red team tests for knowledge metadata privacy filtering."""

# Standard Library
import json
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import QueryKnowledgeInput
from nebula_mcp.server import query_knowledge


def _make_context(pool, enums, agent):
    """Build MCP context with a specific agent."""

    ctx = MagicMock()
    ctx.request_context.lifespan_context = {
        "pool": pool,
        "enums": enums,
        "agent": agent,
    }
    return ctx


async def _make_knowledge(db_pool, enums, title, scopes, metadata):
    """Insert a knowledge item for metadata filtering tests."""

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
        json.dumps(metadata),
    )
    return dict(row)


@pytest.mark.asyncio
async def test_query_knowledge_filters_context_segments(db_pool, enums):
    """Query results should not include context segments outside agent scopes."""

    metadata = {
        "context_segments": [
            {"text": "public info", "scopes": ["public"]},
            {"text": "private info", "scopes": ["personal"]},
        ]
    }
    await _make_knowledge(db_pool, enums, "Mixed Scope", ["public", "personal"], metadata)

    public_agent = {
        "id": "public-agent",
        "scopes": [enums.scopes.name_to_id["public"]],
    }
    ctx = _make_context(db_pool, enums, public_agent)

    payload = QueryKnowledgeInput(scopes=["public"])
    rows = await query_knowledge(payload, ctx)
    assert rows
    segments = rows[0]["metadata"].get("context_segments", [])

    assert all("personal" not in seg.get("scopes", []) for seg in segments)
