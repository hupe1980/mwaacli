package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		hasError bool
	}{
		{
			name: "Valid key-value pairs",
			input: `
            KEY1=value1
            KEY2=value2
            `,
			expected: []string{
				"KEY1=value1",
				"KEY2=value2",
			},
			hasError: false,
		},
		{
			name: "Quoted values with spaces",
			input: `
            KEY1="value with spaces"
            KEY2='another value with spaces'
			`,
			expected: []string{
				"KEY1=value with spaces",
				"KEY2=another value with spaces",
			},
			hasError: false,
		},
		{
			name: "Escaped characters in double quotes",
			input: `
            KEY1="value with \"escaped quotes\" and \n newlines"
            `,
			expected: []string{
				"KEY1=value with \"escaped quotes\" and \n newlines",
			},
			hasError: false,
		},
		{
			name: "Empty values",
			input: `
            KEY1=
            KEY2=""
            `,
			expected: []string{
				"KEY1=",
				"KEY2=",
			},
			hasError: false,
		},
		{
			name: "Unquoted values with inline comments",
			input: `
            KEY1=value1 # This is a comment
            KEY2=value2
            `,
			expected: []string{
				"KEY1=value1",
				"KEY2=value2",
			},
			hasError: false,
		},
		{
			name: "Invalid lines",
			input: `
            KEY1=value1
            invalid_line
            `,
			expected: nil,
			hasError: true,
		},
		{
			name: "Comments and empty lines",
			input: `
            # This is a comment
            KEY1=value1

            # Another comment
            KEY2=value2
            `,
			expected: []string{
				"KEY1=value1",
				"KEY2=value2",
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := ParseEnv(reader)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
