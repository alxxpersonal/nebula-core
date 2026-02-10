"""Unit tests for database helpers (no real DB connection)."""

# Third-Party
import pytest

from nebula_mcp.db import build_dsn, get_agent

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
