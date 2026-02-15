"""Integration tests for MCP-native enrollment bootstrap flow."""

# Standard Library
import json
from datetime import UTC, datetime, timedelta
from uuid import uuid4

# Third-Party
import pytest

# Local
from nebula_mcp.helpers import approve_request as do_approve
from nebula_mcp.helpers import reject_request as do_reject
from nebula_mcp.models import (
    AgentEnrollRedeemInput,
    AgentEnrollStartInput,
    AgentEnrollWaitInput,
    QueryEntitiesInput,
)
from nebula_mcp.server import (
    agent_enroll_redeem,
    agent_enroll_start,
    agent_enroll_wait,
    query_entities,
)

pytestmark = pytest.mark.integration


async def _get_enrollment_row(db_pool, registration_id: str) -> dict:
    row = await db_pool.fetchrow(
        "SELECT * FROM agent_enrollment_sessions WHERE id = $1::uuid",
        registration_id,
    )
    return dict(row) if row else {}


async def test_bootstrap_blocks_non_enroll_tools(bootstrap_mcp_context):
    """Non-enrollment tools should fail with ENROLLMENT_REQUIRED in bootstrap mode."""

    with pytest.raises(ValueError) as exc:
        await query_entities(QueryEntitiesInput(), bootstrap_mcp_context)

    payload = json.loads(str(exc.value))
    assert payload["error"]["code"] == "ENROLLMENT_REQUIRED"
    assert payload["error"]["next_steps"] == [
        "agent_enroll_start",
        "agent_enroll_wait",
        "agent_enroll_redeem",
    ]


async def test_enroll_start_creates_pending_approval(bootstrap_mcp_context, db_pool):
    """Enrollment start should create approval + registration token."""

    name = f"mcp-enroll-{uuid4().hex[:8]}"
    started = await agent_enroll_start(
        AgentEnrollStartInput(
            name=name,
            requested_scopes=["public"],
            requested_requires_approval=True,
        ),
        bootstrap_mcp_context,
    )
    assert started["status"] == "pending_approval"
    assert started["registration_id"]
    assert started["enrollment_token"].startswith("nbe_")

    session = await _get_enrollment_row(db_pool, started["registration_id"])
    assert session["status"] == "pending_approval"

    approval = await db_pool.fetchrow(
        "SELECT request_type FROM approval_requests WHERE id = $1::uuid",
        session["approval_request_id"],
    )
    assert approval["request_type"] == "register_agent"


async def test_enroll_wait_timeout_returns_pending(bootstrap_mcp_context):
    """Wait should return pending state and retry hint when no reviewer action occurred."""

    name = f"mcp-enroll-{uuid4().hex[:8]}"
    started = await agent_enroll_start(
        AgentEnrollStartInput(name=name, requested_scopes=["public"]),
        bootstrap_mcp_context,
    )
    waited = await agent_enroll_wait(
        AgentEnrollWaitInput(
            registration_id=started["registration_id"],
            enrollment_token=started["enrollment_token"],
            timeout_seconds=1,
        ),
        bootstrap_mcp_context,
    )
    assert waited["status"] == "pending_approval"
    assert waited["retry_after_ms"] >= 1000
    assert waited["can_redeem"] is False


async def test_enroll_approve_with_grants_applies_final_scope_and_trust(
    bootstrap_mcp_context, db_pool, enums, test_entity
):
    """Reviewer grants should override requested values at approval execution."""

    name = f"mcp-enroll-{uuid4().hex[:8]}"
    started = await agent_enroll_start(
        AgentEnrollStartInput(
            name=name,
            requested_scopes=["public"],
            requested_requires_approval=True,
        ),
        bootstrap_mcp_context,
    )
    session = await _get_enrollment_row(db_pool, started["registration_id"])

    await do_approve(
        db_pool,
        enums,
        str(session["approval_request_id"]),
        str(test_entity["id"]),
        review_details={
            "grant_scopes": ["public", "code"],
            "grant_scope_ids": [
                str(enums.scopes.name_to_id["public"]),
                str(enums.scopes.name_to_id["code"]),
            ],
            "grant_requires_approval": False,
        },
        review_notes="approved with grants",
    )

    waited = await agent_enroll_wait(
        AgentEnrollWaitInput(
            registration_id=started["registration_id"],
            enrollment_token=started["enrollment_token"],
            timeout_seconds=1,
        ),
        bootstrap_mcp_context,
    )
    assert waited["status"] == "approved"
    assert waited["can_redeem"] is True

    refreshed_agent = await db_pool.fetchrow(
        "SELECT scopes, requires_approval FROM agents WHERE id = $1::uuid",
        session["agent_id"],
    )
    assert refreshed_agent["requires_approval"] is False
    assert set(refreshed_agent["scopes"]) == {
        enums.scopes.name_to_id["public"],
        enums.scopes.name_to_id["code"],
    }


