.PHONY: all
all: tidy fix build test docs man notice

.PHONY: test
test:
	go test -race $(PKGS)

.PHONY: coverage
coverage:
	@go test -covermode=count -coverprofile=coverage.out $(PKGS)
	@go tool cover -html=coverage.out -o coverage.html

.PHONY: build
build:
	go build -o ./bin/meta1v .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run ./...

.PHONY: fmt
fmt:
	golangci-lint format ./...

.PHONY: fix
fix:
	golangci-lint run ./... --fix

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
	@go run internal/tools/docgen/main.go --out ./docs --format markdown
	@echo "CLI documentation generated in ./docs"

.PHONY: man
man:
	@echo "Generating man pages..."
	@go run internal/tools/docgen/main.go --out ./man --format man
	@echo "Man pages generated in ./man"


.PHONY: notice
notice:
	@echo "Generating NOTICE file..."
	@go-licenses report . --template=notice.tpl --include_tests > NOTICE
	@echo "NOTICE file updated"
