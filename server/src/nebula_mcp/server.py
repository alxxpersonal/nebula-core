"""Nebula MCP Server."""

# Standard Library
import json
import sys
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any, Callable

# Third-Party
from asyncpg import Pool
from dotenv import load_dotenv
from mcp.server.fastmcp import Context, FastMCP

# Module Path Bootstrap
if __package__ in (None, ""):
    sys.path.append(str(Path(__file__).resolve().parents[1]))

# Local
from nebula_mcp.context import (
    authenticate_agent,
    maybe_require_approval,
    require_context,
    require_pool,
)
from nebula_mcp.db import get_pool
from nebula_mcp.enums import (
    load_enums,
    require_entity_type,
    require_log_type,
    require_scopes,
    require_status,
)
from nebula_mcp.executors import (
    execute_create_entity,
    execute_create_file,
    execute_create_job,
    execute_create_knowledge,
    execute_create_log,
    execute_create_protocol,
    execute_create_relationship,
    execute_update_entity,
    execute_update_log,
    execute_update_protocol,
)
from nebula_mcp.helpers import (
    bulk_update_entity_scopes as do_bulk_update_entity_scopes,
)
from nebula_mcp.helpers import (
    bulk_update_entity_tags as do_bulk_update_entity_tags,
)
from nebula_mcp.helpers import (
    enforce_scope_subset,
    filter_context_segments,
    normalize_bulk_operation,
    scope_names_from_ids,
)
from nebula_mcp.helpers import (
    get_approval_diff as compute_approval_diff,
)
from nebula_mcp.helpers import (
    get_entity_history as fetch_entity_history,
)
from nebula_mcp.helpers import (
    query_audit_log as fetch_audit_log,
)
from nebula_mcp.helpers import (
    revert_entity as do_revert_entity,
)
from nebula_mcp.imports import (
    extract_items,
    normalize_entity,
    normalize_job,
    normalize_knowledge,
    normalize_relationship,
)
from nebula_mcp.models import (
    MAX_GRAPH_HOPS,
    MAX_PAGE_LIMIT,
    AttachFileInput,
    BulkImportInput,
    BulkUpdateEntityScopesInput,
    BulkUpdateEntityTagsInput,
    CreateEntityInput,
    CreateFileInput,
    CreateJobInput,
    CreateKnowledgeInput,
    CreateLogInput,
    CreateProtocolInput,
    CreateRelationshipInput,
    CreateSubtaskInput,
    GetAgentInfoInput,
    GetApprovalDiffInput,
    GetEntityHistoryInput,
    GetEntityInput,
    GetFileInput,
    GetJobInput,
    GetLogInput,
    GetProtocolInput,
    GetRelationshipsInput,
    GraphNeighborsInput,
    GraphShortestPathInput,
    LinkKnowledgeInput,
    ListAgentsInput,
    QueryAuditLogInput,
    QueryEntitiesInput,
    QueryFilesInput,
    QueryJobsInput,
    QueryKnowledgeInput,
    QueryLogsInput,
    QueryRelationshipsInput,
    RevertEntityInput,
    SearchEntitiesByMetadataInput,
    UpdateEntityInput,
    UpdateJobStatusInput,
    UpdateLogInput,
    UpdateProtocolInput,
    UpdateRelationshipInput,
)
from nebula_mcp.query_loader import QueryLoader

QUERIES = QueryLoader(Path(__file__).resolve().parents[1] / "queries")

load_dotenv()

ADMIN_SCOPES = {"vault-only", "sensitive"}


def _clamp_limit(value: int) -> int:
    if value < 1:
        return 1
    if value > MAX_PAGE_LIMIT:
        return MAX_PAGE_LIMIT
    return value


def _clamp_hops(value: int) -> int:
    if value < 1:
        return 1
    if value > MAX_GRAPH_HOPS:
        return MAX_GRAPH_HOPS
    return value


def _require_admin(agent: dict, enums: Any) -> None:
    scope_names = scope_names_from_ids(agent.get("scopes", []), enums)
    if not any(scope in ADMIN_SCOPES for scope in scope_names):
        raise ValueError("Admin scope required")


def _is_admin(agent: dict, enums: Any) -> bool:
    scope_names = scope_names_from_ids(agent.get("scopes", []), enums)
    return any(scope in ADMIN_SCOPES for scope in scope_names)


