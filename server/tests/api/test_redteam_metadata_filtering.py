"""Red team API tests for metadata privacy filtering in queries."""

# Standard Library
import json

# Third-Party
import pytest


async def _make_entity(db_pool, enums, name, scopes, metadata):
    """Insert a test entity for metadata filtering scenarios."""

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
        json.dumps(metadata),
    )
    return dict(row)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="query_entities should filter context segments by auth scopes")
async def test_api_query_entities_filters_context_segments(api, db_pool, enums):
    """API query results should not include context segments outside scopes."""

    metadata = {
        "context_segments": [
            {"text": "public info", "scopes": ["public"]},
            {"text": "private info", "scopes": ["personal"]},
        ]
    }
    await _make_entity(db_pool, enums, "Mixed Scope", ["public", "personal"], metadata)

    resp = await api.get("/api/entities")
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data
    segments = data[0]["metadata"].get("context_segments", [])

    assert all("personal" not in seg.get("scopes", []) for seg in segments)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="metadata search should filter context segments by auth scopes")
async def test_api_search_entities_filters_context_segments(api, db_pool, enums):
    """API metadata search should not leak context segments outside scopes."""

    metadata = {
        "context_segments": [
            {"text": "public info", "scopes": ["public"]},
            {"text": "private info", "scopes": ["personal"]},
        ],
        "signal": "needle",
    }
    await _make_entity(db_pool, enums, "Metadata Leak", ["public", "personal"], metadata)

    resp = await api.post(
        "/api/entities/search",
        json={"metadata_query": {"signal": "needle"}},
    )
    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data
    segments = data[0]["metadata"].get("context_segments", [])

    assert all("personal" not in seg.get("scopes", []) for seg in segments)
