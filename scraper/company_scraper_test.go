package scraper

import (
	// "bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	ErrUnexpectedType = errors.New("unexpected type for response")
	ErrNetwork        = errors.New("network error")
)

// Mocking the HTTP client
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)

	// Type assert and check for success
	resp, ok := args.Get(0).(*http.Response)
	if !ok {
		return nil, ErrUnexpectedType
	}

	return resp, args.Error(1)
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
	tests := map[string]struct {
		companyNames []string
		expectError  bool
	}{
		"ValidCompanyNames": {
			companyNames: []string{"stanbic bank", "Pearl Technologies", "clinicpesa"},
			expectError:  false,
		},
		"EmptyFile": {
			companyNames: []string{},
			expectError:  false,
		},
		"FileNotFound": {
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Prepare a temporary directory and file for testing, except for the "FileNotFound" case
			var tempFile string

			if name != "FileNotFound" {
				tempDir := "companies-list"
				tempFile = tempDir + "/input.txt"

				if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
					t.Fatalf("Failed to create temp directory: %v", err)
				}
				defer os.RemoveAll(tempDir) // Clean up the temp directory after the test

				file, err := os.Create(tempFile)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer file.Close()

				// Write the company names to the file
				for i := range tt.companyNames {
					if _, err := file.WriteString(tt.companyNames[i] + "\n"); err != nil {
						t.Fatalf("Failed to write to temp file: %v", err)
					}
				}
			} else {
				// Set a non-existing file path for "FileNotFound" case
				tempFile = "non-existing-file.txt"
			}

			// Call the function under test
			actualNames, err := ReadCompanyNames(tempFile)

			// Check for expected error
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error, but got nil")
				}

				return
			} else if err != nil {
				t.Fatalf("Did not expect an error, but got: %v", err)
			}

			// Check the length and content of slices
			assertCompanyNames(t, actualNames, tt.companyNames)
		})
	}
}

func assertCompanyNames(t *testing.T, actualNames, expectedNames []string) {
	if len(actualNames) != len(expectedNames) {
		t.Errorf("Expected %d company names, but got %d", len(expectedNames), len(actualNames))
	}

	for i, name := range expectedNames {
		if actualNames[i] != name {
			t.Errorf("Expected company name %q at index %d, but got %q", name, i, actualNames[i])
		}
	}
}

func TestGetSearchResults(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		companyName    string
		mockResponse   *http.Response
		mockError      error
		expectedResult string
		expectedError  error
	}{
		{
			name:           "Successful search",
			apiKey:         "valid_api_key",
			companyName:    "TestCompany",
			mockResponse:   mockHTTPResponse(http.StatusOK, `{"organic": [{"link": "https://facebook.com/testcompany"}]}`),
			expectedResult: "https://facebook.com/testcompany",
			expectedError:  nil,
		},
		{
			name:           "Missing API key",
			apiKey:         "",
			companyName:    "TestCompany",
			expectedResult: "",
			expectedError:  ErrAPIKeyNotSet,
		},
		{
			name:           "Non-200 status code",
			apiKey:         "valid_api_key",
			companyName:    "TestCompany",
			mockResponse:   mockHTTPResponse(http.StatusInternalServerError, ""),
			expectedResult: "",
			expectedError:  ErrNonOKStatus,
		},
		{
			name:           "No search results",
			apiKey:         "valid_api_key",
			companyName:    "TestCompany",
			mockResponse:   mockHTTPResponse(http.StatusOK, `{"organic": []}`),
			expectedResult: "",
			expectedError:  ErrNoResultsFound,
		},
		{
			name:           "Request failure",
			apiKey:         "valid_api_key",
			companyName:    "TestCompany",
			mockError:      ErrNetwork,
			expectedResult: "",
			expectedError:  ErrRequestFailed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Mock the config and environment
			viper.Set("serpapi.api_key", tc.apiKey)

			// Create a mock HTTP client
			mockClient := new(MockClient)
			if tc.mockResponse != nil || tc.mockError != nil {
				mockClient.On("Do", mock.Anything).Return(tc.mockResponse, tc.mockError)
			}

			// Call GetSearchResults
			result, err := GetSearchResults(mockClient, tc.companyName)

			// Assert the result and error
			assert.Equal(t, tc.expectedResult, result)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Ensure the mock was called as expected
			mockClient.AssertExpectations(t)

			// Close the mock response body if it's not nil
			if tc.mockResponse != nil && tc.mockResponse.Body != nil {
				_ = tc.mockResponse.Body.Close()
			}
		})
	}
}

func TestGetCompanyEmail(t *testing.T) {
	tests := map[string]struct {
		companyURL   string
		companyName  string
		mockResponse string
		statusCode   int
		wantEmail    string
		expectError  bool
	}{
		"Valid email found": {
			companyURL:   "/test-company",
			companyName:  "Test Company",
			mockResponse: `<html><body>Contact us at test@example.com</body></html>`,
			statusCode:   http.StatusOK,
			wantEmail:    "test@example.com",
			expectError:  false,
		},
		"No email found": {
			companyURL:   "/no-email-company",
			companyName:  "No Email Company",
			mockResponse: `<html><body>No contact information available</body></html>`,
			statusCode:   http.StatusOK,
			wantEmail:    "",
			expectError:  true,
		},
		"Facebook URL skipped": {
			companyURL:   "https://www.facebook.com/test-company",
			companyName:  "Test Company",
			mockResponse: "",
			statusCode:   http.StatusOK,
			wantEmail:    "",
			expectError:  true,
		},
		"Non-OK status": {
			companyURL:   "/server-error",
			companyName:  "Server Error Company",
			mockResponse: "",
			statusCode:   http.StatusInternalServerError,
			wantEmail:    "",
			expectError:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)

				_, err := w.Write([]byte(tc.mockResponse))
				if err != nil {
					t.Error("failed to write mock response")
				}
			}))
			defer server.Close()

			// Use the mock server's URL for the test
			companyURL := server.URL + tc.companyURL

			email, err := GetCompanyEmail(companyURL, tc.companyName)
			if (err != nil) != tc.expectError {
				t.Fatalf("expected error: %v, got: %v", tc.expectError, err)
			}

			if email != tc.wantEmail {
				t.Errorf("expected email: %s, got: %s", tc.wantEmail, email)
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
