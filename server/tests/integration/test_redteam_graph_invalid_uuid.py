"""Red team tests for invalid UUID handling in graph tools."""

# Standard Library
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import GraphNeighborsInput, GraphShortestPathInput
from nebula_mcp.server import graph_neighbors, graph_shortest_path


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
    """Insert an agent for invalid UUID graph tests."""

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
async def test_graph_neighbors_rejects_invalid_uuid(db_pool, enums):
    """Graph neighbors should reject malformed UUIDs cleanly."""

    agent = await _make_agent(db_pool, enums, "graph-invalid-uuid", ["public"])
    ctx = _make_context(db_pool, enums, agent)

    payload = GraphNeighborsInput(
        source_type="entity",
        source_id="not-a-uuid",
        max_hops=2,
        limit=10,
    )

    with pytest.raises(ValueError):
        await graph_neighbors(payload, ctx)


@pytest.mark.asyncio
async def test_graph_shortest_path_rejects_invalid_uuid(db_pool, enums):
    """Graph shortest path should reject malformed UUIDs cleanly."""

    agent = await _make_agent(db_pool, enums, "graph-invalid-uuid-2", ["public"])
    ctx = _make_context(db_pool, enums, agent)

    payload = GraphShortestPathInput(
        source_type="entity",
        source_id="not-a-uuid",
        target_type="entity",
        target_id="also-not-a-uuid",
        max_hops=3,
    )

    with pytest.raises(ValueError):
        await graph_shortest_path(payload, ctx)
