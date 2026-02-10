"""Audit route tests."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_list_audit_scopes(api):
    r = await api.get("/api/audit/scopes")
    assert r.status_code == 200
    data = r.json()["data"]
    assert any(row["name"] == "public" for row in data)


@pytest.mark.asyncio
async def test_list_audit_actors(api, test_entity):
    r = await api.get("/api/audit/actors")
    assert r.status_code == 200
    data = r.json()["data"]
    assert len(data) >= 1


@pytest.mark.asyncio
async def test_list_audit_scope_filter(api, enums, test_entity):
    scope_id = enums.scopes.name_to_id["public"]
    r = await api.get("/api/audit", params={"scope_id": str(scope_id)})
    assert r.status_code == 200
    data = r.json()["data"]
    assert isinstance(data, list)
