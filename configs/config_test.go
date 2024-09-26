package configs

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	type dependencies struct {
		envVar   string
		envValue string
	}

	type args struct {
		fileExists  bool
		fileContent string
	}

	type expected struct {
		errorExpected bool
		expectedKey   string
	}

	tests := []struct {
		name         string
		dependencies dependencies
		args         args
		expected     expected
	}{
		{
			name: "Config file exists with valid API key",
			dependencies: dependencies{
				envVar:   "",
				envValue: "",
			},
			args: args{
				fileExists:  true,
				fileContent: "serpapi:\n  api_key: valid_key_from_file",
			},
			expected: expected{
				errorExpected: false,
				expectedKey:   "valid_key_from_file",
			},
		},
		{
			name: "No config file, API key from environment",
			dependencies: dependencies{
				envVar:   "SERPAPI_KEY",
				envValue: "valid_key_from_env",
			},
			args: args{
				fileExists:  false,
				fileContent: "",
			},
			expected: expected{
				errorExpected: false,
				expectedKey:   "valid_key_from_env",
			},
		},
		{
			name: "Config file exists but empty API key, fallback to environment",
			dependencies: dependencies{
				envVar:   "SERPAPI_KEY",
				envValue: "fallback_env_key",
			},
			args: args{
				fileExists:  true,
				fileContent: "serpapi:\n  api_key: ",
			},
			expected: expected{
				errorExpected: false,
				expectedKey:   "fallback_env_key",
			},
		},
		{
			name: "Config file exists with empty API key, no environment variable",
			dependencies: dependencies{
				envVar:   "",
				envValue: "",
			},
			args: args{
				fileExists:  true,
				fileContent: "serpapi:\n  api_key: ",
			},
			expected: expected{
				errorExpected: false,
				expectedKey:   "", // Empty API key as fallback
			},
		},
		{
			name: "No config file, no environment variable",
			dependencies: dependencies{
				envVar:   "",
				envValue: "",
			},
			args: args{
				fileExists:  false,
				fileContent: "",
			},
			expected: expected{
				errorExpected: false,
				expectedKey:   "", // Default fallback should be empty
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable if needed
			if tc.dependencies.envVar != "" {
				os.Setenv(tc.dependencies.envVar, tc.dependencies.envValue)
				defer os.Unsetenv(tc.dependencies.envVar) // Clean up
			}

			// Reset viper configuration for each test case
			viper.Reset()

			// Simulate config file presence
			if tc.args.fileExists {
				viper.SetConfigType("yaml")

				err := viper.ReadConfig(strings.NewReader(tc.args.fileContent))
				if err != nil && !tc.expected.errorExpected {
					t.Error("failed to simulate config file presence")
				}
			}

			// Run the function
			err := InitConfig()

			// Validate the results
			if tc.expected.errorExpected {
				assert.Error(t, err, "expected an error but did not get one")
			} else {
				assert.NoError(t, err, "expected no error but got one")
			}

			// Check if the API key was set correctly
			apiKey := viper.GetString("serpapi.api_key")
			assert.Equal(t, tc.expected.expectedKey, apiKey)
		})
	}
}
