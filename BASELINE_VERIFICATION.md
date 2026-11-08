# Baseline Verification Report — Nyx Browser Foundation (Phase 01)

**Date**: 2026-11-08T21:38:00Z  
**Milestone**: Milestone 1: Reconnaissance, repository audit, and reproducible development baseline  
**Worktree**: `/home/lumen/.gemini/antigravity/worktrees/Nyx-Browser/nyx_foundation_phase_one`  
**Active Branch**: `nyx_foundation_phase_one`  
**HEAD Commit**: `2a117d55` (`iniital setup trail code and project context addition`)  
**Status**: **VERIFIED & REPRODUCIBLE** — 0 Pre-existing test failures across Go backend and Frontend suites.

---

## 1. Executive Summary

This report establishes the verified, reproducible development baseline for the Nyx Browser foundation prior to executing the identity migration (Milestone 2) and security hardening (Milestone 3).

Every check in this document was independently executed and recorded in the isolated worktree environment. The findings confirm that:
1. **Toolchain Alignment**: Go 1.27 (`go1.27.3-X:nodwarf5`), Node 24 (`v24.19.0`), and pnpm 10 (`10.33.4`) are active, verified, and aligned with project specifications.
2. **Dependency Integrity**: Frontend dependencies install cleanly from the frozen lockfile (`pnpm install --frozen-lockfile`) with zero lockfile drift and zero missing packages.
3. **Backend Go Quality**: All 19 internal Go packages compile without errors. The unit test suite passes 100% with **0 failures**.
4. **Frontend Code Quality**: ESLint (`pnpm run lint`) reports **0 errors and 0 warnings**. TypeScript verification (`pnpm run typecheck`) reports **0 errors**. Vitest (`pnpm run test`) runs 5 test suites and passes **29/29 tests**. Production build (`pnpm run build`) generates all required embedded production assets in `frontend/dist/`.
5. **Defect & Warning Catalog**: All pre-existing warnings (Browserslist data age, unbundled ReCaptcha script tag in template, bundle chunk sizes, pnpm build scripts, and legacy tsconfig component exclusions) are documented in detail for tracking.

---

## 2. Toolchain & Environment Matrix

| Component | Target Version | Active / Detected Version | Verification Command | Status |
|---|---|---|---|---|
| **Operating System** | Linux x86_64 | Linux 6.6+ x86_64 | `uname -srm` | Aligned |
| **Go Toolchain** | Go 1.25+ / 1.27 | `go1.27.3-X:nodwarf5 linux/amd64` | `go version` | Aligned |
| **Node.js Runtime** | Node >=24.0.0 | `v24.19.0` (via fnm `lts-latest`) | `node -v` | Aligned |
| **Package Manager** | pnpm >=10.0.0 | `10.33.4` (pinned in `package.json`) | `pnpm -v` | Aligned |
| **Bundler / Build** | Vite 8.x | `vite v8.0.13` | `pnpm run build` | Aligned |
| **Frontend Test** | Vitest 4.x | `vitest v4.1.6` | `pnpm run test` | Aligned |
| **TypeScript** | Vue-TSC 2.x | `vue-tsc v2.2.8` | `pnpm run typecheck` | Aligned |
| **Linter** | ESLint 9.x | `eslint v9.39.4` | `pnpm run lint` | Aligned |

### Environment Setup Command
```bash
# Activate Node 24 and ensure pnpm 10
eval "$(fnm env)" && fnm use 24
```

---

## 3. Go Backend Baseline Verification

### 3.1 Compilation Baseline (`go build -v ./...`)
- **Command**: `go build -v ./...`
- **Exit Code**: `0`
- **Packages Compiled**: 19 internal packages
- **Compiler Warnings / Errors**: `0`

