package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextUnmarshalCopiesTitleIntoLegacyName(t *testing.T) {
	var ctx Context
	require.NoError(t, json.Unmarshal([]byte(`{"id":"ctx-1","title":"Protocol Doc"}`), &ctx))

	assert.Equal(t, "Protocol Doc", ctx.Title)
	assert.Equal(t, "Protocol Doc", ctx.Name)
}

func TestContextUnmarshalCopiesLegacyNameIntoTitle(t *testing.T) {
	var ctx Context
	require.NoError(t, json.Unmarshal([]byte(`{"id":"ctx-2","name":"Legacy Name"}`), &ctx))

	assert.Equal(t, "Legacy Name", ctx.Title)
	assert.Equal(t, "Legacy Name", ctx.Name)
}

func TestContextUnmarshalPreservesExplicitTitleAndName(t *testing.T) {
	var ctx Context
	require.NoError(
		t,
		json.Unmarshal(
			[]byte(`{"id":"ctx-3","title":"Canonical Title","name":"Display Alias"}`),
			&ctx,
		),
	)

	assert.Equal(t, "Canonical Title", ctx.Title)
	assert.Equal(t, "Display Alias", ctx.Name)
}

func TestContextUnmarshalReturnsErrorForInvalidJSON(t *testing.T) {
	var ctx Context
	err := json.Unmarshal([]byte(`{"title"`), &ctx)
	require.Error(t, err)
}

func TestContextUnmarshalReturnsErrorForTypeMismatchPayload(t *testing.T) {
	var ctx Context
	err := json.Unmarshal([]byte(`{"id":"ctx-4","tags":"not-an-array"}`), &ctx)
	require.Error(t, err)
}
