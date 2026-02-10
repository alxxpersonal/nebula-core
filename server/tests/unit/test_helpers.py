"""Unit tests for pure helper functions (no DB needed)."""

# Third-Party
import copy

import pytest

from nebula_mcp.helpers import filter_context_segments

pytestmark = pytest.mark.unit


# --- filter_context_segments ---


class TestFilterContextSegments:
    """Tests for the filter_context_segments function."""

    def test_none_metadata_returns_none(self):
        """Return None when metadata is None."""

        result = filter_context_segments(None, ["public"])
        assert result is None

    def test_no_context_segments_key_returns_as_is(self):
        """Return metadata unchanged when context_segments key is absent."""

        meta = {"description": "hello"}
        result = filter_context_segments(meta, ["public"])
        assert result == {"description": "hello"}

    def test_empty_segments_returns_empty_list(self):
        """Return empty segments list when input segments are empty."""

        meta = {"context_segments": []}
        result = filter_context_segments(meta, ["public"])
        assert result["context_segments"] == []

    def test_all_match(self):
        """Keep all segments when agent has all required scopes."""

        meta = {
            "context_segments": [
                {"text": "a", "scopes": ["public"]},
                {"text": "b", "scopes": ["personal"]},
            ]
        }
        result = filter_context_segments(meta, ["public", "personal"])
        assert len(result["context_segments"]) == 2

    def test_none_match(self):
        """Remove all segments when agent scopes are disjoint."""

        meta = {
            "context_segments": [
                {"text": "a", "scopes": ["vault-only"]},
                {"text": "b", "scopes": ["sensitive"]},
            ]
        }
        result = filter_context_segments(meta, ["public"])
        assert result["context_segments"] == []

    def test_partial_match(self):
        """Keep only segments whose scopes overlap with agent scopes."""

        meta = {
            "context_segments": [
                {"text": "public note", "scopes": ["public"]},
                {"text": "secret note", "scopes": ["vault-only"]},
                {"text": "personal note", "scopes": ["personal"]},
            ]
        }
        result = filter_context_segments(meta, ["public", "personal"])
        texts = [s["text"] for s in result["context_segments"]]
        assert texts == ["public note", "personal note"]

    def test_multi_scope_segment(self):
        """Keep segment if any of its scopes match agent scopes."""

        meta = {
            "context_segments": [
                {"text": "multi", "scopes": ["public", "personal"]},
            ]
        }
        result = filter_context_segments(meta, ["public"])
        assert len(result["context_segments"]) == 1
        assert result["context_segments"][0]["text"] == "multi"

    def test_returns_copy_original_not_mutated(self):
        """Return a copy so the original metadata dict is not mutated."""

        meta = {
            "context_segments": [
                {"text": "a", "scopes": ["public"]},
                {"text": "b", "scopes": ["vault-only"]},
            ]
        }
        original = copy.deepcopy(meta)
        filter_context_segments(meta, ["public"])
        assert meta == original
