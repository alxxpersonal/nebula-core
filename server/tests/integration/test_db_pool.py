"""Integration tests for database pool and agent lookup."""

# Third-Party
import pytest

from nebula_mcp.db import get_agent

pytestmark = pytest.mark.integration


# --- Pool Connectivity ---


async def test_pool_select_one(db_pool):
    """Pool should connect and execute a simple SELECT 1."""

    row = await db_pool.fetchrow("SELECT 1 AS val")
    assert row["val"] == 1


# --- get_agent ---


async def test_get_agent_existing(db_pool, test_agent):
    """get_agent should return the row for an existing active agent."""

    row = await get_agent(db_pool, "test-agent")
    assert row is not None
    assert row["name"] == "test-agent"


async def test_get_agent_nonexistent(db_pool):
    """get_agent should return None for a name that does not exist."""

    row = await get_agent(db_pool, "does-not-exist")
    assert row is None


async def test_get_agent_inactive(db_pool, enums):
    """get_agent should return None for an agent with an inactive status."""

    status_id = enums.statuses.name_to_id["inactive"]
    scope_ids = [enums.scopes.name_to_id["public"]]

    await db_pool.execute(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        """,
        "inactive-agent",
        "An archived agent",
        scope_ids,
        False,
        status_id,
    )

    row = await get_agent(db_pool, "inactive-agent")
    assert row is None
