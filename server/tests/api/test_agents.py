"""Agent route tests."""

# Third-Party
import pytest


@pytest.fixture
async def agent_row(db_pool, enums):
    """Create a test agent."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id["public"]]
    row = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        "api-test-agent",
        "Test agent",
        scope_ids,
        False,
        status_id,
    )
    return dict(row)


@pytest.mark.asyncio
async def test_get_agent_info(api, agent_row, auth_override, enums):
    """Test get agent info."""

    auth_override["scopes"] = [enums.scopes.name_to_id["admin"]]
    r = await api.get(f"/api/agents/{agent_row['name']}")
    assert r.status_code == 200
    assert r.json()["data"]["name"] == "api-test-agent"


@pytest.mark.asyncio
async def test_get_agent_not_found(api, auth_override, enums):
    """Test get agent not found."""

    auth_override["scopes"] = [enums.scopes.name_to_id["admin"]]
    r = await api.get("/api/agents/nonexistent-agent-xyz")
    assert r.status_code == 404


@pytest.mark.asyncio
async def test_list_agents(api, agent_row, auth_override, enums):
    """Test list agents."""

    auth_override["scopes"] = [enums.scopes.name_to_id["admin"]]
    r = await api.get("/api/agents/")
    assert r.status_code == 200
    data = r.json()["data"]
    assert any(a["name"] == "api-test-agent" for a in data)


@pytest.mark.asyncio
async def test_reload_enums(api, auth_override, enums):
    """Test reload enums."""

    auth_override["scopes"] = [enums.scopes.name_to_id["admin"]]
    r = await api.post("/api/agents/reload-enums")
    assert r.status_code == 200
    assert r.json()["data"]["message"] == "Enums reloaded"


@pytest.mark.asyncio
async def test_update_agent_toggle_trust(api, agent_row, auth_override, enums):
    """Test updating agent trust level."""

    auth_override["scopes"] = [enums.scopes.name_to_id["admin"]]
    r = await api.patch(
        f"/api/agents/{agent_row['id']}",
        json={"requires_approval": True},
    )
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["requires_approval"] is True

    # Toggle back
    r2 = await api.patch(
        f"/api/agents/{agent_row['id']}",
        json={"requires_approval": False},
    )
    assert r2.status_code == 200
    assert r2.json()["data"]["requires_approval"] is False


@pytest.mark.asyncio
async def test_update_agent_description(api, agent_row, auth_override, enums):
    """Test updating agent description."""

    auth_override["scopes"] = [enums.scopes.name_to_id["admin"]]
    r = await api.patch(
        f"/api/agents/{agent_row['id']}",
        json={"description": "Updated description"},
    )
    assert r.status_code == 200
    assert r.json()["data"]["description"] == "Updated description"


@pytest.mark.asyncio
async def test_update_agent_not_found(api, auth_override, enums):
    """Test update nonexistent agent."""

    auth_override["scopes"] = [enums.scopes.name_to_id["admin"]]
    r = await api.patch(
        "/api/agents/00000000-0000-0000-0000-000000000000",
        json={"description": "nope"},
    )
    assert r.status_code == 404
