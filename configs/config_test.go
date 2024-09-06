package configs

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name          string
		envVar        string
		envValue      string
		fileExists    bool
		fileContent   string
		expectedError bool
		expectedKey   string
	}{
		{
			name:          "Config file exists with valid API key",
			envVar:        "",
			envValue:      "",
			fileExists:    true,
			fileContent:   "serpapi:\n  api_key: valid_key_from_file",
			expectedError: false,
			expectedKey:   "valid_key_from_file",
		},
		{
			name:          "No config file, API key from environment",
			envVar:        "SERPAPI_KEY",
			envValue:      "valid_key_from_env",
			fileExists:    false,
			fileContent:   "",
			expectedError: false,
			expectedKey:   "valid_key_from_env",
		},
		{
			name:          "Config file exists but empty API key, fallback to environment",
			envVar:        "SERPAPI_KEY",
			envValue:      "fallback_env_key",
			fileExists:    true,
			fileContent:   "serpapi:\n  api_key: ",
			expectedError: false,
			expectedKey:   "fallback_env_key",
		},
		{
			name:          "Config file exists with empty API key, no environment variable",
			envVar:        "",
			envValue:      "",
			fileExists:    true,
			fileContent:   "serpapi:\n  api_key: ",
			expectedError: false,
			expectedKey:   "", // Empty API key as fallback
		},
		{
			name:          "No config file, no environment variable",
			envVar:        "",
			envValue:      "",
			fileExists:    false,
			fileContent:   "",
			expectedError: false,
			expectedKey:   "", // Default fallback should be empty
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable if needed
			if tc.envVar != "" {
				os.Setenv(tc.envVar, tc.envValue)
				defer os.Unsetenv(tc.envVar) // Clean up
			}

			// Reset viper configuration for each test case
			viper.Reset()

			// Simulate config file presence
			if tc.fileExists {
				viper.SetConfigType("yaml")

				err := viper.ReadConfig(strings.NewReader(tc.fileContent))
				if err != nil {
					t.Error("failed to simulate config file presence")
				}
			}

			// Run the function
			err := InitConfig()

			// Validate the results
			if tc.expectedError {
				assert.Error(t, err, "expected an error but did not get one")
			} else {
				assert.NoError(t, err, "expected no error but got one")
			}

			// Check if the API key was set correctly
			apiKey := viper.GetString("serpapi.api_key")
			assert.Equal(t, tc.expectedKey, apiKey)
		})
	}
}
