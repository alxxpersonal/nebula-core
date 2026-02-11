"""Red team tests for invalid scope handling in export routes."""

# Third-Party
import pytest


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid scopes raise ValueError without 400 handling")
async def test_export_entities_rejects_invalid_scope(api):
    """Invalid scope names should not crash entity export."""

    resp = await api.get("/api/export/entities", params={"scopes": "not-a-scope"})
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid scopes raise ValueError without 400 handling")
async def test_export_knowledge_rejects_invalid_scope(api):
    """Invalid scope names should not crash knowledge export."""

    resp = await api.get("/api/export/knowledge", params={"scopes": "not-a-scope"})
    assert resp.status_code in {400, 404}
