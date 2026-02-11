"""Red team tests for invalid UUID handling in MCP tools."""

# Third-Party
import pytest

# Local
from nebula_mcp.models import GetEntityInput
from nebula_mcp.server import get_entity


@pytest.mark.asyncio
async def test_get_entity_rejects_invalid_uuid(untrusted_mcp_context):
    """Invalid UUIDs should return a clean validation error."""

    payload = GetEntityInput(entity_id="not-a-uuid")
    with pytest.raises(ValueError):
        await get_entity(payload, untrusted_mcp_context)
