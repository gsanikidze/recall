.PHONY: fmt fmt-check tidy tidy-check vet test race cover generate generate-check install-ui lint-ui test-ui build-ui audit-ui test-ui-tag ui build build-nui dev clean check

BIN_DIR := bin
BIN := $(BIN_DIR)/recall
GO_FILES := $(shell git ls-files '*.go')
SQLC_VERSION := v1.30.0
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@test -z "$$(gofmt -l $(GO_FILES))" || (gofmt -l $(GO_FILES); exit 1)

tidy:
	go mod tidy

tidy-check:
	@tmp=$$(mktemp -d); \
	cp go.mod go.sum $$tmp/; \
	go mod tidy; \
	diff -u $$tmp/go.mod go.mod; \
	diff -u $$tmp/go.sum go.sum; \
	rm -rf $$tmp

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

cover:
	go test -cover ./...

generate:
	$(SQLC) generate

generate-check: generate
	git diff --exit-code internal/index/db

install-ui:
	npm --prefix ui ci
	# Go treats every subdirectory under the module root as a package for ./... unless
	# it finds a nested go.mod. npm packages can contain .go files, so this sentinel
	# prevents `go test ./...` and friends from traversing ui/node_modules.
	@echo 'module nodemodules' > ui/node_modules/go.mod

lint-ui: install-ui
	npm --prefix ui run lint

test-ui: install-ui
	npm --prefix ui run test

build-ui: install-ui
	npm --prefix ui run build

audit-ui: install-ui
	npm --prefix ui audit --audit-level=moderate

test-ui-tag:
	go test -tags ui ./...

# Build the React UI assets (requires Node/npm)
ui: build-ui

# Build the full binary with embedded UI
build: ui
	mkdir -p $(BIN_DIR)
	go build -tags ui -o $(BIN) .

# Build without UI (no Node required; `recall ui` will show a 503)
build-nui:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) .

# Run the Go API server and Vite UI dev server together.
dev:
	go run . dev

check: fmt-check tidy-check vet test race cover generate-check lint-ui test-ui build-ui audit-ui build-nui build test-ui-tag

clean:
	rm -rf $(BIN_DIR)
	rm -f recall
	rm -rf ui/dist ui/node_modules
