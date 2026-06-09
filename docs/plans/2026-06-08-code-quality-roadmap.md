# Recall Code Quality Roadmap Implementation Plan

> **For Hermes:** Use `subagent-driven-development` or `executing-plans` skill to implement this plan task-by-task. Keep tasks small, verify after each task, and commit code changes separately from docs when requested.

**Goal:** Improve Recall code quality, safety, test coverage, CI reliability, and maintainability while preserving the local-first Markdown + SQLite architecture.

**Architecture:** Recall uses Go packages for CLI, engine, vault, SQLite index, MCP server, REST API, and a React/Vite UI. Improvements should harden shared core layers first (`memory`, `vault`, `index`, `recall`) so CLI/API/MCP/UI inherit safer behavior. CI should enforce formatting, dependency hygiene, tests, race checks, UI lint/build, and generated-code freshness.

**Tech Stack:** Go 1.26, SQLite via `modernc.org/sqlite`, goose migrations, sqlc-generated DB code, Model Context Protocol Go SDK, Fiber REST API, React/Vite/TypeScript UI, npm.

---

## Tracking Notes

- Use checkboxes below as persistent tracker.
- Prefer TDD: write failing test first, run it, implement minimal fix, run full relevant tests.
- Keep commits focused. Suggested commit style: `fix:`, `test:`, `ci:`, `chore:`, `refactor:`.
- Do not mix unrelated UI/backend/CI changes unless task explicitly says so.
- After each phase, run the verification commands listed in that phase.

---

## Current Review Snapshot

Observed on 2026-06-08:

- [x] `go test ./...` passed.
- [x] `go test -race ./...` passed.
- [x] `go vet ./...` passed.
- [x] `npm --prefix ui run lint` passed.
- [x] `npm --prefix ui run build` passed.
- [x] `npm --prefix ui audit --audit-level=moderate` found 0 vulnerabilities.
- [x] `gofmt -l $(git ls-files '*.go')` reports clean.
- [x] `go mod tidy` reports clean; `github.com/gofiber/fiber/v2` is a direct dependency.
- [x] CI workflow exists.
- [ ] API server has tests. Current coverage: `0.0%`.
- [ ] UI has tests. Current: no test files.
- [ ] UI initial bundle warning resolved. Current JS: about `1,366.01 kB` minified / `458.92 kB` gzip.

---

## Phase 1 — Baseline Automation and Hygiene

**Objective:** Make current quality checks repeatable and enforceable.

### Task 1.1 — Fix Go formatting

**Files:**
- Modify: `internal/apiserver/server.go`

**To-do:**
- [x] Run `gofmt -w internal/apiserver/server.go`.
- [x] Run `gofmt -l $(git ls-files '*.go')`.
- [x] Verify command prints no files.
- [x] Run `go test ./...`.
- [x] Commit included in Phase 1 combined commit.

**Verification:**
```bash
gofmt -l $(git ls-files '*.go')
go test ./...
```

### Task 1.2 — Fix Go module tidy drift

**Files:**
- Modify: `go.mod`
- Modify: `go.sum` if changed

**To-do:**
- [x] Run `go mod tidy`.
- [x] Confirm `github.com/gofiber/fiber/v2` is direct dependency in `go.mod`.
- [x] Run `go test ./...`.
- [x] Run `go vet ./...`.
- [x] Commit included in Phase 1 combined commit.

**Verification:**
```bash
go mod tidy
git diff --exit-code go.mod go.sum
go test ./...
go vet ./...
```

### Task 1.3 — Add CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

**To-do:**
- [x] Create GitHub Actions workflow for pushes and pull requests.
- [x] Add Go setup using `go-version-file: go.mod` or explicit Go 1.26.
- [x] Add Node setup with npm cache for `ui/package-lock.json`.
- [x] Run formatting check.
- [x] Run tidy check.
- [x] Run `go vet ./...`.
- [x] Run `go test ./...`.
- [x] Run `go test -race ./...`.
- [x] Run `go test -cover ./...`.
- [x] Run `npm --prefix ui ci`.
- [x] Run `npm --prefix ui run lint`.
- [x] Run `npm --prefix ui run build`.
- [x] Run `npm --prefix ui audit --audit-level=moderate`.
- [x] Run `make build-nui`.
- [x] Run `make build`.
- [x] Run `go test -tags ui ./...` after UI build.
- [x] Commit included in Phase 1 combined commit.