Compiled packages:
1. `github.com/filebrowser/filebrowser/v2` (root / `main.go`)
2. `github.com/filebrowser/filebrowser/v2/auth`
3. `github.com/filebrowser/filebrowser/v2/cmd`
4. `github.com/filebrowser/filebrowser/v2/diskcache`
5. `github.com/filebrowser/filebrowser/v2/errors`
6. `github.com/filebrowser/filebrowser/v2/files`
7. `github.com/filebrowser/filebrowser/v2/fileutils`
8. `github.com/filebrowser/filebrowser/v2/frontend`
9. `github.com/filebrowser/filebrowser/v2/http`
10. `github.com/filebrowser/filebrowser/v2/img`
11. `github.com/filebrowser/filebrowser/v2/rules`
12. `github.com/filebrowser/filebrowser/v2/runner`
13. `github.com/filebrowser/filebrowser/v2/search`
14. `github.com/filebrowser/filebrowser/v2/settings`
15. `github.com/filebrowser/filebrowser/v2/share`
16. `github.com/filebrowser/filebrowser/v2/storage`
17. `github.com/filebrowser/filebrowser/v2/storage/bolt`
18. `github.com/filebrowser/filebrowser/v2/users`
19. `github.com/filebrowser/filebrowser/v2/version`

### 3.2 Test Suite Execution (`go test -count=1 ./...`)
- **Command**: `go test -count=1 ./...`
- **Exit Code**: `0`
- **Result**: All test suites pass. **0 failures**.

#### Package Test Breakdown & Coverage Matrix
| Package | Test Functions | Top-Level Tests | Subtests / Examples | Statement Coverage | Test Duration | Result |
|---|---|---|---|---|---|---|
| `auth` | 6 | 6 | 0 | 41.7% | 0.141s | PASS |
| `cmd` | 2 | 2 | 0 | 16.8% | 0.045s | PASS |
| `diskcache` | 1 | 1 | 0 | 75.5% | 0.008s | PASS |
| `files` | 9 | 9 | 0 | 34.2% | 0.019s | PASS |
| `fileutils` | 4 | 4 | 0 | 72.8% | 0.017s | PASS |
| `http` | 33 | 33 | 40+ subtests | 40.4% | 0.305s | PASS |
| `img` | 2 | 2 | 0 | 51.8% | 0.173s | PASS |
| `rules` | 3 | 3 | 0 | 100.0% | 0.007s | PASS |
| `runner` | 3 | 3 | 1 example | 41.4% | 0.043s | PASS |
| `settings` | 2 | 2 | 0 | 22.6% | 0.041s | PASS |
| `storage/bolt` | 4 | 4 | 0 | 25.8% | 0.043s | PASS |
| `users` | 3 | 3 | 0 | 34.0% | 0.153s | PASS |
| `errors` | 0 | 0 | 0 | 0.0% (no test files) | - | - |
| `frontend` | 0 | 0 | 0 | 0.0% (no test files) | - | - |
| `search` | 0 | 0 | 0 | 0.0% (no test files) | - | - |
| `share` | 0 | 0 | 0 | 0.0% (no test files) | - | - |
| `storage` | 0 | 0 | 0 | 0.0% (no test files) | - | - |
| `version` | 0 | 0 | 0 | 0.0% (no test files) | - | - |
| `.` (root) | 0 | 0 | 0 | 0.0% (no test files) | - | - |
| **Total** | **72 tests + 1 example** | **72** | **40+** | **Overall: 39.8%** | **~0.85s** | **100% PASS** |

> **Audit Note on Initial Survey Count (59 vs 72)**:  
> The Explorer 1 reconnaissance survey initially summarized the test count as "59 test functions". Forensic verification of `go test -list '.*' ./...` reveals that the survey's text accidentally noted `http: 23 test functions` while in fact pasting all 33 test function names (33 - 10 = 23). The actual verified baseline consists of **72 top-level test functions plus 1 runnable example** (`ExampleSplitCommandAndArgs`), totaling 73 test entries across 11 packages, with 0 failures.

---

## 4. Frontend Baseline Verification

### 4.1 Frozen Lockfile Dependency Installation
- **Command**: `cd frontend && pnpm install --frozen-lockfile`
- **Execution Time**: `506ms`
- **Exit Code**: `0`
- **Packages Installed**: 514 packages from local cache
- **Lockfile Drift**: `0` changes (`pnpm-lock.yaml` is fully synchronized with `package.json`)
- **Pnpm Warning**:
  ```
  Ignored build scripts: core-js@3.48.0, core-js@3.49.0, es5-ext@0.10.64, esbuild@0.25.12, esbuild@0.27.3.
  Run "pnpm approve-builds" to pick which dependencies should be allowed to run scripts.
  ```
  *(Standard pnpm 10 default security posture; does not impact build or runtime).*

### 4.2 Linter Baseline (`pnpm run lint`)
- **Command**: `cd frontend && pnpm run lint` (`eslint src/`)
- **Exit Code**: `0`
- **Output**:
  ```
  > filebrowser-frontend@3.0.0 lint /home/lumen/.../frontend
  > eslint src/
  ```
