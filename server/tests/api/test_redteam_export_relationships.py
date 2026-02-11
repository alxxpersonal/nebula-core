"""Red team tests for export relationships."""

# Standard Library

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_export_relationships_default(api):
    """Export relationships should return a response."""

    resp = await api.get("/api/export/relationships")
    assert resp.status_code == 200
