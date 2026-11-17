package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOriginParser(t *testing.T) {
	var allowList []string
	parser := newOriginParser(&allowList)

	require.NoError(t, parser(""))
	assert.Empty(t, allowList)

	require.NoError(t, parser("http://a.example.com, http://b.example.com ,"))
	assert.Equal(t, []string{"http://a.example.com", "http://b.example.com"}, allowList)

	// Each invocation replaces the previous value.
	require.NoError(t, parser("http://c.example.com"))
	assert.Equal(t, []string{"http://c.example.com"}, allowList)
}

func TestNewDialMapParser(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		expectedMap   map[string]string
		expectedError string
	}{
		{
			name:        "empty value yields nil map",
			value:       "",
			expectedMap: nil,
		},
		{
			name:        "single entry",
			value:       "my-guardrail.my-ns.svc.cluster.local=127.0.0.1:9443",
			expectedMap: map[string]string{"my-guardrail.my-ns.svc.cluster.local": "127.0.0.1:9443"},
		},
		{
			name:  "multiple comma-separated entries with whitespace",
			value: " host-a=127.0.0.1:9443 , host-b=127.0.0.1:9444 ,",
			expectedMap: map[string]string{
				"host-a": "127.0.0.1:9443",
				"host-b": "127.0.0.1:9444",
			},
		},
		{
			name:          "entry without separator is rejected",
			value:         "host-without-address",
			expectedError: `invalid dial map entry "host-without-address", expected host=address`,
		},
		{
			name:          "entry with empty host is rejected",
			value:         "=127.0.0.1:9443",
			expectedError: `invalid dial map entry "=127.0.0.1:9443", expected host=address`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dialMap map[string]string
			err := newDialMapParser(&dialMap)(tt.value)
			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMap, dialMap)
		})
	}

	// Each invocation replaces the previous value.
	t.Run("replaces previous value", func(t *testing.T) {
		var dialMap map[string]string
		parser := newDialMapParser(&dialMap)
		require.NoError(t, parser("host-a=127.0.0.1:1"))
		require.NoError(t, parser("host-b=127.0.0.1:2"))
		assert.Equal(t, map[string]string{"host-b": "127.0.0.1:2"}, dialMap)
	})
}
