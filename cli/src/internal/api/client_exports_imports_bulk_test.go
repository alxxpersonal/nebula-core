package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestExportMethodsBuildPathQueryAndDecode(t *testing.T) {
	cases := []struct {
		name string
		path string
		call func(c *Client) (*ExportResult, error)
	}{
		{
			name: "entities",
			path: "/api/exports/entities",
			call: func(c *Client) (*ExportResult, error) {
				return c.ExportEntities(QueryParams{"format": "json", "limit": "10"})
			},
		},
		{
			name: "knowledge",
			path: "/api/exports/knowledge",
			call: func(c *Client) (*ExportResult, error) {
				return c.ExportKnowledge(QueryParams{"format": "json", "limit": "10"})
			},
		},
		{
			name: "relationships",
			path: "/api/exports/relationships",
			call: func(c *Client) (*ExportResult, error) {
				return c.ExportRelationships(QueryParams{"format": "json", "limit": "10"})
			},
		},
		{
			name: "jobs",
			path: "/api/exports/jobs",
			call: func(c *Client) (*ExportResult, error) {
				return c.ExportJobs(QueryParams{"format": "json", "limit": "10"})
			},
		},
		{
			name: "context",
			path: "/api/exports/context",
			call: func(c *Client) (*ExportResult, error) {
				return c.ExportContext(QueryParams{"format": "json", "limit": "10"})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, tc.path, r.URL.Path)
				assert.Equal(t, "json", r.URL.Query().Get("format"))
				assert.Equal(t, "10", r.URL.Query().Get("limit"))

				w.Write(jsonResponse(map[string]any{
					"format": "json",
					"items":  []map[string]any{{"id": "x"}},
					"count":  1,
				}))
			})

			out, err := tc.call(client)
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, "json", out.Format)
			assert.Equal(t, 1, out.Count)
		})
	}
}

func TestImportMethodsEncodeBodyAndDecode(t *testing.T) {
	cases := []struct {
		name string
		path string
		call func(c *Client) (*BulkImportResult, error)
	}{
		{
			name: "knowledge",
			path: "/api/imports/knowledge",
			call: func(c *Client) (*BulkImportResult, error) {
				return c.ImportKnowledge(BulkImportRequest{Format: "json", Data: "[]"})
			},
		},
		{
			name: "relationships",
			path: "/api/imports/relationships",
			call: func(c *Client) (*BulkImportResult, error) {
				return c.ImportRelationships(BulkImportRequest{Format: "json", Data: "[]"})
			},
		},
		{
			name: "jobs",
			path: "/api/imports/jobs",
			call: func(c *Client) (*BulkImportResult, error) {
				return c.ImportJobs(BulkImportRequest{Format: "json", Data: "[]"})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, tc.path, r.URL.Path)

				var body BulkImportRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, "json", body.Format)
				assert.Equal(t, "[]", body.Data)

				w.Write(jsonResponse(map[string]any{
					"created": 1,
					"failed":  0,
					"errors":  []map[string]any{},
					"items":   []map[string]any{{"id": "ok"}},
				}))
			})

			out, err := tc.call(client)
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, 1, out.Created)
		})
	}
}

func TestBulkUpdateCallsEncodeBodyAndDecode(t *testing.T) {
	cases := []struct {
		name string
		path string
		call func(c *Client) (*BulkUpdateResult, error)
	}{
		{
			name: "tags",
			path: "/api/entities/bulk/tags",
			call: func(c *Client) (*BulkUpdateResult, error) {
				return c.BulkUpdateEntityTags(BulkUpdateEntityTagsInput{
					EntityIDs: []string{"e1", "e2"},
					Tags:      []string{"t"},
					Op:        "add",
				})
			},
		},
		{
			name: "scopes",
			path: "/api/entities/bulk/scopes",
			call: func(c *Client) (*BulkUpdateResult, error) {
				return c.BulkUpdateEntityScopes(BulkUpdateEntityScopesInput{
					EntityIDs: []string{"e1"},
					Scopes:    []string{"public"},
					Op:        "add",
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, tc.path, r.URL.Path)

				// Keep this minimal: just confirm the payload contains the expected operation.
				var raw map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
				assert.Equal(t, "add", raw["op"])

				w.Write(jsonResponse(map[string]any{
					"updated":    2,
					"entity_ids": []string{"e1", "e2"},
				}))
			})

			out, err := tc.call(client)
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, 2, out.Updated)
		})
	}
}

func TestClientErrorEnvelopeReturnsCodeMessage(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"BAD_REQUEST","message":"nope"}}`))
	})

	_, err := client.ExportEntities(QueryParams{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "BAD_REQUEST: nope")
}

func TestClientTransportFailureSurfacesDeterministicError(t *testing.T) {
	client := NewClient("http://example.com", "nbl_testkey")
	client.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})

	_, err := client.ExportEntities(QueryParams{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "request failed:")
	assert.ErrorContains(t, err, "boom")
}
