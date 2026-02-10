package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListKeysFiltered(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/keys", r.URL.Path)
		w.Write(jsonResponse([]map[string]any{
			{"id": "key-1", "prefix": "nbl_abc", "name": "my-key", "active": true},
		}))
	})

	keys, err := client.ListKeys()
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "my-key", keys[0].Name)
}

func TestListAllKeys(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/keys/all", r.URL.Path)
		w.Write(jsonResponse([]map[string]any{
			{"id": "key-1", "prefix": "nbl_abc", "name": "key1", "active": true},
			{"id": "key-2", "prefix": "nbl_def", "name": "key2", "active": false},
		}))
	})

	keys, err := client.ListAllKeys()
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestRevokeKey(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Contains(t, r.URL.Path, "/api/keys/key-1")
		w.Write(jsonResponse(map[string]any{}))
	})

	err := client.RevokeKey("key-1")
	require.NoError(t, err)
}

func TestRevokeKeyNotFound(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "NOT_FOUND",
				"message": "key not found",
			},
		})
		w.Write(b)
	})

	err := client.RevokeKey("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestLoginUnauthenticated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Login endpoint should not require auth
		assert.Empty(t, r.Header.Get("Authorization"))

		w.Write(jsonResponse(map[string]any{
			"api_key":   "nbl_newkey",
			"entity_id": "ent-1",
			"username":  "testuser",
		}))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "")
	resp, err := client.Login("testuser")
	require.NoError(t, err)
	assert.Equal(t, "nbl_newkey", resp.APIKey)
}

func TestLoginInvalidUsername(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "INVALID_USERNAME",
				"message": "username must be alphanumeric",
			},
		})
		w.Write(b)
	})

	_, err := client.Login("invalid@user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_USERNAME")
}

func TestCreateKeyDuplicateName(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "DUPLICATE",
				"message": "key name already exists",
			},
		})
		w.Write(b)
	})

	_, err := client.CreateKey("existing-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DUPLICATE")
}
