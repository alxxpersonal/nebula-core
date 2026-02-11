"""Knowledge API routes."""

# Standard Library
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, HTTPException, Query, Request
from pydantic import BaseModel, field_validator

# Local
from nebula_api.auth import maybe_check_agent_approval, require_auth
from nebula_api.response import paginated, success
from nebula_mcp.enums import require_status
from nebula_mcp.executors import execute_create_knowledge, execute_create_relationship
from nebula_mcp.helpers import enforce_scope_subset, scope_names_from_ids
from nebula_mcp.models import MAX_PAGE_LIMIT, MAX_TAG_LENGTH, MAX_TAGS
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()


def _validate_tag_list(tags: list[str] | None) -> list[str] | None:
    if tags is None:
        return None
    cleaned = [t.strip() for t in tags if t and t.strip()]
    if len(cleaned) > MAX_TAGS:
        raise ValueError("Too many tags")
    for tag in cleaned:
        if len(tag) > MAX_TAG_LENGTH:
            raise ValueError("Tag too long")
    return cleaned


class CreateKnowledgeBody(BaseModel):
    """Payload for creating a knowledge item.

    Attributes:
        title: Knowledge title.
        url: Optional URL.
        source_type: Knowledge source type.
        content: Optional content text.
        scopes: Privacy scopes.
        tags: Tag list.
        metadata: Arbitrary metadata.
    """

    title: str
    url: str | None = None
    source_type: str = "article"
    content: str | None = None
    scopes: list[str] = []
    tags: list[str] = []
    metadata: dict | None = None

    @field_validator("tags", mode="before")
    @classmethod
    def _clean_tags(cls, v: list[str] | None) -> list[str] | None:
        return _validate_tag_list(v)


class LinkKnowledgeBody(BaseModel):
    """Payload for linking knowledge to an entity.

    Attributes:
        entity_id: Target entity id.
        relationship_type: Relationship type name.
    """

    entity_id: str
    relationship_type: str = "related-to"


class UpdateKnowledgeBody(BaseModel):
    """Payload for updating a knowledge item.

    Attributes:
        title: Updated title.
        url: Updated URL.
        source_type: Updated source type.
        content: Updated content.
        status: Updated status name.
        tags: Updated tags.
        scopes: Updated scopes.
        metadata: Updated metadata.
    """

    title: str | None = None
    url: str | None = None
    source_type: str | None = None
    content: str | None = None
    status: str | None = None
    tags: list[str] | None = None
    scopes: list[str] | None = None
    metadata: dict | None = None

    @field_validator("tags", mode="before")
    @classmethod
    def _clean_tags(cls, v: list[str] | None) -> list[str] | None:
        return _validate_tag_list(v)


@router.post("/")
async def create_knowledge(
    payload: CreateKnowledgeBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Create a knowledge item.

    Args:
        payload: Knowledge creation payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with created knowledge or approval requirement.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    data = payload.model_dump()
    data.setdefault("metadata", {})
    if data["metadata"] is None:
        data["metadata"] = {}
    if auth["caller_type"] == "agent":
        allowed = scope_names_from_ids(auth.get("scopes", []), enums)
        data["scopes"] = enforce_scope_subset(data["scopes"], allowed)
    if resp := await maybe_check_agent_approval(pool, auth, "create_knowledge", data):
        return resp
    result = await execute_create_knowledge(pool, enums, data)
    return success(result)


@router.get("/")
async def query_knowledge(
    request: Request,
    auth: dict = Depends(require_auth),
    source_type: str | None = None,
    tags: str | None = None,
    search_text: str | None = None,
    limit: int = Query(50, le=MAX_PAGE_LIMIT),
    offset: int = 0,
) -> dict[str, Any]:
    """Query knowledge items with filters.

    Args:
        request: FastAPI request.
        auth: Auth context.
        source_type: Source type filter.
        tags: Comma-separated tag filters.
        search_text: Full-text search filter.
        limit: Max rows.
        offset: Offset for pagination.

    Returns:
        Paginated API response with knowledge items.
    """
    pool = request.app.state.pool
    scope_ids = auth.get("scopes", [])
    tag_list = tags.split(",") if tags else None

    rows = await pool.fetch(
        QUERIES["knowledge/query"],
        source_type,
        tag_list,
        search_text,
        scope_ids,
        limit,
        offset,
    )
    return paginated([dict(r) for r in rows], len(rows), limit, offset)


@router.get("/{knowledge_id}")
async def get_knowledge(
    knowledge_id: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Fetch a knowledge item by id.

    Args:
        knowledge_id: Knowledge id.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with knowledge data.
    """
    pool = request.app.state.pool
    scope_ids = auth.get("scopes", [])
    row = await pool.fetchrow(
        QUERIES["knowledge/get"],
        knowledge_id,
        scope_ids,
    )
    if not row:
        raise HTTPException(status_code=404, detail="Not Found")
    return success(dict(row))


@router.post("/{knowledge_id}/link")
async def link_to_entity(
    knowledge_id: str,
    payload: LinkKnowledgeBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Create a relationship from knowledge to an entity.

    Args:
        knowledge_id: Knowledge id.
        payload: Link payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with created relationship or approval requirement.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    relationship_payload = {
        "source_type": "knowledge",
        "source_id": knowledge_id,
        "target_type": "entity",
        "target_id": payload.entity_id,
        "relationship_type": payload.relationship_type,
        "properties": {},
    }
    if resp := await maybe_check_agent_approval(
        pool, auth, "create_relationship", relationship_payload
    ):
        return resp

    result = await execute_create_relationship(pool, enums, relationship_payload)
    return success(result)


@router.patch("/{knowledge_id}")
async def update_knowledge(
    knowledge_id: str,
    payload: UpdateKnowledgeBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Update a knowledge item.

    Args:
        knowledge_id: Knowledge id.
        payload: Knowledge update payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with updated knowledge or approval requirement.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums

    data = payload.model_dump()
    status_id = None
    if data.get("status"):
        status_id = require_status(data["status"], enums)
    if data.get("metadata") is None:
        data.pop("metadata", None)
    if auth["caller_type"] == "agent" and data.get("scopes") is not None:
        allowed = scope_names_from_ids(auth.get("scopes", []), enums)
        data["scopes"] = enforce_scope_subset(data["scopes"], allowed)
    change = {"knowledge_id": knowledge_id, **data}
    if resp := await maybe_check_agent_approval(pool, auth, "update_knowledge", change):
        return resp
    row = await pool.fetchrow(
        QUERIES["knowledge/update"],
        knowledge_id,
        data.get("title"),
        data.get("url"),
        data.get("source_type"),
        data.get("content"),
        status_id,
        data.get("tags"),
        data.get("scopes"),
        data.get("metadata"),
    )
    if not row:
        raise HTTPException(status_code=404, detail="Not Found")
    return success(dict(row))
