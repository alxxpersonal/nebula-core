"""Pure helper functions for Nebula MCP."""

# Standard Library
import json
from pathlib import Path

# Third-Party
from asyncpg import Pool

# Local
from .enums import EnumRegistry, require_scopes
from .query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[1] / "queries")


# --- Privacy Filtering ---


def filter_context_segments(metadata: dict | str, agent_scopes: list[str]) -> dict:
    """Filter context segments by agent's privacy scopes.

    Args:
        metadata: Entity metadata dict or JSON string.
        agent_scopes: List of scope names the agent has access to.

    Returns:
        Metadata dict with context_segments filtered to matching scopes.
    """

    # Handle JSONB string from asyncpg
    if isinstance(metadata, str):
        metadata = json.loads(metadata)

    if not metadata or "context_segments" not in metadata:
        return metadata

    filtered = metadata.copy()
    segments = []

    for seg in metadata["context_segments"]:
        seg_scopes = seg.get("scopes", [])
        if any(scope in agent_scopes for scope in seg_scopes):
            segments.append(seg)

    filtered["context_segments"] = segments
    return filtered


# --- Approval Workflow ---


async def create_approval_request(
    pool: Pool,
    agent_id: str,
    request_type: str,
    change_details: dict,
    job_id: str | None = None,
) -> dict:
    """Create an approval request for untrusted agent actions.

    Args:
        pool: Database connection pool.
        agent_id: UUID of the requesting agent.
        request_type: Action type (e.g., create_entity).
        change_details: Full payload of requested change.
        job_id: Optional related job ID.

    Returns:
        Created approval request row as dict.
    """

    row = await pool.fetchrow(
        QUERIES["approvals/create_request"],
        request_type,
        agent_id,
        (
            json.dumps(change_details)
            if isinstance(change_details, dict)
            else change_details
        ),
        job_id,
    )
    return dict(row) if row else {}


async def get_pending_approvals_all(pool: Pool) -> list[dict]:
    """Get all pending approval requests for admin review.

    Args:
        pool: Database connection pool.

    Returns:
        List of pending approval request dicts.
    """

    rows = await pool.fetch(QUERIES["approvals/get_pending"])
    return [dict(r) for r in rows]


async def approve_request(
    pool: Pool, enums: EnumRegistry, approval_id: str, reviewed_by: str
) -> dict:
    """Approve request and execute the action.

    Args:
        pool: Database connection pool.
        enums: Enum registry for validation.
        approval_id: UUID of approval request.
        reviewed_by: UUID of approving entity.

    Returns:
        Dict containing approval record and created entity.

    Raises:
        ValueError: If approval not found or no executor available.
    """

    from .executors import EXECUTORS

    approval = await pool.fetchrow(
        QUERIES["approvals/approve"],
        approval_id,
        reviewed_by,
    )

    if not approval:
        raise ValueError("Approval request not found or already processed")

    executor = EXECUTORS.get(approval["request_type"])
    if not executor:
        raise ValueError(f"No executor for: {approval['request_type']}")

    try:
        await pool.execute("SET app.changed_by_type = 'entity'")
        await pool.execute(f"SET app.changed_by_id = '{reviewed_by}'")

        result = await executor(pool, enums, approval["change_details"])

        await pool.execute(
            QUERIES["approvals/link_audit"], str(approval_id), str(result["id"])
        )

        return {"approval": dict(approval), "entity": result}

    except Exception as e:
        await pool.execute(QUERIES["approvals/mark_failed"], str(e), approval_id)
        raise

    finally:
        await pool.execute("RESET app.changed_by_type")
        await pool.execute("RESET app.changed_by_id")


async def reject_request(
    pool: Pool, approval_id: str, reviewed_by: str, review_notes: str
) -> dict:
    """Reject an approval request.

    Args:
        pool: Database connection pool.
        approval_id: UUID of approval request.
        reviewed_by: UUID of rejecting entity.
        review_notes: Reason for rejection.

    Returns:
        Rejected approval request row as dict.

    Raises:
        ValueError: If approval not found.
    """

    row = await pool.fetchrow(
        QUERIES["approvals/reject"],
        approval_id,
        reviewed_by,
        review_notes,
    )

    if not row:
        raise ValueError("Approval request not found or already processed")

    return dict(row)


