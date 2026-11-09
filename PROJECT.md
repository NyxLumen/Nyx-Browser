# Project: Nyx Browser (NYX FOUNDATION: Phase 01 & Phase 01.1)

## Architecture
Nyx Browser is an independent, self-hosted file management and media browsing platform built from a fork of File Browser.
- **Backend Architecture**: Go 1.27 (`go1.27.3-X:nodwarf5`), module path `github.com/nyxlumen/nyx-browser/v2`.
  - Entry point: `main.go` -> `cmd/root.go`
  - CLI & Config: Cobra CLI (`nyx`), Viper configuration loader with strict 6-tier precedence:
    $$\mathbf{Flags} \;\;>\;\; \mathbf{NYX\_*} \;\;>\;\; \mathbf{FB\_*} \;\;>\;\; \mathbf{FILEBROWSER\_*} \;\;>\;\; \mathbf{Config\;Files} \;\;>\;\; \mathbf{Defaults}$$
  - Storage Layer: BoltDB (`go.etcd.io/bbolt`) with Storm ORM (`github.com/asdine/storm/v3`). Buckets: `User`, `Link`, and `config` (`settings`, `server`, `auther`, `version`). Default DB path: `./nyx.db`, with non-destructive fallback to `./filebrowser.db`.
  - Scoped Filesystem: `files.ScopedFs` wrapping `afero.BasePathFs`, enforcing path traversal boundaries and symlink confinement.
  - Authentication: JWT HMAC tokens with pluggable backends (`JSONAuth`, `ProxyAuth`, `NoAuth`, `HookAuth`).
  - HTTP Routing: Gorilla Mux router with embedded frontend assets (`frontend.Assets()`).
  - Upload Cache: Ephemeral chunk metadata cache with memory backend and multi-replica Redis backend (`nyx:upload:` namespace with non-destructive fallback to `filebrowser:upload:`).
  - Image/Media Preview: Semaphore-bounded downscaling pool (`img/service.go`) with disk caching (`diskcache/`).
- **Frontend Architecture**: Vue 3, TypeScript, Pinia, Vue Router, Dayjs, Vite 8, pnpm 10.33.4, Node 22/24.
  - Assets compiled into `frontend/dist/` and embedded into Go binary via `assets.go`.
  - Runtime configuration injected via Go HTML template into `window.Nyx` (with `window.FileBrowser` fallback).
  - Responsive dark-first desktop-grade interface.

---

## Feature Inventory

### Phase 01 Foundation Features (Completed)
| # | Feature | Description | Milestone | Status |
|---|---------|-------------|-----------|--------|
| 1 | Baseline Go Build & Test Verification | Compile 19 packages, run 59 unit tests | Phase 01 M1 | DONE (`5e9e39a7`) |
| 2 | Baseline Frontend Verification | Node 24/pnpm 10, typecheck, vitest, build | Phase 01 M1 | DONE (`5e9e39a7`) |
| 3 | Baseline Documentation & Defect Catalog | Baseline toolchain status and matrices | Phase 01 M1 | DONE (`5e9e39a7`) |
| 4 | Go Module Path Migration | `github.com/nyxlumen/nyx-browser/v2` migration | Phase 01 M2 | DONE (`d5bf51de`) |
| 5 | CLI & Binary Identity Migration | Rename to `nyx`, update build outputs | Phase 01 M2 | DONE (`d5bf51de`) |
| 6 | Database Name & Non-Destructive Fallback | Default `./nyx.db` with `./filebrowser.db` fallback | Phase 01 M2 | DONE (`d5bf51de`) |
| 7 | Config File Precedence & Dual-Env Support | `.nyx.*` > `.filebrowser.*`; `NYX_` > `FB_` | Phase 01 M2 | DONE (`d5bf51de`) |
| 8 | Frontend Identity & Metadata Update | Update manifest, title, sidebar, remove sunset card | Phase 01 M2 | DONE (`d5bf51de`) |
| 9 | Upstream License, Copyright & Attribution | Apache 2.0 `LICENSE`, preserve copyright, add `NOTICE` | Phase 01 M2 | DONE (`d5bf51de`) |
| 10 | Regression Test Coverage Expansion | Auth, permissions, CRUD, previews, config | Phase 01 M3 | DONE (`132b0030`) |
| 11 | Filesystem Boundary Hardening | Symlink traversal and scoped filesystem guards | Phase 01 M3 | DONE (`132b0030`) |
| 12 | Security Vulnerability Fixes | Fix NYX-SEC-001 (proxy auth) and NYX-SEC-002 (share) | Phase 01 M3 | DONE (`132b0030`) |
| 13 | Operational Deployment Audit | Dockerfile, compose, storage permissions, DEPLOYMENT.md | Phase 01 M3 | DONE (`132b0030`) |
| 14 | Backend Architecture & API Mapping | REST API map, ZDD BoltDB overlay blueprints | Phase 01 M4 | DONE (`6696a642`) |
| 15 | Frontend Redesign Proposal | 55 component inventory, OLED dark mode design tokens | Phase 01 M4 | DONE (`6696a642`) |
| 16 | Gallery Media Specification | Preview pipeline, downscaling, caching, FFmpeg | Phase 01 M4 | DONE (`6696a642`) |
| 17 | Code Quality Audit & Tech Debt | Dead code, dependencies, debt catalog | Phase 01 M4 | DONE (`6696a642`) |
| 18 | E2E Integration & Upgrade Compatibility | Clean install, upgrade, dual-env integration tests | Phase 01 M5 | DONE (`ee15cae8`) |
| 19 | Adversarial Security Verification | Stress-test traversal, collisions, database integrity | Phase 01 M5 | DONE (`ee15cae8`) |
| 20 | Milestone Integrity & Audit Review | 5 linear commits verified, POST_CAMPAIGN_REVIEW.md | Phase 01 M5 | DONE (`ee15cae8`) |