def _scope_filter_ids(agent: dict, enums: Any) -> list | None:
    if _is_admin(agent, enums):
        return None
    return agent.get("scopes", []) or []


async def _node_allowed(
    pool: Pool, enums: Any, agent: dict, node_type: str, node_id: str
) -> bool:
    if _is_admin(agent, enums):
        return True
    if node_type == "entity":
        row = await pool.fetchrow(QUERIES["entities/get_by_id"], node_id)
        if not row:
            return False
        scopes = row.get("privacy_scope_ids") or []
        if not scopes:
            return True
        return any(s in agent.get("scopes", []) for s in scopes)
    if node_type == "knowledge":
        scope_ids = agent.get("scopes", []) or []
        row = await pool.fetchrow(QUERIES["knowledge/get"], node_id, scope_ids)
        return row is not None
    return True


async def _validate_relationship_node(
    pool: Pool,
    enums: Any,
    agent: dict,
    node_type: str,
    node_id: str,
    label: str,
) -> None:
    if node_type == "entity":
        row = await pool.fetchrow(QUERIES["entities/get_by_id"], node_id)
        if not row:
            raise ValueError(f"{label} entity not found")
        if not await _node_allowed(pool, enums, agent, node_type, node_id):
            raise ValueError("Access denied")
        return
    if node_type == "knowledge":
        scope_ids = _scope_filter_ids(agent, enums)
        row = await pool.fetchrow(QUERIES["knowledge/get"], node_id, scope_ids)
        if not row:
            raise ValueError(f"{label} knowledge not found")
        return
    if node_type == "job":
        row = await pool.fetchrow(QUERIES["jobs/get"], node_id)
        if not row:
            raise ValueError(f"{label} job not found")
        return
    raise ValueError(f"Unsupported {label.lower()} type")


@asynccontextmanager
async def lifespan(app: FastMCP) -> AsyncIterator[dict[str, Any]]:
    """Initialize and teardown shared application resources.

    Args:
        app: FastMCP application instance.

    Yields:
        Dict with pool, enums, and agent context.
    """

    pool = await get_pool()
    try:
        enums = await load_enums(pool)
        agent = await authenticate_agent(pool)
        yield {"pool": pool, "enums": enums, "agent": agent}
    finally:
        await pool.close()


mcp = FastMCP("NebulaMCP", json_response=True, lifespan=lifespan)


async def _run_bulk_import(
    payload: BulkImportInput,
    ctx: Context,
    normalizer: Callable[[dict[str, Any], dict[str, Any] | None], dict[str, Any]],
    executor: Callable[..., Any],
    action: str,
) -> dict:
    """Run a bulk import with normalization and approval gating.

    Args:
        payload: Bulk import request payload.
        ctx: MCP request context.
        normalizer: Callable that validates and normalizes item rows.
        executor: Callable that persists a normalized item.
        action: Approval action name for audit/approval workflow.

    Returns:
        Dict with created count, error list, and created items.
    """
    pool, enums, agent = await require_context(ctx)
    items = extract_items(payload.format, payload.data, payload.items)
    if approval := await maybe_require_approval(
        pool, agent, action, {"format": payload.format, "items": items}
    ):
        return approval

    allowed_scopes = scope_names_from_ids(agent.get("scopes", []), enums)
    created: list[dict] = []
    errors: list[dict] = []

    async with pool.acquire() as conn:
        async with conn.transaction():
            await conn.execute(
                "SELECT set_config('app.changed_by_type', $1, true)", "agent"
            )
            await conn.execute(
                "SELECT set_config('app.changed_by_id', $1, true)", str(agent["id"])
            )

            for idx, item in enumerate(items, start=1):
                try:
                    normalized = normalizer(item, payload.defaults)
                    if "scopes" in normalized:
                        normalized["scopes"] = enforce_scope_subset(
                            normalized["scopes"], allowed_scopes
                        )
                    result = await executor(conn, enums, normalized)
                    created.append(result)
                except Exception as exc:
                    errors.append({"row": idx, "error": str(exc)})

    return {
        "created": len(created),
        "failed": len(errors),
        "errors": errors,
        "items": created,
    }


# --- Admin Tools ---


