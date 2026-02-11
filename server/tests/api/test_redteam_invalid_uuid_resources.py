"""Red team tests for invalid UUID handling in resource routes."""

# Third-Party
import pytest


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_get_file_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash file detail routes."""

    resp = await api.get("/api/files/not-a-uuid")
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_update_file_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash file update routes."""

    resp = await api.patch(
        "/api/files/not-a-uuid",
        json={"metadata": {"note": "bad"}},
    )
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_get_log_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash log detail routes."""

    resp = await api.get("/api/logs/not-a-uuid")
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_update_log_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash log update routes."""

    resp = await api.patch(
        "/api/logs/not-a-uuid",
        json={"metadata": {"note": "bad"}},
    )
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_get_knowledge_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash knowledge detail routes."""

    resp = await api.get("/api/knowledge/not-a-uuid")
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_update_knowledge_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash knowledge update routes."""

    resp = await api.patch(
        "/api/knowledge/not-a-uuid",
        json={"metadata": {"note": "bad"}},
    )
    assert resp.status_code in {400, 404}


@pytest.mark.asyncio
@pytest.mark.xfail(reason="invalid UUIDs raise asyncpg DataError")
async def test_api_link_knowledge_rejects_invalid_uuid(api):
    """Invalid UUIDs should not crash knowledge link routes."""

    resp = await api.post(
        "/api/knowledge/not-a-uuid/link",
        json={"entity_id": "not-a-uuid"},
    )
    assert resp.status_code in {400, 404}
