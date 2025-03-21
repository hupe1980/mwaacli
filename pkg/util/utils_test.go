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

func TestMergeEnvVars(t *testing.T) {
	tests := []struct {
		name              string
		input             []string
		ignoreEmptyValues bool
		expected          []string
	}{
		{
			name: "No duplicates, ignoreEmptyValues=false",
			input: []string{
				"KEY1=value1",
				"KEY2=value2",
			},
			ignoreEmptyValues: false,
			expected: []string{
				"KEY1=value1",
				"KEY2=value2",
			},
		},
		{
			name: "With duplicates, last occurrence wins, ignoreEmptyValues=false",
			input: []string{
				"KEY1=value1",
				"KEY2=value2",
				"KEY1=new_value1", // Duplicate key
			},
			ignoreEmptyValues: false,
			expected: []string{
				"KEY1=new_value1", // Last occurrence is kept
				"KEY2=value2",
			},
		},
		{
			name: "Empty values ignored, ignoreEmptyValues=true",
			input: []string{
				"KEY1=value1",
				"KEY2=",
				"KEY3=value3",
				"KEY2=new_value2", // Duplicate key
			},
			ignoreEmptyValues: true,
			expected: []string{
				"KEY1=value1",
				"KEY3=value3",
				"KEY2=new_value2", // Last occurrence is kept
			},
		},
		{
			name: "Empty values not ignored, ignoreEmptyValues=false",
			input: []string{
				"KEY1=value1",
				"KEY2=",
				"KEY3=value3",
				"KEY2=new_value2", // Duplicate key
			},
			ignoreEmptyValues: false,
			expected: []string{
				"KEY1=value1",
				"KEY2=new_value2", // Last occurrence is kept
				"KEY3=value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeEnvVars(tt.input, tt.ignoreEmptyValues)
			assert.ElementsMatch(t, tt.expected, result) // Order doesn't matter
		})
	}
}