# --- Audit + History ---


async def get_entity_history(
    pool: Pool, entity_id: str, limit: int = 50, offset: int = 0
) -> list[dict]:
    """List audit history entries for a single entity.

    Args:
        pool: Database connection pool.
        entity_id: Entity UUID.
        limit: Max rows to return.
        offset: Pagination offset.

    Returns:
        List of audit entries as dicts.
    """

    rows = await pool.fetch(QUERIES["audit/entity_history"], entity_id, limit, offset)
    return [dict(r) for r in rows]


async def revert_entity(pool: Pool, entity_id: str, audit_id: str) -> dict:
    """Revert an entity to a historical audit snapshot.

    Args:
        pool: Database connection pool.
        entity_id: Entity UUID to revert.
        audit_id: Audit log entry to restore.

    Returns:
        Updated entity row as dict.

    Raises:
        ValueError: If audit entry is missing or mismatched.
    """

    audit_row = await pool.fetchrow(QUERIES["audit/get"], audit_id)
    if not audit_row:
        raise ValueError("Audit entry not found")

    audit = dict(audit_row)
    if audit.get("table_name") != "entities":
        raise ValueError("Audit entry is not for entities")
    if audit.get("record_id") != entity_id:
        raise ValueError("Audit entry does not match entity")

    snapshot = audit.get("new_data")
    if audit.get("action") == "delete":
        snapshot = audit.get("old_data")
    if snapshot is None:
        raise ValueError("Audit snapshot is empty")

    if isinstance(snapshot, str):
        snapshot = json.loads(snapshot)

    metadata = snapshot.get("metadata")
    metadata_json = None
    if metadata is not None:
        metadata_json = json.dumps(metadata)

    row = await pool.fetchrow(
        QUERIES["entities/revert"],
        entity_id,
        snapshot.get("privacy_scope_ids") or [],
        snapshot.get("name"),
        snapshot.get("type_id"),
        snapshot.get("status_id"),
        snapshot.get("status_changed_at"),
        snapshot.get("status_reason"),
        snapshot.get("tags") or [],
        metadata_json,
        snapshot.get("vault_file_path"),
    )
    return dict(row) if row else {}


def normalize_bulk_operation(op: str) -> str:
    """Normalize bulk operation name."""

    key = (op or "").strip().lower()
    if key in {"add", "+"}:
        return "add"
    if key in {"remove", "rm", "del", "delete", "-"}:
        return "remove"
    if key in {"set", "="}:
        return "set"
    raise ValueError("Invalid bulk operation. Use add, remove, or set.")


async def bulk_update_entity_tags(
    pool: Pool, entity_ids: list[str], tags: list[str], op: str
) -> list[str]:
    """Bulk update entity tags.

    Args:
        pool: Database connection pool.
        entity_ids: Entity UUIDs.
        tags: Tag values.
        op: add, remove, or set.

    Returns:
        Updated entity ids.
    """

    rows = await pool.fetch(QUERIES["entities/bulk_update_tags"], entity_ids, op, tags)
    return [str(r["id"]) for r in rows]


async def bulk_update_entity_scopes(
    pool: Pool, enums: EnumRegistry, entity_ids: list[str], scopes: list[str], op: str
) -> list[str]:
    """Bulk update entity privacy scopes.

    Args:
        pool: Database connection pool.
        enums: Enum registry for validation.
        entity_ids: Entity UUIDs.
        scopes: Scope names.
        op: add, remove, or set.

    Returns:
        Updated entity ids.
    """

    scope_ids = require_scopes(scopes, enums)
    rows = await pool.fetch(
        QUERIES["entities/bulk_update_scopes"], entity_ids, op, scope_ids
    )
    return [str(r["id"]) for r in rows]


