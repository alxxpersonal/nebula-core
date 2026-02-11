"""Red team tests for graph traversal privacy leaks."""

# Standard Library
import json

# Third-Party
import pytest

# Local
from nebula_mcp.models import GraphNeighborsInput
from nebula_mcp.server import graph_neighbors


@pytest.mark.asyncio
@pytest.mark.xfail(reason="graph traversal should respect privacy scopes")
async def test_graph_neighbors_hides_private_nodes(
    db_pool, enums, test_entity, untrusted_mcp_context
):
    """Graph traversal should not expose nodes outside agent scopes."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.entity_types.name_to_id["person"]
    private_scope_id = enums.scopes.name_to_id["sensitive"]

    private_entity = await db_pool.fetchrow(
        """
        INSERT INTO entities (name, type_id, status_id, privacy_scope_ids, tags, metadata)
        VALUES ($1, $2, $3, $4, $5, $6::jsonb)
        RETURNING *
        """,
        "Private Node",
        type_id,
        status_id,
        [private_scope_id],
        ["private"],
        json.dumps({"context_segments": [{"text": "secret", "scopes": ["sensitive"]}]}),
    )

    relationship_type_id = enums.relationship_types.name_to_id["related-to"]

    await db_pool.execute(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id, properties)
        VALUES ('entity', $1, 'entity', $2, $3, $4, $5::jsonb)
        """,
        str(test_entity["id"]),
        str(private_entity["id"]),
        relationship_type_id,
        status_id,
        json.dumps({"note": "secret link"}),
    )

    payload = GraphNeighborsInput(
        source_type="entity",
        source_id=str(test_entity["id"]),
        max_hops=1,
        limit=10,
    )
    results = await graph_neighbors(payload, untrusted_mcp_context)
    leaked_ids = {row["node_id"] for row in results}

    assert str(private_entity["id"]) not in leaked_ids
