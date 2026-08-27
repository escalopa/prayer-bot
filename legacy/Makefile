# Prayer-bot is a multi-module Go repository: the root module holds the shared
# packages (domain, config, log, internal/db) and each serverless function under
# serverless/ is its own module. These targets fan every command out across all
# of them so local checks match CI.

# Serverless modules (each has its own go.mod).
MODULES := serverless/dispatcher serverless/reminder serverless/loader

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Run tests for the root module and all serverless modules
	go test ./...
	@for m in $(MODULES); do \
		echo ">> testing $$m"; \
		(cd $$m && go test ./...) || exit 1; \
	done

.PHONY: vet
vet: ## Run go vet across every module
	go vet ./...
	@for m in $(MODULES); do \
		echo ">> vetting $$m"; \
		(cd $$m && go vet ./...) || exit 1; \
	done

.PHONY: fmt
fmt: ## Format all Go code
	gofmt -w -s .

.PHONY: fmt-check
fmt-check: ## Fail if any Go code is not gofmt-clean
	@out=$$(gofmt -l -s .); \
	if [ -n "$$out" ]; then \
		echo "gofmt needed on:"; echo "$$out"; exit 1; \
	fi

.PHONY: lint
lint: ## Run revive using the repo config (installs it if missing)
	@command -v revive >/dev/null 2>&1 || go install github.com/mgechev/revive@latest
	revive -config revive.toml -formatter friendly ./...

.PHONY: tidy
tidy: ## Run go mod tidy for the root module and all serverless modules
	go mod tidy
	@for m in $(MODULES); do \
		echo ">> tidying $$m"; \
		(cd $$m && go mod tidy) || exit 1; \
	done

.PHONY: check
check: fmt-check vet lint test ## Run the full local verification suite
