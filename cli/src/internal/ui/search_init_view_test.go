package ui

import (
	"testing"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestSearchInitReturnsNilCmd(t *testing.T) {
	model := NewSearchModel(nil)
	assert.Nil(t, model.Init())
}

func TestSearchViewRendersEmptyAndPopulatedStates(t *testing.T) {
	model := NewSearchModel(nil)
	model.width = 80

	out := model.View()
	assert.Contains(t, out, "Type to search.")

	// Inject a results message directly to avoid needing a live client.
	model.query = "a"
	model.mode = searchModeText
	model, _ = model.Update(searchResultsMsg{
		query:    "a",
		mode:     searchModeText,
		entities: []api.Entity{{ID: "ent-1", Name: "Alpha", Type: "person"}},
	})

	out = model.View()
	assert.Contains(t, out, "Alpha")
}
