"""Unit tests for nebula_mcp.executors branch-heavy paths."""

from __future__ import annotations

# Standard Library
from uuid import uuid4
from unittest.mock import AsyncMock

# Third-Party
import pytest

# Local
import nebula_mcp.executors as executors


class _AsyncContext:
    """Small async context manager used by pool transaction stubs."""

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, tb):
        return False


class _PoolStub:
    """Minimal asyncpg-like stub for executor unit tests."""

    def __init__(
        self,
        *,
        fetchrow_rows: list[object] | None = None,
        fetch_rows: list[object] | None = None,
        fetchval_rows: list[object] | None = None,
        in_transaction: bool = True,
    ):
        self.fetchrow_rows = list(fetchrow_rows or [])
        self.fetch_rows = list(fetch_rows or [])
        self.fetchval_rows = list(fetchval_rows or [])
        self._in_transaction = in_transaction
        self.transaction_calls = 0
        self.fetchrow_calls: list[tuple] = []
        self.fetch_calls: list[tuple] = []
        self.fetchval_calls: list[tuple] = []
        self.execute = AsyncMock()

    def is_in_transaction(self) -> bool:
        return self._in_transaction

    def transaction(self):
        self.transaction_calls += 1
        return _AsyncContext()

    async def fetchrow(self, *args):
        self.fetchrow_calls.append(args)
        if not self.fetchrow_rows:
            return None
        value = self.fetchrow_rows.pop(0)
        if isinstance(value, Exception):
            raise value
        return value

    async def fetch(self, *args):
        self.fetch_calls.append(args)
        if not self.fetch_rows:
            return []
        value = self.fetch_rows.pop(0)
        if isinstance(value, Exception):
            raise value
        return value

    async def fetchval(self, *args):
        self.fetchval_calls.append(args)
        if not self.fetchval_rows:
            return None
        value = self.fetchval_rows.pop(0)
        if isinstance(value, Exception):
            raise value
        return value


def test_scope_name_from_id_resolves_known_uuid(mock_enums):
    """Known scope UUIDs should map back to names."""

    scope_name, scope_id = next(iter(mock_enums.scopes.name_to_id.items()))
    assert executors._scope_name_from_id(mock_enums, scope_id) == scope_name


def test_scope_name_from_id_returns_raw_for_non_uuid(mock_enums):
    """Non-UUID scope ids should pass through as strings."""

    assert executors._scope_name_from_id(mock_enums, "not-a-uuid") == "not-a-uuid"


def test_scope_name_from_id_returns_uuid_text_for_unknown_uuid(mock_enums):
    """Unknown UUID scope ids should fall back to UUID text."""

    unknown_scope = uuid4()
    assert executors._scope_name_from_id(mock_enums, unknown_scope) == str(unknown_scope)


def test_decode_json_object_handles_double_encoded_payload():
    """Double-encoded JSON objects should decode into dicts."""

    assert executors._decode_json_object("\"{\\\"k\\\": 1}\"") == {"k": 1}


def test_decode_json_object_handles_invalid_payloads():
    """Invalid/non-object values should normalize to empty dicts."""

    assert executors._decode_json_object("{bad json}") == {}
    assert executors._decode_json_object("\"plain-string\"") == {}
    assert executors._decode_json_object(42) == {}


def test_normalize_entity_row_handles_none():
    """Entity row normalizer should return empty dict for missing rows."""

    assert executors._normalize_entity_row(None) == {}


@pytest.mark.asyncio
async def test_execute_create_entity_uses_explicit_transaction_when_not_in_tx(mock_enums):
    """Non-transactional connection stubs should wrap create in transaction()."""

    pool = _PoolStub(
        fetchrow_rows=[
            None,
            {"id": str(uuid4()), "name": "n", "metadata": "{\"ok\": true}"},
        ],
        in_transaction=False,
    )

    result = await executors.execute_create_entity(
        pool,
        mock_enums,
        {
            "name": "n",
            "type": "project",
            "status": "active",
            "scopes": ["public"],
        },
    )

    assert pool.transaction_calls == 1
    assert result["metadata"] == {"ok": True}


