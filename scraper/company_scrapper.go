package scraper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/spf13/viper"

	"github.com/Businge931/company-email-scraper/config"
)

// SerpAPI response struct visit: https://serper.dev/playground
type SerpAPIResponse struct {
	Organic []struct {
		Link string `json:"link"`
	} `json:"organic"`
}

func ReadCompanyNames(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var companyNames []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		companyNames = append(companyNames, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return companyNames, nil
}

func GetSearchResults(client *http.Client, companyName string) (string, error) {
	if err := config.InitConfig(); err != nil {
		return "", fmt.Errorf("error initializing configuration: %w", err)
	}

	apiKey := viper.GetString("serpapi.api_key")
	if apiKey == "" {
		return "", fmt.Errorf("SERPAPI_KEY not set in config or environment")
	}

	baseURL := "https://google.serper.dev/search"
	params := struct {
		Query  string `url:"q"`
		APIKey string `url:"api_key"`
		Num    int    `url:"num"`
		Engine string `url:"engine"`
	}{
		Query:  companyName,
		APIKey: apiKey,
		Num:    1, // Fetch only the first result
		Engine: "google",
	}

	// Encode the query parameters
	queryParams, _ := query.Values(params)
	searchURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

	// Make the HTTP request to SerpAPI
	resp, err := client.Get(searchURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request to SerpAPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	// Decode the JSON response
	var serpResponse SerpAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&serpResponse)
	if err != nil {
		return "", fmt.Errorf("failed to decode SerpAPI response: %w", err)
	}

	// Ensure that we have at least one result
	if len(serpResponse.Organic) == 0 {
		return "", fmt.Errorf("no results found for %s", companyName)
	}

	// Return the first result URL
	return serpResponse.Organic[0].Link, nil
}

func GetCompanyEmail(companyURL, companyName string) (string, error) {
	// skip Facebook URLs
	if strings.Contains(companyURL, "facebook.com") {
		return "", fmt.Errorf("skipping Facebook URL: %s", companyURL)
	}

	resp, err := http.Get(companyURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch the page: %w", err)
	}
	defer resp.Body.Close()

	// Check for non-OK status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	// Read the body of the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert the body to a string for easier searching
	bodyStr := string(body)

	// Regular expression to find emails
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailRegex.FindAllString(bodyStr, -1)

	// If no emails are found, return an error
	if len(emails) == 0 {
		return "", fmt.Errorf("no email found on the page: %s", companyName)
	}

	// Return the first email found
	return emails[0], nil
}

func WriteEmailsToFile(file *os.File, companyName, email string) error {
	_, err := file.WriteString(fmt.Sprintf("%s : %s\n", companyName, email))
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	return nil
}
