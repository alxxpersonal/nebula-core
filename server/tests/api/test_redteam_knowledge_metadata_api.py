"""Red team API tests for knowledge metadata privacy filtering."""

# Standard Library
import json

# Third-Party
from httpx import ASGITransport, AsyncClient
import pytest

# Local
from nebula_api.app import app
from nebula_api.auth import require_auth


async def _make_agent(db_pool, enums, name):
    """Insert a test agent for knowledge metadata scenarios."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id["public"]]

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


async def _make_knowledge(db_pool, enums, title, scopes, metadata):
    """Insert a test knowledge item for metadata filtering tests."""

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


def _auth_override(agent_id, enums):
    """Build an auth override for public agent requests."""

    auth_dict = {
        "key_id": None,
        "caller_type": "agent",
        "entity_id": None,
        "entity": None,
        "agent_id": agent_id,
        "agent": {"id": agent_id},
        "scopes": [enums.scopes.name_to_id["public"]],
    }

    async def mock_auth():
        """Mock auth for public agent."""

        return auth_dict

    return mock_auth


@pytest.mark.asyncio
async def test_api_query_knowledge_filters_context_segments(db_pool, enums):
    """API query results should not include context segments outside scopes."""

    metadata = {
        "context_segments": [
            {"text": "public info", "scopes": ["public"]},
            {"text": "private info", "scopes": ["personal"]},
        ]
    }
    knowledge = await _make_knowledge(db_pool, enums, "Mixed Scope", ["public", "personal"], metadata)
    agent = await _make_agent(db_pool, enums, "knowledge-viewer")

    app.dependency_overrides[require_auth] = _auth_override(agent["id"], enums)
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(
        transport=transport, base_url="http://test", follow_redirects=True
    ) as client:
        resp = await client.get("/api/knowledge/")
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    data = resp.json()["data"]
    assert data
    segments = data[0]["metadata"].get("context_segments", [])

    assert all("personal" not in seg.get("scopes", []) for seg in segments)


@pytest.mark.asyncio
async def test_api_get_knowledge_filters_context_segments(db_pool, enums):
    """API get should not include context segments outside scopes."""

    metadata = {
        "context_segments": [
            {"text": "public info", "scopes": ["public"]},
            {"text": "private info", "scopes": ["personal"]},
        ]
    }
    knowledge = await _make_knowledge(db_pool, enums, "Mixed Scope", ["public", "personal"], metadata)
    agent = await _make_agent(db_pool, enums, "knowledge-viewer-2")

    app.dependency_overrides[require_auth] = _auth_override(agent["id"], enums)
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get(f"/api/knowledge/{knowledge['id']}")
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    segments = resp.json()["data"]["metadata"].get("context_segments", [])
    assert all("personal" not in seg.get("scopes", []) for seg in segments)
