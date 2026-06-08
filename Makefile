.PHONY: ui build build-nui dev clean

BIN_DIR := bin
BIN := $(BIN_DIR)/recall

# Build the React UI assets (requires Node/npm)
ui:
	cd ui && npm ci && npm run build
	@echo 'module nodemodules' > ui/node_modules/go.mod

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

clean:
	rm -rf $(BIN_DIR)
	rm -f recall
	rm -rf ui/dist ui/node_modules
