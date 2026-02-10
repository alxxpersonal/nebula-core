"""Protocol API routes."""

# Standard Library
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, HTTPException, Query, Request
from pydantic import BaseModel

# Local
from nebula_api.auth import maybe_check_agent_approval, require_auth
from nebula_api.response import paginated, success
from nebula_mcp.enums import require_status
from nebula_mcp.executors import execute_create_protocol, execute_update_protocol
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()


class CreateProtocolBody(BaseModel):
    """Payload for creating a protocol."""

    name: str
    title: str
    version: str | None = None
    content: str
    protocol_type: str | None = None
    applies_to: list[str] = []
    status: str = "active"
    tags: list[str] = []
    metadata: dict | None = None
    vault_file_path: str | None = None


class UpdateProtocolBody(BaseModel):
    """Payload for updating a protocol."""

    title: str | None = None
    version: str | None = None
    content: str | None = None
    protocol_type: str | None = None
    applies_to: list[str] | None = None
    status: str | None = None
    tags: list[str] | None = None
    metadata: dict | None = None
    vault_file_path: str | None = None


@router.get("/")
async def query_protocols(
    request: Request,
    auth: dict = Depends(require_auth),
    status_category: str | None = None,
    protocol_type: str | None = None,
    search: str | None = None,
    limit: int = Query(50, le=100),
) -> dict[str, Any]:
    """Query protocols with optional filters."""

    pool = request.app.state.pool
    rows = await pool.fetch(
        QUERIES["protocols/query"],
        status_category,
        protocol_type,
        search,
        limit,
    )
    return paginated([dict(r) for r in rows], len(rows), limit, 0)


@router.get("/{protocol_name}")
async def get_protocol(
    protocol_name: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Fetch a protocol by name."""

    pool = request.app.state.pool
    row = await pool.fetchrow(QUERIES["protocols/get"], protocol_name)
    if not row:
        raise HTTPException(status_code=404, detail="Not Found")
    return success(dict(row))


@router.post("/")
async def create_protocol(
    payload: CreateProtocolBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Create a protocol."""

    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    if data["metadata"] is None:
        data["metadata"] = {}
    if resp := await maybe_check_agent_approval(pool, auth, "create_protocol", data):
        return resp
    result = await execute_create_protocol(pool, enums, data)
    return success(result)


@router.patch("/{protocol_name}")
async def update_protocol(
    protocol_name: str,
    payload: UpdateProtocolBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Update a protocol."""

    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    data["name"] = protocol_name
    if data.get("status") is not None:
        require_status(data["status"], enums)
    if resp := await maybe_check_agent_approval(pool, auth, "update_protocol", data):
        return resp
    result = await execute_update_protocol(pool, enums, data)
    return success(result)
