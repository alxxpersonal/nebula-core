"""Entity route tests."""

# Third-Party
import pytest


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
