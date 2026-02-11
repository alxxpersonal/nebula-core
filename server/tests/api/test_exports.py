"""Export route tests."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_export_entities_json(api, test_entity):
    """Export entities in json format."""

    r = await api.get("/api/export/entities")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["format"] == "json"
    assert len(data["items"]) >= 1


@pytest.mark.asyncio
async def test_export_entities_csv(api, test_entity):
    """Export entities in csv format."""

    r = await api.get("/api/export/entities", params={"format": "csv"})
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["format"] == "csv"
    assert "name" in data["content"]


@pytest.mark.asyncio
async def test_export_knowledge_json(api):
    """Export knowledge in json format."""

    await api.post(
        "/api/knowledge",
        json={"title": "Export Knowledge", "source_type": "note", "scopes": ["public"]},
    )
    r = await api.get("/api/export/knowledge")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["format"] == "json"
    assert len(data["items"]) >= 1


@pytest.mark.asyncio
async def test_export_relationships_json(api):
    """Export relationships in json format."""

    r1 = await api.post(
        "/api/entities",
        json={"name": "ExportSource", "type": "person", "scopes": ["public"]},
    )
    r2 = await api.post(
        "/api/entities",
        json={"name": "ExportTarget", "type": "person", "scopes": ["public"]},
    )
    await api.post(
        "/api/relationships",
        json={
            "source_type": "entity",
            "source_id": r1.json()["data"]["id"],
            "target_type": "entity",
            "target_id": r2.json()["data"]["id"],
            "relationship_type": "related-to",
        },
    )
    r = await api.get("/api/export/relationships")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["format"] == "json"
    assert len(data["items"]) >= 1


@pytest.mark.asyncio
async def test_export_context_json(api):
    """Export full context in json format."""

    r = await api.get("/api/export/context")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["format"] == "json"
    assert "entities" in data
    assert "knowledge" in data
