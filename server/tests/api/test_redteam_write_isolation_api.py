"""Red team API tests for write isolation on private records."""

# Standard Library
import json

# Third-Party
from httpx import ASGITransport, AsyncClient
import pytest

# Local
from nebula_api.app import app
from nebula_api.auth import require_auth


async def _make_agent(db_pool, enums, name, scopes, requires_approval):
    """Insert a test agent for write isolation scenarios."""

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
    """Insert a test entity for write isolation scenarios."""

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


async def _make_knowledge(db_pool, enums, title, scopes):
    """Insert a test knowledge item for write isolation scenarios."""

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


async def _make_log(db_pool, enums):
    """Insert a test log for write isolation scenarios."""

    status_id = enums.statuses.name_to_id["active"]
    log_type_id = enums.log_types.name_to_id["note"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO logs (log_type_id, status_id, value, metadata)
        VALUES ($1, $2, $3::jsonb, $4::jsonb)
        RETURNING *
        """,
        log_type_id,
        status_id,
        json.dumps({"note": "secret"}),
        json.dumps({"meta": "secret"}),
    )
    return dict(row)


async def _make_file(db_pool, enums):
    """Insert a test file for write isolation scenarios."""

    status_id = enums.statuses.name_to_id["active"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO files (filename, file_path, status_id, metadata)
        VALUES ($1, $2, $3, $4::jsonb)
        RETURNING *
        """,
        "secret.txt",
        "/vault/secret.txt",
        status_id,
        json.dumps({"meta": "secret"}),
    )
    return dict(row)


async def _attach_relationship(
    db_pool, enums, source_type, source_id, target_type, target_id, rel_name
):
    """Attach a relationship between two nodes for isolation tests."""

    status_id = enums.statuses.name_to_id["active"]
    rel_type_id = enums.relationship_types.name_to_id[rel_name]

    await db_pool.execute(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id, properties)
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
        """,
        source_type,
        str(source_id),
        target_type,
        str(target_id),
        rel_type_id,
        status_id,
        json.dumps({"note": "link"}),
    )


def _auth_override(agent_id, enums):
    """Build an auth override for public agent requests."""

    auth_dict = {
        "key_id": None,
        "caller_type": "agent",
        "entity_id": None,
        "entity": None,
        "agent_id": agent_id,
        "agent": {"id": agent_id},
        "scopes": [enums.scopes.name_to_id["public"]],
    }

    async def mock_auth():
        """Mock auth for public agent."""

        return auth_dict

    return mock_auth


@pytest.mark.asyncio
@pytest.mark.xfail(reason="knowledge updates should enforce write scopes")
async def test_api_update_knowledge_denies_private_scope(db_pool, enums):
    """Public agents should not update private knowledge items."""

    private_knowledge = await _make_knowledge(db_pool, enums, "Private", ["sensitive"])
    viewer = await _make_agent(db_pool, enums, "knowledge-viewer", ["public"], False)

    app.dependency_overrides[require_auth] = _auth_override(viewer["id"], enums)
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.patch(
            f"/api/knowledge/{private_knowledge['id']}",
            json={"title": "Hijacked"},
        )
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 403


@pytest.mark.asyncio
@pytest.mark.xfail(reason="log updates should enforce attachment privacy")
async def test_api_update_log_denies_private_attachment(db_pool, enums):
    """Public agents should not update logs attached to private entities."""

    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    log_row = await _make_log(db_pool, enums)
    await _attach_relationship(
        db_pool,
        enums,
        "entity",
        private_entity["id"],
        "log",
        log_row["id"],
        "logged-by",
    )
    viewer = await _make_agent(db_pool, enums, "log-viewer", ["public"], False)

    app.dependency_overrides[require_auth] = _auth_override(viewer["id"], enums)
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.patch(
            f"/api/logs/{log_row['id']}",
            json={"metadata": {"note": "hijack"}},
        )
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 403


@pytest.mark.asyncio
@pytest.mark.xfail(reason="file updates should enforce attachment privacy")
async def test_api_update_file_denies_private_attachment(db_pool, enums):
    """Public agents should not update files attached to private entities."""

    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    file_row = await _make_file(db_pool, enums)
    await _attach_relationship(
        db_pool,
        enums,
        "entity",
        private_entity["id"],
        "file",
        file_row["id"],
        "has-file",
    )
    viewer = await _make_agent(db_pool, enums, "file-viewer", ["public"], False)

    app.dependency_overrides[require_auth] = _auth_override(viewer["id"], enums)
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.patch(
            f"/api/files/{file_row['id']}",
            json={"metadata": {"note": "hijack"}},
        )
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 403
