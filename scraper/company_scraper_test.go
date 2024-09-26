package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/Businge931/company-email-scraper/models"
)

type MockClient struct {
	MockDo func(req *http.Request) (*http.Response, error) // Function to simulate the Do behavior
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.MockDo(req)
}

// Helper function to mock HTTP responses
func mockHTTPResponse(statusCode int, body string) *http.Response {
	// Create a new response body reader
	bodyReader := strings.NewReader(body)

	// Return a mock response
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bodyReader), // Use io.NopCloser to simulate body closing
		Header:     make(http.Header),
	}
}

func TestReadCompanyNames(t *testing.T) {
	tests := []struct {
		name         string
		dependencies struct {
			filePath    string
			fileContent string
		}
		args struct {
			filePath string
		}
		before   func(t *testing.T)
		after    func(t *testing.T)
		expected struct {
			companyNames  []string
			errorExpected bool
		}
	}{
		{
			name: "success/Valid file with company names",
			dependencies: struct {
				filePath    string
				fileContent string
			}{
				filePath:    "valid_companies.txt",
				fileContent: "Company A\nCompany B\nCompany C\n",
			},
			args: struct {
				filePath string
			}{
				filePath: "valid_companies.txt",
			},
			before: func(t *testing.T) {
				err := os.WriteFile("valid_companies.txt", []byte("Company A\nCompany B\nCompany C\n"), 0o600)
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				err := os.Remove("valid_companies.txt")
				assert.NoError(t, err)
			},
			expected: struct {
				companyNames  []string
				errorExpected bool
			}{
				companyNames:  []string{"Company A", "Company B", "Company C"},
				errorExpected: false,
			},
		},
		{
			name: "error/File does not exist",
			dependencies: struct {
				filePath    string
				fileContent string
			}{
				filePath:    "nonexistent.txt",
				fileContent: "",
			},
			args: struct {
				filePath string
			}{
				filePath: "nonexistent.txt",
			},
			before: func(_ *testing.T) {},
			after:  func(_ *testing.T) {},
			expected: struct {
				companyNames  []string
				errorExpected bool
			}{
				companyNames:  nil,
				errorExpected: true,
			},
		},
		{
			name: "error/Malformed file content",
			dependencies: struct {
				filePath    string
				fileContent string
			}{
				filePath:    "malformed.txt",
				fileContent: "Company A\n\nCompany C\n", // Note: empty line
			},
			args: struct {
				filePath string
			}{
				filePath: "malformed.txt",
			},
			before: func(t *testing.T) {
				err := os.WriteFile("malformed.txt", []byte("Company A\n\nCompany C\n"), 0o600)
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				err := os.Remove("malformed.txt")
				assert.NoError(t, err)
			},
			expected: struct {
				companyNames  []string
				errorExpected bool
			}{
				companyNames:  []string{"Company A", "", "Company C"}, // Empty line included
				errorExpected: false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tc.before(t)

			// Run the function
			names, err := ReadCompanyNames(tc.args.filePath)

			// Cleanup
			tc.after(t)

			// Validate the results
			if tc.expected.errorExpected {
				assert.Error(t, err, "expected an error but did not get one")
			} else {
				assert.NoError(t, err, "expected no error but got one")
				assert.ElementsMatch(t, tc.expected.companyNames, names, "company names mismatch")
			}
		})
	}
}

