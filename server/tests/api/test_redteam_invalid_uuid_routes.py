"""Red team tests for invalid UUID handling in API write routes."""

# Third-Party
import pytest


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_update_entity_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash update entity routes."""

    resp = await api.patch(
        "/api/entities/not-a-uuid",
        json={"metadata": {"note": "bad"}},
    )
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_get_relationships_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash relationship list routes."""

    resp = await api.get("/api/relationships/entity/not-a-uuid")
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_update_relationship_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash relationship update routes."""

    resp = await api.patch(
        "/api/relationships/not-a-uuid",
        json={"properties": {"note": "bad"}},
    )
    assert resp.status_code in {400, 404}
