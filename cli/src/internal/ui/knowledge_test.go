package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func knowledgeTestClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestKnowledgeSaveLinksEntities(t *testing.T) {
	var linked []string
	_, client := knowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/knowledge" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "k-1", "name": "note"}})
		case strings.HasPrefix(r.URL.Path, "/api/knowledge/") && strings.HasSuffix(r.URL.Path, "/link"):
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			linked = append(linked, body["entity_id"])
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewKnowledgeModel(client)
	model.fields[fieldTitle].value = "Test"
	model.fields[fieldNotes].value = "Notes"
	model.linkEntities = []api.Entity{{ID: "ent-1", Name: "Alpha"}, {ID: "ent-2", Name: "Beta"}}

	model, cmd := model.save()
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.ElementsMatch(t, []string{"ent-1", "ent-2"}, linked)
	assert.True(t, model.saved)
}

func TestKnowledgeLinkSearchAddsEntity(t *testing.T) {
	model := NewKnowledgeModel(nil)
	model.linkSearching = true
	model.linkResults = []api.Entity{{ID: "ent-1", Name: "Alpha"}}
	model.linkList.SetItems([]string{"Alpha"})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Len(t, model.linkEntities, 1)
	assert.False(t, model.linkSearching)
}

func TestKnowledgeLinkSearchCommand(t *testing.T) {
	_, client := knowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/entities" {
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "ent-1", "name": "Alpha", "tags": []string{}}}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	model := NewKnowledgeModel(client)
	model.linkSearching = true
	model.linkQuery = ""

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.Len(t, model.linkResults, 1)
	assert.Equal(t, "ent-1", model.linkResults[0].ID)
}

func TestNormalizeTag(t *testing.T) {
	assert.Equal(t, "hello-world", normalizeTag(" Hello_World "))
	assert.Equal(t, "foo-bar-baz", normalizeTag("#Foo  Bar   Baz"))
	assert.Equal(t, "", normalizeTag(""))
}

func TestNormalizeScope(t *testing.T) {
	assert.Equal(t, "personal", normalizeScope(" Personal "))
	assert.Equal(t, "vault-only", normalizeScope("#Vault Only"))
}

func TestCommitTagDedupes(t *testing.T) {
	model := NewKnowledgeModel(nil)
	model.tags = []string{"alpha"}
	model.tagBuf = "ALPHA"
	model.commitTag()
	assert.Equal(t, []string{"alpha"}, model.tags)
}

func TestKnowledgeToggleModeLoadsList(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, client := knowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/knowledge" {
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{"id": "k-1", "name": "Alpha", "source_type": "note", "tags": []string{}, "created_at": now},
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	model := NewKnowledgeModel(client)
	model.view = knowledgeViewAdd

	model, cmd := model.toggleMode()
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.Equal(t, knowledgeViewList, model.view)
	assert.Len(t, model.items, 1)
}

func TestKnowledgeListEnterShowsDetail(t *testing.T) {
	model := NewKnowledgeModel(nil)
	model.view = knowledgeViewList
	model.items = []api.Knowledge{
		{ID: "k-1", Name: "Alpha", SourceType: "note"},
	}
	model.list.SetItems([]string{formatKnowledgeLine(model.items[0])})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	require.NotNil(t, model.detail)
	assert.Equal(t, "k-1", model.detail.ID)
	assert.Equal(t, knowledgeViewDetail, model.view)
}
