"""Red team tests for agent update access control."""

# Standard Library

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_agent_can_update_other_agent(api_agent_auth, db_pool, enums):
    """Non-admin agents should not be able to update other agents."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id["public"]]

    victim = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        "victim-agent",
        "Victim agent",
        scope_ids,
        True,
        status_id,
    )

    resp = await api_agent_auth.patch(
        f"/api/agents/{victim['id']}",
        json={
            "requires_approval": False,
            "scopes": ["sensitive"],
        },
    )

    assert resp.status_code == 403


@pytest.mark.asyncio
async def test_user_can_update_agent(api, db_pool, enums):
    """Non-admin users should not be able to update agents."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id["public"]]

    victim = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        "victim-user-agent",
        "Victim agent",
        scope_ids,
        True,
        status_id,
    )

    resp = await api.patch(
        f"/api/agents/{victim['id']}",
        json={
            "requires_approval": False,
        },
    )

    assert resp.status_code == 403
