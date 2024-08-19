package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	fileName := fmt.Sprintf("company_emails_%d.txt", time.Now().Unix())
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	for _, companyName := range companyNames {
		companyURL, err := scraper.GetSearchResults(
			client,
			companyName)
		if err != nil {
			log.Printf("Error getting search results for %s: %v", companyName, err)
			output[companyName] = ""
			continue
		}
		output[companyName] = companyURL

		email, err := scraper.GetCompanyEmail(companyURL, companyName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		scraper.WriteEmailsToFile(file, companyName, email)

	}

}