func TestGetSearchResults(t *testing.T) {
	type dependencies struct {
		apiKey string
	}

	type args struct {
		companyName string
	}

	type expected struct {
		result string
		err    error
	}

	tests := []struct {
		name         string
		client       HTTPClient // Mocked client
		dependencies dependencies
		args         args
		expected     expected
	}{
		{
			name: "success/Successful search",
			client: &MockClient{
				MockDo: func(_ *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK, `{"organic": [{"link": "https://facebook.com/testcompany"}]}`), nil
				},
			},
			dependencies: dependencies{
				apiKey: "valid_api_key",
			},
			args: args{
				companyName: "TestCompany",
			},
			expected: expected{
				result: "https://facebook.com/testcompany",
				err:    nil,
			},
		},
		{
			name: "error/Missing API key",
			client: &MockClient{
				MockDo: func(_ *http.Request) (*http.Response, error) {
					return nil, models.ErrMockNoCall // No call expected
				},
			},
			dependencies: dependencies{
				apiKey: "",
			},
			args: args{
				companyName: "TestCompany",
			},
			expected: expected{
				result: "",
				err:    models.ErrAPIKeyNotSet,
			},
		},
		{
			name: "error/Non-200 status code",
			client: &MockClient{
				MockDo: func(_ *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusInternalServerError, ""), nil
				},
			},
			dependencies: dependencies{
				apiKey: "valid_api_key",
			},
			args: args{
				companyName: "TestCompany",
			},
			expected: expected{
				result: "",
				err:    models.ErrNonOKStatus,
			},
		},
		{
			name: "error/No search results",
			client: &MockClient{
				MockDo: func(_ *http.Request) (*http.Response, error) {
					return mockHTTPResponse(http.StatusOK, `{"organic": []}`), nil
				},
			},
			dependencies: dependencies{
				apiKey: "valid_api_key",
			},
			args: args{
				companyName: "TestCompany",
			},
			expected: expected{
				result: "",
				err:    models.ErrNoResultsFound,
			},
		},
		{
			name: "error/Request failure",
			client: &MockClient{
				MockDo: func(_ *http.Request) (*http.Response, error) {
					return nil, models.ErrNetwork
				},
			},
			dependencies: dependencies{
				apiKey: "valid_api_key",
			},
			args: args{
				companyName: "TestCompany",
			},
			expected: expected{
				result: "",
				err:    models.ErrRequestFailed,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Set API key from dependencies
			viper.Set("serpapi.api_key", tt.dependencies.apiKey)

			// Call GetSearchResults with args
			result, err := GetSearchResults(tt.client, tt.args.companyName)

			// Assert the expected result and error
			assert.Equal(t, tt.expected.result, result)

			if tt.expected.err != nil {
				assert.ErrorIs(t, err, tt.expected.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetCompanyEmail(t *testing.T) {
	// Define structs for dependencies, args, and expected results
	type dependencies struct {
		mockResponse string
		statusCode   int
	}

	type args struct {
		companyURL  string
		companyName string
	}

	type expected struct {
		wantEmail   string
		expectError bool
	}

	// Define the test cases
	tests := []struct {
		name         string
		dependencies dependencies
		args         args
		expected     expected
	}{
		{
			name: "Valid email found",
			dependencies: dependencies{
				mockResponse: `<html><body>Contact us at test@example.com</body></html>`,
				statusCode:   http.StatusOK,
			},
			args: args{
				companyURL:  "/test-company",
				companyName: "Test Company",
			},

			expected: expected{
				wantEmail:   "test@example.com",
				expectError: false,
			},
		},
		{
			name: "No email found",
			dependencies: dependencies{
				mockResponse: `<html><body>No contact information available</body></html>`,
				statusCode:   http.StatusOK,
			},
			args: args{
				companyURL:  "/no-email-company",
				companyName: "No Email Company",
			},

			expected: expected{
				wantEmail:   "",
				expectError: true,
			},
		},
		{
			name: "Facebook URL skipped",
			dependencies: dependencies{
				mockResponse: "",
				statusCode:   http.StatusOK,
			},
			args: args{
				companyURL:  "https://www.facebook.com/test-company",
				companyName: "Test Company",
			},

			expected: expected{
				wantEmail:   "",
				expectError: true,
			},
		},
		{
			name: "Non-OK status",
			dependencies: dependencies{
				mockResponse: "",
				statusCode:   http.StatusInternalServerError,
			},
			args: args{
				companyURL:  "/server-error",
				companyName: "Server Error Company",
			},

			expected: expected{
				wantEmail:   "",
				expectError: true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.dependencies.statusCode)

				_, err := w.Write([]byte(tc.dependencies.mockResponse))
				if err != nil {
					t.Error("failed to write mock response")
				}
			}))
			defer server.Close()

			// Use the mock server's URL for the test
			companyURL := server.URL + tc.args.companyURL

			// Call the function under test
			email, err := GetCompanyEmail(companyURL, tc.args.companyName)
			if (err != nil) != tc.expected.expectError {
				t.Fatalf("expected error: %v, got: %v", tc.expected.expectError, err)
			}

			if email != tc.expected.wantEmail {
				t.Errorf("expected email: %s, got: %s", tc.expected.wantEmail, email)
			}
		})
	}
}

func TestWriteEmailsToFile(t *testing.T) {
	tests := map[string]struct {
		companyName string
		email       string
		wantOutput  string
	}{
		"Test Company": {
			companyName: "Test Company",
			email:       "test@example.com",
			wantOutput:  "Test Company : test@example.com\n",
		},
		"Another Company": {
			companyName: "Another Company",
			email:       "another@example.com",
			wantOutput:  "Another Company : another@example.com\n",
		},
		"Empty Email": {
			companyName: "Empty Email",
			email:       "",
			wantOutput:  "Empty Email : \n",
		},
	}

	for name, tc := range tests {
		t.Run(fmt.Sprintf("Writing %s", name), func(t *testing.T) {
			// Create a temporary file
			tmpFile, err := os.CreateTemp("", "test_emails_*.txt")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Call WriteEmailsToFile
			err = WriteEmailsToFile(tmpFile, tc.companyName, tc.email)
			if err != nil {
				t.Fatalf("WriteEmailsToFile() error = %v", err)
			}

			// Close the file to flush the write
			tmpFile.Close()

			// Read the content of the file
			content, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("failed to read temp file: %v", err)
			}

			// Check if the content matches the expected output
			if string(content) != tc.wantOutput {
				t.Errorf("expected %q, got %q", tc.wantOutput, string(content))
			}
		})
	}
}
