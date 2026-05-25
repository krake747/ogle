.PHONY: all
all: tidy generate build fix test docs man notice

PKGS := $(shell go list ./internal/...)

.PHONY: test
test:
	go test -race $(PKGS)

.PHONY: coverage
coverage:
	@go test -covermode=count -coverprofile=coverage.out $(PKGS)
	@go tool cover -html=coverage.out -o coverage.html

.PHONY: build
build:
	go build -o ./bin/ogle .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: generate
generate:
	go tool mockery

.PHONY: lint
lint:
	go vet ./internal/...
	command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./internal/... || echo "golangci-lint not installed, skipping"

.PHONY: lint-md
lint-md:
	markdownlint-cli2 "**/*.md"

.PHONY: fmt
fmt:
	golangci-lint format ./...

.PHONY: fix
fix:
	golangci-lint run ./internal/... --fix
	markdownlint-cli2 --fix "**/*.md"

.PHONY: launch
launch:
	go run main.go

.PHONY: clean
clean:
	rm -rf ./bin

.PHONY: update
update:
	go get -u ./...
	go mod tidy

.PHONY: docs
docs:
	@echo "Generating CLI documentation..."
	@go run internal/tools/docgen/main.go --out ./docs/cli --format markdown
	@echo "CLI documentation generated in ./docs/cli"

.PHONY: man
man:
	@echo "Generating man pages..."
	@go run internal/tools/docgen/main.go --out ./docs/cli/man --format man
	@echo "Man pages generated in ./docs/cli/man"


.PHONY: notice
notice:
	@echo "Generating NOTICE file..."
	@go-licenses report . --template=notice.tpl --include_tests > NOTICE
	@echo "NOTICE file updated"

.PHONY: tools
tools:
	go install github.com/evilmartians/lefthook@latest
	go install github.com/conventionalcommit/commitlint@latest
	go install github.com/vektra/mockery/v3@latest
