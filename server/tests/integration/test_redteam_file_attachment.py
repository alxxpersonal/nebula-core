"""Red team tests for file attachment isolation."""

# Standard Library
import json
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import AttachFileInput
from nebula_mcp.server import (
    attach_file_to_entity,
    attach_file_to_job,
    attach_file_to_knowledge,
)


def _make_context(pool, enums, agent):
    """Build a mock MCP context for file attachment tests."""

    ctx = MagicMock()
    ctx.request_context.lifespan_context = {
        "pool": pool,
        "enums": enums,
        "agent": agent,
    }
    return ctx


async def _make_agent(db_pool, enums, name, scopes, requires_approval):
    """Insert a test agent for file attachment scenarios."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]

    row = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        name,
        "redteam agent",
        scope_ids,
        requires_approval,
        status_id,
    )
    return dict(row)


async def _make_entity(db_pool, enums, name, scopes):
    """Insert a test entity for file attachment scenarios."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.entity_types.name_to_id["person"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]

    row = await db_pool.fetchrow(
        """
        INSERT INTO entities (name, type_id, status_id, privacy_scope_ids, tags, metadata)
        VALUES ($1, $2, $3, $4, $5, $6::jsonb)
        RETURNING *
        """,
        name,
        type_id,
        status_id,
        scope_ids,
        ["test"],
        json.dumps({"context_segments": [{"text": "secret", "scopes": scopes}]}),
    )
    return dict(row)


async def _make_file(db_pool, enums):
    """Insert a test file for attachment scenarios."""

    status_id = enums.statuses.name_to_id["active"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO files (filename, file_path, status_id, metadata)
        VALUES ($1, $2, $3, $4::jsonb)
        RETURNING *
        """,
        "attached.txt",
        "/vault/attached.txt",
        status_id,
        json.dumps({"note": "attachment"}),
    )
    return dict(row)


async def _make_job(db_pool, enums, agent_id):
    """Insert a test job for attachment scenarios."""

    status_id = enums.statuses.name_to_id["active"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO jobs (title, status_id, agent_id, metadata)
        VALUES ($1, $2, $3, $4::jsonb)
        RETURNING *
        """,
        "Private Job",
        status_id,
        agent_id,
        json.dumps({"secret": "job"}),
    )
    return dict(row)


async def _make_knowledge(db_pool, enums, title, scopes):
    """Insert a test knowledge item for attachment scenarios."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id[s] for s in scopes]

    row = await db_pool.fetchrow(
        """
        INSERT INTO knowledge_items (title, source_type, content, privacy_scope_ids, status_id, tags, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
        RETURNING *
        """,
        title,
        "note",
        "secret",
        scope_ids,
        status_id,
        ["test"],
        json.dumps({"note": "secret"}),
    )
    return dict(row)


@pytest.mark.asyncio
async def test_attach_file_to_private_entity_denied(db_pool, enums):
    """Public agents should not attach files to private entities."""

    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    file_row = await _make_file(db_pool, enums)

    viewer = await _make_agent(db_pool, enums, "file-attacher", ["public"], False)
    ctx = _make_context(db_pool, enums, viewer)

    payload = AttachFileInput(
        file_id=str(file_row["id"]), target_id=str(private_entity["id"])
    )

    with pytest.raises(ValueError):
        await attach_file_to_entity(payload, ctx)


@pytest.mark.asyncio
async def test_attach_file_to_foreign_job_denied(db_pool, enums):
    """Agents should not attach files to jobs owned by other agents."""

    owner = await _make_agent(db_pool, enums, "job-owner", ["public"], False)
    viewer = await _make_agent(db_pool, enums, "job-viewer", ["public"], False)
    job = await _make_job(db_pool, enums, owner["id"])
    file_row = await _make_file(db_pool, enums)

    ctx = _make_context(db_pool, enums, viewer)
    payload = AttachFileInput(file_id=str(file_row["id"]), target_id=str(job["id"]))

    with pytest.raises(ValueError):
        await attach_file_to_job(payload, ctx)


@pytest.mark.asyncio
async def test_attach_file_to_private_knowledge_denied(db_pool, enums):
    """Public agents should not attach files to private knowledge."""

    knowledge = await _make_knowledge(
        db_pool, enums, "Private Knowledge", ["sensitive"]
    )
    file_row = await _make_file(db_pool, enums)
    viewer = await _make_agent(db_pool, enums, "knowledge-attacher", ["public"], False)
    ctx = _make_context(db_pool, enums, viewer)

    payload = AttachFileInput(
        file_id=str(file_row["id"]), target_id=str(knowledge["id"])
    )

    with pytest.raises(ValueError):
        await attach_file_to_knowledge(payload, ctx)
