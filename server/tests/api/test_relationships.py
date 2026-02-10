"""Relationship route tests."""

# Third-Party
import pytest


async def _make_entity(api, name="RelEntity"):
    """Make entity."""

    r = await api.post(
        "/api/entities",
        json={
            "name": name,
            "type": "person",
            "scopes": ["public"],
        },
    )
    return r.json()["data"]


@pytest.mark.asyncio
async def test_create_relationship(api):
    """Test create relationship."""

    e1 = await _make_entity(api, "Source")
    e2 = await _make_entity(api, "Target")

    r = await api.post(
        "/api/relationships",
        json={
            "source_type": "entity",
            "source_id": str(e1["id"]),
            "target_type": "entity",
            "target_id": str(e2["id"]),
            "relationship_type": "related-to",
        },
    )
    assert r.status_code == 200
    assert "id" in r.json()["data"]


@pytest.mark.asyncio
async def test_get_relationships(api):
    """Test get relationships."""

    e1 = await _make_entity(api, "GetSrc")
    e2 = await _make_entity(api, "GetTgt")

    await api.post(
        "/api/relationships",
        json={
            "source_type": "entity",
            "source_id": str(e1["id"]),
            "target_type": "entity",
            "target_id": str(e2["id"]),
            "relationship_type": "related-to",
        },
    )

    r = await api.get(f"/api/relationships/entity/{e1['id']}")
    assert r.status_code == 200
    assert len(r.json()["data"]) >= 1


@pytest.mark.asyncio
async def test_query_relationships(api):
    """Test query relationships."""

    e1 = await _make_entity(api, "QSrc")
    e2 = await _make_entity(api, "QTgt")

    await api.post(
        "/api/relationships",
        json={
            "source_type": "entity",
            "source_id": str(e1["id"]),
            "target_type": "entity",
            "target_id": str(e2["id"]),
            "relationship_type": "related-to",
        },
    )

    r = await api.get("/api/relationships", params={"source_type": "entity"})
    assert r.status_code == 200
    assert len(r.json()["data"]) >= 1


@pytest.mark.asyncio
async def test_update_relationship(api):
    """Test update relationship."""

    e1 = await _make_entity(api, "UpdSrc")
    e2 = await _make_entity(api, "UpdTgt")

    # use asymmetric type to avoid symmetric trigger recursion bug
    cr = await api.post(
        "/api/relationships",
        json={
            "source_type": "entity",
            "source_id": str(e1["id"]),
            "target_type": "entity",
            "target_id": str(e2["id"]),
            "relationship_type": "works-on",
        },
    )
    rel_id = cr.json()["data"]["id"]

    r = await api.patch(
        f"/api/relationships/{rel_id}",
        json={
            "properties": {"note": "updated"},
        },
    )
    assert r.status_code == 200


@pytest.mark.asyncio
async def test_get_relationships_direction_filter(api):
    """Test get relationships direction filter."""

    e1 = await _make_entity(api, "DirSrc")
    e2 = await _make_entity(api, "DirTgt")

    await api.post(
        "/api/relationships",
        json={
            "source_type": "entity",
            "source_id": str(e1["id"]),
            "target_type": "entity",
            "target_id": str(e2["id"]),
            "relationship_type": "related-to",
        },
    )

    r = await api.get(
        f"/api/relationships/entity/{e1['id']}", params={"direction": "outgoing"}
    )
    assert r.status_code == 200