@mcp.tool()
async def reload_enums(ctx: Context) -> str:
    """Reload enum cache from the database.

    Args:
        ctx: MCP request context.

    Returns:
        Confirmation message.
    """

    lifespan_ctx = ctx.request_context.lifespan_context
    if not lifespan_ctx or "pool" not in lifespan_ctx:
        raise ValueError("Pool not initialized")

    lifespan_ctx["enums"] = await load_enums(lifespan_ctx["pool"])
    return "Enums reloaded."


@mcp.tool()
async def get_pending_approvals(ctx: Context) -> list[dict]:
    """Get pending approval requests for the authenticated agent.

    Args:
        ctx: MCP request context.

    Returns:
        List of pending approval request dicts.
    """

    pool, enums, agent = await require_context(ctx)

    rows = await pool.fetch(QUERIES["approvals/get_pending_by_agent"], agent["id"])
    return [dict(r) for r in rows]


@mcp.tool()
async def get_approval_diff(payload: GetApprovalDiffInput, ctx: Context) -> dict:
    """Compute diff for an approval request."""

    pool = await require_pool(ctx)
    return await compute_approval_diff(pool, payload.approval_id)


# --- Bulk Import Tools ---


@mcp.tool()
async def bulk_import_entities(payload: BulkImportInput, ctx: Context) -> dict:
    """Bulk import entities from CSV or JSON."""

    return await _run_bulk_import(
        payload,
        ctx,
        normalize_entity,
        execute_create_entity,
        "bulk_import_entities",
    )


@mcp.tool()
async def bulk_import_knowledge(payload: BulkImportInput, ctx: Context) -> dict:
    """Bulk import knowledge items from CSV or JSON."""

    return await _run_bulk_import(
        payload,
        ctx,
        normalize_knowledge,
        execute_create_knowledge,
        "bulk_import_knowledge",
    )


@mcp.tool()
async def bulk_import_relationships(payload: BulkImportInput, ctx: Context) -> dict:
    """Bulk import relationships from CSV or JSON."""

    return await _run_bulk_import(
        payload,
        ctx,
        normalize_relationship,
        execute_create_relationship,
        "bulk_import_relationships",
    )


@mcp.tool()
async def bulk_import_jobs(payload: BulkImportInput, ctx: Context) -> dict:
    """Bulk import jobs from CSV or JSON."""

    return await _run_bulk_import(
        payload,
        ctx,
        normalize_job,
        execute_create_job,
        "bulk_import_jobs",
    )


# --- Entity Tools ---


@mcp.tool()
async def get_entity(payload: GetEntityInput, ctx: Context) -> dict:
    """Retrieve entity by ID with privacy filtering.

    Args:
        payload: Input with entity_id.
        ctx: MCP request context.

    Returns:
        Entity dict with filtered metadata.

    Raises:
        ValueError: If entity not found or access denied.
    """

    pool, enums, agent = await require_context(ctx)

    row = await pool.fetchrow(QUERIES["entities/get"], payload.entity_id)
    if not row:
        raise ValueError("Entity not found")

    entity = dict(row)

    # Privacy check
    entity_scopes = entity.get("privacy_scope_ids", [])
    agent_scopes = agent.get("scopes", [])

    if entity_scopes and not any(s in agent_scopes for s in entity_scopes):
        raise ValueError("Access denied: entity not in agent scopes")

    # Filter context segments
    if entity.get("metadata"):
        scope_names = [enums.scopes.id_to_name.get(s, "") for s in agent_scopes]
        entity["metadata"] = filter_context_segments(entity["metadata"], scope_names)

    return entity


@mcp.tool()
async def query_entities(payload: QueryEntitiesInput, ctx: Context) -> list[dict]:
    """Search entities with filters and full-text search."""

    pool, enums, agent = await require_context(ctx)

    type_id = require_entity_type(payload.type, enums) if payload.type else None
    allowed_scopes = scope_names_from_ids(agent.get("scopes", []), enums)
    requested_scopes = enforce_scope_subset(payload.scopes, allowed_scopes)
    scope_ids = require_scopes(requested_scopes, enums)
    limit = _clamp_limit(payload.limit)
    offset = max(0, payload.offset)

    rows = await pool.fetch(
        QUERIES["entities/query"],
        type_id,
        payload.tags or None,
        payload.search_text,
        payload.status_category,
        scope_ids,
        limit,
        offset,
    )
    return [dict(r) for r in rows]


