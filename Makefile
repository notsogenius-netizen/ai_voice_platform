# AI Voice Support Platform — bootstrap Makefile
# Placeholders only. Targets will be filled in as services are implemented.

.PHONY: help test build lint tidy

help:
	@echo "Available targets (placeholders until services are implemented):"
	@echo "  make test   - run tests across services"
	@echo "  make build  - build all services"
	@echo "  make lint   - lint all services"
	@echo "  make tidy   - run go mod tidy in each service module"

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