@pytest.mark.asyncio
async def test_execute_update_context_raises_when_metadata_target_missing(mock_enums):
    """Metadata updates should fail when the target context item does not exist."""

    pool = _PoolStub(fetchrow_rows=[None])

    with pytest.raises(ValueError, match="Context not found"):
        await executors.execute_update_context(
            pool,
            mock_enums,
            {"context_id": str(uuid4()), "metadata": {"k": "v"}},
        )


@pytest.mark.asyncio
async def test_execute_update_context_status_and_scope_paths(mock_enums):
    """Status/scopes branches should be exercised for context updates."""

    context_id = str(uuid4())
    pool = _PoolStub(
        fetchrow_rows=[
            {"id": context_id, "metadata": {"nested": {"a": 1}}},
            {"id": context_id, "metadata": {"nested": {"a": 2}}},
        ]
    )

    result = await executors.execute_update_context(
        pool,
        mock_enums,
        {
            "context_id": context_id,
            "status": "active",
            "scopes": ["public"],
            "metadata": {"nested": {"a": 2}},
        },
    )

    assert result["id"] == context_id
    assert len(pool.fetchrow_calls) == 2


@pytest.mark.asyncio
async def test_execute_create_relationship_reraises_unexpected_unique_violation(
    monkeypatch, mock_enums
):
    """Unexpected unique constraints should be re-raised unchanged."""

    class _FakeUniqueViolation(Exception):
        def __init__(self, constraint_name: str):
            super().__init__(constraint_name)
            self.constraint_name = constraint_name

    pool = _PoolStub(
        fetchrow_rows=[_FakeUniqueViolation("some_other_constraint")],
    )
    monkeypatch.setattr(executors, "UniqueViolationError", _FakeUniqueViolation)

    with pytest.raises(_FakeUniqueViolation):
        await executors.execute_create_relationship(
            pool,
            mock_enums,
            {
                "source_type": "entity",
                "source_id": str(uuid4()),
                "target_type": "entity",
                "target_id": str(uuid4()),
                "relationship_type": "related-to",
            },
        )


@pytest.mark.asyncio
async def test_execute_create_relationship_rejects_cycle(mock_enums):
    """Cycle-sensitive relationship types should reject cycle paths."""

    pool = _PoolStub(fetchval_rows=[True])

    with pytest.raises(ValueError, match="create a cycle"):
        await executors.execute_create_relationship(
            pool,
            mock_enums,
            {
                "source_type": "entity",
                "source_id": str(uuid4()),
                "target_type": "entity",
                "target_id": str(uuid4()),
                "relationship_type": "depends-on",
            },
        )


@pytest.mark.asyncio
async def test_execute_update_job_raises_when_missing(mock_enums):
    """Job updates should raise not found when row update returns nothing."""

    pool = _PoolStub(fetchrow_rows=[None])

    with pytest.raises(ValueError, match="Job not found"):
        await executors.execute_update_job(
            pool,
            mock_enums,
            {"job_id": "2026Q1-ABCD"},
        )


@pytest.mark.asyncio
async def test_execute_update_job_status_raises_when_missing(mock_enums):
    """Status updates should raise when the target job does not exist."""

    pool = _PoolStub(fetchrow_rows=[None])

    with pytest.raises(ValueError, match="Job not found"):
        await executors.execute_update_job_status(
            pool,
            mock_enums,
            {"job_id": "2026Q1-ABCD", "status": "active"},
        )


@pytest.mark.asyncio
async def test_execute_update_relationship_raises_when_missing(mock_enums):
    """Relationship updates should raise not found for missing ids."""

    pool = _PoolStub(fetchrow_rows=[None])

    with pytest.raises(ValueError, match="Relationship not found"):
        await executors.execute_update_relationship(
            pool,
            mock_enums,
            {"relationship_id": str(uuid4()), "status": "active"},
        )


@pytest.mark.asyncio
async def test_execute_update_file_status_branch(mock_enums):
    """File updates should resolve status names when supplied."""

    file_id = str(uuid4())
    pool = _PoolStub(fetchrow_rows=[{"id": file_id}])
    result = await executors.execute_update_file(
        pool,
        mock_enums,
        {"file_id": file_id, "status": "active"},
    )
    assert result == {"id": file_id}


