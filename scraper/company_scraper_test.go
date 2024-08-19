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
	// Prepare a temporary directory and file for testing
	tempDir := "companies-list"
	tempFile := tempDir + "/input.txt"
	expectedNames := []string{"stanbic bank", "Pearl Technologies", "clinicpesa"}

	// Ensure the directory exists
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create the input file with expected company names
	file, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temp directory after the test
	defer file.Close()

	// Write the expected company names to the file
	for _, name := range expectedNames {
		if _, err := file.WriteString(name + "\n"); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}

	// Call the function under test
	actualNames, err := ReadCompanyNames(tempFile)
	if err != nil {
		t.Fatalf("ReadCompanyNames returned an error: %v", err)
	}

	// Compare the actual output with the expected output
	if !reflect.DeepEqual(actualNames, expectedNames) {
		t.Errorf("Expected %v, but got %v", expectedNames, actualNames)
	}
}

func TestGetSearchResults(t *testing.T) {
	// Mock response data
	mockResponse := SerpAPIResponse{
		Organic: []struct {
			Link string `json:"link"`
		}{
			{Link: "https://google.serper.dev/search"},
		},
	}

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Encode the mock response as JSON and write it to the response writer
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create an http.Client that directs requests to the mock server
	client := server.Client()

	// Call the function under test with the mock client
	companyName := "stanbic bank"
	result, err := GetSearchResults(client, companyName)
	if err != nil {
		t.Fatalf("GetSearchResults returned an error: %v", err)
	}

	// Expected URL from the mock response
	expectedURL := "https://www.stanbicbank.co.ug/uganda/personal"

	// Check if the result matches the expected URL
	if result != expectedURL {
		t.Errorf("Expected %s, but got %s", expectedURL, result)
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