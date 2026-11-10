.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' ${MAKEFILE_LIST} \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: deps
deps: ## Download dependencies
	yarn
	go mod download

.PHONY: upgrade
upgrade: ## Upgrade dependencies
	go get -u ./...

.PHONY: tidy
tidy: ## Go mod tidy
	go mod tidy

.PHONY: tidy-check
tidy-check: ## Fail if go.mod/go.sum are not tidy
	go mod tidy -diff

.PHONY: fmt-check
fmt-check: ## Fail if any Go file is not gofmt-formatted
	@out="$$(gofmt -l $$(git ls-files '*.go'))"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: ## Run tests
	go test ./... -cover -race

.PHONY: lint
lint: ## Run linter
	golangci-lint run ./...

.PHONY: vuln
vuln: ## Run govulncheck
	govulncheck ./...

.PHONY: check
check: fmt-check vet lint test tidy-check ## Definition of done for every change

.PHONY: protoc
protoc: ## Generate protocol buffer
	protoc --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			devtool/*.proto