"""Red team tests for invalid UUID handling in audit API routes."""

# Third-Party
import pytest


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_audit_rejects_invalid_actor_id(api):
    """Invalid UUIDs should not crash audit list routes."""

    resp = await api.get("/api/audit", params={"actor_id": "not-a-uuid"})
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_audit_rejects_invalid_scope_id(api):
    """Invalid UUIDs should not crash audit list routes."""

    resp = await api.get("/api/audit", params={"scope_id": "not-a-uuid"})
    assert resp.status_code in {400, 404}
