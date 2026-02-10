"""File API routes."""

# Standard Library
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, Query, Request
from pydantic import BaseModel

# Local
from nebula_api.auth import maybe_check_agent_approval, require_auth
from nebula_api.response import api_error, success
from nebula_mcp.executors import execute_create_file, execute_update_file
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()


class CreateFileBody(BaseModel):
    """Payload for creating a file entry."""

    filename: str
    file_path: str
    mime_type: str | None = None
    size_bytes: int | None = None
    checksum: str | None = None
    status: str = "active"
    tags: list[str] = []
    metadata: dict | None = None


class UpdateFileBody(BaseModel):
    """Payload for updating a file entry."""

    filename: str | None = None
    file_path: str | None = None
    mime_type: str | None = None
    size_bytes: int | None = None
    checksum: str | None = None
    status: str | None = None
    tags: list[str] | None = None
    metadata: dict | None = None


@router.get("/")
async def list_files(
    request: Request,
    auth: dict = Depends(require_auth),
    tags: list[str] = Query(default_factory=list),
    mime_type: str | None = None,
    status_category: str = "active",
    limit: int = Query(50, le=500),
    offset: int = 0,
) -> dict[str, Any]:
    """List files with optional filters."""

    pool = request.app.state.pool
    rows = await pool.fetch(
        QUERIES["files/list"],
        tags or None,
        mime_type,
        status_category,
        limit,
        offset,
    )
    return success([dict(r) for r in rows])


@router.get("/{file_id}")
async def get_file(
    file_id: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Fetch a file by id."""

    pool = request.app.state.pool
    row = await pool.fetchrow(QUERIES["files/get"], file_id)
    if not row:
        api_error("NOT_FOUND", f"File '{file_id}' not found", 404)
    return success(dict(row))


@router.post("/")
async def create_file(
    payload: CreateFileBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Create a file entry."""

    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    if data.get("metadata") is None:
        data["metadata"] = {}

    if resp := await maybe_check_agent_approval(pool, auth, "create_file", data):
        return resp

    result = await execute_create_file(pool, enums, data)
    return success(result)


@router.patch("/{file_id}")
async def update_file(
    file_id: str,
    payload: UpdateFileBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Update a file entry."""

    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    data["file_id"] = file_id

    if resp := await maybe_check_agent_approval(pool, auth, "update_file", data):
        return resp

    result = await execute_update_file(pool, enums, data)
    if not result:
        api_error("NOT_FOUND", f"File '{file_id}' not found", 404)
    return success(result)
