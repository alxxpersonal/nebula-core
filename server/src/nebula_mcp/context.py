"""Context extraction and validation helpers for MCP tools."""

# Standard Library
import os
from pathlib import Path
from typing import Any

# Third-Party
from asyncpg import Pool
from mcp.server.fastmcp import Context

# Local
from .db import get_agent
from .enums import EnumRegistry
from .query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[1] / "queries")

# --- Type Aliases ---

AgentDict = dict[str, Any]


async def require_context(ctx: Context) -> tuple[Pool, EnumRegistry, AgentDict]:
    """Extract pool, enums, and agent from context or raise.

    Args:
        ctx: MCP request context.

    Returns:
        Tuple of (pool, enums, agent) from lifespan context.

    Raises:
        ValueError: If pool, enums, or agent not initialized.
    """

    lifespan_ctx = ctx.request_context.lifespan_context

    if not lifespan_ctx or "pool" not in lifespan_ctx:
        raise ValueError("Pool not initialized")

    if "enums" not in lifespan_ctx:
        raise ValueError("Enums not initialized")

    if "agent" not in lifespan_ctx:
        raise ValueError("Agent not initialized")

    return lifespan_ctx["pool"], lifespan_ctx["enums"], lifespan_ctx["agent"]


async def require_pool(ctx: Context) -> Pool:
    """Extract pool from context when enums not needed.

    Args:
        ctx: MCP request context.

    Returns:
        Database connection pool.

    Raises:
        ValueError: If pool not initialized.
    """

    lifespan_ctx = ctx.request_context.lifespan_context

    if not lifespan_ctx or "pool" not in lifespan_ctx:
        raise ValueError("Pool not initialized")

    return lifespan_ctx["pool"]


async def require_agent(pool: Pool, agent_name: str) -> AgentDict:
    """Validate agent exists and is active.

    Args:
        pool: Database connection pool.
        agent_name: Agent name to validate.

    Returns:
        Agent row with id, scopes, requires_approval, etc.

    Raises:
        ValueError: If agent not found or inactive.
    """

    agent = await get_agent(pool, agent_name)

    if not agent:
        raise ValueError("Agent not found or inactive")

    return agent


async def maybe_require_approval(
    pool: Pool,
    agent: AgentDict,
    action: str,
    payload_dict: dict,
) -> dict | None:
    """Return approval response if agent requires it, else None.

    Checks agent trust level and routes to approval workflow if needed.
    Trusted agents return None to proceed with direct execution.

    Args:
        pool: Database connection pool.
        agent: Agent dict with requires_approval field.
        action: Action name (e.g., create_entity).
        payload_dict: Full payload for the action.

    Returns:
        Approval response if untrusted, None if trusted.
    """

    # Import here to avoid circular dependency
    from .helpers import create_approval_request

    if not agent.get("requires_approval", True):
        return None

    approval = await create_approval_request(
        pool,
        agent["id"],
        action,
        payload_dict,
        None,
    )

    return {
        "status": "approval_required",
        "approval_request_id": str(approval["id"]) if approval else None,
        "message": "Approval request created. Waiting for review.",
        "requested_action": action,
    }


async def authenticate_agent(pool: Pool) -> AgentDict:
    """Authenticate MCP server agent via NEBULA_API_KEY env var.

    Reads the API key from environment, validates against DB, loads agent row.

    Args:
        pool: Database connection pool.

    Returns:
        Agent row with id, name, scopes, requires_approval, etc.

    Raises:
        ValueError: If key missing, invalid, or agent inactive.
    """
    from pathlib import Path

    from argon2 import PasswordHasher
    from argon2.exceptions import VerifyMismatchError

    from .query_loader import QueryLoader

    QUERIES = QueryLoader(Path(__file__).resolve().parents[1] / "queries")
    ph = PasswordHasher()

    api_key = os.environ.get("NEBULA_API_KEY")
    if not api_key:
        raise ValueError(
            "NEBULA_API_KEY environment variable is required for "
            "MCP server authentication"
        )

    if len(api_key) < 8:
        raise ValueError("NEBULA_API_KEY is too short")

    prefix = api_key[:8]

    row = await pool.fetchrow(QUERIES["api_keys/get_by_prefix"], prefix)
    if not row:
        raise ValueError("NEBULA_API_KEY is invalid or revoked")

    try:
        ph.verify(row["key_hash"], api_key)
    except VerifyMismatchError:
        raise ValueError("NEBULA_API_KEY hash mismatch")

    if not row["agent_id"]:
        raise ValueError("NEBULA_API_KEY is not an agent key")

    agent = await pool.fetchrow(QUERIES["agents/get_by_id"], row["agent_id"])
    if not agent:
        raise ValueError("Agent not found or inactive for NEBULA_API_KEY")

    return dict(agent)
