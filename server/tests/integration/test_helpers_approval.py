"""Integration tests for the approval workflow helpers."""

# Third-Party
import pytest

from nebula_mcp.helpers import (
    approve_request,
    create_approval_request,
    get_pending_approvals_all,
    reject_request,
)

pytestmark = pytest.mark.integration


# --- create_approval_request ---


async def test_create_approval_request_returns_pending(db_pool, enums, untrusted_agent):
    """Creating an approval request should return a row with pending status."""

    result = await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "create_entity",
        {
            "name": "Pending Entity",
            "type": "project",
            "status": "active",
            "scopes": ["public"],
        },
    )

    assert result["status"] == "pending"


# --- get_pending_approvals_all ---


async def test_get_pending_approvals_count(db_pool, enums, untrusted_agent):
    """get_pending_approvals_all should return the correct number of pending requests."""

    await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "create_entity",
        {
            "name": "A",
            "type": "project",
            "status": "active",
            "scopes": ["public"],
        },
    )
    await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "create_entity",
        {
            "name": "B",
            "type": "project",
            "status": "active",
            "scopes": ["public"],
        },
    )

    pending = await get_pending_approvals_all(db_pool)
    assert len(pending) == 2


# --- approve_request ---


async def test_approve_request_creates_entity_and_links_audit(
    db_pool,
    enums,
    untrusted_agent,
    test_entity,
):
    """Approving a request should create the entity and link an audit trail."""

    approval = await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "create_entity",
        {
            "name": "Approved Entity",
            "type": "project",
            "status": "active",
            "scopes": ["public"],
        },
    )

    result = await approve_request(
        db_pool,
        enums,
        str(approval["id"]),
        str(test_entity["id"]),
    )

    assert "entity" in result
    assert result["entity"]["name"] == "Approved Entity"
    assert "approval" in result


async def test_approve_nonexistent_raises(db_pool, enums, test_entity):
    """Approving a nonexistent approval ID should raise ValueError."""

    with pytest.raises(
        ValueError, match="Approval request not found or already processed"
    ):
        await approve_request(
            db_pool,
            enums,
            "00000000-0000-0000-0000-000000000000",
            str(test_entity["id"]),
        )


async def test_approve_twice_raises(db_pool, enums, untrusted_agent, test_entity):
    """Approving the same request twice should raise ValueError."""

    approval = await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "create_entity",
        {
            "name": "Double Approve Entity",
            "type": "project",
            "status": "active",
            "scopes": ["public"],
        },
    )

    await approve_request(
        db_pool,
        enums,
        str(approval["id"]),
        str(test_entity["id"]),
    )

    with pytest.raises(ValueError, match="already processed"):
        await approve_request(
            db_pool,
            enums,
            str(approval["id"]),
            str(test_entity["id"]),
        )


async def test_approve_unknown_executor_type_raises(
    db_pool, enums, untrusted_agent, test_entity
):
    """Approving a request with an unknown executor type should raise ValueError."""

    approval = await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "nonexistent_action",
        {},
    )

    with pytest.raises(ValueError, match="No executor for"):
        await approve_request(
            db_pool,
            enums,
            str(approval["id"]),
            str(test_entity["id"]),
        )


async def test_approve_with_bad_data_marks_failed(
    db_pool, enums, untrusted_agent, test_entity
):
    """Approving a request with invalid data should mark it as approved-failed."""

    approval = await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "create_entity",
        {
            "name": "Will Fail",
            "type": "INVALID_TYPE",
            "status": "active",
            "scopes": ["public"],
        },
    )

    with pytest.raises(ValueError):
        await approve_request(
            db_pool,
            enums,
            str(approval["id"]),
            str(test_entity["id"]),
        )

    # Verify the approval_requests row was marked as failed
    row = await db_pool.fetchrow(
        "SELECT status FROM approval_requests WHERE id = $1::uuid",
        str(approval["id"]),
    )
    assert row["status"] == "approved-failed"


# --- reject_request ---


async def test_reject_request_success(db_pool, enums, untrusted_agent, test_entity):
    """Rejecting a request should set status to rejected and store review notes."""

    approval = await create_approval_request(
        db_pool,
        str(untrusted_agent["id"]),
        "create_entity",
        {
            "name": "To Reject",
            "type": "project",
            "status": "active",
            "scopes": ["public"],
        },
    )

    result = await reject_request(
        db_pool,
        str(approval["id"]),
        str(test_entity["id"]),
        "Not needed",
    )

    assert result["status"] == "rejected"
    assert result["review_notes"] == "Not needed"


async def test_reject_nonexistent_raises(db_pool, enums, test_entity):
    """Rejecting a nonexistent approval ID should raise ValueError."""

    with pytest.raises(
        ValueError, match="Approval request not found or already processed"
    ):
        await reject_request(
            db_pool,
            "00000000-0000-0000-0000-000000000000",
            str(test_entity["id"]),
            "Does not exist",
        )