@mcp.tool()
async def update_entity(payload: UpdateEntityInput, ctx: Context) -> dict:
    """Update entity metadata, tags, or status."""

    pool, enums, agent = await require_context(ctx)

    if resp := await maybe_require_approval(
        pool, agent, "update_entity", payload.model_dump()
    ):
        return resp

    return await execute_update_entity(pool, enums, payload.model_dump())


@mcp.tool()
async def bulk_update_entity_tags(
    payload: BulkUpdateEntityTagsInput, ctx: Context
) -> dict:
    """Bulk update entity tags."""

    pool, enums, agent = await require_context(ctx)

    if resp := await maybe_require_approval(
        pool, agent, "bulk_update_entity_tags", payload.model_dump()
    ):
        return resp

    op = normalize_bulk_operation(payload.op)
    updated = await do_bulk_update_entity_tags(
        pool, payload.entity_ids, payload.tags, op
    )
    return {"updated": len(updated), "entity_ids": updated}


@mcp.tool()
async def bulk_update_entity_scopes(
    payload: BulkUpdateEntityScopesInput, ctx: Context
) -> dict:
    """Bulk update entity privacy scopes."""

    pool, enums, agent = await require_context(ctx)

    op = normalize_bulk_operation(payload.op)
    allowed_scopes = scope_names_from_ids(agent.get("scopes", []), enums)
    requested_scopes = enforce_scope_subset(payload.scopes, allowed_scopes)
    data = payload.model_dump()
    data["scopes"] = requested_scopes
    if resp := await maybe_require_approval(
        pool, agent, "bulk_update_entity_scopes", data
    ):
        return resp
    updated = await do_bulk_update_entity_scopes(
        pool, enums, payload.entity_ids, requested_scopes, op
    )
    return {"updated": len(updated), "entity_ids": updated}


@mcp.tool()
async def search_entities_by_metadata(
    payload: SearchEntitiesByMetadataInput, ctx: Context
) -> list[dict]:
    """Search entities by JSONB metadata containment."""

    pool, enums, agent = await require_context(ctx)
    scope_ids = agent.get("scopes", [])
    limit = _clamp_limit(payload.limit)

    rows = await pool.fetch(
        QUERIES["entities/search_by_metadata"],
        json.dumps(payload.metadata_query),
        limit,
        scope_ids,
    )
    return [dict(r) for r in rows]


@mcp.tool()
async def create_entity(payload: CreateEntityInput, ctx: Context) -> dict:
    """Create an entity in the Nebula database."""

    pool, enums, agent = await require_context(ctx)
    allowed_scopes = scope_names_from_ids(agent.get("scopes", []), enums)
    requested_scopes = enforce_scope_subset(payload.scopes, allowed_scopes)
    data = payload.model_dump()
    data["scopes"] = requested_scopes

    if resp := await maybe_require_approval(pool, agent, "create_entity", data):
        return resp

    return await execute_create_entity(pool, enums, data)


@mcp.tool()
async def get_entity_history(
    payload: GetEntityHistoryInput, ctx: Context
) -> list[dict]:
    """List audit history entries for an entity."""

    pool, enums, agent = await require_context(ctx)
    limit = _clamp_limit(payload.limit)
    offset = max(0, payload.offset)
    return await fetch_entity_history(pool, payload.entity_id, limit, offset)


@mcp.tool()
async def query_audit_log(payload: QueryAuditLogInput, ctx: Context) -> list[dict]:
    """Query audit log entries with filters."""

    pool, enums, agent = await require_context(ctx)
    _require_admin(agent, enums)
    limit = _clamp_limit(payload.limit)
    offset = max(0, payload.offset)
    return await fetch_audit_log(
        pool,
        payload.table_name,
        payload.action,
        payload.actor_type,
        payload.actor_id,
        payload.record_id,
        payload.scope_id,
        limit,
        offset,
    )


