"""Context route tests."""

# Third-Party
import pytest

# Local
from nebula_mcp.models import MAX_TAGS


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


@pytest.mark.asyncio
async def test_create_context_validation_errors(api):
    """Create route should reject invalid URL, tags, and scopes."""

    invalid_url = await api.post(
        "/api/context",
        json={"title": "BadUrl", "url": "ftp://bad", "scopes": ["public"]},
    )
    assert invalid_url.status_code == 422

    too_many_tags = await api.post(
        "/api/context",
        json={
            "title": "TooManyTags",
            "scopes": ["public"],
            "tags": [f"t{i}" for i in range(MAX_TAGS + 1)],
        },
    )
    assert too_many_tags.status_code == 422

    bad_scope = await api.post(
        "/api/context",
        json={"title": "BadScope", "scopes": ["not-a-scope"]},
    )
    assert bad_scope.status_code == 400


@pytest.mark.asyncio
async def test_get_context_validation_and_not_found(api):
    """Get route should validate context ids."""

    invalid = await api.get("/api/context/not-a-uuid")
    assert invalid.status_code == 400

    missing = await api.get("/api/context/00000000-0000-0000-0000-000000000001")
    assert missing.status_code == 404


@pytest.mark.asyncio
async def test_link_context_validation_and_relationship_type_errors(api):
    """Link route should reject invalid ids and unknown relationship types."""

    context = (
        await api.post("/api/context", json={"title": "CtxLink", "scopes": ["public"]})
    ).json()["data"]
    entity = (
        await api.post(
            "/api/entities",
            json={"name": "EntityLink", "type": "person", "scopes": ["public"]},
        )
    ).json()["data"]

    invalid_ids = await api.post(
        f"/api/context/{context['id']}/link",
        json={"entity_id": "not-a-uuid", "relationship_type": "related-to"},
    )
    assert invalid_ids.status_code == 400

    bad_rel_type = await api.post(
        f"/api/context/{context['id']}/link",
        json={"entity_id": entity["id"], "relationship_type": "does-not-exist"},
    )
    assert bad_rel_type.status_code == 400


@pytest.mark.asyncio
async def test_update_context_validation_errors(api):
    """Update route should validate ids, URL, status, and scopes."""

    context = (
        await api.post(
            "/api/context",
            json={"title": "CtxUpdate", "url": "https://ok", "scopes": ["public"]},
        )
    ).json()["data"]

    bad_id = await api.patch("/api/context/not-a-uuid", json={"title": "X"})
    assert bad_id.status_code == 400

    bad_url = await api.patch(f"/api/context/{context['id']}", json={"url": "file://bad"})
    assert bad_url.status_code == 422

    bad_status = await api.patch(
        f"/api/context/{context['id']}",
        json={"status": "does-not-exist"},
    )
    assert bad_status.status_code == 400

    bad_scope = await api.patch(
        f"/api/context/{context['id']}",
        json={"scopes": ["invalid-scope"]},
    )
    assert bad_scope.status_code == 400


@pytest.mark.asyncio
async def test_update_context_agent_scope_subset_enforced(api_agent_auth):
    """Agent updates should reject scope expansion outside caller scopes."""

    created = await api_agent_auth.post(
        "/api/context",
        json={"title": "AgentCtx", "scopes": ["public"]},
    )
    assert created.status_code == 200
    context_id = created.json()["data"]["id"]

    expanded = await api_agent_auth.patch(
        f"/api/context/{context_id}",
        json={"scopes": ["public", "sensitive"]},
    )
    assert expanded.status_code == 400
