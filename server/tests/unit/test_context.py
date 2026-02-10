"""Unit tests for context extraction and validation helpers."""

# Standard Library
# Third-Party
from unittest.mock import MagicMock, patch
from uuid import uuid4

import pytest

from nebula_mcp.context import (
    maybe_require_approval,
    require_agent,
    require_context,
    require_pool,
)

pytestmark = pytest.mark.unit


# --- require_context ---


class TestRequireContext:
    """Tests for the require_context function."""

    async def test_valid_returns_pool_enums_agent(
        self, mock_context, mock_pool, mock_enums, mock_agent
    ):
        """Return (pool, enums, agent) tuple from a valid context."""

        pool, enums, agent = await require_context(mock_context)
        assert pool is mock_pool
        assert enums is mock_enums
        assert agent is mock_agent

    async def test_no_pool_raises(self, mock_enums, mock_agent):
        """Raise ValueError when pool is missing from lifespan context."""

        ctx = MagicMock()
        ctx.request_context.lifespan_context = {
            "enums": mock_enums,
            "agent": mock_agent,
        }

        with pytest.raises(ValueError, match="Pool not initialized"):
            await require_context(ctx)

    async def test_no_enums_raises(self, mock_pool, mock_agent):
        """Raise ValueError when enums is missing from lifespan context."""

        ctx = MagicMock()
        ctx.request_context.lifespan_context = {"pool": mock_pool, "agent": mock_agent}

        with pytest.raises(ValueError, match="Enums not initialized"):
            await require_context(ctx)

    async def test_no_agent_raises(self, mock_pool, mock_enums):
        """Raise ValueError when agent is missing from lifespan context."""

        ctx = MagicMock()
        ctx.request_context.lifespan_context = {"pool": mock_pool, "enums": mock_enums}

        with pytest.raises(ValueError, match="Agent not initialized"):
            await require_context(ctx)

    async def test_no_lifespan_raises(self):
        """Raise ValueError when lifespan_context is None."""

        ctx = MagicMock()
        ctx.request_context.lifespan_context = None

        with pytest.raises(ValueError, match="Pool not initialized"):
            await require_context(ctx)


# --- require_pool ---


class TestRequirePool:
    """Tests for the require_pool function."""

    async def test_valid_returns_pool(self, mock_context, mock_pool):
        """Return pool from a valid context."""

        pool = await require_pool(mock_context)
        assert pool is mock_pool

    async def test_missing_pool_raises(self):
        """Raise ValueError when pool is missing."""

        ctx = MagicMock()
        ctx.request_context.lifespan_context = {}

        with pytest.raises(ValueError, match="Pool not initialized"):
            await require_pool(ctx)


# --- require_agent ---


class TestRequireAgent:
    """Tests for the require_agent function."""

    @patch("nebula_mcp.context.get_agent")
    async def test_valid_agent(self, mock_get_agent, mock_pool):
        """Return agent dict when agent is found."""

        agent_row = {
            "id": uuid4(),
            "name": "test-agent",
            "scopes": [],
            "requires_approval": False,
        }
        mock_get_agent.return_value = agent_row

        result = await require_agent(mock_pool, "test-agent")
        assert result["name"] == "test-agent"
        mock_get_agent.assert_awaited_once_with(mock_pool, "test-agent")

    @patch("nebula_mcp.context.get_agent")
    async def test_agent_not_found_raises(self, mock_get_agent, mock_pool):
        """Raise ValueError when agent is not found."""

        mock_get_agent.return_value = None

        with pytest.raises(ValueError, match="Agent not found or inactive"):
            await require_agent(mock_pool, "ghost")


# --- maybe_require_approval ---


class TestMaybeRequireApproval:
    """Tests for the maybe_require_approval function."""

    async def test_trusted_returns_none(self, mock_pool, mock_agent):
        """Return None for a trusted agent (requires_approval=False)."""

        result = await maybe_require_approval(
            mock_pool, mock_agent, "create_entity", {"name": "test"}
        )
        assert result is None

    @patch("nebula_mcp.helpers.create_approval_request")
    async def test_untrusted_returns_approval_dict(
        self, mock_create_approval, mock_pool, mock_untrusted_agent
    ):
        """Return approval response dict for an untrusted agent."""

        approval_id = uuid4()
        mock_create_approval.return_value = {"id": approval_id}

        result = await maybe_require_approval(
            mock_pool,
            mock_untrusted_agent,
            "create_entity",
            {"name": "test"},
        )

        assert result is not None
        assert result["status"] == "approval_required"
        assert result["approval_request_id"] == str(approval_id)
        assert result["requested_action"] == "create_entity"
