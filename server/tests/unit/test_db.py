"""Unit tests for database helpers (no real DB connection)."""

# Third-Party
import pytest

from nebula_mcp.db import build_dsn, get_agent, get_pool

pytestmark = pytest.mark.unit


# --- build_dsn ---


class TestBuildDsn:
    """Tests for the build_dsn function."""

    def test_all_env_vars_set(self, monkeypatch):
        """Build DSN using all explicit environment variables."""

        monkeypatch.setenv("POSTGRES_HOST", "db.example.com")
        monkeypatch.setenv("POSTGRES_PORT", "5432")
        monkeypatch.setenv("POSTGRES_DB", "mydb")
        monkeypatch.setenv("POSTGRES_USER", "admin")
        monkeypatch.setenv("POSTGRES_PASSWORD", "secret")

        dsn = build_dsn()
        assert dsn == "postgresql://admin:secret@db.example.com:5432/mydb"

    def test_defaults_only_password(self, monkeypatch):
        """Use default values when only password is provided."""

        monkeypatch.delenv("POSTGRES_HOST", raising=False)
        monkeypatch.delenv("POSTGRES_PORT", raising=False)
        monkeypatch.delenv("POSTGRES_DB", raising=False)
        monkeypatch.delenv("POSTGRES_USER", raising=False)
        monkeypatch.setenv("POSTGRES_PASSWORD", "pass123")

        dsn = build_dsn()
        assert dsn == "postgresql://nebula:pass123@localhost:6432/nebula"

    def test_no_password_raises(self, monkeypatch):
        """Raise ValueError when POSTGRES_PASSWORD is not set."""

        monkeypatch.delenv("POSTGRES_PASSWORD", raising=False)

        with pytest.raises(ValueError, match="POSTGRES_PASSWORD is required"):
            build_dsn()

    def test_special_chars_in_password(self, monkeypatch):
        """URL-encode special characters in the password."""

        monkeypatch.setenv("POSTGRES_PASSWORD", "p@ss/w0rd#!")
        monkeypatch.setenv("POSTGRES_HOST", "localhost")
        monkeypatch.setenv("POSTGRES_PORT", "5432")
        monkeypatch.setenv("POSTGRES_DB", "nebula")
        monkeypatch.setenv("POSTGRES_USER", "nebula")

        dsn = build_dsn()
        assert "p%40ss%2Fw0rd%23%21" in dsn

    def test_custom_host_port(self, monkeypatch):
        """Build DSN with a custom host and port."""

        monkeypatch.setenv("POSTGRES_HOST", "10.0.0.5")
        monkeypatch.setenv("POSTGRES_PORT", "9999")
        monkeypatch.setenv("POSTGRES_PASSWORD", "pw")
        monkeypatch.delenv("POSTGRES_DB", raising=False)
        monkeypatch.delenv("POSTGRES_USER", raising=False)

        dsn = build_dsn()
        assert "@10.0.0.5:9999/" in dsn


# --- get_agent ---


class TestGetAgent:
    """Tests for the get_agent function."""

    async def test_empty_name_raises(self, mock_pool):
        """Raise ValueError when agent_name is empty."""

        with pytest.raises(ValueError, match="agent_name required"):
            await get_agent(mock_pool, "")

    async def test_returns_fetchrow_payload(self, mock_pool):
        """Valid names should delegate to fetchrow and return payload."""

        expected = {"id": "agent-1", "name": "alpha"}
        mock_pool.fetchrow.return_value = expected

        result = await get_agent(mock_pool, "alpha")

        assert result == expected
        assert mock_pool.fetchrow.await_count == 1
        assert mock_pool.fetchrow.await_args.args[1] == "alpha"


# --- get_pool ---


class TestGetPool:
    """Tests for the get_pool function."""

    async def test_create_pool_success_uses_passed_parameters(self, monkeypatch):
        """get_pool should pass dsn and override params to asyncpg.create_pool."""

        called = {}

        async def _fake_create_pool(**kwargs):
            called.update(kwargs)
            return "pool-object"

        monkeypatch.setattr("nebula_mcp.db.build_dsn", lambda: "postgresql://dsn")
        monkeypatch.setattr("nebula_mcp.db.asyncpg.create_pool", _fake_create_pool)

        pool = await get_pool(min_size=2, max_size=5, command_timeout=9)

        assert pool == "pool-object"
        assert called["dsn"] == "postgresql://dsn"
        assert called["min_size"] == 2
        assert called["max_size"] == 5
        assert called["command_timeout"] == 9

    async def test_connection_refused_maps_to_runtime_error(self, monkeypatch):
        """Connection refused errors should surface Docker hint RuntimeError."""

        monkeypatch.setattr("nebula_mcp.db.build_dsn", lambda: "postgresql://dsn")

        async def _raise_refused(**_kwargs):
            raise ConnectionRefusedError(61, "connection refused")

        monkeypatch.setattr("nebula_mcp.db.asyncpg.create_pool", _raise_refused)

        with pytest.raises(RuntimeError, match="Database connection failed"):
            await get_pool()

    async def test_non_connection_oserror_bubbles(self, monkeypatch):
        """Non-connectivity OSErrors should re-raise untouched."""

        monkeypatch.setattr("nebula_mcp.db.build_dsn", lambda: "postgresql://dsn")

        async def _raise_permission(**_kwargs):
            raise OSError(13, "permission denied")

        monkeypatch.setattr("nebula_mcp.db.asyncpg.create_pool", _raise_permission)

        with pytest.raises(OSError, match="permission denied"):
            await get_pool()
