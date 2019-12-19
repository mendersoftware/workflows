GO ?= go
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
GOFILES := $(shell find . -name "*.go" -type f -not -path './vendor/*')
COMPOSECMD := docker-compose -f "docker-compose.yaml" "up" -d

.PHONY: all
all: fmt lint vet test

.PHONY: build
build:
	$(GO) build -o bin/workflows .

.PHONY: test
test:
	$(GO) test -cover -coverprofile=coverage.txt $(PACKAGES) && echo "\n==>\033[32m Ok\033[m\n" || exit 1

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: docker
docker:
	$(COMPOSECMD)

.PHONY: lint
lint:
	for pkg in ${PACKAGES}; do \
		golint -set_exit_status $$pkg || GOLINT_FAILED=1; \
	done; \
	[ -z "$$GOLINT_FAILED" ]

.PHONY: vet
vet:
	$(GO) vet $(PACKAGES)

.PHONY: clean
clean:
	$(GO) clean -modcache -x -i ./...
	find . -name coverage.txt -delete
	rm bin/*
