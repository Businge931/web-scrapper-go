GOBASE = $(shell pwd)
LINT_PATH = $(GOBASE)/build/lint

help:
    @grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

lint: install-golangci ## Linter for developers
    $(LINT_PATH)/golangci-lint run --timeout=5m -c .golangci.yml
	
install-golangci: ## Install the correct version of lint
    GOBIN=$(LINT_PATH) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.58.1