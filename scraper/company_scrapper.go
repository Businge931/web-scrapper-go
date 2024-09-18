package scraper

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/spf13/viper"

	"github.com/Businge931/company-email-scraper/configs"
	"github.com/Businge931/company-email-scraper/models"
)

// SerpAPI response struct visit: https://serper.dev/playground
type SerpAPIResponse struct {
	Organic []struct {
		Link string `json:"link"`
	} `json:"organic"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
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

func GetSearchResults(client HTTPClient, companyName string) (string, error) {
	if err := configs.InitConfig(); err != nil {
		return "", fmt.Errorf("%w: %w",models.ErrInitConfig, err)
	}

	apiKey, err := getAPIKey()
	if err != nil {
		return "", err
	}

	searchURL, err := buildSearchURL(companyName, apiKey)
	if err != nil {
		return "", err
	}

	resp, err := makeHTTPRequest(client, searchURL)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	serpResponse, err := decodeResponse(resp)
	if err != nil {
		return "", err
	}

	return extractFirstResultURL(serpResponse, companyName)
}

func getAPIKey() (string, error) {
	apiKey := viper.GetString("serpapi.api_key")
	if apiKey == "" {
		return "", models.ErrAPIKeyNotSet
	}

	return apiKey, nil
}

func buildSearchURL(companyName, apiKey string) (string, error) {
	baseURL := "https://google.serper.dev/search"
	params := struct {
		Query  string `url:"q"`
		APIKey string `url:"api_key"`
		Num    int    `url:"num"`
		Engine string `url:"engine"`
	}{
		Query:  companyName,
		APIKey: apiKey,
		Num:    1,
		Engine: "google",
	}

	queryParams, err := query.Values(params)
	if err != nil {
		return "", fmt.Errorf("failed to encode query parameters: %w", err)
	}

	return fmt.Sprintf("%s?%s", baseURL, queryParams.Encode()), nil
}

func makeHTTPRequest(client HTTPClient, url string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", models.ErrRequestFailed, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", models.ErrRequestFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", models.ErrNonOKStatus, resp.Status)
	}

	return resp, nil
}

func decodeResponse(resp *http.Response) (SerpAPIResponse, error) {
	var serpResponse SerpAPIResponse

	err := json.NewDecoder(resp.Body).Decode(&serpResponse)
	if err != nil {
		return serpResponse, fmt.Errorf("%w: %w", models.ErrDecodeFailed, err)
	}

	return serpResponse, nil
}

func extractFirstResultURL(serpResponse SerpAPIResponse, companyName string) (string, error) {
	if len(serpResponse.Organic) == 0 {
		return "", fmt.Errorf("%w: %s", models.ErrNoResultsFound, companyName)
	}

	return serpResponse.Organic[0].Link, nil
}

func GetCompanyEmail(companyURL, companyName string) (string, error) {
	// skip Facebook URLs
	if strings.Contains(companyURL, "facebook.com") {
		return "", fmt.Errorf("%w: %s", models.ErrSkippingFacebookURL, companyURL)
	}

	// Validate the URL
	parsedURL, err := url.ParseRequestURI(companyURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("%w: %s", models.ErrInvalidCompanyURL, companyURL)
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a new HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, companyURL, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", models.ErrFetchFailed, err)
	}

	// Make the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", models.ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	// Check for non-OK status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s", models.ErrNonOKStatus, resp.Status)
	}

	// Read the body of the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %w", models.ErrReadFailed, err)
	}

	// Convert the body to a string for easier searching
	bodyStr := string(body)

	// Regular expression to find emails
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailRegex.FindAllString(bodyStr, -1)

	// If no emails are found, return an error
	if len(emails) == 0 {
		return "", fmt.Errorf("%w: %s", models.ErrNoEmailFound, companyName)
	}

	// Return the first email found
	return emails[0], nil
}

func WriteEmailsToFile(file *os.File, companyName, email string) error {
	_, err := file.WriteString(fmt.Sprintf("%s : %s\n", companyName, email))
	if err != nil {
		return fmt.Errorf("%w: %w", models.ErrWriteFileFailed, err)
	}

	return nil
}
