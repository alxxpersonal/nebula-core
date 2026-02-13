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

func testProtocolsClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestProtocolsListToDetailToEditFlow(t *testing.T) {
	now := time.Now()
	_, client := testProtocolsClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/protocols") && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":         "proto-1",
						"name":       "p1",
						"title":      "Protocol 1",
						"content":    "hello",
						"status":     "active",
						"tags":       []string{},
						"metadata":   map[string]any{},
						"created_at": now,
						"updated_at": now,
					},
				},
			})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewProtocolsModel(client)
	cmd := model.Init()
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.False(t, model.loading)
	assert.Len(t, model.items, 1)

	// Enter detail
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, model.detail)
	assert.Equal(t, protocolsViewDetail, model.view)

	// Enter edit
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	assert.Equal(t, protocolsViewEdit, model.view)
}

func TestProtocolsAddValidationErrorOnEmpty(t *testing.T) {
	_, client := testProtocolsClient(t, func(w http.ResponseWriter, r *http.Request) {})
	model := NewProtocolsModel(client)
	model.view = protocolsViewAdd

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	assert.Equal(t, "Name is required", model.addErr)
}

func TestProtocolsAddFlowSubmitsCreateProtocol(t *testing.T) {
	now := time.Now()
	var created api.CreateProtocolInput
	var posted bool

	_, client := testProtocolsClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/protocols") && r.Method == http.MethodGet:
			// Used both for initial load and post-create reload.
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{
				"id":         "proto-1",
				"name":       "p1",
				"title":      "Protocol 1",
				"content":    "hello",
				"status":     "active",
				"tags":       []string{"t1"},
				"metadata":   map[string]any{},
				"created_at": now,
				"updated_at": now,
			}}})
			return
		case r.URL.Path == "/api/protocols" && r.Method == http.MethodPost:
			posted = true
			require.NoError(t, json.NewDecoder(r.Body).Decode(&created))
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "proto-1"}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewProtocolsModel(client)
	cmd := model.Init()
	require.NotNil(t, cmd)
	model, _ = model.Update(cmd())

	// Enter Add.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, protocolsViewAdd, model.view)

	// Name.
	for _, r := range []rune("p1") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Title.
	for _, r := range []rune("Protocol 1") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Jump to Applies and Tags, leaving buffers uncommitted so saveAdd() commits them.
	for i := 0; i < 3; i++ { // Version, Type, Applies To
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Equal(t, protoFieldApplies, model.addFocus)
	for _, r := range []rune("entity") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown}) // Status
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown}) // Tags
	assert.Equal(t, protoFieldTags, model.addFocus)
	for _, r := range []rune("t1") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Content (required).
	for i := 0; i < 1; i++ { // Content
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Equal(t, protoFieldContent, model.addFocus)
	for _, r := range []rune("hello") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Save.
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	require.NotNil(t, cmd)
	model, cmd = model.Update(cmd())
	require.NotNil(t, cmd)
	model, _ = model.Update(cmd())

	require.True(t, posted)
	assert.Equal(t, "p1", created.Name)
	assert.Equal(t, "Protocol 1", created.Title)
	assert.Equal(t, "hello", created.Content)
	assert.Equal(t, []string{"entity"}, created.AppliesTo)
	assert.Equal(t, []string{"t1"}, created.Tags)
}

func TestProtocolsEditFlowCommitsTagAndApplyAndSaves(t *testing.T) {
	now := time.Now()
	var patched api.UpdateProtocolInput
	var patchedName string

	_, client := testProtocolsClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/protocols") && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{
				"id":         "proto-1",
				"name":       "p1",
				"title":      "Protocol 1",
				"content":    "hello",
				"status":     "active",
				"tags":       []string{},
				"metadata":   map[string]any{},
				"created_at": now,
				"updated_at": now,
			}}})
			return
		case strings.HasPrefix(r.URL.Path, "/api/protocols/") && r.Method == http.MethodPatch:
			patchedName = strings.TrimPrefix(r.URL.Path, "/api/protocols/")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&patched))
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "proto-1"}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewProtocolsModel(client)
	cmd := model.Init()
	require.NotNil(t, cmd)
	model, _ = model.Update(cmd())

	// List -> detail -> edit.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, model.detail)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	assert.Equal(t, protocolsViewEdit, model.view)

	// Focus Applies To and type buf (commit happens on save).
	for i := 0; i < protoEditFieldApplies; i++ {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Equal(t, protoEditFieldApplies, model.editFocus)
	for _, r := range []rune("entity") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Focus Tags and type buf.
	for i := model.editFocus; i < protoEditFieldTags; i++ {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Equal(t, protoEditFieldTags, model.editFocus)
	for _, r := range []rune("t2") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	require.NotNil(t, cmd)
	model, cmd = model.Update(cmd())
	require.NotNil(t, cmd)
	model, _ = model.Update(cmd())

	assert.Equal(t, "p1", patchedName)
	require.NotNil(t, patched.AppliesTo)
	assert.Equal(t, []string{"entity"}, *patched.AppliesTo)
	require.NotNil(t, patched.Tags)
	assert.Equal(t, []string{"t2"}, *patched.Tags)
	assert.Equal(t, protocolsViewList, model.view)
}

func TestProtocolPtrHelpers(t *testing.T) {
	assert.Nil(t, stringPtr(""))
	assert.Nil(t, stringPtr("  "))
	require.NotNil(t, stringPtr("x"))
	assert.Equal(t, "x", *stringPtr("x"))

	assert.Nil(t, slicePtr(nil))
	assert.Nil(t, slicePtr([]string{}))
	require.NotNil(t, slicePtr([]string{"a"}))
	assert.Equal(t, []string{"a"}, *slicePtr([]string{"a"}))
}
