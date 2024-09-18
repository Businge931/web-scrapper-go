package models

import "errors"

var (
	// static error variables for GetSearchResults
	ErrInitConfig     = errors.New("error initializing configuration")
	ErrAPIKeyNotSet   = errors.New("SERPAPI_KEY not set in config or environment")
	ErrRequestFailed  = errors.New("failed to make request to SerpAPI")
	ErrDecodeFailed   = errors.New("failed to decode SerpAPI response")
	ErrNoResultsFound = errors.New("no results found")

	// static error variables for GetCompanyEmail
	ErrSkippingFacebookURL = errors.New("skipping Facebook URL")
	ErrFetchFailed         = errors.New("failed to fetch the page")
	ErrNonOKStatus         = errors.New("received non-OK HTTP status")
	ErrReadFailed          = errors.New("failed to read response body")
	ErrNoEmailFound        = errors.New("no email found on the page")
	ErrInvalidCompanyURL   = errors.New("invalid company URL")
	ErrWriteFileFailed     = errors.New("failed to write to file")
)
