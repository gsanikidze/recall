.PHONY: ui build build-nui dev clean

# Build the React UI assets (requires Node/npm)
ui:
	cd ui && npm install && npm run build
	@echo 'module nodemodules' > ui/node_modules/go.mod

# Build the full binary with embedded UI
build: ui
	go build -tags ui -o recall .

# Build without UI (no Node required; `recall ui` will show a 503)
build-nui:
	go build -o recall .

# Run the Go API server for development (pair with `cd ui && npm run dev`)
dev:
	go run . ui --no-browser

clean:
	rm -f recall
	rm -rf ui/dist ui/node_modules