**Suggested workflow skeleton:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
          cache-dependency-path: ui/package-lock.json

      - name: Go format check
        run: test -z "$(gofmt -l $(git ls-files '*.go'))"

      - name: Go mod tidy check
        run: |
          go mod tidy
          git diff --exit-code go.mod go.sum

      - name: Go vet
        run: go vet ./...

      - name: Go tests
        run: go test ./...

      - name: Go race tests
        run: go test -race ./...

      - name: Go coverage
        run: go test -cover ./...

      - name: UI install
        run: npm --prefix ui ci

      - name: UI lint
        run: npm --prefix ui run lint

      - name: UI build
        run: npm --prefix ui run build

      - name: UI audit
        run: npm --prefix ui audit --audit-level=moderate

      - name: Build without UI
        run: make build-nui

      - name: Build with UI
        run: make build

      - name: Test embedded UI tag
        run: go test -tags ui ./...
```

### Task 1.4 — Expand Makefile quality contract

**Files:**
- Modify: `Makefile`

**To-do:**
- [x] Add `fmt` target.
- [x] Add `fmt-check` target.
- [x] Add `tidy` target.
- [x] Add `tidy-check` target.
- [x] Add `vet` target.
- [x] Add `test` target.
- [x] Add `race` target.
- [x] Add `cover` target.
- [x] Add `lint-ui` target.
- [x] Add `build-ui` target.
- [x] Add `check` target chaining core quality gates.
- [x] Document why `ui/node_modules/go.mod` workaround exists or remove in later task.
- [x] Run `make check`.
- [x] Commit included in Phase 1 combined commit.

**Verification:**
```bash
make check
```

---

## Phase 2 — Core Data Safety and Validation

**Objective:** Prevent corrupted/low-quality memories and reduce vault/index drift.

### Task 2.1 — Require `updated` and non-empty body in memory validation

**Files:**
- Modify: `internal/memory/memory.go`
- Modify: `internal/memory/memory_test.go`

**To-do:**
- [ ] Add failing test: `Memory.Validate` rejects missing `Updated`.
- [ ] Add failing test: `Memory.Validate` rejects blank `Body`.
- [ ] Run targeted test and verify failure.
- [ ] Update `Validate()` to enforce `Updated` non-zero and non-empty body.
- [ ] Run targeted test and verify pass.
- [ ] Run `go test ./internal/memory ./internal/recall ./cmd ./internal/mcpserver`.
- [ ] Commit:
  ```bash
  git add internal/memory/memory.go internal/memory/memory_test.go
  git commit -m "fix: validate memory body and updated date"
  ```

**Verification:**
```bash
go test ./internal/memory ./internal/recall ./cmd ./internal/mcpserver
```

### Task 2.2 — Validate parsed memories on `Get`

**Files:**
- Modify: `internal/vault/files.go` or `internal/recall/engine.go`
- Modify: `internal/vault/vault_test.go` or `internal/recall/engine_test.go`

**To-do:**
- [ ] Add failing test for hand-edited invalid memory returned by `Get`.
- [ ] Choose validation layer: prefer `vault.Read` if all reads should be valid.
- [ ] Add path-context error for invalid memory.
- [ ] Run targeted tests.
- [ ] Run `go test ./internal/vault ./internal/recall`.
- [ ] Commit:
  ```bash
  git add internal/vault/files.go internal/vault/vault_test.go internal/recall/engine_test.go
  git commit -m "fix: reject invalid memories on read"
  ```

**Verification:**
```bash
go test ./internal/vault ./internal/recall
```

### Task 2.3 — Add write/index rollback or compensation

**Files:**
- Modify: `internal/recall/engine.go`
- Modify: `internal/recall/engine_test.go`

**To-do:**
- [ ] Refactor engine to allow injectable index interface for failure tests, or add a test hook with minimal API.
- [ ] Add failing test: `Add` index failure removes newly written vault file.
- [ ] Add failing test: `Update` index failure restores old file content or leaves old path unchanged.
- [ ] Add failing test: `Delete` index failure does not leave stale searchable row, or documents/reindexes compensation.
- [ ] Implement compensation strategy.
- [ ] Update package comment to describe actual consistency model.
- [ ] Run targeted tests.
- [ ] Run `go test ./internal/recall ./internal/vault ./internal/index`.
- [ ] Commit:
  ```bash
  git add internal/recall/engine.go internal/recall/engine_test.go
  git commit -m "fix: compensate failed index updates"
  ```

**Verification:**
```bash
go test ./internal/recall ./internal/vault ./internal/index
```

### Task 2.4 — Reject vault symlink escapes

**Files:**
- Modify: `internal/vault/files.go`
- Modify: `internal/vault/vault.go`
- Modify: `internal/vault/vault_test.go`

**To-do:**
- [ ] Add failing test: domain symlink to outside dir is rejected by `HasDomain`.
- [ ] Add failing test: `Write` cannot write through symlinked domain.
- [ ] Add failing test: `WriteAt`, `Read`, `Delete` reject symlink escape paths.
- [ ] Use `os.Lstat` for domain checks.
- [ ] Use `filepath.EvalSymlinks` for root and final parent validation.
- [ ] Ensure resolved path stays under resolved vault root.
- [ ] Run `go test ./internal/vault`.
- [ ] Commit:
  ```bash
  git add internal/vault/files.go internal/vault/vault.go internal/vault/vault_test.go
  git commit -m "fix: prevent vault symlink escapes"
  ```

**Verification:**
```bash
go test ./internal/vault
```

---

## Phase 3 — Search, SQLite, and Index Integrity

**Objective:** Improve robustness under load, concurrency, and malformed filters.

### Task 3.1 — Clamp search limits centrally

**Files:**
- Modify: `internal/index/search.go`
- Modify: `internal/index/index_test.go` or create/modify search tests
- Modify API/MCP tests later if needed

**To-do:**
- [ ] Add `MaxLimit = 200` or similar.
- [ ] Add tests for `Limit <= 0` uses default.
- [ ] Add tests for huge limit clamped to max.
- [ ] Implement clamping in `Index.Search`.
- [ ] Run `go test ./internal/index`.
- [ ] Commit:
  ```bash
  git add internal/index/search.go internal/index/*_test.go
  git commit -m "fix: clamp search result limits"
  ```

**Verification:**
```bash
go test ./internal/index
```

### Task 3.2 — Validate search filters

**Files:**
- Modify: `internal/index/search.go`
- Modify: `internal/index/index_test.go`

**To-do:**
- [ ] Add failing test: invalid lifecycle returns clear error.
- [ ] Add failing test: invalid `since` date returns clear error.
- [ ] Add failing test: invalid `until` date returns clear error.
- [ ] Add failing test: `since > until` returns clear error.
- [ ] Implement validation before building SQL.
- [ ] Run `go test ./internal/index`.
- [ ] Commit:
  ```bash
  git add internal/index/search.go internal/index/*_test.go
  git commit -m "fix: validate search filters"
  ```

**Verification:**
```bash
go test ./internal/index
```

### Task 3.3 — Configure SQLite busy/WAL/foreign keys

**Files:**
- Modify: `internal/index/index.go`
- Modify: `internal/index/index_test.go`

**To-do:**
- [ ] Add test or integration check for concurrent writes.
- [ ] Set `sqlDB.SetMaxOpenConns(1)` or deliberate WAL strategy.
- [ ] Apply `PRAGMA busy_timeout = 5000`.
- [ ] Apply `PRAGMA journal_mode = WAL`.
- [ ] Apply `PRAGMA foreign_keys = ON`.
- [ ] Run `go test -race ./internal/index ./internal/recall`.
- [ ] Commit:
  ```bash
  git add internal/index/index.go internal/index/*_test.go
  git commit -m "fix: configure sqlite for local concurrency"
  ```

**Verification:**
```bash
go test -race ./internal/index ./internal/recall
```

### Task 3.4 — Add DB uniqueness and foreign-key constraints

**Files:**
- Create: `internal/index/migrations/0002_constraints.sql`
- Modify generated sqlc files if queries need regeneration
- Modify tests under `internal/index/`

**To-do:**
- [ ] Add migration for unique path.
- [ ] Add unique index for `(memory_id, tag)`.
- [ ] Add unique index for `(memory_id, target_id)`.
- [ ] Add foreign keys where feasible; if SQLite table rebuild needed, do carefully.
- [ ] Run migrations against temp DB in tests.
- [ ] Run `go test ./internal/index`.
- [ ] If sqlc schema changed, run `sqlc generate` and commit generated files.
- [ ] Commit:
  ```bash
  git add internal/index/migrations internal/index
  git commit -m "fix: enforce index constraints"
  ```

**Verification:**
```bash
go test ./internal/index
```

### Task 3.5 — Detect duplicate IDs during reindex

**Files:**
- Modify: `internal/recall/engine.go`
- Modify: `internal/recall/engine_test.go`

**To-do:**
- [ ] Add failing test with two Markdown files sharing same frontmatter `id`.
- [ ] Track `id -> relPath` during `Reindex`.
- [ ] Return error including both paths.
- [ ] Run `go test ./internal/recall`.
- [ ] Commit:
  ```bash
  git add internal/recall/engine.go internal/recall/engine_test.go
  git commit -m "fix: detect duplicate memory ids on reindex"
  ```

**Verification:**
```bash
go test ./internal/recall
```

---

## Phase 4 — API and MCP Hardening

**Objective:** Make local REST/MCP behavior predictable, validated, and tested.

### Task 4.1 — Add API server test suite

**Files:**
- Create: `internal/apiserver/server_test.go`

**To-do:**
- [ ] Add test helper that creates temp Recall project and engine.
- [ ] Test allowed hosts: `localhost`, `127.0.0.1`, `::1`.
- [ ] Test non-loopback host rejected.
- [ ] Test CORS allowlist for Vite dev origins.
- [ ] Test CRUD happy path.
- [ ] Test invalid JSON returns 400.
- [ ] Test unknown domain returns 422.
- [ ] Test missing memory returns 404.
- [ ] Test reindex endpoint returns stats.
- [ ] Run `go test ./internal/apiserver`.
- [ ] Commit:
  ```bash
  git add internal/apiserver/server_test.go
  git commit -m "test: cover api server routes"
  ```

**Verification:**
```bash
go test ./internal/apiserver
```

### Task 4.2 — Replace stringly error classification

**Files:**
- Modify: `internal/apiserver/server.go`
- Modify: `internal/recall/engine.go` if adding sentinel errors
- Modify tests from Task 4.1

**To-do:**
- [ ] Define typed/sentinel validation errors where appropriate.
- [ ] Replace `strings.Contains` validation mapping with `errors.Is` or typed error unwrap.
- [ ] Keep user-facing messages clear.
- [ ] Run API tests.
- [ ] Commit:
  ```bash
  git add internal/apiserver/server.go internal/recall/engine.go internal/apiserver/server_test.go
  git commit -m "refactor: use typed api errors"
  ```

**Verification:**
```bash
go test ./internal/apiserver ./internal/recall
```

### Task 4.3 — Add MCP coverage for validation and limits

**Files:**
- Modify: `internal/mcpserver/server_test.go`

**To-do:**
- [ ] Test `recall_search` respects limit cap.
- [ ] Test invalid lifecycle/date returns useful error.
- [ ] Test `recall_add` rejects blank body.
- [ ] Test unknown domain error remains useful for agents.
- [ ] Run `go test ./internal/mcpserver`.
- [ ] Commit:
  ```bash
  git add internal/mcpserver/server_test.go
  git commit -m "test: cover mcp validation paths"
  ```

**Verification:**
```bash
go test ./internal/mcpserver
```

---

## Phase 5 — UI Quality

**Objective:** Add frontend tests, tighten types, improve UX and bundle size.

### Task 5.1 — Add Vitest and Testing Library

**Files:**
- Modify: `ui/package.json`
- Modify: `ui/package-lock.json`
- Create: `ui/src/test/setup.ts` if needed
- Modify: `ui/tsconfig*.json` or `vite.config.ts` if needed

**To-do:**
- [ ] Install test deps: `vitest`, `jsdom`, Testing Library packages, `msw` or fetch mock choice.
- [ ] Add `test` and `test:watch` scripts.
- [ ] Add minimal smoke test.
- [ ] Run `npm --prefix ui test`.
- [ ] Commit:
  ```bash
  git add ui/package.json ui/package-lock.json ui/src/test ui/vite.config.ts
  git commit -m "test: add ui test harness"
  ```

**Verification:**
```bash
npm --prefix ui run test
npm --prefix ui run lint
npm --prefix ui run build
```

### Task 5.2 — Test API client behavior

**Files:**
- Create: `ui/src/api/client.test.ts`
- Modify: `ui/src/api/client.ts` if bugs found

**To-do:**
- [ ] Test successful JSON response.
- [ ] Test JSON error response.
- [ ] Test non-JSON error response.
- [ ] Test 204/no-body response if relevant.
- [ ] Ensure `Accept: application/json` is sent.
- [ ] Ensure `Content-Type` only sent when body exists.
- [ ] Encode path params with `encodeURIComponent`.
- [ ] Run UI tests.
- [ ] Commit:
  ```bash
  git add ui/src/api/client.ts ui/src/api/client.test.ts
  git commit -m "test: cover ui api client"
  ```

**Verification:**
```bash
npm --prefix ui run test
```

### Task 5.3 — Tighten TypeScript types

**Files:**
- Modify: `ui/tsconfig.app.json`
- Modify: `ui/src/api/types.ts`
- Modify UI components as needed

**To-do:**
- [ ] Enable `strict`.
- [ ] Consider `exactOptionalPropertyTypes`.
- [ ] Consider `noUncheckedIndexedAccess`.
- [ ] Add `type Lifecycle = 'evergreen' | 'expires'`.
- [ ] Use `Lifecycle` in create/update/filter/detail types.
- [ ] Model `expires_on` behavior better or validate in forms.
- [ ] Run UI lint/build/tests.
- [ ] Commit:
  ```bash
  git add ui/tsconfig.app.json ui/src
  git commit -m "refactor: tighten ui types"
  ```

**Verification:**
```bash
npm --prefix ui run lint
npm --prefix ui run build
npm --prefix ui run test
```

### Task 5.4 — Validate lifecycle/expiry in UI forms

**Files:**
- Modify: `ui/src/components/MemoryEditor.tsx`
- Modify: `ui/src/components/MetadataPanel.tsx`
- Add/modify related tests

**To-do:**
- [ ] Add test: selecting `expires` with no date blocks save and shows inline error.
- [ ] Add test: selecting `evergreen` clears `expires_on` before save.
- [ ] Implement validation.
- [ ] Run UI tests.
- [ ] Commit:
  ```bash
  git add ui/src/components ui/src/**/*.test.tsx
  git commit -m "fix: validate memory expiry in ui"
  ```

**Verification:**
```bash
npm --prefix ui run test
```

### Task 5.5 — Protect unsaved edits

**Files:**
- Modify: `ui/src/App.tsx`
- Modify: `ui/src/components/MemoryEditor.tsx`
- Add/modify related tests

**To-do:**
- [ ] Add test: navigating away with dirty editor prompts user.
- [ ] Add browser `beforeunload` guard for dirty state.
- [ ] Add route/selection guard.
- [ ] Consider local draft autosave only if needed.
- [ ] Run UI tests.
- [ ] Commit:
  ```bash
  git add ui/src/App.tsx ui/src/components/MemoryEditor.tsx ui/src/**/*.test.tsx
  git commit -m "fix: guard unsaved ui edits"
  ```

**Verification:**
```bash
npm --prefix ui run test
npm --prefix ui run build
```

### Task 5.6 — Reduce initial UI bundle size

**Files:**
- Modify: `ui/src/App.tsx`
- Modify: `ui/vite.config.ts` if adding chunking
- Modify tests if needed

**To-do:**
- [ ] Lazy-load `MemoryEditor`.
- [ ] Lazy-load heavy markdown editor path if possible.
- [ ] Add `Suspense` fallback.
- [ ] Configure manual chunks if useful.
- [ ] Add bundle size budget or documented threshold.
- [ ] Run build and compare bundle output.
- [ ] Commit:
  ```bash
  git add ui/src/App.tsx ui/vite.config.ts
  git commit -m "perf: lazy load memory editor"
  ```

**Verification:**
```bash
npm --prefix ui run build
```

### Task 5.7 — Clean UI dependency hygiene

**Files:**
- Modify: `ui/package.json`
- Modify: `ui/package-lock.json`

**To-do:**
- [ ] Move build-only deps to devDependencies: `autoprefixer`, `postcss`, `tailwindcss`.
- [ ] Audit unused Radix dependencies and `class-variance-authority`.
- [ ] Remove unused deps if not needed.
- [ ] Run `npm --prefix ui install` or `npm --prefix ui prune` as appropriate.
- [ ] Run lint/build/tests.
- [ ] Commit:
  ```bash
  git add ui/package.json ui/package-lock.json
  git commit -m "chore: clean ui dependencies"
  ```

**Verification:**
```bash
npm --prefix ui ci
npm --prefix ui run lint
npm --prefix ui run build
npm --prefix ui run test
npm --prefix ui audit --audit-level=moderate
```

---

## Phase 6 — UX/API Feature Completeness

**Objective:** Remove misleading UI and improve end-to-end behavior.

### Task 6.1 — Implement create-domain API or hide button

**Files if implementing:**
- Modify: `internal/apiserver/server.go`
- Modify: `internal/apiserver/server_test.go`
- Modify: `ui/src/components/DomainSidebar.tsx`
- Modify: `ui/src/App.tsx`
- Add UI tests

**To-do:**
- [ ] Decide: implement API/UI flow or hide/disable button.
- [ ] If implementing: add `POST /api/domains`.
- [ ] Add API test for create domain.
- [ ] Add UI dialog/form test.
- [ ] Ensure domain name validation messages are clear.
- [ ] Run API/UI tests.
- [ ] Commit:
  ```bash
  git add internal/apiserver ui/src
  git commit -m "feat: support creating domains in ui"
  ```

**Verification:**
```bash
go test ./internal/apiserver
npm --prefix ui run test
npm --prefix ui run build
```

### Task 6.2 — Encode/decode route params safely

**Files:**
- Modify: `ui/src/App.tsx`
- Modify: `ui/src/api/client.ts`
- Add/modify UI tests

**To-do:**
- [ ] Use `encodeURIComponent` for path segments.
- [ ] Decode `useParams()` values if needed.
- [ ] Add tests for special chars in IDs/domains if supported.
- [ ] Run UI tests/build.
- [ ] Commit:
  ```bash
  git add ui/src/App.tsx ui/src/api/client.ts ui/src/**/*.test.tsx
  git commit -m "fix: encode ui route params"
  ```

**Verification:**
```bash
npm --prefix ui run test
npm --prefix ui run build
```

---

## Phase 7 — Generated Code and Documentation

**Objective:** Make Recall easier to build and maintain over time. Release metadata/automation moved to Phase 8.

### Task 7.1 — Add generated-code freshness checks

**Files:**
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`
- Possibly modify generated files under `internal/index/db/`

