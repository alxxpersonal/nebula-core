"""Red team API tests for relationship write isolation."""

# Standard Library
import json

# Third-Party
from httpx import ASGITransport, AsyncClient
import pytest

# Local
from nebula_api.app import app
from nebula_api.auth import require_auth


async def _make_agent(db_pool, enums, name, scopes, requires_approval):
    """Insert a test agent for relationship write scenarios."""

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
    """Insert a test entity for relationship write scenarios."""

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


async def _make_relationship(db_pool, enums, source_id, target_id):
    """Insert a relationship between two entities."""

    status_id = enums.statuses.name_to_id["active"]
    rel_type_id = enums.relationship_types.name_to_id["related-to"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id, properties)
        VALUES ('entity', $1, 'entity', $2, $3, $4, $5::jsonb)
        RETURNING *
        """,
        str(source_id),
        str(target_id),
        rel_type_id,
        status_id,
        json.dumps({"note": "private link"}),
    )
    return dict(row)


def _auth_override(agent_id, enums):
    """Build an auth override for public agent API requests."""

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
@pytest.mark.xfail(reason="relationship creation should enforce scope access")
async def test_api_create_relationship_denies_private_target(db_pool, enums):
    """Public agents should not create relationships to private entities."""

    public_entity = await _make_entity(db_pool, enums, "Public", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    viewer = await _make_agent(db_pool, enums, "rel-viewer", ["public"], False)

    app.dependency_overrides[require_auth] = _auth_override(viewer["id"], enums)
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post(
            "/api/relationships/",
            json={
                "source_type": "entity",
                "source_id": str(public_entity["id"]),
                "target_type": "entity",
                "target_id": str(private_entity["id"]),
                "relationship_type": "related-to",
                "properties": {"note": "link"},
            },
        )
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 403


@pytest.mark.asyncio
@pytest.mark.xfail(reason="relationship updates should enforce scope access")
async def test_api_update_relationship_denies_private_target(db_pool, enums):
    """Public agents should not update relationships to private entities."""

    public_entity = await _make_entity(db_pool, enums, "Public", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    relationship = await _make_relationship(
        db_pool, enums, public_entity["id"], private_entity["id"]
    )
    viewer = await _make_agent(db_pool, enums, "rel-viewer-2", ["public"], False)

    app.dependency_overrides[require_auth] = _auth_override(viewer["id"], enums)
    app.state.pool = db_pool
    app.state.enums = enums
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.patch(
            f"/api/relationships/{relationship['id']}",
            json={"properties": {"note": "hijack"}},
        )
    app.dependency_overrides.pop(require_auth, None)

    assert resp.status_code == 403