- **Violations**: **0 errors, 0 warnings**. Clean.

### 4.3 Typecheck Baseline (`pnpm run typecheck`)
- **Command**: `cd frontend && pnpm run typecheck` (`vue-tsc -p ./tsconfig.app.json --noEmit`)
- **Exit Code**: `0`
- **Output**:
  ```
  > filebrowser-frontend@3.0.0 typecheck /home/lumen/.../frontend
  > vue-tsc -p ./tsconfig.app.json --noEmit
  ```
- **Violations**: **0 TypeScript errors**.

### 4.4 Unit Test Baseline (`pnpm run test`)
- **Command**: `cd frontend && pnpm run test` (`vitest run`)
- **Exit Code**: `0`
- **Test Runner**: Vitest v4.1.6
- **Execution Time**: `2.85s`
- **Test Suites**: 5 passed (5 total)
- **Unit Tests**: 29 passed (29 total, **0 failures**)

Detailed test suite results:
| Suite File | Tests | Duration | Status |
|---|---|---|---|
| `src/css/__tests__/mobile.test.ts` | 1 | 4ms | PASS |
| `src/utils/__tests__/upload.test.ts` | 2 | 8ms | PASS |
| `src/utils/__tests__/check-conflict.test.ts` | 19 | 9ms | PASS |
| `src/views/files/__tests__/upload-conflict-resolution.test.ts` | 5 | 19ms | PASS |
| `src/components/prompts/__tests__/copy-move-conflict.test.ts` | 2 | 7ms | PASS |
| **Total** | **29 tests** | **2.85s** | **100% PASS** |

### 4.5 Production Build Baseline (`pnpm run build`)
- **Command**: `cd frontend && pnpm run build` (`vue-tsc -p ./tsconfig.app.json --noEmit && vite build`)
- **Exit Code**: `0`
- **Build Duration**: `30.64s`
- **Modules Transformed**: `1079 modules`
- **Output Directory**: `frontend/dist/`
- **Embed Verification**: Generates `index.html`, `manifest.json`, and all `.js`, `.css`, `.woff2`, `.gz`, and `.br` assets required by `frontend/assets.go` (`//go:embed dist/*`).

Key compiled chunk sizes:
- `dist/assets/Editor-DU8e4lG3.js`: 401.01 kB (gzip: 121.88 kB)
- `dist/assets/index-BqkXVoMb.js`: 792.10 kB (gzip: 225.17 kB)
- `dist/assets/i18n-C0UlxgXD.js`: 828.86 kB (gzip: 194.83 kB)
- `dist/assets/Preview-CaXSHOvx.js`: 1,257.51 kB (gzip: 377.44 kB)
- `dist/assets/Preview-D3v_iDY5.css`: 58.23 kB (gzip: 14.67 kB)
- `dist/assets/index-BOEsmRAc.css`: 66.93 kB (gzip: 18.69 kB)

---

## 5. Pre-Existing Defect, Warning & Technical Debt Catalog

This catalog documents all warnings and architectural limitations inherited from upstream. None of these represent regressions introduced in Phase 01.