**To-do:**
- [x] Add `generate` target for `sqlc generate`.
- [x] Add `generate-check` target: run generation then `git diff --exit-code internal/index/db`.
- [x] Pin/document sqlc version (`v1.30.0` observed in generated files).
- [x] Add CI step for generate check.
- [x] Run target locally.
- [ ] Commit:
  ```bash
  git add Makefile .github/workflows/ci.yml internal/index/db
  git commit -m "ci: verify generated sqlc code"
  ```

**Verification:**
```bash
make generate-check
```

### Task 7.2 — Add README and development docs

**Files:**
- Create: `README.md`
- Create or modify docs under `docs/`

**To-do:**
- [x] Document what Recall does.
- [x] Document install/build.
- [x] Document `make build` vs `make build-nui`.
- [x] Document `RECALL_HOME` and `RECALL_PROJECT`.
- [x] Document CLI examples: init/add/search/get/delete/reindex/mcp/ui.
- [x] Document MCP setup example.
- [x] Document API/UI dev flow.
- [x] Document test/check commands.
- [x] Document local security model: unauthenticated loopback API, CORS allowlist, DNS-rebinding guard.
- [ ] Commit:
  ```bash
  git add README.md docs
  git commit -m "docs: add recall development guide"
  ```

**Verification:**
```bash
make check
```

