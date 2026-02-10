"""Approval route tests."""

# Standard Library
import json

# Third-Party
import pytest


@pytest.fixture
async def untrusted_agent(db_pool, enums):
    """Create an untrusted agent that triggers approvals."""

    status_id = enums.statuses.name_to_id["active"]
    scope_ids = [enums.scopes.name_to_id["public"]]
    row = await db_pool.fetchrow(
        """
        INSERT INTO agents (name, description, scopes, requires_approval, status_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        "api-untrusted",
        "Untrusted agent for API tests",
        scope_ids,
        True,
        status_id,
    )
    return dict(row)


@pytest.fixture
async def pending_approval(db_pool, untrusted_agent):
    """Create a pending approval request."""

    row = await db_pool.fetchrow(
        """
        INSERT INTO approval_requests (request_type, requested_by, change_details, status)
        VALUES ($1, $2, $3::jsonb, $4)
        RETURNING *
        """,
        "create_entity",
        untrusted_agent["id"],
        json.dumps(
            {
                "name": "ApprovalTest",
                "type": "person",
                "scopes": ["public"],
                "tags": [],
                "metadata": {},
                "vault_file_path": None,
                "status": "active",
            }
        ),
        "pending",
    )
    return dict(row)


@pytest.mark.asyncio
async def test_get_pending(api, pending_approval):
    """Test get pending."""

    r = await api.get("/api/approvals/pending")
    assert r.status_code == 200
    data = r.json()["data"]
    assert len(data) >= 1


@pytest.mark.asyncio
async def test_get_approval(api, pending_approval):
    """Test get approval."""

    r = await api.get(f"/api/approvals/{pending_approval['id']}")
    assert r.status_code == 200
    assert r.json()["data"]["request_type"] == "create_entity"


@pytest.mark.asyncio
async def test_approve_request(api, pending_approval, auth_override):
    """Test approve request."""

    r = await api.post(f"/api/approvals/{pending_approval['id']}/approve")
    assert r.status_code == 200


@pytest.mark.asyncio
async def test_reject_request(api, pending_approval, auth_override):
    """Test reject request."""

    r = await api.post(
        f"/api/approvals/{pending_approval['id']}/reject",
        json={
            "review_notes": "nah bro",
        },
    )
    assert r.status_code == 200


@pytest.mark.asyncio
async def test_get_approval_not_found(api):
    """Test get approval not found."""

    r = await api.get("/api/approvals/00000000-0000-0000-0000-000000000000")
    assert r.status_code == 404


@pytest.mark.asyncio
async def test_get_approval_diff_create_job(api, db_pool, untrusted_agent):
    """Approval diff should include create_job fields."""

    row = await db_pool.fetchrow(
        """
        INSERT INTO approval_requests (request_type, requested_by, change_details, status)
        VALUES ($1, $2, $3::jsonb, $4)
        RETURNING *
        """,
        "create_job",
        untrusted_agent["id"],
        json.dumps({"title": "Diff Job", "priority": "high"}),
        "pending",
    )

    r = await api.get(f"/api/approvals/{row['id']}/diff")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["request_type"] == "create_job"
    assert data["changes"]["title"]["to"] == "Diff Job"


@pytest.mark.asyncio
async def test_get_approval_diff_create_knowledge(api, db_pool, untrusted_agent):
    """Approval diff should include create_knowledge fields."""

    row = await db_pool.fetchrow(
        """
        INSERT INTO approval_requests (request_type, requested_by, change_details, status)
        VALUES ($1, $2, $3::jsonb, $4)
        RETURNING *
        """,
        "create_knowledge",
        untrusted_agent["id"],
        json.dumps(
            {"title": "Diff Knowledge", "source_type": "note", "scopes": ["public"]}
        ),
        "pending",
    )

    r = await api.get(f"/api/approvals/{row['id']}/diff")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["request_type"] == "create_knowledge"
    assert data["changes"]["title"]["to"] == "Diff Knowledge"


@pytest.mark.asyncio
async def test_get_approval_diff_update_relationship(
    api, db_pool, enums, untrusted_agent
):
    """Approval diff should include relationship updates."""

    status_id = enums.statuses.name_to_id["active"]
    type_id = enums.relationship_types.name_to_id["related-to"]

    source = await db_pool.fetchrow(
        """
        INSERT INTO entities (name, type_id, status_id, privacy_scope_ids)
        VALUES ($1, $2, $3, $4)
        RETURNING *
        """,
        "Diff Source",
        enums.entity_types.name_to_id["person"],
        status_id,
        [enums.scopes.name_to_id["public"]],
    )
    target = await db_pool.fetchrow(
        """
        INSERT INTO entities (name, type_id, status_id, privacy_scope_ids)
        VALUES ($1, $2, $3, $4)
        RETURNING *
        """,
        "Diff Target",
        enums.entity_types.name_to_id["person"],
        status_id,
        [enums.scopes.name_to_id["public"]],
    )

    relationship = await db_pool.fetchrow(
        """
        INSERT INTO relationships (source_type, source_id, target_type, target_id, type_id, status_id, properties)
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
        RETURNING *
        """,
        "entity",
        str(source["id"]),
        "entity",
        str(target["id"]),
        type_id,
        status_id,
        json.dumps({"note": "old"}),
    )

    row = await db_pool.fetchrow(
        """
        INSERT INTO approval_requests (request_type, requested_by, change_details, status)
        VALUES ($1, $2, $3::jsonb, $4)
        RETURNING *
        """,
        "update_relationship",
        untrusted_agent["id"],
        json.dumps(
            {
                "relationship_id": str(relationship["id"]),
                "properties": {"note": "new"},
            }
        ),
        "pending",
    )

    r = await api.get(f"/api/approvals/{row['id']}/diff")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["request_type"] == "update_relationship"
    assert data["changes"]["properties"]["to"] == {"note": "new"}


@pytest.mark.asyncio
async def test_get_approval_diff_update_job_status(
    api, db_pool, enums, untrusted_agent
):
    """Approval diff should include job status updates."""

    status_id = enums.statuses.name_to_id["in-progress"]
    job = await db_pool.fetchrow(
        """
        INSERT INTO jobs (title, status_id, priority)
        VALUES ($1, $2, $3)
        RETURNING *
        """,
        "Diff Job Status",
        status_id,
        "medium",
    )

    row = await db_pool.fetchrow(
        """
        INSERT INTO approval_requests (request_type, requested_by, change_details, status)
        VALUES ($1, $2, $3::jsonb, $4)
        RETURNING *
        """,
        "update_job_status",
        untrusted_agent["id"],
        json.dumps(
            {
                "job_id": job["id"],
                "status": "completed",
                "status_reason": "done",
            }
        ),
        "pending",
    )

    r = await api.get(f"/api/approvals/{row['id']}/diff")
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["request_type"] == "update_job_status"
    assert data["changes"]["status"]["to"] == "completed"