@mcp.tool()
async def revert_entity(payload: RevertEntityInput, ctx: Context) -> dict:
    """Revert an entity to a historical audit entry."""

    pool, enums, agent = await require_context(ctx)
    audit_row = await pool.fetchrow(QUERIES["audit/get"], payload.audit_id)
    if not audit_row:
        raise ValueError("Audit entry not found")
    if audit_row.get("table_name") != "entities":
        raise ValueError("Audit entry is not for entities")
    if audit_row.get("record_id") != payload.entity_id:
        raise ValueError("Audit entry does not match entity")

    if resp := await maybe_require_approval(
        pool, agent, "revert_entity", payload.model_dump()
    ):
        return resp

    async with pool.acquire() as conn:
        await conn.execute("SET app.changed_by_type = 'agent'")
        await conn.execute("SET app.changed_by_id = $1", agent["id"])
        try:
            return await do_revert_entity(conn, payload.entity_id, payload.audit_id)
        finally:
            await conn.execute("RESET app.changed_by_type")
            await conn.execute("RESET app.changed_by_id")


# --- Knowledge Tools ---


@mcp.tool()
async def create_knowledge(payload: CreateKnowledgeInput, ctx: Context) -> dict:
    """Create a knowledge item."""

    pool, enums, agent = await require_context(ctx)
    allowed_scopes = scope_names_from_ids(agent.get("scopes", []), enums)
    requested_scopes = enforce_scope_subset(payload.scopes, allowed_scopes)
    data = payload.model_dump()
    data["scopes"] = requested_scopes

    if resp := await maybe_require_approval(pool, agent, "create_knowledge", data):
        return resp

    return await execute_create_knowledge(pool, enums, data)


@mcp.tool()
async def query_knowledge(payload: QueryKnowledgeInput, ctx: Context) -> list[dict]:
    """Search knowledge items with filters."""

    pool, enums, agent = await require_context(ctx)

    allowed_scopes = scope_names_from_ids(agent.get("scopes", []), enums)
    requested_scopes = enforce_scope_subset(payload.scopes, allowed_scopes)
    scope_ids = require_scopes(requested_scopes, enums)
    limit = _clamp_limit(payload.limit)
    offset = max(0, payload.offset)

    rows = await pool.fetch(
        QUERIES["knowledge/query"],
        payload.source_type,
        payload.tags or None,
        payload.search_text,
        scope_ids,
        limit,
        offset,
    )
    return [dict(r) for r in rows]


@mcp.tool()
async def link_knowledge_to_entity(payload: LinkKnowledgeInput, ctx: Context) -> dict:
    """Link knowledge item to entity via relationship."""

    pool, enums, agent = await require_context(ctx)
    relationship_payload = {
        "source_type": "knowledge",
        "source_id": payload.knowledge_id,
        "target_type": "entity",
        "target_id": payload.entity_id,
        "relationship_type": payload.relationship_type,
        "properties": {},
    }
    if resp := await maybe_require_approval(
        pool, agent, "create_relationship", relationship_payload
    ):
        return resp

    return await execute_create_relationship(pool, enums, relationship_payload)


# --- Log Tools ---


@mcp.tool()
async def create_log(payload: CreateLogInput, ctx: Context) -> dict:
    """Create a log entry."""

    pool, enums, agent = await require_context(ctx)
    if resp := await maybe_require_approval(
        pool, agent, "create_log", payload.model_dump()
    ):
        return resp
    return await execute_create_log(pool, enums, payload.model_dump())


@mcp.tool()
async def get_log(payload: GetLogInput, ctx: Context) -> dict:
    """Retrieve a log entry by id."""

    pool, enums, agent = await require_context(ctx)
    row = await pool.fetchrow(QUERIES["logs/get"], payload.log_id)
    if not row:
        raise ValueError(f"Log '{payload.log_id}' not found")
    return dict(row)


@mcp.tool()
async def query_logs(payload: QueryLogsInput, ctx: Context) -> list[dict]:
    """Query log entries."""

    pool, enums, agent = await require_context(ctx)
    log_type_id = None
    if payload.log_type:
        log_type_id = require_log_type(payload.log_type, enums)
    tags = payload.tags if payload.tags else None
    limit = _clamp_limit(payload.limit)
    offset = max(0, payload.offset)
    rows = await pool.fetch(
        QUERIES["logs/query"],
        log_type_id,
        tags,
        payload.status_category,
        limit,
        offset,
    )
    return [dict(r) for r in rows]


@mcp.tool()
async def update_log(payload: UpdateLogInput, ctx: Context) -> dict:
    """Update a log entry."""

    pool, enums, agent = await require_context(ctx)
    if resp := await maybe_require_approval(
        pool, agent, "update_log", payload.model_dump()
    ):
        return resp
    return await execute_update_log(pool, enums, payload.model_dump())


