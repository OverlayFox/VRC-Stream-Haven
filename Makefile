.DEFAULT_GOAL := all

PPROF_DIR            ?= .pprof
PPROF_PORT           ?= 6060
PPROF_PROFILE        ?= heap
PPROF_DIFF_HTTP_PORT ?= 8081


.PHONY: build
build:
	@echo "==> Building..."
	go build -v ./...

.PHONY: test
test:
	@echo "==> Running tests with race detector..."
	go test -race -v ./...

.PHONY: lint
lint:
	golangci-lint run 

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: install-dev
install-dev:
	@echo "==> Installing development dependencies..."
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.11.2
	golangci-lint --version