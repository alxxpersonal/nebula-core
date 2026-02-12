"""Red team tests for invalid UUID handling in MCP resource tools."""

# Third-Party
import pytest

# Local
from nebula_mcp.models import (
    GetFileInput,
    GetLogInput,
    LinkKnowledgeInput,
    UpdateLogInput,
)
from nebula_mcp.server import get_file, get_log, link_knowledge_to_entity, update_log


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_get_log_rejects_invalid_uuid(untrusted_mcp_context):
    """Invalid UUIDs should not crash get_log."""

    payload = GetLogInput(log_id="not-a-uuid")
    with pytest.raises(ValueError):
        await get_log(payload, untrusted_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs are accepted without validation")
async def test_update_log_rejects_invalid_uuid(untrusted_mcp_context):
    """Invalid UUIDs should not crash update_log."""

    payload = UpdateLogInput(id="not-a-uuid", metadata={"note": "bad"})
    with pytest.raises(ValueError):
        await update_log(payload, untrusted_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_get_file_rejects_invalid_uuid(untrusted_mcp_context):
    """Invalid UUIDs should not crash get_file."""

    payload = GetFileInput(file_id="not-a-uuid")
    with pytest.raises(ValueError):
        await get_file(payload, untrusted_mcp_context)


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_link_knowledge_rejects_invalid_uuid(untrusted_mcp_context):
    """Invalid UUIDs should not crash link_knowledge_to_entity."""

    payload = LinkKnowledgeInput(
        knowledge_id="not-a-uuid",
        entity_id="also-not-a-uuid",
        relationship_type="about",
    )
    with pytest.raises(ValueError):
        await link_knowledge_to_entity(payload, untrusted_mcp_context)