### Task 7.3 — Review `ui/node_modules/go.mod` workaround

**Files:**
- Modify: `Makefile`
- Modify docs if keeping workaround

**To-do:**
- [x] Determine why `@echo 'module nodemodules' > ui/node_modules/go.mod` is needed.
- [x] Prefer cleaner layout if possible: embed built assets from Go-only directory.
- [x] If keeping workaround, document exact reason in Makefile comment.
- [x] Ensure `make clean` removes generated artifacts.
- [x] Run build/test.
- [ ] Commit:
  ```bash
  git add Makefile ui
  git commit -m "chore: clarify ui build workaround"
  ```

**Verification:**
```bash
make clean
make build
make check
```

---

## Phase 8 — Release Metadata and Automation

**Objective:** Add version metadata and release packaging separately from Phase 7.

### Task 8.1 — Add release metadata and automation

**Files:**
- Modify version definition file (`main.go` or relevant cmd file)
- Create: `.goreleaser.yaml` or release script
- Modify: `Makefile`

**To-do:**
- [ ] Replace hardcoded-only version with ldflags variables: `version`, `commit`, `date`.
- [ ] Add Makefile target for local release snapshot.
- [ ] Add cross-platform release config: linux/darwin/windows, amd64/arm64.
- [ ] Include checksums.
- [ ] Run snapshot build.
- [ ] Commit:
  ```bash
  git add .goreleaser.yaml Makefile main.go cmd
  git commit -m "chore: add release metadata"
  ```

