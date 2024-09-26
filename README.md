# Company Email Scraper

This Go program reads a list of company names from a text file, performs a Google search, identifies the company's About page, scrapes the email address from the "About" page, and writes the company name and email address to an output file.

The output file is a .txt and its content will have the structure below:
 company_name : email
 company_name_2 : email_2


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
    make deps
    ```

## Running the Program

```bash
make run
