"""Red team tests for entity history access isolation."""

# Standard Library
import json
from unittest.mock import MagicMock

# Third-Party
import pytest

# Local
from nebula_mcp.models import GetEntityHistoryInput
from nebula_mcp.server import get_entity_history


def _make_context(pool, enums, agent):
    """Build MCP context with a specific agent."""

    ctx = MagicMock()
    ctx.request_context.lifespan_context = {
        "pool": pool,
        "enums": enums,
        "agent": agent,
    }
    return ctx


async def _make_entity(db_pool, enums, name, scopes):
    """Insert an entity for history access tests."""

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


@pytest.mark.asyncio
async def test_get_entity_history_denies_private_entity(db_pool, enums):
    """Entity history should be denied when entity is outside agent scopes."""

    private_entity = await _make_entity(db_pool, enums, "Private", ["sensitive"])
    await db_pool.execute(
        "UPDATE entities SET name = $1 WHERE id = $2",
        "Private Updated",
        private_entity["id"],
    )

    public_agent = {
        "id": "history-viewer",
        "scopes": [enums.scopes.name_to_id["public"]],
    }
    ctx = _make_context(db_pool, enums, public_agent)

    payload = GetEntityHistoryInput(entity_id=str(private_entity["id"]))

    with pytest.raises(ValueError):
        await get_entity_history(payload, ctx)