**Verification:**
```bash
make release-snapshot
./bin/recall version
```

---

## Master Checklist

### Baseline
- [x] Go formatting clean.
- [x] Go modules tidy.
- [x] CI workflow added.
- [x] Makefile has quality targets.

### Core safety
- [ ] Memory validation requires `updated`.
- [ ] Memory validation requires non-empty body.
- [ ] Parsed memories validated on read/get.
- [ ] Add/update/delete compensate index/vault failures.
- [ ] Vault rejects symlink escapes.

### Index/search
- [ ] Search limits clamped.
- [ ] Search filters validated.
- [ ] SQLite busy timeout configured.
- [ ] SQLite WAL configured.
- [ ] SQLite foreign keys enabled.
- [ ] DB uniqueness constraints added.
- [ ] Duplicate IDs detected during reindex.

### API/MCP
- [ ] API server tests added.
- [ ] API error mapping uses typed errors.
- [ ] MCP validation tests added.
- [ ] MCP limit tests added.

### UI
- [ ] UI test harness added.
- [ ] API client tests added.
- [ ] TypeScript strictness improved.
- [ ] Lifecycle/expiry validation added.
- [ ] Unsaved-edit guard added.
- [ ] Editor lazy-loaded.
- [ ] Bundle warning reduced/accepted with budget.
- [ ] UI dependency hygiene cleaned.

