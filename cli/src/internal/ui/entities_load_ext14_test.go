package ui

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEntitiesOmitsSearchParamWhenEmpty(t *testing.T) {
	var rawQuery string
	_, client := testEntitiesClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/entities" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		rawQuery = r.URL.RawQuery
		require.NoError(
			t,
			json.NewEncoder(w).Encode(
				map[string]any{
					"data": []map[string]any{
						{"id": "ent-1", "name": "alpha", "type": "person", "tags": []string{}},
					},
				},
			),
		)
	})

	model := NewEntitiesModel(client)
	cmd := model.loadEntities("")
	msg := cmd()

	loaded, ok := msg.(entitiesLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded.items, 1)
	assert.Equal(t, "", rawQuery)
	assert.Equal(t, "ent-1", loaded.items[0].ID)
}

func TestLoadEntitiesReturnsErrMsgOnQueryFailure(t *testing.T) {
	_, client := testEntitiesClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/entities" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		require.NoError(
			t,
			json.NewEncoder(w).Encode(
				map[string]any{
					"error": map[string]any{
						"code":    "INTERNAL_ERROR",
						"message": "db exploded",
					},
				},
			),
		)
	})

	model := NewEntitiesModel(client)
	cmd := model.loadEntities("al")
	msg := cmd()

	errResult, ok := msg.(errMsg)
	require.True(t, ok)
	require.Error(t, errResult.err)
	assert.Contains(t, errResult.err.Error(), "db exploded")
}
