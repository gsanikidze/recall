# Recall development guide

This guide covers local development, generated code, checks, and the UI build workaround.

## Repository layout

```text
cmd/                    CLI commands and command-specific tests
internal/memory/        Memory model, date parsing, validation
internal/vault/         Markdown vault source of truth
internal/index/         SQLite index/search layer
internal/index/db/      sqlc-generated Go code
internal/apiserver/     Local REST API for UI
internal/mcpserver/     MCP stdio server
ui/                     React/Vite/TypeScript app and embedded-asset package
docs/plans/             Roadmaps and implementation plans
```

## Local toolchain

Required:

- Go version from `go.mod`.
- Node.js 22+ and npm.
- Git.

Generated sqlc checks use:

```text
sqlc v1.30.0
```

The Makefile runs sqlc through `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0`, so a separate sqlc install is optional.

## Common commands

Format Go:

```bash
make fmt
make fmt-check
```

Check Go dependencies:

```bash
make tidy
make tidy-check
```

Run Go checks:

```bash
make vet
make test
make race
make cover
```

Run UI checks:

```bash
make install-ui
make lint-ui
make test-ui
make build-ui
make audit-ui
```

Run generated-code check:

```bash
make generate-check
```

Run everything:

```bash
make check
```

## Generated sqlc code

Inputs:

- `internal/index/schema.sql`
- `internal/index/query.sql`
- `sqlc.yaml`

Output:

- `internal/index/db/`

Regenerate:

```bash
make generate
```

Verify freshness:

```bash
make generate-check
```

`make generate-check` runs generation and then:

```bash
git diff --exit-code internal/index/db
```

If this fails, review generated diffs and commit them with the SQL/schema/query change that required regeneration.

## Building with and without UI

Full build with embedded React app:

```bash
make build
```

This does:

1. `npm --prefix ui ci`
2. creates `ui/node_modules/go.mod` sentinel
3. `npm --prefix ui run build`
4. `go build -tags ui -o bin/recall .`

Build without UI assets:

```bash
make build-nui
```

This does not need Node/npm. The binary still includes the API/server command, but SPA serving uses the stub filesystem and returns `503` until built with UI assets.

## Why `ui/node_modules/go.mod` exists

The UI package is under the Go module root because Go embeds `ui/dist` via the `recall/ui` package. After `npm ci`, `ui/node_modules` may contain packages with `.go` files. Go commands using `./...` walk every directory under the current module unless they encounter a nested `go.mod`.

The generated sentinel file:

```text
ui/node_modules/go.mod
```

contains:

```text
module nodemodules
```

This marks `ui/node_modules` as a nested Go module, so commands like `go test ./...`, `go vet ./...`, and `go test -tags ui ./...` do not traverse npm dependency trees.

`make clean` removes `ui/node_modules`, including the sentinel. `make build-ui` recreates it after npm install/build.

Cleaner alternatives considered:

- Move built assets to a Go-only directory outside `ui/node_modules` traversal concerns.
- Run Go commands with explicit package lists instead of `./...`.
- Keep current layout and sentinel because it is small, local to generated dependencies, and works with normal Go tooling.

Current decision: keep sentinel and document it.

## API/UI development loop

Run Go API server and Vite UI dev server together:

```bash
go run . dev
```

Or use Make:

```bash
make dev
```

Options:

```bash
go run . dev --api-port 8888 --ui-port 5173 --install
```

This starts the API at `http://localhost:8888` and Vite at
`http://localhost:5173`. `--install` runs `npm --prefix ui ci` first.

For separate processes, run Go server with no browser:

```bash
go run . ui --no-browser --port 8888
```

Run Vite dev server:

```bash
cd ui
npm ci
npm run dev
```

The API server has a CORS allowlist for Vite local origins and DNS-rebinding protection for non-loopback hosts.

## Local security model

Recall’s API is unauthenticated and local-only by design.

- Listen address is `localhost:<port>`.
- Non-loopback Host headers are rejected.
- CORS is restricted to local Vite dev origins.
- Do not expose the API port publicly without adding authentication and transport security.

## Environment overrides

`RECALL_PROJECT` overrides the project path containing `vault/` and `db/`.

```bash
RECALL_PROJECT=/tmp/recall-dev go test ./cmd
```

`RECALL_HOME` overrides Recall config home.

```bash
RECALL_HOME=/tmp/recall-home recall doctor
```

Use these in tests, throwaway local projects, and agent sandboxes.

## CI mapping

GitHub Actions runs the same Make targets used locally:

- `make fmt-check`
- `make tidy-check`
- `make vet`
- `make test`
- `make race`
- `make cover`
- `make generate-check`
- `make lint-ui`
- `make test-ui`
- `make build-ui`
- `make audit-ui`
- `make build-nui`
- `make build`
- `make test-ui-tag`

Keep local `make check` green before pushing.