### UX/features
- [ ] New domain UX implemented or hidden.
- [ ] Route/API path params encoded safely.

### Release/docs
- [x] Generated-code freshness check added.
- [x] README added.
- [x] Dev/security docs added.
- [x] `ui/node_modules/go.mod` workaround reviewed.

### Release automation
- [ ] Release metadata added.
- [ ] Release automation added.

---

## Recommended Iteration Order

1. [x] Phase 1: CI + format + tidy + Makefile.
2. [ ] Phase 2: validation + consistency + symlink safety.
3. [ ] Phase 3: SQLite/search/index integrity.
4. [ ] Phase 4: API/MCP tests and errors.
5. [ ] Phase 5: UI tests/types/bundle.
6. [ ] Phase 6: UX completeness.
7. [x] Phase 7: docs/generated checks.
8. [ ] Phase 8: release metadata/automation.

---

## Global Verification Command

After each phase:

```bash
cd /root/Documents/recall
make check
npm --prefix ui run build
git status --short
```

Until `make check` exists, use:

```bash
cd /root/Documents/recall
test -z "$(gofmt -l $(git ls-files '*.go'))"
go mod tidy && git diff --exit-code go.mod go.sum
go vet ./...
go test ./...
go test -race ./...
make generate-check
npm --prefix ui ci
npm --prefix ui run lint
npm --prefix ui run test
npm --prefix ui run build
npm --prefix ui audit --audit-level=moderate
git status --short
```
