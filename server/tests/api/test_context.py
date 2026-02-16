"""Context route tests."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_create_context(api):
    """Test create context."""

    r = await api.post(
        "/api/context",
        json={
            "title": "Test Article",
            "url": "https://example.com/article",
            "source_type": "article",
            "content": "some content",
            "scopes": ["public"],
            "tags": ["test"],
        },
    )
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["title"] == "Test Article"


@pytest.mark.asyncio
async def test_query_context(api):
    """Test query context."""

    await api.post(
        "/api/context",
        json={
            "title": "QueryContext",
            "source_type": "video",
            "scopes": ["public"],
        },
    )
    r = await api.get("/api/context", params={"source_type": "video"})
    assert r.status_code == 200
    data = r.json()["data"]
    assert len(data) >= 1


@pytest.mark.asyncio
async def test_link_context_to_entity(api):
    """Test link context to entity."""

    kr = await api.post(
        "/api/context",
        json={
            "title": "LinkTest",
            "scopes": ["public"],
        },
    )
    k_id = kr.json()["data"]["id"]

    er = await api.post(
        "/api/entities",
        json={
            "name": "LinkTarget",
            "type": "person",
            "scopes": ["public"],
        },
    )
    e_id = er.json()["data"]["id"]

    r = await api.post(
        f"/api/context/{k_id}/link",
        json={
            "entity_id": str(e_id),
            "relationship_type": "related-to",
        },
    )
    assert r.status_code == 200


@pytest.mark.asyncio
async def test_query_context_pagination(api):
    """Test query context pagination."""

    for i in range(3):
        await api.post(
            "/api/context",
            json={
                "title": f"KPage{i}",
                "scopes": ["public"],
            },
        )
    r = await api.get("/api/context", params={"limit": 2})
    assert r.status_code == 200
    meta = r.json()["meta"]
    assert meta["limit"] == 2