async def query_audit_log(
    pool: Pool,
    table_name: str | None = None,
    action: str | None = None,
    actor_type: str | None = None,
    actor_id: str | None = None,
    record_id: str | None = None,
    scope_id: str | None = None,
    limit: int = 50,
    offset: int = 0,
) -> list[dict]:
    """List audit log entries with optional filters.

    Args:
        pool: Database connection pool.
        table_name: Table name filter.
        action: Action filter (insert, update, delete).
        actor_type: Actor type filter (agent, entity, system).
        actor_id: Actor UUID filter.
        record_id: Record id filter.
        limit: Max rows to return.
        offset: Pagination offset.

    Returns:
        List of audit entries as dicts.
    """

    actor_id = actor_id or None
    rows = await pool.fetch(
        QUERIES["audit/list"],
        table_name,
        action,
        actor_type,
        actor_id,
        record_id,
        scope_id,
        limit,
        offset,
    )
    return [dict(r) for r in rows]


async def list_audit_scopes(pool: Pool) -> list[dict]:
    """List privacy scopes with usage counts."""

    rows = await pool.fetch(QUERIES["audit/scopes"])
    return [dict(r) for r in rows]


async def list_audit_actors(pool: Pool, actor_type: str | None = None) -> list[dict]:
    """List audit actors with activity counts."""

    rows = await pool.fetch(QUERIES["audit/actors"], actor_type)
    return [dict(r) for r in rows]


def _normalize_diff_value(value: object) -> object:
    """Normalize diff values for stable comparisons.

    Args:
        value: Value to normalize, often dict or list.

    Returns:
        JSON-encoded string for complex types, otherwise original value.
    """
    if isinstance(value, (dict, list)):
        return json.dumps(value, sort_keys=True)
    return value


async def get_approval_diff(pool: Pool, approval_id: str) -> dict:
    """Compute diff between approval request and current entity state.

    Args:
        pool: Database connection pool.
        approval_id: Approval request UUID.

    Returns:
        Dict with request_type and changes map.
    """

    row = await pool.fetchrow(QUERIES["approvals/get_request"], approval_id)
    if not row:
        raise ValueError("Approval request not found")

    approval = dict(row)
    change_details = approval.get("change_details") or {}
    if isinstance(change_details, str):
        change_details = json.loads(change_details)

    request_type = approval.get("request_type")
    changes: dict[str, dict[str, object]] = {}

    if request_type == "update_entity":
        entity_id = change_details.get("entity_id")
        if not entity_id:
            raise ValueError("Approval request missing entity_id")

        entity_row = await pool.fetchrow(QUERIES["entities/get"], entity_id)
        if not entity_row:
            raise ValueError("Entity not found for approval diff")

        entity = dict(entity_row)
        for key, new_val in change_details.items():
            if key == "entity_id":
                continue
            old_val = entity.get(key)
            if _normalize_diff_value(old_val) != _normalize_diff_value(new_val):
                changes[key] = {"from": old_val, "to": new_val}
    elif request_type == "create_entity":
        for key, new_val in change_details.items():
            changes[key] = {"from": None, "to": new_val}
    elif request_type in {"create_knowledge", "create_relationship", "create_job"}:
        for key, new_val in change_details.items():
            changes[key] = {"from": None, "to": new_val}
    elif request_type == "update_relationship":
        relationship_id = change_details.get("relationship_id")
        if not relationship_id:
            raise ValueError("Approval request missing relationship_id")

        rel_row = await pool.fetchrow(
            QUERIES["relationships/get_by_id"], relationship_id
        )
        if not rel_row:
            raise ValueError("Relationship not found for approval diff")

        relationship = dict(rel_row)
        for key, new_val in change_details.items():
            if key == "relationship_id":
                continue
            old_val = relationship.get(key)
            if _normalize_diff_value(old_val) != _normalize_diff_value(new_val):
                changes[key] = {"from": old_val, "to": new_val}
    elif request_type == "update_job_status":
        job_id = change_details.get("job_id")
        if not job_id:
            raise ValueError("Approval request missing job_id")

        job_row = await pool.fetchrow(QUERIES["jobs/get"], job_id)
        if not job_row:
            raise ValueError("Job not found for approval diff")

        job = dict(job_row)
        for key, new_val in change_details.items():
            if key == "job_id":
                continue
            old_val = job.get(key)
            if _normalize_diff_value(old_val) != _normalize_diff_value(new_val):
                changes[key] = {"from": old_val, "to": new_val}

    return {
        "approval_id": approval_id,
        "request_type": request_type,
        "changes": changes,
    }
