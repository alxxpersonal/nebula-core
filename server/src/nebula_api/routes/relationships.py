"""Relationship API routes."""

# Standard Library
import json
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, Query, Request
from pydantic import BaseModel

# Local
from nebula_api.auth import maybe_check_agent_approval, require_auth
from nebula_api.response import api_error, success
from nebula_mcp.enums import EnumRegistry, require_status
from nebula_mcp.executors import execute_create_relationship
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()
ADMIN_SCOPE_NAMES = {"vault-only", "sensitive"}


def _is_admin(auth: dict, enums: EnumRegistry) -> bool:
    scope_ids = set(auth.get("scopes", []))
    allowed_ids = {
        enums.scopes.name_to_id.get(name)
        for name in ADMIN_SCOPE_NAMES
        if enums.scopes.name_to_id.get(name)
    }
    return bool(scope_ids.intersection(allowed_ids))


class CreateRelationshipBody(BaseModel):
    """Payload for creating a relationship.

    Attributes:
        source_type: Source node type.
        source_id: Source node id.
        target_type: Target node type.
        target_id: Target node id.
        relationship_type: Relationship type name.
        properties: Optional relationship properties.
    """

    source_type: str
    source_id: str
    target_type: str
    target_id: str
    relationship_type: str
    properties: dict | None = None


class UpdateRelationshipBody(BaseModel):
    """Payload for updating a relationship.

    Attributes:
        properties: Updated properties.
        status: Updated status name.
    """

    properties: dict | None = None
    status: str | None = None


@router.post("/")
async def create_relationship(
    payload: CreateRelationshipBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Create a relationship.

    Args:
        payload: Relationship creation payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with created relationship or approval requirement.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    if data.get("properties") is None:
        data["properties"] = {}
    if resp := await maybe_check_agent_approval(
        pool, auth, "create_relationship", data
    ):
        return resp
    result = await execute_create_relationship(pool, enums, data)
    return success(result)


@router.get("/{source_type}/{source_id}")
async def get_relationships(
    source_type: str,
    source_id: str,
    request: Request,
    auth: dict = Depends(require_auth),
    direction: str = "both",
    relationship_type: str | None = None,
) -> dict[str, Any]:
    """Get relationships for a source node.

    Args:
        source_type: Source node type.
        source_id: Source node id.
        request: FastAPI request.
        auth: Auth context.
        direction: Direction filter.
        relationship_type: Relationship type filter.

    Returns:
        API response with relationships.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    scope_ids = None if _is_admin(auth, enums) else auth.get("scopes", [])

    rows = await pool.fetch(
        QUERIES["relationships/get"],
        source_type,
        source_id,
        direction,
        relationship_type,
        scope_ids,
    )
    return success([dict(r) for r in rows])


@router.get("/")
async def query_relationships(
    request: Request,
    auth: dict = Depends(require_auth),
    source_type: str | None = None,
    target_type: str | None = None,
    relationship_types: str | None = None,
    status_category: str = "active",
    limit: int = Query(50, le=100),
) -> dict[str, Any]:
    """Query relationships with filters.

    Args:
        request: FastAPI request.
        auth: Auth context.
        source_type: Source type filter.
        target_type: Target type filter.
        relationship_types: Comma-separated relationship types.
        status_category: Status category filter.
        limit: Max rows.

    Returns:
        API response with relationship list.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    scope_ids = None if _is_admin(auth, enums) else auth.get("scopes", [])

    type_list = relationship_types.split(",") if relationship_types else None

    rows = await pool.fetch(
        QUERIES["relationships/query"],
        source_type,
        target_type,
        type_list,
        status_category,
        limit,
        scope_ids,
    )
    return success([dict(r) for r in rows])


@router.patch("/{relationship_id}")
async def update_relationship(
    relationship_id: str,
    payload: UpdateRelationshipBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Update a relationship.

    Args:
        relationship_id: Relationship id.
        payload: Relationship update payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with updated relationship or approval requirement.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    change = {
        "relationship_id": relationship_id,
        "properties": payload.properties,
        "status": payload.status,
    }
    if resp := await maybe_check_agent_approval(
        pool, auth, "update_relationship", change
    ):
        return resp

    status_id = require_status(payload.status, enums) if payload.status else None

    row = await pool.fetchrow(
        QUERIES["relationships/update"],
        relationship_id,
        json.dumps(payload.properties) if payload.properties else None,
        status_id,
    )
    if not row:
        api_error("NOT_FOUND", "Relationship not found", 404)

    return success(dict(row))
