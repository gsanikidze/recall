.PHONY: fmt fmt-check tidy tidy-check vet test race cover lint-ui build-ui audit-ui test-ui-tag ui build build-nui dev clean check

BIN_DIR := bin
BIN := $(BIN_DIR)/recall
GO_FILES := $(shell git ls-files '*.go')

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

lint-ui:
	npm --prefix ui run lint

build-ui:
	npm --prefix ui ci
	npm --prefix ui run build
	# Keep Go commands from traversing npm dependency trees as nested packages.
	@echo 'module nodemodules' > ui/node_modules/go.mod

audit-ui:
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

# Run the Go API server for development (pair with `cd ui && npm run dev`)
dev:
	go run . ui --no-browser

check: fmt-check tidy-check vet test race cover lint-ui build-ui audit-ui build-nui build test-ui-tag

clean:
	rm -rf $(BIN_DIR)
	rm -f recall
	rm -rf ui/dist ui/node_modules
