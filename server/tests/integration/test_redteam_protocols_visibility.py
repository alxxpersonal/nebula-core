"""Red team tests for MCP protocol visibility controls."""

# Standard Library
import json

# Third-Party
import pytest

# Local
from nebula_mcp.models import GetProtocolInput
from nebula_mcp.server import get_protocol


async def _make_trusted_protocol(db_pool, enums, name: str) -> None:
    """Insert a trusted protocol row for MCP visibility tests."""

    status_id = enums.statuses.name_to_id["active"]
    await db_pool.execute(
        """
        INSERT INTO protocols (
            name,
            title,
            version,
            content,
            protocol_type,
            applies_to,
            status_id,
            tags,
            trusted,
            metadata,
            vault_file_path
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, $9::jsonb, $10)
        """,
        name,
        "Trusted Internal Protocol",
        "1.0.0",
        "internal system prompt material",
        "system",
        ["agents"],
        status_id,
        ["internal"],
        json.dumps({"classification": "internal"}),
        None,
    )


@pytest.mark.asyncio
async def test_non_admin_agent_cannot_read_trusted_protocol_content(
    db_pool,
    enums,
    untrusted_mcp_context,
):
    """Non-admin agents should not fetch trusted protocol content by name."""

    protocol_name = "rt-trusted-protocol-agent-read"
    await _make_trusted_protocol(db_pool, enums, protocol_name)

    payload = GetProtocolInput(protocol_name=protocol_name)
    with pytest.raises(ValueError, match="Access denied"):
        await get_protocol(payload, untrusted_mcp_context)