async def test_enroll_reject_returns_reason(bootstrap_mcp_context, db_pool, test_entity):
    """Rejected enrollment should return reviewer reason via wait."""

    name = f"mcp-enroll-{uuid4().hex[:8]}"
    started = await agent_enroll_start(
        AgentEnrollStartInput(name=name, requested_scopes=["public"]),
        bootstrap_mcp_context,
    )
    session = await _get_enrollment_row(db_pool, started["registration_id"])

    await do_reject(
        db_pool,
        str(session["approval_request_id"]),
        str(test_entity["id"]),
        "missing trust signals",
    )

    waited = await agent_enroll_wait(
        AgentEnrollWaitInput(
            registration_id=started["registration_id"],
            enrollment_token=started["enrollment_token"],
            timeout_seconds=1,
        ),
        bootstrap_mcp_context,
    )
    assert waited["status"] == "rejected"
    assert waited["reason"] == "missing trust signals"
    assert waited["can_redeem"] is False


async def test_enroll_redeem_is_one_time(
    bootstrap_mcp_context, db_pool, enums, test_entity
):
    """Redeem should mint one API key and block replay."""

    name = f"mcp-enroll-{uuid4().hex[:8]}"
    started = await agent_enroll_start(
        AgentEnrollStartInput(name=name, requested_scopes=["public"]),
        bootstrap_mcp_context,
    )
    session = await _get_enrollment_row(db_pool, started["registration_id"])
    await do_approve(
        db_pool,
        enums,
        str(session["approval_request_id"]),
        str(test_entity["id"]),
    )

    redeemed = await agent_enroll_redeem(
        AgentEnrollRedeemInput(
            registration_id=started["registration_id"],
            enrollment_token=started["enrollment_token"],
        ),
        bootstrap_mcp_context,
    )
    assert redeemed["api_key"].startswith("nbl_")
    assert redeemed["agent_id"] == str(session["agent_id"])
    assert "public" in redeemed["scopes"]

    with pytest.raises(ValueError, match="already redeemed"):
        await agent_enroll_redeem(
            AgentEnrollRedeemInput(
                registration_id=started["registration_id"],
                enrollment_token=started["enrollment_token"],
            ),
            bootstrap_mcp_context,
        )


async def test_enroll_expired_wait_and_redeem(bootstrap_mcp_context, db_pool):
    """Expired enrollment should report expired and deny redemption."""

    name = f"mcp-enroll-{uuid4().hex[:8]}"
    started = await agent_enroll_start(
        AgentEnrollStartInput(name=name, requested_scopes=["public"]),
        bootstrap_mcp_context,
    )
    await db_pool.execute(
        """
        UPDATE agent_enrollment_sessions
        SET expires_at = $2
        WHERE id = $1::uuid
        """,
        started["registration_id"],
        datetime.now(UTC) - timedelta(minutes=1),
    )

    waited = await agent_enroll_wait(
        AgentEnrollWaitInput(
            registration_id=started["registration_id"],
            enrollment_token=started["enrollment_token"],
            timeout_seconds=1,
        ),
        bootstrap_mcp_context,
    )
    assert waited["status"] == "expired"
    assert waited["can_redeem"] is False

    with pytest.raises(ValueError, match="expired"):
        await agent_enroll_redeem(
            AgentEnrollRedeemInput(
                registration_id=started["registration_id"],
                enrollment_token=started["enrollment_token"],
            ),
            bootstrap_mcp_context,
        )
