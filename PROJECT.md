# Project: Nyx Browser (NYX FOUNDATION: Phase 01)

## Architecture
Nyx Browser is an independent, self-hosted file management and media browsing platform built from a fork of File Browser.
- **Backend Architecture**: Go 1.27 (`go1.27.3-X:nodwarf5`), migrating module path to `github.com/nyxlumen/nyx-browser/v2`.
  - Entry point: `main.go` -> `cmd/root.go`
  - CLI & Config: Cobra CLI (`nyx`), Viper configuration loader with strict precedence: CLI Flags > `NYX_` env vars > `FB_` env vars > `.nyx.*` configs > `.filebrowser.*` configs > Database stored settings > Built-in defaults.
  - Storage Layer: BoltDB (`go.etcd.io/bbolt`) with Storm ORM (`github.com/asdine/storm/v3`). Buckets: `User`, `Link`, and `config` (`settings`, `server`, `auther`, `version`). Default DB path: `./nyx.db`, with non-destructive fallback to `./filebrowser.db`.
  - Scoped Filesystem: `files.ScopedFs` wrapping `afero.BasePathFs`, enforcing path traversal boundaries and symlink confinement.
  - Authentication: JWT HMAC tokens with pluggable backends (`JSONAuth`, `ProxyAuth`, `NoAuth`, `HookAuth`).
  - HTTP Routing: Gorilla Mux router with embedded frontend assets (`frontend.Assets()`).
  - Image/Media Preview: Semaphore-bounded downscaling pool (`img/service.go`) with disk caching (`diskcache/`).
- **Frontend Architecture**: Vue 3, TypeScript, Pinia, Vue Router, Dayjs, Vite 8, pnpm 10.33.4, Node 24.
  - Assets compiled into `frontend/dist/` and embedded into Go binary via `assets.go`.
  - Runtime configuration injected via Go HTML template into `window.Nyx` (with `window.FileBrowser` fallback).
  - Responsive dark-first desktop-grade interface.

---

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Baseline Go Build & Test Verification | Verify Go 1.27 toolchain, compile all 19 packages, run 59 existing unit tests, catalog 0 pre-existing baseline failures | M1 | R1 |
| 2 | Baseline Frontend Verification | Verify Node 24 (`fnm use 24`) & pnpm 10 (`10.33.4`), install 514 packages from frozen lockfile, verify lint, typecheck, vitest (29 tests), and production build | M1 | R1 |
| 3 | Baseline Documentation & Defect Catalog | Document baseline toolchain status, pre-existing warnings (Browserslist, unbundled recaptcha, large chunks), and baseline test matrices | M1 | R1 |
| 4 | Go Module Path Migration | Migrate `github.com/filebrowser/filebrowser/v2` to `github.com/nyxlumen/nyx-browser/v2` across `go.mod` and all 94 internal Go import files | M2 | R2 |
| 5 | CLI & Binary Identity Migration | Rename CLI command to `nyx`, update version template, build outputs, and Taskfile/Goreleaser | M2 | R2 |
| 6 | Database Name & Non-Destructive Fallback | Change default DB to `./nyx.db`. If `./nyx.db` is missing but legacy `./filebrowser.db` exists, open legacy DB without deletion or silent mutation | M2 | R2 |
| 7 | Config File Precedence & Dual-Env Support | Support `.nyx.{json,yaml,yml,toml}` with fallback to `.filebrowser.*`. Support `NYX_` env vars with fallback to `FB_` (`NYX_` takes precedence) | M2 | R2 |
| 8 | Frontend Identity & Metadata Update | Update `package.json`, `index.html`, `manifest.json`, `window.Nyx` namespace (with legacy `window.FileBrowser` proxy), sidebar branding, and translation strings. Remove sunset archival warning card | M2 | R2 |
| 9 | Upstream License, Copyright & Attribution | Retain Apache 2.0 `LICENSE`, preserve all copyright headers, add `NOTICE` acknowledging upstream File Browser derivation | M2 | R2 |
| 10 | Regression Test Coverage Expansion | Add automated regression tests for authentication, permissions, file CRUD operations, uploads, downloads, search, previews, config precedence, and DB compatibility | M3 | R3 |
| 11 | Filesystem Boundary & Path Traversal Hardening | Audit and add regression tests for arbitrary directory escapes, symlink traversal, case sensitivity bypasses, and scoped filesystem guards | M3 | R3 |
| 12 | Security Audit & Low-Risk Vulnerability Fixes | Fix demonstrable low-risk vulnerabilities with automated tests; document higher-risk findings with reproduction steps and remediation advice | M3 | R3 |
| 13 | Operational & Deployment Readiness Audit | Audit Dockerfile, Docker Compose, container startup/shutdown, persistent storage permissions, and improve deployment documentation | M3 | R3 |
| 14 | Backend Architecture & API Mapping | Document comprehensive backend architecture, REST API map, auth, storage, and identify reuse points for future Nyx Gallery without duplicate media DB | M4 | R4 |
| 15 | Frontend Redesign Architecture Proposal | Deliver practical frontend redesign and restructuring proposal aligned with minimal dark theme, PWA-first, and media gallery direction | M4 | R4 |
| 16 | Nyx Gallery & Media Processing Specification | Deliver specification for image/video preview handling, caching, metadata extraction, poster generation, and FFmpeg/FFprobe integration | M4 | R4 |
| 17 | Code Quality Audit & Tech Debt Backlog | Audit unused dependencies, dead code, duplicated utilities, TypeScript `any` boundaries, and produce prioritized technical debt backlog | M4 | R4 |
| 18 | End-to-End Integration & Upgrade Compatibility | Verify clean installation, upgrade from File Browser database/config, and dual `NYX_`/`FB_` configuration interoperability | M5 | R5 |
| 19 | Independent Adversarial Verification | Stress-test path traversal, configuration collisions, session handling, and legacy database integrity under adversarial conditions | M5 | R5 |
| 20 | Milestone Integrity & Final Deliverables | Validate 5 milestone Git commits on `nyx_foundation_phase_one` worktree branch and deliver final campaign report to Sentinel | M5 | R5 |

