"""Red team tests for graph traversal privacy with knowledge nodes."""

# Standard Library
import json
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
    """Insert an agent for graph privacy tests."""

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
    """Insert an entity for graph privacy tests."""

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
    """Insert a knowledge node for graph privacy tests."""

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


async def _attach_relationship(
    db_pool, enums, source_type, source_id, target_type, target_id, rel_name
):
    """Attach relationship between graph nodes."""

    status_id = enums.statuses.name_to_id["active"]
    rel_type_id = enums.relationship_types.name_to_id[rel_name]

    await db_pool.execute(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id, properties)
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
        """,
        source_type,
        source_id,
        target_type,
        target_id,
        rel_type_id,
        status_id,
        json.dumps({"note": "graph link"}),
    )


@pytest.mark.asyncio
async def test_graph_neighbors_hides_private_knowledge(db_pool, enums):
    """Graph neighbors should not expose private knowledge nodes."""

    public_entity = await _make_entity(db_pool, enums, "Public Node", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private Node", ["sensitive"])
    knowledge = await _make_knowledge(db_pool, enums, "Secret Knowledge", ["sensitive"])

    await _attach_relationship(
        db_pool,
        enums,
        "knowledge",
        str(knowledge["id"]),
        "entity",
        str(public_entity["id"]),
        "related-to",
    )
    await _attach_relationship(
        db_pool,
        enums,
        "knowledge",
        str(knowledge["id"]),
        "entity",
        str(private_entity["id"]),
        "related-to",
    )

    viewer = await _make_agent(db_pool, enums, "graph-knowledge-viewer", ["public"])
    ctx = _make_context(db_pool, enums, viewer)

    payload = GraphNeighborsInput(
        source_type="entity",
        source_id=str(public_entity["id"]),
        max_hops=2,
        limit=10,
    )
    results = await graph_neighbors(payload, ctx)
    ids = {row["node_id"] for row in results}

    assert str(knowledge["id"]) not in ids


@pytest.mark.asyncio
async def test_graph_shortest_path_hides_private_knowledge(db_pool, enums):
    """Shortest path should not expose private knowledge nodes."""

    public_entity = await _make_entity(db_pool, enums, "Public Node", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private Node", ["sensitive"])
    knowledge = await _make_knowledge(db_pool, enums, "Secret Knowledge", ["sensitive"])

    await _attach_relationship(
        db_pool,
        enums,
        "knowledge",
        str(knowledge["id"]),
        "entity",
        str(public_entity["id"]),
        "related-to",
    )
    await _attach_relationship(
        db_pool,
        enums,
        "knowledge",
        str(knowledge["id"]),
        "entity",
        str(private_entity["id"]),
        "related-to",
    )

    viewer = await _make_agent(db_pool, enums, "path-knowledge-viewer", ["public"])
    ctx = _make_context(db_pool, enums, viewer)

    payload = GraphShortestPathInput(
        source_type="entity",
        source_id=str(public_entity["id"]),
        target_type="knowledge",
        target_id=str(knowledge["id"]),
        max_hops=3,
    )

    with pytest.raises(ValueError):
        await graph_shortest_path(payload, ctx)
