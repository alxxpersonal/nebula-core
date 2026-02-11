"""Approval API routes."""

# Standard Library
import os
from pathlib import Path
from typing import Any

# Third-Party
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel

# Local
from nebula_api.auth import require_auth
from nebula_api.response import api_error, success
from nebula_mcp.helpers import (
    approve_request as do_approve,
)
from nebula_mcp.helpers import (
    get_approval_diff as compute_approval_diff,
)
from nebula_mcp.helpers import (
    get_pending_approvals_all,
)
from nebula_mcp.helpers import (
    reject_request as do_reject,
)
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[2] / "queries")

router = APIRouter()
ADMIN_SCOPE_NAMES = {"vault-only", "sensitive"}


def _require_admin_scope(auth: dict, enums: Any) -> None:
    if os.getenv("NEBULA_STRICT_ADMIN") != "1":
        return
    if auth.get("caller_type") != "agent":
        return
    scope_ids = set(auth.get("scopes", []))
    allowed_ids = {
        enums.scopes.name_to_id.get(name)
        for name in ADMIN_SCOPE_NAMES
        if enums.scopes.name_to_id.get(name)
    }
    if not scope_ids.intersection(allowed_ids):
        api_error("FORBIDDEN", "Admin scope required", 403)


class RejectBody(BaseModel):
    """Payload for rejecting an approval request.

    Attributes:
        review_notes: Optional reviewer notes explaining the rejection.
    """

    review_notes: str = ""


@router.get("/pending")
async def get_pending(
    request: Request, auth: dict = Depends(require_auth)
) -> dict[str, Any]:
    """List pending approval requests.

    Args:
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with pending approvals.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    _require_admin_scope(auth, enums)
    results = await get_pending_approvals_all(pool)
    return success(results)


@router.get("/{approval_id}")
async def get_approval(
    approval_id: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Fetch a single approval request by id.

    Args:
        approval_id: Approval request UUID.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with approval request data.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    _require_admin_scope(auth, enums)

    row = await pool.fetchrow(QUERIES["approvals/get_request"], approval_id)
    if not row:
        api_error("NOT_FOUND", "Approval request not found", 404)

    return success(dict(row))


@router.post("/{approval_id}/approve")
async def approve(
    approval_id: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Approve an approval request.

    Args:
        approval_id: Approval request UUID.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with approval result.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    _require_admin_scope(auth, enums)

    result = await do_approve(pool, enums, approval_id, str(auth["entity_id"]))
    return success(result)


@router.post("/{approval_id}/reject")
async def reject(
    approval_id: str,
    payload: RejectBody,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Reject an approval request.

    Args:
        approval_id: Approval request UUID.
        payload: Rejection payload with optional notes.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with rejection result.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    _require_admin_scope(auth, enums)

    result = await do_reject(
        pool, approval_id, str(auth["entity_id"]), payload.review_notes
    )
    return success(result)


@router.get("/{approval_id}/diff")
async def get_diff(
    approval_id: str,
    request: Request,
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    """Compute the diff for an approval request.

    Args:
        approval_id: Approval request UUID.
        request: FastAPI request.
        auth: Auth context.

    Returns:
        API response with approval diff data.
    """
    pool = request.app.state.pool
    enums = request.app.state.enums
    _require_admin_scope(auth, enums)
    result = await compute_approval_diff(pool, approval_id)
    return success(result)
