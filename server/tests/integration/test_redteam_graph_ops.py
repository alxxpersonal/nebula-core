"""Red team graph operation edge cases."""

# Standard Library
import json

# Third-Party
import pytest

# Local
from nebula_mcp.models import GraphNeighborsInput, GraphShortestPathInput
from nebula_mcp.server import graph_neighbors, graph_shortest_path


async def _make_entity(db_pool, enums, name):
    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.entity_types.name_to_id["person"]
    scope_ids = [enums.scopes.name_to_id["public"]]
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
        json.dumps({}),
    )
    return dict(row)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="graph_neighbors should allow limit=0 safely")
async def test_graph_neighbors_zero_limit(db_pool, enums, mock_mcp_context):
    """Limit=0 should return empty list not crash."""

    node = await _make_entity(db_pool, enums, "Node")
    payload = GraphNeighborsInput(
        source_type="entity",
        source_id=str(node["id"]),
        max_hops=1,
        limit=0,
    )

    rows = await graph_neighbors(payload, mock_mcp_context)
    assert rows == []


@pytest.mark.asyncio
@pytest.mark.xfail(reason="graph_neighbors should reject invalid node types")
async def test_graph_neighbors_invalid_type(db_pool, enums, mock_mcp_context):
    """Invalid node type should be rejected."""

    node = await _make_entity(db_pool, enums, "Node")
    payload = GraphNeighborsInput(
        source_type="invalid",
        source_id=str(node["id"]),
        max_hops=1,
        limit=10,
    )

    with pytest.raises(ValueError):
        await graph_neighbors(payload, mock_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="graph_shortest_path should reject invalid node types")
async def test_graph_shortest_path_invalid_type(db_pool, enums, mock_mcp_context):
    """Invalid node type should be rejected."""

    node = await _make_entity(db_pool, enums, "Node")
    payload = GraphShortestPathInput(
        source_type="invalid",
        source_id=str(node["id"]),
        target_type="entity",
        target_id=str(node["id"]),
        max_hops=2,
    )

    with pytest.raises(ValueError):
        await graph_shortest_path(payload, mock_mcp_context)
