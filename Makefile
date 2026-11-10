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

PORTED_DIRS := internal log pattern trace

.PHONY: conventions
conventions: ## Fail if ported packages break the mechanical conventions in CLAUDE.md
	@for dir in $(PORTED_DIRS); do [ -d "$$dir" ] || { echo "PORTED_DIRS: $$dir does not exist"; exit 1; }; done
	@go fix -embedlit=false -diff $(addprefix ./,$(addsuffix /...,$(PORTED_DIRS))) || { echo "Go idioms: go fix wants the changes above"; exit 1; }
	@out="$$(grep -rnE '^[[:space:]]*(type[[:space:]]+)?[A-Za-z0-9_]*(Config|Opts|Cfg)(\[[^]]*\])?[[:space:]]+struct|^func[^"]*[(,][[:space:]]*(cfg|config|conf)[[:space:]]+\*?[A-Za-z0-9_.]+' --include='*.go' $(PORTED_DIRS))"; \
	if [ -n "$$out" ]; then echo "D-032: constructor settings are opts *XxxOptions:"; echo "$$out"; exit 1; fi
	@out="$$(grep -rnE '(^|[^A-Za-z0-9_])[tb]\.((Fatalf|Errorf)\(|(Fatal|Error)\(("|fmt\.Sprint))' --include='*_test.go' $(PORTED_DIRS) | grep -v '^internal/test/')"; \
	if [ -n "$$out" ]; then echo "D-002: use test.DiffMessage in ported tests:"; echo "$$out"; exit 1; fi

.PHONY: check
check: fmt-check vet lint conventions test tidy-check ## Definition of done for every change

.PHONY: protoc
protoc: ## Generate protocol buffer
	protoc --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			devtool/*.proto