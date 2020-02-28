GO ?= go
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
GOFILES := $(shell find . -name "*.go" -type f -not -path './vendor/*')
COMPOSECMD := docker-compose -f "docker-compose.yaml" "up" -d
COMPOSEFILES_ACCEPTANCE_TESTING = -f tests/docker-compose.yaml -f tests/docker-compose.acceptance.yaml


.PHONY: all
all: fmt lint vet test

.PHONY: build
build:
	$(GO) build -o bin/workflows .

.PHONY: test
test:
	WORKFLOWS_MONGO_URL="mongodb://localhost" $(GO) test -cover -coverprofile=coverage.txt $(PACKAGES) && echo "\n==>\033[32m Ok\033[m\n" || exit 1

.PHONY: test-short
test-short:
	$(GO) test -cover -coverprofile=coverage.txt --short $(PACKAGES) && echo "\n==>\033[32m Ok\033[m\n" || exit 1

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

.PHONY: acceptance-testing-up
acceptance-testing-up:
	docker-compose $(COMPOSEFILES_ACCEPTANCE_TESTING) up -d

.PHONY: acceptance-testing-run
acceptance-testing-run:
	docker exec tests_acceptance-testing_1 /testing/run.sh

.PHONY: acceptance-testing-logs
acceptance-testing-logs:
	docker-compose $(COMPOSEFILES_ACCEPTANCE_TESTING) ps -a
	docker-compose $(COMPOSEFILES_ACCEPTANCE_TESTING) logs

.PHONY: acceptance-testing-down
acceptance-testing-down:
	docker-compose $(COMPOSEFILES_ACCEPTANCE_TESTING) down
