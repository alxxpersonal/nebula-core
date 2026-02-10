"""Agent API routes."""

# Standard Library
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel
from starlette.responses import JSONResponse

# Local
from nebula_api.auth import require_auth
from nebula_api.response import api_error, success
from nebula_mcp.enums import load_enums, require_scopes
from nebula_mcp.helpers import create_approval_request
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()


class RegisterAgentBody(BaseModel):
    """Payload for registering a new agent.

    Attributes:
        name: Unique agent name.
        description: Optional agent description.
        requested_scopes: Requested privacy scopes.
        capabilities: Agent capability tags.
    """

    name: str
    description: str | None = None
    requested_scopes: list[str] = ["public"]
    capabilities: list[str] = []


class UpdateAgentBody(BaseModel):
    """Payload for updating agent settings.

    Attributes:
        description: Updated description.
        requires_approval: Whether the agent requires approval.
        scopes: New scope list.
    """

    description: str | None = None
    requires_approval: bool | None = None
    scopes: list[str] | None = None


@router.post("/register")
async def register_agent(payload: RegisterAgentBody, request: Request) -> JSONResponse:
    """Register a new agent and create an approval request.

    Args:
        payload: Agent registration payload.
        request: FastAPI request.

    Returns:
        JSON response with approval request info.
    """

    pool = request.app.state.pool
    enums = request.app.state.enums

    # Check name uniqueness
    existing = await pool.fetchrow(QUERIES["agents/check_name"], payload.name)
    if existing:
        api_error("CONFLICT", f"Agent '{payload.name}' already exists", 409)

    # Resolve scope names to UUIDs
    scope_ids = require_scopes(payload.requested_scopes, enums)

    # Get pending status
    pending_status_id = enums.statuses.name_to_id.get("inactive")
    if not pending_status_id:
        api_error("INTERNAL", "Status 'inactive' not found", 500)

    # Create agent row with inactive status
    agent = await pool.fetchrow(
        QUERIES["agents/create"],
        payload.name,
        payload.description,
        scope_ids,
        payload.capabilities,
        pending_status_id,
        True,  # requires_approval
    )

    # Create approval request for registration
    approval = await create_approval_request(
        pool,
        agent["id"],
        "register_agent",
        {
            "agent_id": str(agent["id"]),
            "name": payload.name,
            "description": payload.description,
            "requested_scopes": payload.requested_scopes,
            "capabilities": payload.capabilities,
        },
    )

    return JSONResponse(
        status_code=201,
        content={
            "data": {
                "agent_id": str(agent["id"]),
                "approval_request_id": str(approval["id"]),
                "status": "pending_approval",
            }
        },
    )


@router.patch("/{agent_id}")
async def update_agent(
    agent_id: str,
    payload: UpdateAgentBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Update agent fields.

    Args:
        agent_id: Agent id.
        payload: Agent update payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with updated agent data.
    """

    pool = request.app.state.pool
    enums = request.app.state.enums

    # Resolve scope names to UUIDs if provided
    scope_ids = None
    if payload.scopes is not None:
        scope_ids = require_scopes(payload.scopes, enums)

    row = await pool.fetchrow(
        QUERIES["agents/update"],
        agent_id,
        payload.description,
        payload.requires_approval,
        scope_ids,
    )

    if not row:
        api_error("NOT_FOUND", "Agent not found", 404)

    return success(dict(row))


@router.get("/{agent_name}")
async def get_agent_info(
    agent_name: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict:
    """Get agent info by name.

    Args:
        agent_name: Agent name.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with agent info.
    """

    pool = request.app.state.pool

    row = await pool.fetchrow(QUERIES["agents/get_info"], agent_name)
    if not row:
        api_error("NOT_FOUND", f"Agent '{agent_name}' not found", 404)

    return success(dict(row))


@router.get("/")
async def list_agents(
    request: Request,
    auth: dict = Depends(require_auth),
    status_category: str = "active",
) -> dict:
    """List agents by status category.

    Args:
        request: FastAPI request.
        auth: Auth context.
        status_category: Status category filter.

    Returns:
        API response with agent list.
    """

    pool = request.app.state.pool

    rows = await pool.fetch(QUERIES["agents/list"], status_category)
    return success([dict(r) for r in rows])


@router.post("/reload-enums")
async def reload_enums_route(
    request: Request, auth: dict = Depends(require_auth)
) -> dict:
    """Reload enum cache.

    Args:
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with confirmation message.
    """

    pool = request.app.state.pool
    request.app.state.enums = await load_enums(pool)
    return success({"message": "Enums reloaded"})