@pytest.mark.asyncio
async def test_execute_update_protocol_status_branch(mock_enums):
    """Protocol updates should resolve status names when supplied."""

    pool = _PoolStub(fetchrow_rows=[{"name": "alpha"}])
    result = await executors.execute_update_protocol(
        pool,
        mock_enums,
        {"name": "alpha", "status": "active"},
    )
    assert result == {"name": "alpha"}


@pytest.mark.asyncio
async def test_execute_update_log_log_type_and_status_branches(mock_enums):
    """Log updates should resolve both log type and status names."""

    log_id = str(uuid4())
    pool = _PoolStub(fetchrow_rows=[{"id": log_id}])
    result = await executors.execute_update_log(
        pool,
        mock_enums,
        {"id": log_id, "log_type": "event", "status": "active"},
    )
    assert result == {"id": log_id}


@pytest.mark.asyncio
async def test_execute_bulk_update_entity_tags_falls_back_to_first_row_value(mock_enums):
    """Bulk tag update should collect ids even when row key is not named 'id'."""

    raw_id = str(uuid4())
    pool = _PoolStub(fetch_rows=[[{"entity_id": raw_id}]])
    result = await executors.execute_bulk_update_entity_tags(
        pool,
        mock_enums,
        {"entity_ids": [str(uuid4())], "op": "add", "tags": ["alpha"]},
    )
    assert result == {"updated": 1, "entity_ids": [raw_id]}


@pytest.mark.asyncio
async def test_execute_bulk_update_entity_scopes_falls_back_to_first_row_value(
    mock_enums,
):
    """Bulk scope update should collect ids even when row key is not named 'id'."""

    raw_id = str(uuid4())
    pool = _PoolStub(fetch_rows=[[{"entity_id": raw_id}]])
    result = await executors.execute_bulk_update_entity_scopes(
        pool,
        mock_enums,
        {"entity_ids": [str(uuid4())], "op": "remove", "scopes": ["public"]},
    )
    assert result == {"updated": 1, "entity_ids": [raw_id]}


@pytest.mark.asyncio
async def test_execute_register_agent_parses_review_details_json_and_raises_missing(
    mock_enums,
):
    """JSON review payloads should parse before activation and still raise if missing."""

    agent_id = str(uuid4())
    pool = _PoolStub(fetchrow_rows=[{"requires_approval": True}, None])

    with pytest.raises(ValueError, match="not found"):
        await executors.execute_register_agent(
            pool,
            mock_enums,
            {"agent_id": agent_id, "requested_scopes": ["public"]},
            review_details='{"grant_scopes":["public"]}',
        )


@pytest.mark.asyncio
async def test_execute_register_agent_preserves_trusted_agent_on_reenroll(mock_enums):
    """Trusted agents should not be flipped back to approval mode implicitly."""

    agent_id = str(uuid4())
    pool = _PoolStub(
        fetchrow_rows=[
            {"requires_approval": False},
            {"id": uuid4(), "name": "trusted-agent"},
        ]
    )

    result = await executors.execute_register_agent(
        pool,
        mock_enums,
        {
            "agent_id": agent_id,
            "requested_scopes": ["public"],
            "requested_requires_approval": True,
        },
        review_details={},
    )

    assert result["requires_approval"] is False
    activate_call = pool.fetchrow_calls[1]
    assert activate_call[3] is False


@pytest.mark.asyncio
async def test_execute_register_agent_marks_approved_enrollment_when_metadata_present(
    mock_enums,
):
    """Enrollment should be marked approved when reviewer metadata is provided."""

    agent_id = str(uuid4())
    approval_id = str(uuid4())
    reviewed_by = str(uuid4())
    granted_scope_id = next(iter(mock_enums.scopes.name_to_id.values()))
    pool = _PoolStub(
        fetchrow_rows=[
            {"requires_approval": True},
            {"id": uuid4(), "name": "approved-agent"},
        ]
    )

    result = await executors.execute_register_agent(
        pool,
        mock_enums,
        {"agent_id": agent_id, "requested_scopes": ["public"]},
        review_details={
            "grant_scope_ids": [granted_scope_id],
            "grant_requires_approval": True,
            "_approval_id": approval_id,
            "_reviewed_by": reviewed_by,
        },
    )

    assert result["approval_id"] == approval_id
    pool.execute.assert_awaited_once()
