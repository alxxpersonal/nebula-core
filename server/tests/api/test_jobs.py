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
async def test_create_job_accepts_iso_due_at(api):
    """Create job should accept ISO due_at strings."""

    r = await api.post(
        "/api/jobs",
        json={
            "title": "Timed Job",
            "due_at": "2026-02-18T18:00:00Z",
        },
    )
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["title"] == "Timed Job"
    assert data["due_at"] is not None


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
async def test_query_jobs_accepts_iso_due_filters(api):
    """Query jobs should parse ISO due filter params without 500 errors."""

    await api.post(
        "/api/jobs",
        json={
            "title": "Due Filter Job",
            "due_at": "2026-02-18T18:00:00Z",
        },
    )
    r = await api.get(
        "/api/jobs",
        params={"due_before": "2026-12-31T00:00:00Z"},
    )
    assert r.status_code == 200
    assert isinstance(r.json()["data"], list)


@pytest.mark.asyncio
async def test_query_jobs_invalid_due_filter_returns_400(api):
    """Invalid due filter should return INVALID_INPUT."""

    r = await api.get("/api/jobs", params={"due_before": "not-a-date"})
    assert r.status_code == 400
    body = r.json()
    assert body["detail"]["error"]["code"] == "INVALID_INPUT"


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
async def test_update_job_status_accepts_iso_completed_at(api):
    """Status updates should accept ISO completed_at values."""

    cr = await api.post("/api/jobs", json={"title": "Status Date Job"})
    job_id = cr.json()["data"]["id"]

    r = await api.patch(
        f"/api/jobs/{job_id}/status",
        json={
            "status": "completed",
            "completed_at": "2026-02-18T18:00:00Z",
        },
    )
    assert r.status_code == 200
    assert r.json()["data"]["completed_at"] is not None


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
