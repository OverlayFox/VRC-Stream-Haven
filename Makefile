.DEFAULT_GOAL := all

PPROF_DIR            ?= .pprof
PPROF_PORT           ?= 6060
PPROF_PROFILE        ?= heap
PPROF_DIFF_HTTP_PORT ?= 8081

.PHONY: all build test lint tidy verify clean pprof-snapshot pprof-diff

all: tidy verify lint build test

build:
	@echo "==> Building..."
	go build -v ./...

test:
	@echo "==> Running tests with race detector..."
	go test -race -v ./...

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

.PHONY: pprof-interactive
pprof-interactive:
	@echo "==> Starting pprof interactive mode..."
	go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap

$(PPROF_DIR):
	@mkdir -p $(PPROF_DIR)

pprof-snapshot: | $(PPROF_DIR)
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	FILE="$(PPROF_DIR)/$(PPROF_PROFILE)_$${TIMESTAMP}.pprof"; \
	echo "==> Capturing $(PPROF_PROFILE) profile from localhost:$(PPROF_PORT)..."; \
	curl -sf "http://localhost:$(PPROF_PORT)/debug/pprof/$(PPROF_PROFILE)" -o "$${FILE}" \
	  && echo "    Saved: $${FILE}" \
	  || { echo "ERROR: Could not reach pprof server at localhost:$(PPROF_PORT)"; exit 1; }

pprof-diff:
	@NEWEST=$$(ls -t $(PPROF_DIR)/$(PPROF_PROFILE)_*.pprof 2>/dev/null | head -n1); \
	SECOND=$$(ls -t $(PPROF_DIR)/$(PPROF_PROFILE)_*.pprof 2>/dev/null | sed -n '2p'); \
	if [ -z "$$NEWEST" ] || [ -z "$$SECOND" ]; then \
	  echo "ERROR: Need at least 2 snapshots. Run 'make pprof-snapshot' twice."; exit 1; \
	fi; \
	echo "==> Diffing profiles:"; \
	echo "    base: $$SECOND"; \
	echo "    head: $$NEWEST"; \
	go tool pprof -diff_base="$$SECOND" -http=:$(PPROF_DIFF_HTTP_PORT) "$$NEWEST"