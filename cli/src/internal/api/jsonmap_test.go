package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONMapUnmarshalObject(t *testing.T) {
	var m JSONMap
	require.NoError(t, json.Unmarshal([]byte(`{"a":1}`), &m))
	assert.Equal(t, float64(1), m["a"])
}

func TestJSONMapUnmarshalJSONString(t *testing.T) {
	var m JSONMap
	require.NoError(t, json.Unmarshal([]byte(`"{\"a\":2}"`), &m))
	assert.Equal(t, float64(2), m["a"])
}

func TestJSONMapUnmarshalNullOrEmptyReturnsEmptyMap(t *testing.T) {
	cases := []struct {
		name       string
		payload    string
		wantNonNil bool
	}{
		{name: "null", payload: `null`, wantNonNil: false},
		{name: "empty string", payload: `""`, wantNonNil: true},
		{name: "quoted null", payload: `"null"`, wantNonNil: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m JSONMap
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &m))
			if tc.wantNonNil {
				assert.NotNil(t, m)
			}
			assert.Len(t, m, 0)
		})
	}
}
