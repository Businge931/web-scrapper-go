package main

import (
	"os"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Businge931/company-email-scraper/scraper"
)

func main() {
	companyNames, err := scraper.ReadCompanyNames("companies-list/input.txt")
	if err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}
	output := make(map[string]string)

	// Create a new http.Client
	client := &http.Client{}

	// Create the output file once
	fileName := "output/company_emails.txt"

	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	for i := range companyNames {
		companyURL, err := scraper.GetSearchResults(
			client,
			companyNames[i],
		)
		if err != nil {
			log.Printf("Error getting search results for %s: %v", companyNames[i], err)
			output[companyNames[i]] = ""

			continue
		}

		output[companyNames[i]] = companyURL

		email, err := scraper.GetCompanyEmail(companyURL, companyNames[i])
		if err != nil {
			log.Printf("Error fetching company email for %s: %v", companyNames[i], err)
			continue
		}

		err = scraper.WriteEmailsToFile(file, companyNames[i], email)
		if err != nil {
			log.Printf("Error writing to file for %s: %v", companyNames[i], err)
			continue
		}
	}
}
