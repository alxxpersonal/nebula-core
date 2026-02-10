package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportEntities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/exports/entities", r.URL.Path)
		assert.Equal(t, "csv", r.URL.Query().Get("format"))
		w.Write(jsonResponse(map[string]any{
			"format":  "csv",
			"content": "id,name\n1,test\n",
			"count":   1,
		}))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "nbl_testkey")
	resp, err := client.ExportEntities(QueryParams{"format": "csv"})
	require.NoError(t, err)
	assert.Equal(t, "csv", resp.Format)
	assert.Equal(t, 1, resp.Count)
}