| # | Category | Location | Description & Warning Text | Impact & Recommendation |
|---|---|---|---|---|
| **W1** | Tooling / Data Age | Frontend Vite / Vitest build | `Browserslist: browsers data (caniuse-lite) is 7 months old. Please run: npx update-browserslist-db@latest` | Non-blocking. In Phase 02 / maintenance, run database update or pin in lockfile. |
| **W2** | Template Asset Packaging | `frontend/public/index.html:43` | `<script src="[{[ .ReCaptchaHost ]}]/recaptcha/api.js?render=explicit"> in "/public/index.html" can't be bundled without type="module" attribute` | Vite warning regarding external Go template tag in HTML. Expected behavior for dynamic backend template injection. |
| **W3** | Template Runtime CSS | `frontend/public/index.html:36` | `[{[ .StaticURL ]}]/custom.css doesn't exist at build time, it will remain unchanged to be resolved at runtime` | Expected behavior; `custom.css` is an optional user-provided file served at runtime by Go backend. |
| **W4** | Bundle Chunk Size | `frontend/vite.config.ts` | `(!) Some chunks are larger than 500 kB after minification.` (`Preview`, `i18n`, `index`) | Large chunks due to bundled Monaco/Ace editors, pdf.js, and KaTeX. Addressed in Milestone 4 frontend restructuring proposal. |
| **W5** | Build Plugin Timing | `unplugin-vue-i18n:resource` | `[PLUGIN_TIMINGS] Your build spent significant time in plugin unplugin-vue-i18n:resource.` | 33 translation JSON files are parsed at build time. Normal for full multi-language compilation. |
| **W6** | Options API Type Exclusion | `frontend/tsconfig.app.json:9-16` | Explicit exclusion of 7 legacy Vue components: `Shell.vue`, `Copy.vue`, `Move.vue`, `Delete.vue`, `FileList.vue`, `Rename.vue`, `Share.vue`. | These legacy components use Options API and fail strict TypeScript checking. Targeted for Composition API rewrite in Phase 02. |
| **W7** | Relaxed Index Access | `frontend/tsconfig.app.json:27` | `"noUncheckedIndexedAccess": false` | Relaxed array/object index checking. Should be enabled during future TypeScript hardening. |
| **W8** | Backend Test Gaps | Go packages: `search/`, `share/`, `storage/`, `errors/`, `frontend/`, `version/`, `.` | 8 internal Go packages lack test files (0% coverage). `search/` is a key user-facing feature with 0% coverage. | Targeted for test suite expansion in Milestone 3. |
| **W9** | Upstream Archival Notice | `frontend/src/views/Settings.vue:37-53`, `cmd/root.go:195-199`, `frontend/src/i18n/*.json` | Upstream deprecation notice warning users that File Browser was being archived on 2026-09-01. | To be removed/replaced with active Nyx Browser Foundation notices during Milestone 2 identity migration. |

---

## 6. Reproducible Build & Verification Instructions

Follow these exact steps to reproduce the baseline in a clean environment:

### Prerequisites
- Linux x86_64
- Go 1.25 or 1.27 (`go version`)
- fnm or nvm with Node 24 (`v24.19.0`)
- pnpm 10 (`10.33.4`)

### Step 1: Environment Alignment
```bash
# Enter project root
cd /home/lumen/.gemini/antigravity/worktrees/Nyx-Browser/nyx_foundation_phase_one

# Activate Node 24
eval "$(fnm env)" && fnm use 24

# Verify tool versions
go version
# Output: go version go1.27.3-X:nodwarf5 linux/amd64

node -v
# Output: v24.19.0

cd frontend && pnpm -v && cd ..
# Output: 10.33.4
```

### Step 2: Go Backend Build & Tests
```bash
# Compile all 19 Go packages
go build -v ./...

# Run the full Go test suite
go test -count=1 ./...

# Run with statement coverage
go test -cover ./...
```

### Step 3: Frontend Installation, Quality Checks & Build
```bash
cd frontend

# Clean frozen installation
pnpm install --frozen-lockfile

# ESLint check
pnpm run lint

# TypeScript verification
pnpm run typecheck

# Vitest unit test suite
pnpm run test

# Production asset compilation
pnpm run build
```

### Step 4: Standalone Binary Execution
```bash
cd /home/lumen/.gemini/antigravity/worktrees/Nyx-Browser/nyx_foundation_phase_one

# Compile standalone executable
go build -o nyx-baseline .

# Verify help output and options
./nyx-baseline --help
```

---

## 7. Milestone 1 Sign-Off & Next Steps

The development baseline is verified, stable, and completely reproducible. All tests pass with **0 baseline failures**. The repository is in an ideal state to begin Milestone 2: Complete Nyx Identity Migration.

### Transition to Milestone 2:
- **Go Module Path**: Rename `github.com/filebrowser/filebrowser/v2` -> `github.com/nyxlumen/nyx-browser/v2` across `go.mod` and all 94 internal Go import files.
- **CLI & Binary**: Rename to `nyx`.
- **Database & Config**: Default to `./nyx.db` and `.nyx.*` configs with non-destructive fallback to `./filebrowser.db` and `.filebrowser.*`. Support `NYX_` env vars with fallback to `FB_`.
- **Frontend Identity**: Rebrand metadata, titles, manifest, `window.Nyx` namespace (with `window.FileBrowser` proxy), and remove the obsolete sunset notice card.
- **Attribution**: Retain Apache 2.0 `LICENSE` and add top-level `NOTICE` file.
