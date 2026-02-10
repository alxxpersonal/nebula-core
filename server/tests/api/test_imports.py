"""Bulk import route tests."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_import_entities_json(api):
    payload = {
        "format": "json",
        "items": [
            {
                "name": "Import Entity",
                "type": "person",
                "scopes": ["public"],
                "tags": ["import"],
            }
        ],
    }
    r = await api.post("/api/import/entities", json=payload)
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["created"] == 1
    assert data["failed"] == 0


@pytest.mark.asyncio
async def test_import_entities_csv(api):
    payload = {
        "format": "csv",
        "data": "name,type,scopes,tags\nCSV Entity,person,public,import",
    }
    r = await api.post("/api/import/entities", json=payload)
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["created"] == 1


@pytest.mark.asyncio
async def test_import_knowledge_json(api):
    payload = {
        "format": "json",
        "items": [
            {
                "title": "Import Knowledge",
                "source_type": "note",
                "scopes": ["public"],
                "content": "test",
            }
        ],
    }
    r = await api.post("/api/import/knowledge", json=payload)
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["created"] == 1


@pytest.mark.asyncio
async def test_import_relationships_json(api):
    r1 = await api.post(
        "/api/entities",
        json={"name": "ImportSource", "type": "person", "scopes": ["public"]},
    )
    r2 = await api.post(
        "/api/entities",
        json={"name": "ImportTarget", "type": "person", "scopes": ["public"]},
    )
    source_id = r1.json()["data"]["id"]
    target_id = r2.json()["data"]["id"]

    payload = {
        "format": "json",
        "items": [
            {
                "source_type": "entity",
                "source_id": source_id,
                "target_type": "entity",
                "target_id": target_id,
                "relationship_type": "related-to",
            }
        ],
    }
    r = await api.post("/api/import/relationships", json=payload)
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["created"] == 1


@pytest.mark.asyncio
async def test_import_jobs_json(api):
    payload = {
        "format": "json",
        "items": [
            {
                "title": "Import Job",
                "description": "test",
                "priority": "high",
            }
        ],
    }
    r = await api.post("/api/import/jobs", json=payload)
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["created"] == 1