# --- Relationship Tools ---


@mcp.tool()
async def create_relationship(payload: CreateRelationshipInput, ctx: Context) -> dict:
    """Create a polymorphic relationship between items."""

    pool, enums, agent = await require_context(ctx)
    await _validate_relationship_node(
        pool, enums, agent, payload.source_type, payload.source_id, "Source"
    )
    await _validate_relationship_node(
        pool, enums, agent, payload.target_type, payload.target_id, "Target"
    )

    if resp := await maybe_require_approval(
        pool, agent, "create_relationship", payload.model_dump()
    ):
        return resp

    return await execute_create_relationship(pool, enums, payload.model_dump())


@mcp.tool()
async def get_relationships(payload: GetRelationshipsInput, ctx: Context) -> list[dict]:
    """Get relationships for an item with direction filter."""

    pool, enums, agent = await require_context(ctx)
    scope_ids = _scope_filter_ids(agent, enums)

    rows = await pool.fetch(
        QUERIES["relationships/get"],
        payload.source_type,
        payload.source_id,
        payload.direction,
        payload.relationship_type,
        scope_ids,
    )
    return [dict(r) for r in rows]


@mcp.tool()
async def query_relationships(
    payload: QueryRelationshipsInput, ctx: Context
) -> list[dict]:
    """Search relationships with filters."""

    pool, enums, agent = await require_context(ctx)
    limit = _clamp_limit(payload.limit)
    scope_ids = _scope_filter_ids(agent, enums)

    rows = await pool.fetch(
        QUERIES["relationships/query"],
        payload.source_type,
        payload.target_type,
        payload.relationship_types or None,
        payload.status_category,
        limit,
        scope_ids,
    )
    return [dict(r) for r in rows]


@mcp.tool()
async def update_relationship(payload: UpdateRelationshipInput, ctx: Context) -> dict:
    """Update relationship properties or status."""

    pool, enums, agent = await require_context(ctx)
    if resp := await maybe_require_approval(
        pool, agent, "update_relationship", payload.model_dump()
    ):
        return resp

    status_id = require_status(payload.status, enums) if payload.status else None

    row = await pool.fetchrow(
        QUERIES["relationships/update"],
        payload.relationship_id,
        json.dumps(payload.properties) if payload.properties else None,
        status_id,
    )
    return dict(row) if row else {}


# --- Graph Tools ---


def _decode_graph_path(path: list[str] | None) -> list[dict]:
    """Decode a graph path list into typed node dictionaries.

    Args:
        path: List of encoded path entries in the form "type:id".

    Returns:
        A list of node dictionaries with "type" and "id".
    """
    if not path:
        return []

    nodes: list[dict] = []
    for item in path:
        if ":" not in item:
            continue
        node_type, node_id = item.split(":", 1)
        nodes.append({"type": node_type, "id": node_id})
    return nodes


@mcp.tool()
async def graph_neighbors(payload: GraphNeighborsInput, ctx: Context) -> list[dict]:
    """Return neighbors within max hops."""

    pool, enums, agent = await require_context(ctx)
    max_hops = _clamp_hops(payload.max_hops)
    limit = _clamp_limit(payload.limit)

    if not await _node_allowed(
        pool, enums, agent, payload.source_type, payload.source_id
    ):
        raise ValueError("Access denied")

    rows = await pool.fetch(
        QUERIES["graph/neighbors"],
        payload.source_type,
        payload.source_id,
        max_hops,
        limit,
    )
    results = []
    for row in rows:
        path_nodes = _decode_graph_path(row["path"])
        allowed = True
        for node in path_nodes:
            if not await _node_allowed(pool, enums, agent, node["type"], node["id"]):
                allowed = False
                break
        if not allowed:
            continue
        results.append(
            {
                "node_type": row["node_type"],
                "node_id": row["node_id"],
                "depth": row["depth"],
                "path": path_nodes,
            }
        )
    return results


