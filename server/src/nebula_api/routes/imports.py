"""Bulk import API routes."""

# Standard Library
from typing import Any, Callable

# Third-Party
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel

# Local
from nebula_api.auth import maybe_check_agent_approval, require_auth
from nebula_api.response import success
from nebula_mcp.executors import (
    execute_create_entity,
    execute_create_job,
    execute_create_knowledge,
    execute_create_relationship,
)
from nebula_mcp.imports import (
    extract_items,
    normalize_entity,
    normalize_job,
    normalize_knowledge,
    normalize_relationship,
)

router = APIRouter()


class BulkImportBody(BaseModel):
    """Payload for bulk imports across resource types.

    Attributes:
        format: Input format, json or csv.
        data: CSV string data when format is csv.
        items: JSON items when format is json.
        defaults: Default values applied to each item.
    """

    format: str = "json"
    data: str | None = None
    items: list[dict[str, Any]] | None = None
    defaults: dict[str, Any] | None = None


async def _run_import(
    request: Request,
    auth: dict[str, Any],
    payload: BulkImportBody,
    normalizer: Callable[[dict[str, Any], dict[str, Any] | None], dict[str, Any]],
    executor: Callable[..., Any],
    approval_action: str,
) -> dict[str, Any]:
    """Run a bulk import with normalization and approval gating.

    Args:
        request: FastAPI request.
        auth: Auth context.
        payload: Bulk import payload.
        normalizer: Normalizer function for items.
        executor: Executor to persist normalized items.
        approval_action: Approval action name for audit/approval workflow.

    Returns:
        API response with created items and errors.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums

    items = extract_items(payload.format, payload.data, payload.items)
    if resp := await maybe_check_agent_approval(
        pool, auth, approval_action, {"format": payload.format, "items": items}
    ):
        return resp

    created: list[dict[str, Any]] = []
    errors: list[dict[str, Any]] = []

    async with pool.acquire() as conn:
        async with conn.transaction():
            if auth["caller_type"] == "user":
                await conn.execute(
                    "SELECT set_config('app.changed_by_type', $1, true)", "entity"
                )
                await conn.execute(
                    "SELECT set_config('app.changed_by_id', $1, true)",
                    str(auth["entity_id"]),
                )
            else:
                await conn.execute(
                    "SELECT set_config('app.changed_by_type', $1, true)", "agent"
                )
                await conn.execute(
                    "SELECT set_config('app.changed_by_id', $1, true)",
                    str(auth["agent_id"]),
                )

            for idx, item in enumerate(items, start=1):
                try:
                    normalized = normalizer(item, payload.defaults)
                    result = await executor(conn, enums, normalized)
                    created.append(result)
                except Exception as exc:
                    errors.append({"row": idx, "error": str(exc)})

    return success(
        {
            "created": len(created),
            "failed": len(errors),
            "errors": errors,
            "items": created,
        }
    )


@router.post("/entities")
async def import_entities(
    payload: BulkImportBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Bulk import entities.

    Args:
        payload: Bulk import payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with created entities or approval requirement.
    """
    return await _run_import(
        request,
        auth,
        payload,
        normalize_entity,
        execute_create_entity,
        "bulk_import_entities",
    )


@router.post("/knowledge")
async def import_knowledge(
    payload: BulkImportBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Bulk import knowledge items.

    Args:
        payload: Bulk import payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with created knowledge or approval requirement.
    """
    return await _run_import(
        request,
        auth,
        payload,
        normalize_knowledge,
        execute_create_knowledge,
        "bulk_import_knowledge",
    )


@router.post("/relationships")
async def import_relationships(
    payload: BulkImportBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Bulk import relationships.

    Args:
        payload: Bulk import payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with created relationships or approval requirement.
    """
    return await _run_import(
        request,
        auth,
        payload,
        normalize_relationship,
        execute_create_relationship,
        "bulk_import_relationships",
    )


@router.post("/jobs")
async def import_jobs(
    payload: BulkImportBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Bulk import jobs.

    Args:
        payload: Bulk import payload.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with created jobs or approval requirement.
    """
    return await _run_import(
        request,
        auth,
        payload,
        normalize_job,
        execute_create_job,
        "bulk_import_jobs",
    )
