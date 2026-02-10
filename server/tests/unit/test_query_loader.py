"""Unit tests for the SQL query loader."""

# Standard Library
from pathlib import Path

import pytest

from nebula_mcp.query_loader import QueryLoader

pytestmark = pytest.mark.unit


# --- QueryLoader Tests ---


class TestQueryLoader:
    """Tests for the QueryLoader class."""

    def test_load_existing_query(self, tmp_path):
        """Load a SQL file from the base directory."""

        sql_file = tmp_path / "my_query.sql"
        sql_file.write_text("SELECT 1;", encoding="utf-8")

        loader = QueryLoader(tmp_path)
        assert loader["my_query"] == "SELECT 1;"

    def test_load_nested_path(self, tmp_path):
        """Load a SQL file from a nested subdirectory."""

        subdir = tmp_path / "entities"
        subdir.mkdir()
        sql_file = subdir / "create.sql"
        sql_file.write_text("INSERT INTO entities;", encoding="utf-8")

        loader = QueryLoader(tmp_path)
        assert loader["entities/create"] == "INSERT INTO entities;"

    def test_caching_returns_same_content(self, tmp_path):
        """Return cached content on second access even if file changes."""

        sql_file = tmp_path / "cached.sql"
        sql_file.write_text("SELECT 1;", encoding="utf-8")

        loader = QueryLoader(tmp_path)
        first = loader["cached"]

        # Modify the file on disk
        sql_file.write_text("SELECT 2;", encoding="utf-8")
        second = loader["cached"]

        assert first == second == "SELECT 1;"

    def test_missing_file_raises(self, tmp_path):
        """Raise FileNotFoundError for a nonexistent query file."""

        loader = QueryLoader(tmp_path)
        with pytest.raises(FileNotFoundError, match="Query file not found"):
            loader["nonexistent"]

    def test_str_path_conversion(self, tmp_path):
        """Accept a string path and convert it to a Path internally."""

        sql_file = tmp_path / "strtest.sql"
        sql_file.write_text("SELECT 'str';", encoding="utf-8")

        loader = QueryLoader(str(tmp_path))
        assert loader["strtest"] == "SELECT 'str';"
        assert isinstance(loader.path, Path)

    def test_empty_cache_on_init(self, tmp_path):
        """Start with an empty cache on initialization."""

        loader = QueryLoader(tmp_path)
        assert loader.cache == {}
