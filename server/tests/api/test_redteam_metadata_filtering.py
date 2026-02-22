"""Red team API tests for metadata privacy filtering in queries."""

# Standard Library
import json

# Third-Party
from httpx import ASGITransport, AsyncClient
import pytest

# Local
from nebula_api.app import app
from nebula_api.auth import require_auth


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


def _public_user_auth_override(test_entity, enums):
    """Build a user auth override that only has public scope."""

    public_scope = enums.scopes.name_to_id["public"]
    auth_dict = {
        "key_id": None,
        "caller_type": "user",
        "entity_id": test_entity["id"],
        "entity": test_entity,
        "agent_id": None,
        "agent": None,
        "scopes": [public_scope],
    }

    async def mock_auth():
        """Return public-only user auth context."""

        return auth_dict

    return mock_auth


@pytest.mark.asyncio
async def test_api_query_entities_filters_context_segments(
    db_pool, enums, test_entity
):
    """API query results should not include context segments outside scopes."""

    metadata = {
        "context_segments": [
            {"text": "public info", "scopes": ["public"]},
            {"text": "private info", "scopes": ["private"]},
        ]
    }
    await _make_entity(db_pool, enums, "Mixed Scope", ["public", "private"], metadata)

    app.state.pool = db_pool
    app.state.enums = enums
    app.dependency_overrides[require_auth] = _public_user_auth_override(
        test_entity, enums
    )
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/entities/")
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data
    segments = data[0]["metadata"].get("context_segments", [])

    assert all("private" not in seg.get("scopes", []) for seg in segments)


@pytest.mark.asyncio
async def test_api_search_entities_filters_context_segments(
    db_pool, enums, test_entity
):
    """API metadata search should not leak context segments outside scopes."""

    metadata = {
        "context_segments": [
            {"text": "public info", "scopes": ["public"]},
            {"text": "private info", "scopes": ["private"]},
        ],
        "signal": "needle",
    }
    await _make_entity(db_pool, enums, "Metadata Leak", ["public", "private"], metadata)

    app.state.pool = db_pool
    app.state.enums = enums
    app.dependency_overrides[require_auth] = _public_user_auth_override(
        test_entity, enums
    )
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post(
            "/api/entities/search",
            json={"metadata_query": {"signal": "needle"}},
        )
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data
    segments = data[0]["metadata"].get("context_segments", [])

    assert all("private" not in seg.get("scopes", []) for seg in segments)


@pytest.mark.asyncio
async def test_api_search_entities_hides_private_entities(
    db_pool, enums, test_entity
):
    """API metadata search should not return private-only entities."""

    metadata = {"signal": "private-only"}
    await _make_entity(db_pool, enums, "Private Node", ["private"], metadata)

    app.state.pool = db_pool
    app.state.enums = enums
    app.dependency_overrides[require_auth] = _public_user_auth_override(
        test_entity, enums
    )
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post(
            "/api/entities/search",
            json={"metadata_query": {"signal": "private-only"}},
        )
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    data = resp.json()["data"]

    assert not data
