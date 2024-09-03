package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Businge931/company-email-scraper/config"
	"github.com/google/go-querystring/query"
	"github.com/stretchr/testify/assert"
)

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
				for _, name := range tt.companyNames {
					if _, err := file.WriteString(name + "\n"); err != nil {
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
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error, but got: %v", err)
				}
			}

			// Check the length and content of slices
			if len(actualNames) != len(tt.companyNames) {
				t.Errorf("Expected %d company names, but got %d", len(tt.companyNames), len(actualNames))
			}

			for i, name := range tt.companyNames {
				if actualNames[i] != name {
					t.Errorf("Expected company name %q at index %d, but got %q", name, i, actualNames[i])
				}
			}
		})
	}
}

func TestGetSearchResults(t *testing.T) {
	tests := map[string]struct {
		companyName   string
		apiKey        string
		mockServer    *httptest.Server
		expectedError string
		expectedURL   string
	}{
		"Successful search with valid API key and company name": {
			companyName: "testCompany",
			apiKey:      "validApiKey",
			mockServer: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(SerpAPIResponse{
					Organic: []struct {
						Link string `json:"link"`
					}{
						{Link: "https://example.com"},
					},
				})
			})),
			expectedError: "",
			expectedURL:   "https://example.com",
		},
		"Failure due to missing or invalid API key": {
			companyName:   "testCompany",
			apiKey:        "",
			mockServer:    nil,
			expectedError: "SERPAPI_KEY not set in config or environment",
		},
		"Failure due to the search API returning an error status": {
			companyName: "testCompany",
			apiKey:      "validApiKey",
			mockServer: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})),
			expectedError: "received non-OK HTTP status: 500 Internal Server Error",
		},
		"Failure when no results are found for the given company name": {
			companyName: "testCompany",
			apiKey:      "validApiKey",
			mockServer: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(SerpAPIResponse{
					Organic: []struct {
						Link string `json:"link"`
					}{},
				})
			})),
			expectedError: "no results found for testCompany",
		},
		"Failure due to JSON decoding issues": {
			companyName: "testCompany",
			apiKey:      "validApiKey",
			mockServer: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid JSON"))
			})),
			expectedError: "failed to decode SerpAPI response: ",
		},
		"Failure due to issues with the InitConfig function": {
			companyName:   "testCompany",
			apiKey:        "validApiKey",
			mockServer:    nil,
			expectedError: "error initializing configuration: ",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set up environment variable
			cleanup := config.SetupConfigFile(t, tc.apiKey)
			defer cleanup()

			err := config.InitConfig()
			if err != nil {
				t.Fatalf("error initializing config: %v", err)
			}

			if tc.mockServer != nil {
				defer tc.mockServer.Close()
				baseURL := tc.mockServer.URL
				params := struct {
					Query  string `url:"q"`
					APIKey string `url:"api_key"`
					Num    int    `url:"num"`
					Engine string `url:"engine"`
				}{
					Query:  tc.companyName,
					APIKey: tc.apiKey,
					Num:    1,
					Engine: "google",
				}
				queryParams, _ := query.Values(params)
				searchURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

				client := tc.mockServer.Client()

				url, err := GetSearchResults(client, searchURL)
				if tc.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.expectedError)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expectedURL, url)
				}
			} else {
				client := &http.Client{}

				url, err := GetSearchResults(client, tc.companyName)
				if tc.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.expectedError)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expectedURL, url)
				}
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.mockResponse))
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