@mcp.tool()
async def graph_shortest_path(payload: GraphShortestPathInput, ctx: Context) -> dict:
    """Return shortest path between two nodes."""

    pool, enums, agent = await require_context(ctx)
    max_hops = _clamp_hops(payload.max_hops)

    if not await _node_allowed(
        pool, enums, agent, payload.source_type, payload.source_id
    ):
        raise ValueError("Access denied")
    if not await _node_allowed(
        pool, enums, agent, payload.target_type, payload.target_id
    ):
        raise ValueError("Access denied")

    row = await pool.fetchrow(
        QUERIES["graph/shortest_path"],
        payload.source_type,
        payload.source_id,
        payload.target_type,
        payload.target_id,
        max_hops,
    )
    if not row:
        raise ValueError("No path found")

    path_nodes = _decode_graph_path(row["path"])
    for node in path_nodes:
        if not await _node_allowed(pool, enums, agent, node["type"], node["id"]):
            raise ValueError("No path found")

    return {"depth": row["depth"], "path": path_nodes}


# --- Job Tools ---


@mcp.tool()
async def create_job(payload: CreateJobInput, ctx: Context) -> dict:
    """Create a new job with auto-generated ID."""

    pool, enums, agent = await require_context(ctx)

    if resp := await maybe_require_approval(
        pool, agent, "create_job", payload.model_dump()
    ):
        return resp

    return await execute_create_job(pool, enums, payload.model_dump())


@mcp.tool()
async def get_job(payload: GetJobInput, ctx: Context) -> dict:
    """Retrieve job by ID."""

    pool, enums, agent = await require_context(ctx)

    row = await pool.fetchrow(QUERIES["jobs/get"], payload.job_id)
    if not row:
        raise ValueError(f"Job '{payload.job_id}' not found")

    return dict(row)


@mcp.tool()
async def query_jobs(payload: QueryJobsInput, ctx: Context) -> list[dict]:
    """Search jobs with multiple filters."""

    pool, enums, agent = await require_context(ctx)
    limit = _clamp_limit(payload.limit)

    rows = await pool.fetch(
        QUERIES["jobs/query"],
        payload.status_names or None,
        payload.assigned_to,
        payload.agent_id,
        payload.priority,
        payload.due_before,
        payload.due_after,
        payload.overdue_only,
        payload.parent_job_id,
        limit,
    )
    return [dict(r) for r in rows]


@mcp.tool()
async def update_job_status(payload: UpdateJobStatusInput, ctx: Context) -> dict:
    """Update job status with optional completion timestamp."""

    pool, enums, agent = await require_context(ctx)
    if resp := await maybe_require_approval(
        pool, agent, "update_job_status", payload.model_dump()
    ):
        return resp

    status_id = require_status(payload.status, enums)

    row = await pool.fetchrow(
        QUERIES["jobs/update_status"],
        payload.job_id,
        status_id,
        payload.status_reason,
        payload.completed_at,
    )
    if not row:
        raise ValueError(f"Job '{payload.job_id}' not found")

    return dict(row)


@mcp.tool()
async def create_subtask(payload: CreateSubtaskInput, ctx: Context) -> dict:
    """Create a subtask under a parent job."""

    pool, enums, agent = await require_context(ctx)
    subtask_payload = {
        "title": payload.title,
        "description": payload.description,
        "job_type": None,
        "assigned_to": None,
        "agent_id": None,
        "priority": payload.priority,
        "parent_job_id": payload.parent_job_id,
        "due_at": payload.due_at,
        "metadata": {},
    }
    if resp := await maybe_require_approval(pool, agent, "create_job", subtask_payload):
        return resp
    return await execute_create_job(pool, enums, subtask_payload)


# --- File Tools ---


@mcp.tool()
async def create_file(payload: CreateFileInput, ctx: Context) -> dict:
    """Create a file metadata record."""

    pool, enums, agent = await require_context(ctx)

    if resp := await maybe_require_approval(
        pool, agent, "create_file", payload.model_dump()
    ):
        return resp

    return await execute_create_file(pool, enums, payload.model_dump())


@mcp.tool()
async def get_file(payload: GetFileInput, ctx: Context) -> dict:
    """Retrieve a file by ID."""

    pool, enums, agent = await require_context(ctx)

    row = await pool.fetchrow(QUERIES["files/get"], payload.file_id)
    if not row:
        raise ValueError(f"File '{payload.file_id}' not found")

    return dict(row)


@mcp.tool()
async def list_files(payload: QueryFilesInput, ctx: Context) -> list[dict]:
    """List files with filters."""

    pool, enums, agent = await require_context(ctx)
    limit = _clamp_limit(payload.limit)
    offset = max(0, payload.offset)

    rows = await pool.fetch(
        QUERIES["files/list"],
        payload.tags or None,
        payload.mime_type,
        payload.status_category,
        limit,
        offset,
    )
    return [dict(r) for r in rows]