### Phase 01.1 Remediation Features
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 21 | CLI Version Flag Compatibility | Assign `rootCmd.Version = version.Version` in `cmd/root.go`, enabling `nyx --version` and `nyx -v` while preserving `nyx version` | P1.1-M1 | R2 |
| 22 | Environment Variable Precedence & `FILEBROWSER_` Support | Implement deterministic 6-tier binding in `cmd/utils.go` (`Flags > NYX_* > FB_* > FILEBROWSER_* > Config > Defaults`), deprecation check in `cmd/root.go`, container healthchecks | P1.1-M1 | R3 |
| 23 | CLI & Config Automated Regression Suite | Unit tests (`cmd/version_test.go`) and integration tests (`cmd/integration_test.go`) for `--version`, `version`, `FILEBROWSER_` fallback, and precedence | P1.1-M1 | R2, R3 |
| 24 | Redis Upload Cache Namespace Migration | Replace `"filebrowser:upload:"` with canonical `"nyx:upload:"` in `http/upload_cache_redis.go` | P1.1-M2 | R4 |
| 25 | Active Upload Session Safe Read-Fallback | Implement non-destructive read-fallback on `GetLength` and `Touch`, dual `Del` on `Complete` to preserve active legacy upload sessions | P1.1-M2 | R4 |
| 26 | Redis Upload Cache Unit & Regression Suite | Hermetic unit tests (`http/upload_cache_redis_test.go`) validating legacy-key retrieval, new-key creation, TTL refresh, dual deletion | P1.1-M2 | R4 |
| 27 | Root & Docs README Modernization | Remove obsolete File Browser sunset notices, introduce accurate Nyx overview, modern quickstart, Docker run, preserve attribution and Apache 2.0 license | P1.1-M3 | R1 |
| 28 | CLI Documentation Modernization | Rename all 30 `docs/cli/filebrowser-*.md` files to `docs/cli/nyx-*.md`, update headings (`# nyx ...`), command examples, `./nyx.db`, and env vars | P1.1-M3 | R1 |
| 29 | Documentation Cross-Link Integrity | Update caller links (`docs/installation.md`, `customization.md`, `troubleshooting.md`, `docs/README.md`), verify `nyx docs` generation | P1.1-M3 | R1 |
| 30 | Docker Reproducibility Investigation Report | Document investigation of `JSON.sh`, offline build analysis, licensing, and rationale for simplest maintainable solution | P1.1-M3 | R5 |
| 31 | Full Campaign Integration & Baseline Verification | Execute `go test ./...` with race detector, frontend typecheck/lint/test/build, and Docker container verification | P1.1-M4 | R6 |
| 32 | Linear Remediation Git Commits & Final Deliverables | Verify diffs, create meaningful remediation commits on `nyx_foundation_phase_one`, compile deliverables, merge readiness assessment | P1.1-M4 | R6, R7 |

---

## Milestones

| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| **P1.1-M1** | CLI Version & Environment Variable Compatibility | `cmd/root.go`, `cmd/utils.go`, `docker/*/healthcheck.sh`, `cmd/version_test.go`, `cmd/integration_test.go` | Phase 01 HEAD (`ee15cae8`) | DONE |
| **P1.1-M2** | Redis Upload Cache Namespace & In-Flight Session Compatibility | `http/upload_cache_redis.go`, `http/upload_cache_redis_test.go` | Phase 01 HEAD (`ee15cae8`) | DONE |
| **P1.1-M3** | Complete Nyx Branding, CLI Docs Suite & Docker Investigation | `README.md`, `docs/README.md`, `SECURITY.md`, `CONTRIBUTING.md`, `docs/cli/*.md`, caller docs, Docker investigation doc | Phase 01 HEAD (`ee15cae8`) | DONE |
| **P1.1-M4** | Integration Verification, Remediation Commits & Final Deliverables | Full test suites, clean Git diff inspection, remediation commits on `nyx_foundation_phase_one`, final campaign deliverables | P1.1-M1, P1.1-M2, P1.1-M3 | DONE |

---

## Interface Contracts

### 1. CLI & Viper Configuration ↔ Environment & Flags
- Primary Prefix: `v.SetEnvPrefix("NYX")` with automatic binding of flags via `AutomaticEnv()`.
- Legacy Fallback Bindings: In `cmd/utils.go`:
  ```go
  _ = v.BindEnv(f.Name, fbSnake, filebrowserSnake)
  ```
- Deterministic 6-Tier Precedence Order:
  $$\mathbf{Flags} \;\;>\;\; \mathbf{NYX\_*} \;\;>\;\; \mathbf{FB\_*} \;\;>\;\; \mathbf{FILEBROWSER\_*} \;\;>\;\; \mathbf{Config\;Files} \;\;>\;\; \mathbf{Defaults}$$
- Root Version Registration: In `cmd/root.go`:
  ```go
  rootCmd.Version = version.Version
  rootCmd.SetVersionTemplate("Nyx Browser version {{printf \"%s\" .Version}}\n")
  ```
  Both `nyx --version` / `nyx -v` (via Cobra flag) and `nyx version` (via Cobra command) report the authentic project version.

### 2. Redis Upload Cache Namespace & Compatibility
- Canonical Namespace: `"nyx:upload:" + filePath`
- Legacy Namespace: `"filebrowser:upload:" + filePath`
- Operations:
  - `Register(filePath, fileSize, cb)`: Sets `nyx:upload:<path>` with `uploadCacheTTL` (3 min).
  - `GetLength(filePath)`: Queries `nyx:upload:<path>`. If `redis.Nil`, falls back to query `filebrowser:upload:<path>`. Refreshes TTL on active key.
  - `Touch(filePath)`: Refreshes TTL on `nyx:upload:<path>`. If key does not exist, refreshes `filebrowser:upload:<path>`.
  - `Complete(filePath)`: Deletes both `nyx:upload:<path>` and `filebrowser:upload:<path>` atomically.

### 3. Documentation & Asset Identity Contract
- Documentation paths: `docs/cli/nyx-*.md` and `docs/cli/nyx.md`.
- References in documentation cite command `nyx`, database `./nyx.db`, config `.nyx.yaml`, and environment `NYX_`.
- All historical File Browser archival sunset notices removed from `README.md`, `docs/README.md`, `SECURITY.md`, and `CONTRIBUTING.md`.
- Apache 2.0 `LICENSE` and `NOTICE` file attribution preserved verbatim.

---

## Code Layout & Milestone Ownership

- **Milestone P1.1-M1 File Ownership**:
  - `cmd/root.go`
  - `cmd/utils.go`
  - `cmd/version_test.go` (new)
  - `cmd/integration_test.go`
  - `docker/alpine/healthcheck.sh`
  - `docker/common/healthcheck.sh`
- **Milestone P1.1-M2 File Ownership**:
  - `http/upload_cache_redis.go`
  - `http/upload_cache_redis_test.go` (new)
- **Milestone P1.1-M3 File Ownership**:
  - `README.md`
  - `docs/README.md`
  - `SECURITY.md`
  - `CONTRIBUTING.md`
  - `docs/cli/*.md` (all 30 renamed files)
  - `docs/installation.md`
  - `docs/customization.md`
  - `docs/troubleshooting.md`
  - `DOCKER_INVESTIGATION.md` (or equivalent deliverable documentation)
- **Milestone P1.1-M4 File Ownership**:
  - Verification reports, `POST_CAMPAIGN_REVIEW.md` updates, final remediation summary, and git commits.
