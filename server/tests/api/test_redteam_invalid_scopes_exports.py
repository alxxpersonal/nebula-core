"""Red team tests for invalid scope handling in export routes."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_export_entities_rejects_invalid_scope(api):
    """Invalid scope names should not crash entity export."""

    resp = await api.get("/api/export/entities", params={"scopes": "not-a-scope"})
    assert resp.status_code == 400
    body = resp.json()
    assert body["detail"]["error"]["code"] == "VALIDATION_ERROR"


@pytest.mark.asyncio
async def test_export_knowledge_rejects_invalid_scope(api):
    """Invalid scope names should not crash knowledge export."""

    resp = await api.get("/api/export/knowledge", params={"scopes": "not-a-scope"})
    assert resp.status_code == 400
    body = resp.json()
    assert body["detail"]["error"]["code"] == "VALIDATION_ERROR"


@pytest.mark.asyncio
async def test_export_entities_rejects_invalid_type(api):
    """Invalid entity type should not crash entity export."""

    resp = await api.get("/api/export/entities", params={"type": "not-a-type"})
    assert resp.status_code == 400
    body = resp.json()
    assert body["detail"]["error"]["code"] == "VALIDATION_ERROR"
