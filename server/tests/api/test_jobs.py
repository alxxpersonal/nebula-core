"""Job route tests."""

# Third-Party
import pytest


@pytest.mark.asyncio
async def test_create_job(api):
    """Test create job."""

    r = await api.post(
        "/api/jobs",
        json={
            "title": "Test Job",
            "description": "A test job",
            "priority": "medium",
        },
    )
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["title"] == "Test Job"
    assert "id" in data


@pytest.mark.asyncio
async def test_get_job(api):
    """Test get job."""

    cr = await api.post("/api/jobs", json={"title": "GetJob"})
    job_id = cr.json()["data"]["id"]

    r = await api.get(f"/api/jobs/{job_id}")
    assert r.status_code == 200
    assert r.json()["data"]["title"] == "GetJob"


@pytest.mark.asyncio
async def test_get_job_not_found(api):
    """Test get job not found."""

    r = await api.get("/api/jobs/00000000-0000-0000-0000-000000000000")
    assert r.status_code == 404


@pytest.mark.asyncio
async def test_query_jobs(api):
    """Test query jobs."""

    await api.post("/api/jobs", json={"title": "QueryJob", "priority": "high"})
    r = await api.get("/api/jobs", params={"priority": "high"})
    assert r.status_code == 200
    assert len(r.json()["data"]) >= 1


@pytest.mark.asyncio
async def test_update_job_status(api):
    """Test update job status."""

    cr = await api.post("/api/jobs", json={"title": "StatusJob"})
    job_id = cr.json()["data"]["id"]

    r = await api.patch(
        f"/api/jobs/{job_id}/status",
        json={
            "status": "completed",
        },
    )
    assert r.status_code == 200


@pytest.mark.asyncio
async def test_create_subtask(api):
    """Test create subtask."""

    cr = await api.post("/api/jobs", json={"title": "ParentJob"})
    parent_id = cr.json()["data"]["id"]

    r = await api.post(
        f"/api/jobs/{parent_id}/subtasks",
        json={
            "title": "Child Task",
            "priority": "low",
        },
    )
    assert r.status_code == 200