async def _attach_file(
    ctx: Context, target_type: str, payload: AttachFileInput
) -> dict:
    """Attach a file to a target entity or knowledge item.

    Args:
        ctx: MCP request context.
        target_type: Target type for relationship (entity or knowledge).
        payload: File attachment request payload.

    Returns:
        Relationship record or approval response payload.
    """
    pool, enums, agent = await require_context(ctx)

    if resp := await maybe_require_approval(
        pool,
        agent,
        f"attach_file_to_{target_type}",
        {"file_id": payload.file_id, "target_id": payload.target_id},
    ):
        return resp

    file_row = await pool.fetchrow(QUERIES["files/get"], payload.file_id)
    if not file_row:
        raise ValueError("File not found")

    relationship = {
        "source_type": "file",
        "source_id": payload.file_id,
        "target_type": target_type,
        "target_id": payload.target_id,
        "relationship_type": payload.relationship_type,
        "properties": {},
    }

    return await execute_create_relationship(pool, enums, relationship)


@mcp.tool()
async def attach_file_to_entity(payload: AttachFileInput, ctx: Context) -> dict:
    """Attach a file to an entity."""

    return await _attach_file(ctx, "entity", payload)


@mcp.tool()
async def attach_file_to_knowledge(payload: AttachFileInput, ctx: Context) -> dict:
    """Attach a file to a knowledge item."""

    return await _attach_file(ctx, "knowledge", payload)


@mcp.tool()
async def attach_file_to_job(payload: AttachFileInput, ctx: Context) -> dict:
    """Attach a file to a job."""

    return await _attach_file(ctx, "job", payload)


# --- Protocol Tools ---


@mcp.tool()
async def get_protocol(payload: GetProtocolInput, ctx: Context) -> dict:
    """Retrieve protocol by name."""

    pool = await require_pool(ctx)

    row = await pool.fetchrow(QUERIES["protocols/get"], payload.protocol_name)
    if not row:
        raise ValueError(f"Protocol '{payload.protocol_name}' not found")

    return dict(row)


@mcp.tool()
async def create_protocol(payload: CreateProtocolInput, ctx: Context) -> dict:
    """Create a protocol."""

    pool, enums, agent = await require_context(ctx)
    data = payload.model_dump()
    if not _is_admin(agent, enums):
        data["trusted"] = False
    if resp := await maybe_require_approval(pool, agent, "create_protocol", data):
        return resp
    return await execute_create_protocol(pool, enums, data)


@mcp.tool()
async def update_protocol(payload: UpdateProtocolInput, ctx: Context) -> dict:
    """Update a protocol."""

    pool, enums, agent = await require_context(ctx)
    data = payload.model_dump()
    if not _is_admin(agent, enums):
        data["trusted"] = None
    if resp := await maybe_require_approval(pool, agent, "update_protocol", data):
        return resp
    return await execute_update_protocol(pool, enums, data)


@mcp.tool()
async def list_active_protocols(ctx: Context) -> list[dict]:
    """List all active protocols."""

    pool = await require_pool(ctx)
    rows = await pool.fetch(QUERIES["protocols/list_active"])
    return [dict(r) for r in rows]


# --- Agent Tools ---


@mcp.tool()
async def get_agent_info(payload: GetAgentInfoInput, ctx: Context) -> dict:
    """Retrieve agent configuration including system_prompt."""

    pool, enums, agent = await require_context(ctx)
    _require_admin(agent, enums)

    row = await pool.fetchrow(QUERIES["agents/get_info"], payload.name)
    if not row:
        raise ValueError(f"Agent '{payload.name}' not found")

    return dict(row)


@mcp.tool()
async def list_agents(payload: ListAgentsInput, ctx: Context) -> list[dict]:
    """List agents by status category."""

    pool, enums, agent = await require_context(ctx)
    _require_admin(agent, enums)
    rows = await pool.fetch(QUERIES["agents/list"], payload.status_category)
    return [dict(r) for r in rows]


# --- Main ---


def main() -> None:
    """Run the Nebula MCP server."""

    load_dotenv()
    mcp.run()


if __name__ == "__main__":
    main()
