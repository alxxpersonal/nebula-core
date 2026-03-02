package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type metadataNilStringer struct{}

func (metadataNilStringer) String() string { return "<nil>" }

func TestParseStringSliceMatrix(t *testing.T) {
	assert.Equal(t, []string{"public", "admin"}, parseStringSlice([]string{" public ", "#admin", "PUBLIC"}))
	assert.Equal(
		t,
		[]string{"private", "sensitive", "<nil>"},
		parseStringSlice([]any{"private", "", "sensitive", nil}),
	)
	assert.Equal(t, []string{"public", "private"}, parseStringSlice("public, #private, public"))
	assert.Nil(t, parseStringSlice(123))
}

func TestFormatMetadataValueAndInlineFallbacks(t *testing.T) {
	assert.Equal(t, "None", formatMetadataValue(nil))
	assert.Equal(t, "None", formatMetadataValue("   "))
	assert.Equal(t, "[alpha, None]", formatMetadataValue([]any{"alpha", nil}))
	assert.Contains(t, formatMetadataValue(map[string]any{"k": "v"}), "\"k\":\"v\"")

	badMap := map[string]any{"bad": func() {}}
	fallback := formatMetadataValue(badMap)
	assert.Contains(t, fallback, "map")

	inlineFallback := formatMetadataInline(badMap)
	assert.Contains(t, inlineFallback, "map")
}

func TestMetadataPreviewFallbackAndValuePreviewBranches(t *testing.T) {
	data := map[string]any{"zeta": "last", "alpha": "first"}
	assert.Equal(t, "first", metadataPreview(data, 20))
	assert.Equal(t, "", metadataPreview(data, 0))

	withScopes := metadataValuePreview(map[string]any{"text": "note", "scopes": []any{"public", "admin"}}, 40)
	assert.Contains(t, withScopes, "note")
	assert.Contains(t, withScopes, "public")

	listPreview := metadataValuePreview([]any{"alpha", "beta", "gamma"}, 12)
	assert.NotEmpty(t, listPreview)

	mapPreview := metadataValuePreview(map[string]any{"a": "1", "b": "2"}, 30)
	assert.Contains(t, mapPreview, "a")
}

func TestFormatMetadataInlineDefaultAndSanitizeBranches(t *testing.T) {
	assert.Equal(t, "42", formatMetadataInline(42))
	assert.Equal(t, "None", formatMetadataInline(metadataNilStringer{}))

	sanitized := formatMetadataInline("\x1b[31mred\x1b[0m")
	assert.NotContains(t, sanitized, "\x1b[31m")
	assert.Contains(t, sanitized, "red")
}
