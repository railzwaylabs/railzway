# Build variables
BINARY_NAME=railzway
DOCKER_IMAGE=railzwaylabs/railzway
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test coverage lint run docker-build help

all: lint test build

## Build: Compile the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) cmd/railzway/main.go

## Clean: Remove build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

## Test: Run unit tests
test:
	$(GOTEST) -v -race ./...

## Coverage: Run tests with coverage
coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out

## Lint: Run linters
lint:
	$(GOLINT) run

## Run: Run the application locally
run: build
	./$(BINARY_NAME) serve

## Docker Build: Build docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):latest .

## Deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## API Docs: Generate Swagger documentation
api-docs:
	swag init -g cmd/railzway/main.go --output docs

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
