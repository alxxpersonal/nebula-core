"""Entity route tests."""

# Third-Party
from httpx import ASGITransport, AsyncClient
import pytest

# Local
from nebula_api.app import app
from nebula_api.auth import require_auth


@pytest.mark.asyncio
async def test_create_entity(api):
    """Test create entity."""

    r = await api.post(
        "/api/entities",
        json={
            "name": "New Entity",
            "type": "person",
            "scopes": ["public"],
            "tags": ["test-tag"],
        },
    )
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["name"] == "New Entity"
    assert "id" in data


@pytest.mark.asyncio
async def test_get_entity(api, test_entity):
    """Test get entity."""

    r = await api.get(f"/api/entities/{test_entity['id']}")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["name"] == "api-test-user"


@pytest.mark.asyncio
async def test_get_entity_not_found(api):
    """Test get entity not found."""

    r = await api.get("/api/entities/00000000-0000-0000-0000-000000000000")
    assert r.status_code == 404


@pytest.mark.asyncio
async def test_get_entity_invalid_id_returns_400(api):
    """Entity get should reject malformed ids."""

    r = await api.get("/api/entities/not-a-uuid")
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "INVALID_INPUT"


@pytest.mark.asyncio
async def test_query_entities(api):
    """Test query entities."""

    await api.post(
        "/api/entities",
        json={"name": "QueryTest", "type": "person", "scopes": ["public"]},
    )
    r = await api.get("/api/entities", params={"type": "person"})
    assert r.status_code == 200
    data = r.json()["data"]
    assert len(data) >= 1


@pytest.mark.asyncio
async def test_create_entity_invalid_status_returns_400(api):
    """Entity create should reject unknown statuses."""

    r = await api.post(
        "/api/entities",
        json={
            "name": "Bad Status Entity",
            "type": "person",
            "status": "todo",
            "scopes": ["public"],
        },
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "INVALID_INPUT"


@pytest.mark.asyncio
async def test_create_entity_invalid_scope_returns_400(api):
    """Entity create should reject unknown scope names."""

    r = await api.post(
        "/api/entities",
        json={
            "name": "Bad Scope Entity",
            "type": "person",
            "status": "active",
            "scopes": ["does-not-exist"],
        },
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "INVALID_INPUT"


@pytest.mark.asyncio
async def test_update_entity(api, test_entity):
    """Test update entity."""

    r = await api.patch(
        f"/api/entities/{test_entity['id']}",
        json={
            "tags": ["updated"],
            "status": "active",
        },
    )
    assert r.status_code == 200


@pytest.mark.asyncio
async def test_update_entity_invalid_id_returns_400(api):
    """Entity update should reject malformed entity ids."""

    r = await api.patch(
        "/api/entities/not-a-uuid",
        json={"status": "active"},
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "INVALID_INPUT"


@pytest.mark.asyncio
async def test_update_entity_invalid_status_returns_400(api, test_entity):
    """Entity update should reject unknown statuses."""

    r = await api.patch(
        f"/api/entities/{test_entity['id']}",
        json={"status": "todo"},
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "INVALID_INPUT"


@pytest.mark.asyncio
async def test_bulk_update_tags_requires_entity_ids(api):
    """Bulk tag updates should fail without entity ids."""

    r = await api.post(
        "/api/entities/bulk/tags",
        json={"entity_ids": [], "tags": ["x"], "op": "add"},
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "VALIDATION_ERROR"


@pytest.mark.asyncio
async def test_bulk_update_tags_requires_tags_for_add(api, test_entity):
    """Bulk tag add should fail when tag list is empty."""

    r = await api.post(
        "/api/entities/bulk/tags",
        json={"entity_ids": [str(test_entity["id"])], "tags": [], "op": "add"},
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "VALIDATION_ERROR"


@pytest.mark.asyncio
async def test_bulk_update_scopes_requires_entity_ids(api):
    """Bulk scope updates should fail without entity ids."""

    r = await api.post(
        "/api/entities/bulk/scopes",
        json={"entity_ids": [], "scopes": ["public"], "op": "add"},
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "VALIDATION_ERROR"


@pytest.mark.asyncio
async def test_bulk_update_scopes_invalid_scope_returns_400(api, test_entity):
    """Bulk scope updates should reject invalid scope names."""

    r = await api.post(
        "/api/entities/bulk/scopes",
        json={
            "entity_ids": [str(test_entity["id"])],
            "scopes": ["does-not-exist"],
            "op": "add",
        },
    )
    assert r.status_code == 400
    assert r.json()["detail"]["error"]["code"] == "INVALID_INPUT"


@pytest.mark.asyncio
async def test_entity_history_not_found_returns_404(api):
    """History endpoint should return 404 for missing entities."""

    r = await api.get("/api/entities/00000000-0000-0000-0000-000000000000/history")
    assert r.status_code == 404


@pytest.mark.asyncio
async def test_revert_entity_forbidden_for_agents(db_pool, enums, test_entity):
    """Entity revert should be blocked for agent callers."""

    status_id = enums.statuses.name_to_id["active"]
    public_scope = enums.scopes.name_to_id["public"]
    agent = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        "revert-agent",
        "agent",
        [public_scope],
        False,
        status_id,
    )

    async def mock_auth():
        return {
            "key_id": None,
            "caller_type": "agent",
            "entity_id": None,
            "entity": None,
            "agent_id": agent["id"],
            "agent": dict(agent),
            "scopes": [public_scope],
        }

    app.dependency_overrides[require_auth] = mock_auth
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(
        transport=transport, base_url="http://test", follow_redirects=True
    ) as client:
        r = await client.post(
            f"/api/entities/{test_entity['id']}/revert",
            json={"audit_id": "00000000-0000-0000-0000-000000000000"},
        )
    app.dependency_overrides.pop(require_auth, None)

    assert r.status_code == 403
    assert r.json()["detail"]["error"]["code"] == "FORBIDDEN"


@pytest.mark.asyncio
async def test_search_by_metadata(api):
    """Test search by metadata."""

    await api.post(
        "/api/entities",
        json={
            "name": "SearchTarget",
            "type": "person",
            "scopes": ["public"],
            "metadata": {"role": "engineer"},
        },
    )
    r = await api.post(
        "/api/entities/search",
        json={
            "metadata_query": {"role": "engineer"},
        },
    )
    assert r.status_code == 200
    data = r.json()["data"]
    assert any(d["name"] == "SearchTarget" for d in data)


@pytest.mark.asyncio
async def test_query_with_pagination(api):
    """Test query with pagination."""

    for i in range(3):
        await api.post(
            "/api/entities",
            json={"name": f"Page{i}", "type": "person", "scopes": ["public"]},
        )
    r = await api.get("/api/entities", params={"limit": 2, "offset": 0})
    assert r.status_code == 200
    meta = r.json()["meta"]
    assert meta["limit"] == 2
    assert meta["offset"] == 0