---

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Reconnaissance, Repository Audit & Reproducible Baseline | Verify Node 24 & pnpm 10 environment, frozen lockfile installation, Go build/test baseline, document 0 pre-existing Go test failures and clean baseline | Survey (Phase 0) | PLANNED |
| M2 | Complete Nyx Identity Migration | Go module migration (`github.com/nyxlumen/nyx-browser/v2`), CLI binary `nyx`, `nyx.db` with legacy fallback, `.nyx.*` configs, `NYX_` > `FB_` env vars, frontend branding, upstream license `NOTICE` | M1 | PLANNED |
| M3 | Regression Coverage, Security Hardening & Deployment Readiness | Test suite expansion (auth, permissions, CRUD, search, previews, config), path traversal hardening, security fixes, Docker/operational verification | M2 | PLANNED |
| M4 | Architecture Documentation, Future Roadmap Specs & Code Quality | Backend API mapping, Nyx Gallery reuse points, frontend redesign proposal, media pipeline specification (FFmpeg/FFprobe), technical debt backlog | M3 | PLANNED |
| M5 | Integration, Compatibility Testing, Adversarial Review & Final Verification | Full E2E integration test, legacy migration/compatibility verification, independent adversarial testing, forensic integrity audit, final deliverable synthesis | M4 | PLANNED |

---

## Interface Contracts

### 1. CLI & Viper Configuration ↔ Environment & Files
- Prefix: `v.SetEnvPrefix("NYX")` with automatic binding of flags to uppercase snake case keys (`NYX_<FLAG>`).
- Fallback: Explicit secondary bindings for legacy `FB_<FLAG>` environment variables. When both `NYX_<FLAG>` and `FB_<FLAG>` are set, `NYX_` overrides.
- Config File Lookup: Search order `./.nyx.{json,yaml,yml,toml}`, `$HOME/.nyx.*`, `/etc/nyx/.nyx.*`. If none found, fallback search `./.filebrowser.*`, `$HOME/.filebrowser.*`, `/etc/filebrowser/.filebrowser.*` with informative log.
- Database Path: Default is `./nyx.db`. If `./nyx.db` does not exist and `./filebrowser.db` exists in target directory, open `./filebrowser.db` without renaming or mutating.

### 2. Backend ↔ Frontend Asset Contract
- Asset Embedding: `frontend/dist/` embedded into Go binary via `frontend.Assets() fs.FS`.
- Runtime Context: Backend serves `index.html` template injecting JSON payload.
  - Primary global variable: `window.Nyx = [{[ .Json ]}]`
  - Backward compatibility shim: `window.FileBrowser = window.Nyx`
  - Frontend TypeScript constants read `window.Nyx.Name || window.FileBrowser?.Name || "Nyx Browser"`.

### 3. Backend Storage ↔ Database
- ORM: Storm ORM buckets (`User`, `Link`) and Key-Value bucket (`config`).
- Struct reflection names are preserved.
- Package module path changes (`github.com/nyxlumen/nyx-browser/v2`) do NOT affect serialized BoltDB bucket names or data formats.

---

## Code Layout
- Backend Source:
  - `main.go` — CLI invocation
  - `cmd/` — Cobra CLI commands and Viper configuration loader
  - `http/` — HTTP routers, middleware, API handlers
  - `storage/` & `storage/bolt/` — Storage interfaces and BoltDB / Storm implementation
  - `files/` & `fileutils/` — Scoped filesystem abstraction, traversal protection, file utilities
  - `auth/` — Authentication providers (JSON, Proxy, Hook, NoAuth)
  - `img/` & `diskcache/` — Image preview generation, caching, downscaling
  - `settings/`, `users/`, `share/`, `runner/`, `rules/`, `search/`, `errors/`, `version/`
- Frontend Source:
  - `frontend/src/` — Vue 3 components, views, router, pinia stores, utils, i18n
  - `frontend/public/` — HTML template, manifest, icons, static assets
  - `frontend/package.json`, `frontend/pnpm-lock.yaml`, `frontend/vite.config.ts`, `frontend/tsconfig*.json`
- Deployment & CI:
  - `Dockerfile`, `Dockerfile.s6`, `compose.yaml`
  - `docker/`, `.github/workflows/`, `Taskfile.yml`, `.goreleaser.yml`
- Documentation & Specifications:
  - `docs/` — Public user and operational documentation
  - Root architectural deliverables: Backend API docs, Frontend redesign proposal, Media pipeline specification, Technical debt backlog.
