# AI Voice Support Platform — bootstrap Makefile

.PHONY: help test build lint tidy quality quality-report quality-test quality-baseline quality-build

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
QUALITY_DIR := $(ROOT)/tools/quality
DIST := $(ROOT)/dist
BASE ?=
QUALITY_BIN := $(QUALITY_DIR)/bin/quality

help:
	@echo "Available targets:"
	@echo "  make test            - run tests across services"
	@echo "  make build           - build all services"
	@echo "  make lint            - lint all services"
	@echo "  make tidy            - run go mod tidy in each service module"
	@echo "  make quality         - run custom Go quality gate (BASE=origin/main for PR mode)"
	@echo "  make quality-report  - run quality gate and write JSON+SARIF under dist/"
	@echo "  make quality-test    - run quality tool unit tests"
	@echo "  make quality-baseline - analyze working tree without base comparison (full inventory)"

# Run tests in each service module that has packages.
# Empty modules (bootstrap) are skipped so the target stays green.
test:
	@echo "TODO: expand once service packages exist"
	@for dir in services/*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			pkgs=$$(cd "$$dir" && go list ./... 2>/dev/null || true); \
			if [ -z "$$pkgs" ]; then \
				echo "==> test $$dir (no packages yet, skip)"; \
			else \
				echo "==> test $$dir"; \
				(cd "$$dir" && go test ./...); \
			fi; \
		fi; \
	done

# Build each service binary (cmd packages will be added later).
build:
	@echo "TODO: build service binaries once cmd entrypoints exist"
	@for dir in services/*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			pkgs=$$(cd "$$dir" && go list ./... 2>/dev/null || true); \
			if [ -z "$$pkgs" ]; then \
				echo "==> build $$dir (no packages yet, skip)"; \
			else \
				echo "==> build $$dir"; \
				(cd "$$dir" && go build ./...); \
			fi; \
		fi; \
	done

# Lint placeholder — wire golangci-lint (or similar) when code exists.
lint:
	@echo "TODO: run linter once service code and lint config exist"

# Keep module files tidy (no-op when modules have no deps).
tidy:
	@for dir in services/*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "==> tidy $$dir"; \
			(cd "$$dir" && go mod tidy); \
		fi; \
	done
	@echo "==> tidy tools/quality"
	@(cd "$(QUALITY_DIR)" && go mod tidy)

$(QUALITY_BIN): $(shell find "$(QUALITY_DIR)" -name '*.go' -not -path '*/testdata/*')
	@mkdir -p "$(QUALITY_DIR)/bin"
	@(cd "$(QUALITY_DIR)" && go build -o bin/quality ./cmd/quality)

.PHONY: quality-build
quality-build:
	@mkdir -p "$(QUALITY_DIR)/bin"
	@(cd "$(QUALITY_DIR)" && go build -o bin/quality ./cmd/quality)

# Custom AST quality gate. Set BASE=origin/main (or origin/<base_ref>) for PR mode.
quality: $(QUALITY_BIN)
	@cfg="$(ROOT)/.quality.yaml"; \
	args="--root $(ROOT) --config $$cfg"; \
	if [ -n "$(BASE)" ]; then args="$$args --base $(BASE)"; fi; \
	"$(QUALITY_BIN)" $$args

# Generate terminal + JSON + SARIF artifacts under dist/.
quality-report: $(QUALITY_BIN)
	@mkdir -p "$(DIST)"
	@cfg="$(ROOT)/.quality.yaml"; \
	args="--root $(ROOT) --config $$cfg"; \
	if [ -n "$(BASE)" ]; then args="$$args --base $(BASE)"; fi; \
	set +e; \
	"$(QUALITY_BIN)" $$args --format json --output "$(DIST)/quality-report.json"; \
	code=$$?; \
	"$(QUALITY_BIN)" $$args --format sarif --output "$(DIST)/quality-report.sarif" --quiet; \
	sarif_code=$$?; \
	set -e; \
	if [ $$code -ne 0 ]; then exit $$code; fi; \
	if [ $$sarif_code -ne 0 ] && [ $$sarif_code -ne 1 ]; then exit $$sarif_code; fi; \
	exit $$code

# Full inventory without git baseline (useful when inspecting legacy debt).
quality-baseline: $(QUALITY_BIN)
	@"$(QUALITY_BIN)" --root "$(ROOT)" --config "$(ROOT)/.quality.yaml"

quality-test:
	@(cd "$(QUALITY_DIR)" && go test ./...)
