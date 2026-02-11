"""Red team tests for relationship privacy in API routes."""

# Standard Library
import json

# Third-Party
import pytest

# Local
from nebula_api.app import app
from nebula_api.auth import require_auth


async def _make_entity(db_pool, enums, name, scopes):
    """Insert a test entity for relationship privacy scenarios."""

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
        json.dumps({"note": "scope-test"}),
    )
    return dict(row)


async def _make_relationship(db_pool, enums, source_id, target_id):
    """Insert a relationship between two entities."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.relationship_types.name_to_id["related-to"]

    row = await db_pool.fetchrow(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id, properties)
        VALUES ('entity', $1, 'entity', $2, $3, $4, $5::jsonb)
        RETURNING *
        """,
        str(source_id),
        str(target_id),
        type_id,
        status_id,
        json.dumps({"note": "private-link"}),
    )
    return dict(row)


def _public_auth(public_entity, enums):
    """Build auth payload for a public user."""

    return {
        "key_id": None,
        "caller_type": "user",
        "entity_id": public_entity["id"],
        "entity": public_entity,
        "agent_id": None,
        "agent": None,
        "scopes": [enums.scopes.name_to_id["public"]],
    }


@pytest.mark.asyncio
async def test_api_get_relationships_hides_private_entities(api_no_auth, db_pool, enums):
    """API relationships should hide private entity links."""

    public_entity = await _make_entity(db_pool, enums, "Public", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    rel = await _make_relationship(db_pool, enums, public_entity["id"], private_entity["id"])

    async def mock_auth():
        """Mock auth with public only scope."""

        return _public_auth(public_entity, enums)

    app.dependency_overrides[require_auth] = mock_auth
    try:
        resp = await api_no_auth.get(
            f"/api/relationships/entity/{public_entity['id']}"
        )
    finally:
        app.dependency_overrides.pop(require_auth, None)

    data = resp.json()["data"]
    ids = {row["id"] for row in data}
    assert rel["id"] not in ids


@pytest.mark.asyncio
async def test_api_query_relationships_hides_private_entities(api_no_auth, db_pool, enums):
    """API query relationships should hide private entity links."""

    public_entity = await _make_entity(db_pool, enums, "Public 2", ["public"])
    private_entity = await _make_entity(db_pool, enums, "Private 2", ["sensitive"])
    rel = await _make_relationship(db_pool, enums, public_entity["id"], private_entity["id"])

    async def mock_auth():
        """Mock auth with public only scope."""

        return _public_auth(public_entity, enums)

    app.dependency_overrides[require_auth] = mock_auth
    try:
        resp = await api_no_auth.get("/api/relationships", params={"limit": 50})
    finally:
        app.dependency_overrides.pop(require_auth, None)

    data = resp.json()["data"]
    ids = {row["id"] for row in data}
    assert rel["id"] not in ids
