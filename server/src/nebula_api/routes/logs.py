"""Log API routes."""

# Standard Library
from datetime import datetime
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, Query, Request
from pydantic import BaseModel

# Local
from nebula_api.auth import maybe_check_agent_approval, require_auth
from nebula_api.response import api_error, success
from nebula_mcp.enums import require_log_type, require_status
from nebula_mcp.executors import execute_create_log, execute_update_log
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()


class CreateLogBody(BaseModel):
    """Payload for creating a log entry.

    Attributes:
        log_type: Log type name.
        timestamp: Timestamp for the log entry.
        value: Log value payload.
        status: Status name.
        tags: Optional tag list.
        metadata: Optional metadata payload.
    """

    log_type: str
    timestamp: datetime | None = None
    value: dict | None = None
    status: str = "active"
    tags: list[str] = []
    metadata: dict | None = None


class UpdateLogBody(BaseModel):
    """Payload for updating a log entry.

    Attributes:
        log_type: Updated log type name.
        timestamp: Updated timestamp.
        value: Updated value payload.
        status: Updated status name.
        tags: Updated tags.
        metadata: Updated metadata.
    """

    log_type: str | None = None
    timestamp: datetime | None = None
    value: dict | None = None
    status: str | None = None
    tags: list[str] | None = None
    metadata: dict | None = None


@router.post("/")
async def create_log(
    payload: CreateLogBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Create a log entry."""

    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    if data.get("value") is None:
        data["value"] = {}
    if data.get("metadata") is None:
        data["metadata"] = {}

    if resp := await maybe_check_agent_approval(pool, auth, "create_log", data):
        return resp

    result = await execute_create_log(pool, enums, data)
    return success(result)


@router.get("/{log_id}")
async def get_log(
    log_id: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Fetch a log entry by id."""

    pool = request.app.state.pool
    row = await pool.fetchrow(QUERIES["logs/get"], log_id)
    if not row:
        api_error("NOT_FOUND", f"Log '{log_id}' not found", 404)
    return success(dict(row))


@router.get("/")
async def query_logs(
    request: Request,
    auth: dict = Depends(require_auth),
    log_type: str | None = None,
    tags: list[str] = Query(default_factory=list),
    status_category: str = "active",
    limit: int = Query(50, le=500),
    offset: int = 0,
) -> dict[str, Any]:
    """Query log entries with filters."""

    pool = request.app.state.pool
    enums = request.app.state.enums

    log_type_id = require_log_type(log_type, enums) if log_type else None

    rows = await pool.fetch(
        QUERIES["logs/query"],
        log_type_id,
        tags or None,
        status_category,
        limit,
        offset,
    )
    return success([dict(r) for r in rows])


@router.patch("/{log_id}")
async def update_log(
    log_id: str,
    payload: UpdateLogBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Update a log entry."""

    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    data["id"] = log_id

    if resp := await maybe_check_agent_approval(pool, auth, "update_log", data):
        return resp

    if payload.log_type:
        data["log_type"] = payload.log_type

    result = await execute_update_log(pool, enums, data)
    if not result:
        api_error("NOT_FOUND", f"Log '{log_id}' not found", 404)
    return success(result)
