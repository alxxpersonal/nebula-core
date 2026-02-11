"""Red team tests for invalid UUID handling in API routes."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_api_get_entity_rejects_invalid_uuid(api):
    """Invalid UUIDs should return a 400 or 404, not a 500."""

    resp = await api.get("/api/entities/not-a-uuid")
    assert resp.status_code in {400, 404}
