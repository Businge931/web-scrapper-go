# Company Email Scraper

This Go program reads a list of company names from a text file, performs a Google search, identifies the company's About page, scrapes the email address from the "About" page, and writes the company name and email address to an output file.

## Features
- Google Search integration (mocked for simplicity)
- Facebook About page email scraping
- Output to a structured text file
- Automated testing with high test coverage
- Continuous Integration (CI) pipeline with linting and test coverage

## Installation

1. Clone the repository:
    ```bash
    git clone https://github.com/Businge931/web-scrapper.git
    cd web-scraper
    ```

2. Install dependencies:
    ```bash
    go mod tidy
    ```

## Running the Program

```bash
go run main.go
