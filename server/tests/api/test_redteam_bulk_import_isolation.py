"""Red team API tests for bulk import isolation."""

# Standard Library
import json

# Third-Party
from httpx import ASGITransport, AsyncClient
import pytest

# Local
from nebula_api.app import app
from nebula_api.auth import require_auth


async def _make_agent(db_pool, enums, name, scopes, requires_approval):
    """Insert an agent for bulk import tests."""

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
    """Insert an entity for bulk import tests."""

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
        json.dumps({"note": "seed"}),
    )
    return dict(row)


def _auth_override(agent_id, scopes, enums):
    """Build auth override for agent scoped requests."""

    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]
    auth_dict = {
        "key_id": None,
        "caller_type": "agent",
        "entity_id": None,
        "entity": None,
        "agent_id": agent_id,
        "agent": {"id": agent_id, "requires_approval": False},
        "scopes": scope_ids,
    }

    async def mock_auth():
        """Mock auth for public agent."""

        return auth_dict

    return mock_auth


@pytest.mark.asyncio
@pytest.mark.xfail(reason="bulk import should enforce scope subset for agents")
async def test_bulk_import_entities_scope_escalation(db_pool, enums):
    """Agents should not bulk import entities with private scopes."""

    agent = await _make_agent(db_pool, enums, "bulk-import-viewer", ["public"], False)
    app.dependency_overrides[require_auth] = _auth_override(agent["id"], ["public"], enums)
    app.state.pool = db_pool
    app.state.enums = enums

    payload = {
        "format": "json",
        "items": [
            {
                "name": "Escalated Entity",
                "type": "person",
                "status": "active",
                "scopes": ["personal"],
                "tags": ["redteam"],
                "metadata": {"note": "private"},
            }
        ],
    }

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post("/api/import/entities", json=payload)
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    scopes = resp.json()["data"]["items"][0]["scopes"]
    assert "personal" not in scopes


@pytest.mark.asyncio
@pytest.mark.xfail(reason="bulk import jobs should not allow agent_id spoofing")
async def test_bulk_import_jobs_agent_spoofing(db_pool, enums):
    """Agents should not bulk import jobs for other agents."""

    owner = await _make_agent(db_pool, enums, "bulk-owner", ["public"], False)
    viewer = await _make_agent(db_pool, enums, "bulk-viewer", ["public"], False)

    app.dependency_overrides[require_auth] = _auth_override(viewer["id"], ["public"], enums)
    app.state.pool = db_pool
    app.state.enums = enums

    payload = {
        "format": "json",
        "items": [
            {
                "title": "Spoofed Job",
                "agent_id": str(owner["id"]),
                "priority": "high",
                "metadata": {"note": "spoof"},
            }
        ],
    }

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post("/api/import/jobs", json=payload)
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    job = resp.json()["data"]["items"][0]
    assert job["agent_id"] == str(viewer["id"])


@pytest.mark.asyncio
@pytest.mark.xfail(reason="bulk import relationships should enforce target scopes")
async def test_bulk_import_relationships_private_target(db_pool, enums):
    """Agents should not bulk import relationships to private entities."""

    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    public_entity = await _make_entity(db_pool, enums, "Public", ["public"])
    viewer = await _make_agent(db_pool, enums, "bulk-linker", ["public"], False)

    app.dependency_overrides[require_auth] = _auth_override(viewer["id"], ["public"], enums)
    app.state.pool = db_pool
    app.state.enums = enums

    payload = {
        "format": "json",
        "items": [
            {
                "source_type": "entity",
                "source_id": str(public_entity["id"]),
                "target_type": "entity",
                "target_id": str(private_entity["id"]),
                "relationship_type": "related-to",
            }
        ],
    }

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post("/api/import/relationships", json=payload)
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    rel = resp.json()["data"]["items"][0]
    assert rel["target_id"] != str(private_entity["id"])


@pytest.mark.asyncio
@pytest.mark.xfail(reason="bulk import knowledge should enforce scope subset for agents")
async def test_bulk_import_knowledge_scope_escalation(db_pool, enums):
    """Agents should not bulk import knowledge with private scopes."""

    agent = await _make_agent(db_pool, enums, "bulk-knowledge-viewer", ["public"], False)
    app.dependency_overrides[require_auth] = _auth_override(agent["id"], ["public"], enums)
    app.state.pool = db_pool
    app.state.enums = enums

    payload = {
        "format": "json",
        "items": [
            {
                "title": "Escalated Knowledge",
                "source_type": "note",
                "content": "secret",
                "scopes": ["personal"],
                "tags": ["redteam"],
                "metadata": {"note": "private"},
            }
        ],
    }

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post("/api/import/knowledge", json=payload)
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 200
    scopes = resp.json()["data"]["items"][0]["scopes"]
    assert "personal" not in scopes
