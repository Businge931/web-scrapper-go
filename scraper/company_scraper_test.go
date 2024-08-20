package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
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

			// Compare the actual output with the expected output
			if !reflect.DeepEqual(actualNames, tt.companyNames) {
				t.Errorf("Expected %v, but got %v", tt.companyNames, actualNames)
			}
		})
	}
}

func TestGetSearchResults(t *testing.T) {
	tests := map[string]struct {
		companyName   string
		mockResponse  SerpAPIResponse
		expectedURL   string
		expectedError bool
	}{
		"ValidResponse": {
			companyName: "stanbic bank",
			mockResponse: SerpAPIResponse{
				Organic: []struct {
					Link string `json:"link"`
				}{
					{Link: "https://www.stanbicbank.co.ug/uganda/personal"},
				},
			},
			expectedURL:   "https://www.stanbicbank.co.ug/uganda/personal",
			expectedError: false,
		},
		"NoResultsFound": {
			companyName: "unknown company",
			mockResponse: SerpAPIResponse{
				Organic: []struct {
					Link string `json:"link"`
				}{},
			},
			expectedURL:   "",
			expectedError: true,
		},
		"APIError": {
			companyName:   "error company",
			mockResponse:  SerpAPIResponse{},
			expectedURL:   "",
			expectedError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if name == "APIError" {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				// Encode the mock response as JSON and write it to the response writer
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create an http.Client that directs requests to the mock server
			client := server.Client()

			// Call the function under test with the mock client
			result, err := GetSearchResults(client, tt.companyName)

			if tt.expectedError {
				if err == nil {
					t.Fatalf("Expected an error but got none")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error but got: %v", err)
				}
			}

			// Check if the result matches the expected URL
			if result != tt.expectedURL {
				t.Errorf("Expected %s, but got %s", tt.expectedURL, result)
			}
		})
	}
}

func TestGetCompanyEmail(t *testing.T) {
	tests := []struct {
		name         string
		companyURL   string
		companyName  string
		mockResponse string
		statusCode   int
		wantEmail    string
		expectError  bool
	}{
		{
			name:         "Valid email found",
			companyURL:   "/test-company",
			companyName:  "Test Company",
			mockResponse: `<html><body>Contact us at test@example.com</body></html>`,
			statusCode:   http.StatusOK,
			wantEmail:    "test@example.com",
			expectError:  false,
		},
		{
			name:         "No email found",
			companyURL:   "/no-email-company",
			companyName:  "No Email Company",
			mockResponse: `<html><body>No contact information available</body></html>`,
			statusCode:   http.StatusOK,
			wantEmail:    "",
			expectError:  true,
		},
		{
			name:         "Facebook URL skipped",
			companyURL:   "https://www.facebook.com/test-company",
			companyName:  "Test Company",
			mockResponse: "",
			statusCode:   http.StatusOK,
			wantEmail:    "",
			expectError:  true,
		},
		{
			name:         "Non-OK status",
			companyURL:   "/server-error",
			companyName:  "Server Error Company",
			mockResponse: "",
			statusCode:   http.StatusInternalServerError,
			wantEmail:    "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Use the mock server's URL for the test
			companyURL := server.URL + tt.companyURL

			email, err := GetCompanyEmail(companyURL, tt.companyName)
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}
			if email != tt.wantEmail {
				t.Errorf("expected email: %s, got: %s", tt.wantEmail, email)
			}
		})
	}
}

func TestWriteEmailsToFile(t *testing.T) {
	tests := []struct {
		companyName string
		email       string
		wantOutput  string
	}{
		{
			companyName: "Test Company",
			email:       "test@example.com",
			wantOutput:  "Test Company : test@example.com\n",
		},
		{
			companyName: "Another Company",
			email:       "another@example.com",
			wantOutput:  "Another Company : another@example.com\n",
		},
		{
			companyName: "Empty Email",
			email:       "",
			wantOutput:  "Empty Email : \n",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Writing %s", tt.companyName), func(t *testing.T) {
			// Create a temporary file
			tmpFile, err := os.CreateTemp("", "test_emails_*.txt")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Call WriteEmailsToFile
			err = WriteEmailsToFile(tmpFile, tt.companyName, tt.email)
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
			if string(content) != tt.wantOutput {
				t.Errorf("expected %q, got %q", tt.wantOutput, string(content))
			}
		})
	}
}
