package local

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertAirlfowCfgToMap(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedResult map[string]string
		expectError    bool
	}{
		{
			name: "Valid configuration",
			configContent: `
[core]
dags_folder = /usr/local/airflow/dags
executor = LocalExecutor

[scheduler]
dag_dir_list_interval = 300
`,
			expectedResult: map[string]string{
				"core.dags_folder":                "/usr/local/airflow/dags",
				"core.executor":                   "LocalExecutor",
				"scheduler.dag_dir_list_interval": "300",
			},
			expectError: false,
		},
		{
			name: "Configuration with DEFAULT section",
			configContent: `
[DEFAULT]
key_in_default = value_in_default

[core]
dags_folder = /usr/local/airflow/dags
`,
			expectedResult: map[string]string{
				"core.dags_folder": "/usr/local/airflow/dags",
			},
			expectError: false,
		},
		{
			name: "Empty configuration file",
			configContent: `
`,
			expectedResult: map[string]string{},
			expectError:    false,
		},
		{
			name: "Invalid configuration file",
			configContent: `
invalid_line_without_section
`,
			expectedResult: nil,
			expectError:    true,
		},
		{
			name:           "Missing configuration file",
			configContent:  "",
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for the test
			var (
				tempFile *os.File
				err      error
			)

			if tt.configContent != "" {
				tempFile, err = os.CreateTemp("", "airflow.cfg")
				assert.NoError(t, err)
				defer os.Remove(tempFile.Name()) // Clean up the file after the test

				_, err = tempFile.WriteString(tt.configContent)
				assert.NoError(t, err)

				err = tempFile.Close()
				assert.NoError(t, err)
			}

			// Call the function
			var result map[string]string
			if tt.configContent != "" {
				result, err = ConvertAirlfowCfgToMap(tempFile.Name())
			} else {
				result, err = ConvertAirlfowCfgToMap("nonexistent.cfg")
			}

			// Validate the result
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
