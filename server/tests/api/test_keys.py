"""API key management route tests."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_login_creates_entity_and_key(api_no_auth):
    """Test login creates entity and key."""

    r = await api_no_auth.post("/api/keys/login", json={"username": "newuser"})
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["api_key"].startswith("nbl_")
    assert data["username"] == "newuser"
    assert "entity_id" in data


@pytest.mark.asyncio
async def test_login_existing_user(api_no_auth, test_entity):
    """Test login existing user."""

    r = await api_no_auth.post("/api/keys/login", json={"username": "api-test-user"})
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["entity_id"] == str(test_entity["id"])


@pytest.mark.asyncio
async def test_create_additional_key(api):
    """Test create additional key."""

    r = await api.post("/api/keys", json={"name": "second-key"})
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["api_key"].startswith("nbl_")
    assert data["name"] == "second-key"


@pytest.mark.asyncio
async def test_list_keys(api):
    """Test list keys."""

    await api.post("/api/keys", json={"name": "list-test"})
    r = await api.get("/api/keys")
    assert r.status_code == 200
    data = r.json()["data"]
    assert len(data) >= 1


@pytest.mark.asyncio
async def test_revoke_key(api, db_pool, auth_override):
    """Test revoke key."""

    cr = await api.post("/api/keys", json={"name": "revoke-me"})
    key_id = cr.json()["data"]["key_id"]

    r = await api.delete(f"/api/keys/{key_id}")
    assert r.status_code == 200
    assert r.json()["data"]["revoked"] is True

    row = await db_pool.fetchrow(
        "SELECT revoked_at FROM api_keys WHERE id = $1::uuid", key_id
    )
    assert row["revoked_at"] is not None


@pytest.mark.asyncio
async def test_list_all_keys(api, db_pool, test_entity, auth_override):
    """Test list all keys includes user and agent keys."""

    # Create a user key
    await api.post("/api/keys", json={"name": "user-key-for-all"})

    # Create an agent + agent key directly in DB
    from nebula_api.auth import generate_api_key

    agent = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, (SELECT id FROM statuses WHERE name = 'active'))
        RETURNING *
        """,
        "all-keys-test-agent",
        "Agent for list_all test",
        [],
        False,
    )
    raw_key, prefix, key_hash = generate_api_key()
    await db_pool.execute(
        """
        INSERT INTO api_keys (agent_id, key_hash, key_prefix, name)
        VALUES ($1, $2, $3, $4)
        """,
        agent["id"],
        key_hash,
        prefix,
        "agent-key-for-all",
    )

    r = await api.get("/api/keys/all")
    assert r.status_code == 200
    data = r.json()["data"]
    assert len(data) >= 2

    owner_types = {k["owner_type"] for k in data}
    assert "user" in owner_types
    assert "agent" in owner_types
