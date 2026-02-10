package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMetadataInput(t *testing.T) {
	input := "name: alex\nprofile:\n  age: 17\n  tags: [ai, ml]"
	got, err := parseMetadataInput(input)
	assert.NoError(t, err)
	assert.Equal(t, "alex", got["name"])

	profile, ok := got["profile"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "17", profile["age"])

	tags, ok := profile["tags"].([]any)
	assert.True(t, ok)
	assert.Equal(t, []any{"ai", "ml"}, tags)
}

func TestParseMetadataInputErrors(t *testing.T) {
	_, err := parseMetadataInput("name alex")
	assert.Error(t, err)

	_, err = parseMetadataInput(" name: bad")
	assert.Error(t, err)
}

func TestMetadataToInput(t *testing.T) {
	data := map[string]any{
		"name": "alex",
		"profile": map[string]any{
			"age": 17,
		},
	}
	out := metadataToInput(data)
	assert.Contains(t, out, "name: alex")
	assert.Contains(t, out, "profile:")
	assert.Contains(t, out, "  age: 17")
}

func TestMetadataScopeHelpers(t *testing.T) {
	input := map[string]any{
		"scopes": []any{"Public", "work"},
		"name":   "alex",
	}
	scopes := extractMetadataScopes(input)
	assert.Equal(t, []string{"public", "work"}, scopes)

	stripped := stripMetadataScopes(input)
	_, ok := stripped["scopes"]
	assert.False(t, ok)

	merged := mergeMetadataScopes(stripped, []string{"personal"})
	assert.Equal(t, []string{"personal"}, merged["scopes"])
}
