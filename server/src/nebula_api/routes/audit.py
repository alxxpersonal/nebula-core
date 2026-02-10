"""Audit API routes."""

# Third-Party
from fastapi import APIRouter, Depends, Query, Request

# Local
from nebula_api.auth import require_auth
from nebula_api.response import paginated, success
from nebula_mcp.helpers import (
    list_audit_actors,
    list_audit_scopes,
    query_audit_log,
)

router = APIRouter()


@router.get("/")
async def list_audit_log(
    request: Request,
    auth: dict = Depends(require_auth),
    table: str | None = None,
    action: str | None = None,
    actor_type: str | None = None,
    actor_id: str | None = None,
    record_id: str | None = None,
    scope_id: str | None = None,
    limit: int = Query(50, le=200),
    offset: int = 0,
) -> dict:
    """List audit log entries with optional filters.

    Args:
        request: FastAPI request.
        auth: Auth context.
        table: Table name filter.
        action: Action filter.
        actor_type: Actor type filter.
        actor_id: Actor id filter.
        record_id: Record id filter.
        scope_id: Privacy scope filter.
        limit: Max rows.
        offset: Offset for pagination.

    Returns:
        Paginated audit log response.
    """
    pool = request.app.state.pool
    rows = await query_audit_log(
        pool,
        table,
        action,
        actor_type,
        actor_id,
        record_id,
        scope_id,
        limit,
        offset,
    )
    return paginated(rows, len(rows), limit, offset)


@router.get("/scopes")
async def list_scopes(
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict:
    """List audit scopes with usage counts.

    Args:
        request: FastAPI request.
        auth: Auth context.

    Returns:
        List of scopes with counts.
    """
    pool = request.app.state.pool
    rows = await list_audit_scopes(pool)
    return success(rows)


@router.get("/actors")
async def list_actors(
    request: Request,
    auth: dict = Depends(require_auth),
    actor_type: str | None = None,
) -> dict:
    """List audit actors with activity counts.

    Args:
        request: FastAPI request.
        auth: Auth context.
        actor_type: Actor type filter.

    Returns:
        List of audit actors.
    """
    pool = request.app.state.pool
    rows = await list_audit_actors(pool, actor_type)
    return success(rows)
