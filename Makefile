SHELL:=/bin/bash
BIN:=./bin
GOLANGCI_LINT_VERSION?=1.28.0
built_at := $(shell date +%s)
git_commit := $(shell git describe --dirty --always)
version=$(shell cat version | tr -d '\n')

ifeq ($(OS),Windows_NT)
    OSNAME = windows
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        OSNAME = linux
		GOLANGCI_LINT_ARCHIVE=golangci-lint-$(GOLANGCI_LINT_VERSION)-linux-amd64.tar.gz
    endif
    ifeq ($(UNAME_S),Darwin)
        OSNAME = darwin
		GOLANGCI_LINT_ARCHIVE=golangci-lint-$(GOLANGCI_LINT_VERSION)-darwin-amd64.tar.gz
    endif
endif

ifdef os
  OSNAME=$(os)
endif

.PHONY: all
all: build

.PHONY: deps
deps:
	@go mod tidy
	@go mod vendor

.PHONY: lint
lint: $(BIN)/golangci-lint/golangci-lint ## lint
	$(BIN)/golangci-lint/golangci-lint run

$(BIN)/golangci-lint/golangci-lint:
	curl -OL https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/$(GOLANGCI_LINT_ARCHIVE)
	mkdir -p $(BIN)/golangci-lint/
	tar -xf $(GOLANGCI_LINT_ARCHIVE) --strip-components=1 -C $(BIN)/golangci-lint/
	chmod +x $(BIN)/golangci-lint
	rm -f $(GOLANGCI_LINT_ARCHIVE)

.PHONY: unit-test
unit-test:
	go test -v -mod=vendor -cover $$(go list ./...)

.PHONY: build
build: unit-test
	echo $(git_commit) > sha
	CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w \
	-X github.com/90poe/aws-s3-proxy/main.Commit=$(git_commit) \
	-X github.com/90poe/aws-s3-proxy/main.Date=$(built_at) \
    -X github.com/90poe/aws-s3-proxy/main.Ver=$(version)"\
	 -a -o ./artifacts/svc-unpacked ./cmd/aws-s3-proxy/
	rm -rf ./artifacts/svc
	upx -q -o ./artifacts/svc ./artifacts/svc-unpacked
