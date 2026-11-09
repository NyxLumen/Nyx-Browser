# Nyx Browser Technical Debt Audit & Prioritized Backlog

> **Document Version**: 2.0.0  
> **Repository**: `github.com/nyxlumen/nyx-browser/v2`  
> **Target Subsystems**: Backend Go Toolchain, Storage, HTTP Middleware, Error Handling, Frontend TypeScript, State Stores, Styling Pipeline  
> **Status**: Authoritative Technical Debt Reference & Remediation Plan  
> **Branch**: `nyx_foundation_phase_one`  

---

## 1. Executive Summary

A comprehensive architectural and forensic code audit of the Nyx Browser repository was performed across the backend (`http/`, `storage/`, `files/`, `img/`, `diskcache/`, `errors/`, `users/`, `cmd/`) and the frontend (`frontend/src/`, `frontend/package.json`, `frontend/tsconfig*.json`).

While Nyx Browser's core architecture—specifically `files.ScopedFs` filesystem confinement, JWT session handling, and BoltDB transactional reliability—provides an exceptionally solid baseline, legacy code inherited from upstream File Browser carries accumulated technical debt:
1. **Critical Backend Crash Hazards & Memory Leaks**:
   - `http/data.go:95` calls `log.Fatalf` inside request serving paths, terminating the entire server process upon transient database I/O contention.
   - `diskcache/file_cache.go:98` permanently accumulates mutexes in a memory map that is never cleared.
   - `diskcache/file_cache.go:51` (`Load`) omits lock acquisition, exposing callers to data races during concurrent cache writes.
   - HTTP error mapper (`http/utils.go:77`) maps only 6 errors, dropping 12 validation errors into `500 Internal Server Error`.
2. **Abandoned & Unmaintained Dependencies**:
   - `github.com/asdine/storm/v3` (abandoned since 2020), `github.com/flynn/go-shlex` (abandoned since 2015), `github.com/tomasen/realip` (abandoned since 2018).
3. **Frontend Type Safety Gaps**:
   - 7 core Vue components are explicitly excluded from TypeScript compilation in `frontend/tsconfig.app.json`.
   - Over 45 occurrences of unconstrained `: any` and `as any` compromise type safety across file CRUD operations.
   - `"noUncheckedIndexedAccess"` is disabled to suppress compiler warnings.
4. **Duplicate Utilities & Branding Gaps**:
   - Redundant byte sizing libraries (`filesize` vs `pretty-bytes`), multiple overlapping YAML parsers, legacy Redis key prefix (`filebrowser:upload:`), and obsolete route aliases (`monkey`).

This document provides the exhaustive inventory of all findings and establishes a prioritized 17-item remediation backlog with exact file:line citations, risk assessments, and story-point estimates.

---

## 2. Backend Technical Debt Audit

### 2.1 Dependency Inventory & Upstream Health (`go.mod`)

Inspection of `go.mod` reveals several stale, unmaintained, or redundant dependencies:

```
+──────────────────────────────────┬──────────────┬──────────────┬────────────────────────────────────────────────────────+
| Dependency Path                  | Version      | Status       | Architectural Risk & Code Impact                       |
+──────────────────────────────────┼──────────────┼──────────────┼────────────────────────────────────────────────────────+
| github.com/asdine/storm/v3       | v3.2.1       | Abandoned    | Abandoned ORM layer over BoltDB. Uses heavy reflection  |
|                                  |              | (No commits  | and creates internal index buckets preventing clean     |
|                                  |              | since 2020)  | zero-downtime schema evolution.                         |
+──────────────────────────────────┼──────────────┼──────────────┼────────────────────────────────────────────────────────+
| github.com/flynn/go-shlex        | v0.0.0-2015  | Abandoned    | Shell lexer abandoned since May 2015. Used in runner.  |
+──────────────────────────────────┼──────────────┼──────────────┼────────────────────────────────────────────────────────+
| github.com/tomasen/realip        | v0.0.0-2018  | Abandoned    | Blindly trusts X-Forwarded-For headers without proxy    |
|                                  |              |              | CIDR verification. Enables remote client IP spoofing.   |
+──────────────────────────────────┼──────────────┼──────────────┼────────────────────────────────────────────────────────+
| github.com/disintegration/imaging| v1.6.2       | Inactive     | Pure Go image downscaler lacking SIMD acceleration and  |
|                                  |              |              | modern format support (WebP, AVIF, HEIC).               |
+──────────────────────────────────┼──────────────┼──────────────┼────────────────────────────────────────────────────────+
| golang.org/x/image               | v0.44.0      | Unused in    | Declared as direct dependency in go.mod:30, but only    |
|                                  |              | Production   | imported in test code (bmp/tiff).                       |
+──────────────────────────────────┼──────────────┼──────────────┼────────────────────────────────────────────────────────+
| Overlapping YAML parsers         | v2, v3, v3.04| Duplicated   | go.mod pulls in 3 distinct YAML packages:               |
|                                  |              |              | gopkg.in/yaml.v3, gopkg.in/yaml.v2, go.yaml.in/yaml/v3.|
+──────────────────────────────────┴──────────────┴──────────────┴────────────────────────────────────────────────────────+
```

### 2.2 Reliability & Crash Hazards in HTTP Layer

1. **Process Crash on Transient Settings Failure (`http/data.go:93-97`)**:
   ```go
   settings, err := store.Settings.Get()
   if err != nil {
       log.Fatalf("ERROR: couldn't get settings: %v\n", err)
       return
   }
   ```
   `log.Fatalf` executes `os.Exit(1)` inside the request-handling path. Any transient disk I/O contention or read lock on `nyx.db` immediately terminates the entire Nyx Browser server process.
2. **Absence of Panic Recovery Middleware**:
   Neither `http/http.go` nor `http/data.go` install a `recover()` middleware. An unhandled panic in an image decoder, template renderer, or third-party library unwinds the goroutine and crashes the HTTP server.
3. **Inconsistent Error Status Mapping (`http/utils.go:77-98`)**:
   `errors/errors.go` defines 18 domain errors. However, `http/utils.go:77` (`errToStatus`) maps only 6 errors:
   ```go
   func errToStatus(err error) int {
       switch {
       case err == nil:
           return http.StatusOK
       case os.IsPermission(err):
           return http.StatusForbidden
       case os.IsNotExist(err), errors.Is(err, libErrors.ErrNotExist):
           return http.StatusNotFound
       case os.IsExist(err), errors.Is(err, libErrors.ErrExist):
           return http.StatusConflict
       case errors.Is(err, libErrors.ErrPermissionDenied):
           return http.StatusForbidden
       case errors.Is(err, libErrors.ErrInvalidRequestParams):
           return http.StatusBadRequest
       case errors.Is(err, libErrors.ErrRootUserDeletion):
           return http.StatusForbidden
       case errors.Is(err, imgErrors.ErrImageTooLarge):
           return http.StatusRequestEntityTooLarge
       default:
           return http.StatusInternalServerError // Drops 12 domain errors into 500!
       }
   }
   ```
   As a consequence, standard user validation errors (e.g., `ErrEasyPassword`, `ErrEmptyPassword`, `ErrInvalidAuthMethod`) drop into `default` and return `500 Internal Server Error` instead of `400 Bad Request`.
4. **Unstructured Plain Text Error Responses (`http/data.go:116`)**:
   Errors are formatted as plain text strings (`http.Error(w, strconv.Itoa(status)+" "+txt, status)`), preventing frontend clients from parsing structured error codes and field validations.

### 2.3 Disk Cache Concurrency & Memory Defects (`diskcache/file_cache.go`)

1. **Unbounded Memory Leak in Fine-Grained Locks (`diskcache/file_cache.go:90-103`)**:
   ```go
   func (f *FileCache) getScopedLocks(key string) (lock sync.Locker) {
       f.scopedLocks.Do(func() { f.scopedLocks.locks = map[string]sync.Locker{} })
       f.scopedLocks.Lock()
       lock, ok := f.scopedLocks.locks[key]
       if !ok {
           lock = &sync.Mutex{}
           f.scopedLocks.locks[key] = lock
       }
       f.scopedLocks.Unlock()
       return lock
   }
   ```
   Every unique thumbnail request creates an entry in `f.scopedLocks.locks`. **This map is never pruned or cleared.** On servers hosting tens of thousands of media files, this map permanently leaks heap memory.
2. **Missing Concurrency Lock in `Load()` (`diskcache/file_cache.go:51-63`)**:
   `Store()` and `Delete()` acquire fine-grained locks, but `Load()` does not. Concurrent readers accessing a thumbnail while it is being written experience read-after-write races and file truncation errors.
3. **Executable Bit Set on Cached Images (`diskcache/file_cache.go:44`)**:
   Cached thumbnail files are written with Unix permission `0700` (`-rwx------`), marking JPEG images as executable binaries on the filesystem. Standard file mode should be `0644` or `0600`.
4. **Omission of Thumbnail Invalidation on PUT and TUS Uploads**:
   - `http/http.go:67`: `resourcePutHandler` is not passed `fileCache` and never calls `delThumbs`.
   - `http/tus_handlers.go`: Chunked uploads overwriting existing media never trigger cache invalidation.
   - `http/resource.go:116`: Deleting a directory invokes `delThumbs` only on the directory entry itself, leaving all descendant thumbnails orphaned in `diskcache/`.

### 2.4 Code Smells & Legacy Artifacts

1. **Obsolete Route Helper Alias (`http/http.go:41-43`)**:
   ```go
   monkey := func(fn handleFunc, prefix string) http.Handler {
       return handle(fn, prefix, store, server)
   }
   ```
   The alias `monkey` is used in 32 route definitions across `http.go`. This is an obsolete naming artifact from upstream File Browser.
2. **Legacy Redis Key Namespace (`http/upload_cache_redis.go:39`)**:
   ```go
   func (c *redisUploadCache) filePathKey(filePath string) string {
       return "filebrowser:upload:" + filePath
   }
   ```
   The Redis key namespace hardcodes legacy `filebrowser:upload:` instead of `nyx:upload:`.

---

## 3. Frontend Technical Debt Audit

### 3.1 Type Safety Exclusions (`frontend/tsconfig.app.json`)

To bypass compiler errors during the initial Vue 3 migration, 7 core components were explicitly excluded from TypeScript type checking in `frontend/tsconfig.app.json:9-15`:
```json
"exclude": [
  "src/**/__tests__/*",
  "src/components/Shell.vue",
  "src/components/prompts/Copy.vue",
  "src/components/prompts/Move.vue",
  "src/components/prompts/Delete.vue",
  "src/components/prompts/FileList.vue",
  "src/components/prompts/Rename.vue",
  "src/components/prompts/Share.vue"
]
```
These components operate with zero compiler validation, creating severe runtime regression risks whenever store signatures or API models change.

### 3.2 Disabled Strict Index Checking

`frontend/tsconfig.app.json:27` explicitly disables unchecked index access:
```json
"noUncheckedIndexedAccess": false
```
Enabling this standard TypeScript strict flag reveals dozens of potential `undefined` indexing bugs across array lookups and dictionary accesses in `frontend/src/`.

### 3.3 TypeScript `any` Boundaries

A codebase scan reveals over 45 instances of `: any` and `as any` across `frontend/src/`:
- **Global Runtime Shims (`frontend/src/types/global.d.ts:5-12`)**:
  `window.Nyx: any`, `window.FileBrowser: any`, `window.grecaptcha: any`, `window.__vue__: any`.
- **File API Contracts (`frontend/src/api/files.ts`)**:
  - Line 37: `data.items = data.items.map((item: any, index: any) => ...)`
  - Line 58: `resourceAction(url: string, method: ApiMethod, content?: any)`
  - Line 82: `download(format: any, ...files: string[])`
  - Lines 175, 198, 202: `items: any[]` in move and copy handlers.
- **Settings & Layout Types (`frontend/src/types/settings.d.ts`, `frontend/src/types/layout.d.ts`)**:
  - `rules: any[]`, `commands: any[]`, `confirm?: any`, `props?: any`.
- **Video Player (`frontend/src/components/files/VideoPlayer.vue:36,93`)**:
  - `options?: any`, `srcOpt: any[]`.

### 3.4 Duplicate Byte Sizing Libraries

`frontend/package.json` includes two separate byte-formatting libraries:
- `"filesize": "^11.0.13"` (`package.json:30`)
- `"pretty-bytes": "^7.1.0"` (`package.json:39`)

`frontend/src/components/Sidebar.vue:137,183` is the sole importer of `pretty-bytes`, while the rest of the application uses `filesize` via `frontend/src/utils/index.ts:6`.

### 3.5 Fragile Path String Manipulation (`frontend/src/api/utils.ts:76-82`)

```typescript
export function removePrefix(url: string): string {
  url = url.split("/").splice(2).join("/");
  if (url === "") url = "/";
  if (url[0] !== "/") url = "/" + url;
  return url;
}
```
`removePrefix` assumes all application URLs begin with `/<route>/` and unconditionally slices the first two segments. If a path contains a different segment structure, this function silently corrupts the resource URL.

---

## 4. Prioritized Technical Debt Backlog

The technical debt backlog is categorized by severity (High, Medium, Low Priority) and estimated in Story Points (SP) and developer days (d).

```
+─────────────────────────────────────────────────────────────────────────────────────────────+
|                               TECHNICAL DEBT BACKLOG MATRIX                                 |
+──────────┬───────┬──────────────────┬─────────────────┬───────────────────┬────────┬────────+
| Priority | ID    | Category         | Subsystem       | Title             | Impact | Est.   |
+──────────┼───────┼──────────────────┼─────────────────┼───────────────────┼────────┼────────+
| HIGH     | TD-01 | Memory Leak      | Backend Cache   | Unbounded Mutexes | OOM    | 2 SP   |
| HIGH     | TD-02 | Reliability      | Backend Core    | Fatal Settings Exit| Crash | 1 SP   |
| HIGH     | TD-03 | Concurrency      | Backend Cache   | Cache Read Race   | Corrupt| 1 SP   |
| HIGH     | TD-04 | Invalidation     | Backend HTTP    | Missing Cache Purge| Leak  | 3 SP   |
| HIGH     | TD-05 | Error Handling   | Backend HTTP    | 500 on Validation | UX/API | 2 SP   |
| HIGH     | TD-06 | Security         | Backend Auth    | RealIP Spoofing   | Audit  | 2 SP   |
+──────────┼───────┼──────────────────┼─────────────────┼───────────────────┼────────┼────────+
| MEDIUM   | TD-07 | Dependency       | Backend Storage | Abandoned Storm   | Maint. | 5 SP   |
| MEDIUM   | TD-08 | Type Safety      | Frontend Core   | Excluded Vue Files| Bug    | 5 SP   |
| MEDIUM   | TD-09 | Type Safety      | Frontend API    | TypeScript any    | Maint. | 3 SP   |
| MEDIUM   | TD-10 | Duplication      | Frontend Utils  | pretty-bytes Redun| Bloat  | 1 SP   |
| MEDIUM   | TD-11 | Performance      | Backend Media   | 400MB Bitmap Spikes| OOM    | 3 SP   |
| MEDIUM   | TD-12 | Type Safety      | Frontend Config | NoUncheckedIndexed | Bug    | 3 SP   |
+──────────┼───────┼──────────────────┼─────────────────┼───────────────────┼────────┼────────+
| LOW      | TD-13 | Code Quality     | Backend HTTP    | Obsolete monkey   | Read.  | 1 SP   |
| LOW      | TD-14 | Identity         | Backend Cache   | Redis FB Namespace| Brand  | 1 SP   |
| LOW      | TD-15 | Filesystem       | Backend Cache   | Mode 0700 on Cache| Perms  | 1 SP   |
| LOW      | TD-16 | Dependency       | Backend Core    | Unused x/image    | Clean  | 1 SP   |
| LOW      | TD-17 | Dependency       | Backend Core    | Overlapping YAML  | Bloat  | 1 SP   |
+──────────┴───────┴──────────────────┴─────────────────┴───────────────────┴────────┴────────+
```

---

## 5. Itemized Remediation Specifications

### High Priority Backlog

#### TD-01: Fix Unbounded Memory Leak in Disk Cache Mutex Map
- **Location**: `diskcache/file_cache.go:24,90-103`
- **Description**: Replace the unbounded dynamic map `f.scopedLocks.locks = map[string]sync.Locker{}` with a fixed striped mutex pool:
  ```go
  type FileCache struct {
      fs     afero.Fs
      stripes [256]sync.RWMutex
  }
  func (f *FileCache) getLock(key string) *sync.RWMutex {
      h := fnv.New32a()
      _, _ = h.Write([]byte(key))
      return &f.stripes[h.Sum32()%256]
  }
  ```
- **Risk / Impact**: Eliminates permanent heap allocation leak on busy servers.
- **Remediation Effort**: 2 SP (1 developer day).

#### TD-02: Eliminate `log.Fatalf` in HTTP Settings Loader
- **Location**: `http/data.go:93-97`
- **Description**: Replace `log.Fatalf` with structured error handling returning HTTP `500 Internal Server Error` without exiting the process:
  ```go
  settings, err := store.Settings.Get()
  if err != nil {
      log.Printf("ERROR: couldn't get settings: %v\n", err)
      http.Error(w, "Internal Server Error", http.StatusInternalServerError)
      return
  }
  ```
- **Risk / Impact**: Prevents server crashes caused by transient disk locks or database timeouts.
- **Remediation Effort**: 1 SP (0.5 developer days).

#### TD-03: Eliminate Data Race in `FileCache.Load()`
- **Location**: `diskcache/file_cache.go:51-63`
- **Description**: Acquire read lock from the striped mutex pool in `Load()` to serialize readers against concurrent `Store()` and `Delete()` operations.
- **Risk / Impact**: Eliminates corrupted thumbnail reads and truncated image payloads.
- **Remediation Effort**: 1 SP (0.5 developer days).

#### TD-04: Implement Comprehensive Thumbnail Invalidation
- **Location**: `http/http.go:67`, `http/resource.go:116-124,190-220`, `http/tus_handlers.go:179-294`
- **Description**:
  1. Pass `fileCache` to `resourcePutHandler` and invoke `delThumbs()` on file overwrite.
  2. Call `delThumbs()` upon successful completion of TUS chunked uploads.
  3. On recursive directory deletion, query cache index for prefix `file.RealPath() + "/"` and purge all descendant thumbnail files.
- **Risk / Impact**: Prevents stale thumbnail display and eliminates permanent disk storage leakage.
- **Remediation Effort**: 3 SP (1.5 developer days).

#### TD-05: Map All Domain Errors to Meaningful HTTP Status Codes
- **Location**: `http/utils.go:77-98`
- **Description**: Expand `errToStatus()` to map all 18 domain errors defined in `errors/errors.go`:
  - `ErrEmptyPassword`, `ErrEasyPassword`, `ErrEmptyUsername`, `ErrEmptyRequest`, `ErrInvalidDataType`, `ErrInvalidOption`: Map to `http.StatusBadRequest` (400).
  - `ErrInvalidAuthMethod`: Map to `http.StatusForbidden` (403).
  - `ErrSourceIsParent`, `ErrScopeIsRelative`: Map to `http.StatusBadRequest` (400).
- **Risk / Impact**: Prevents client confusion caused by validation errors returning HTTP 500.
- **Remediation Effort**: 2 SP (1 developer day).

#### TD-06: Replace Abandoned `tomasen/realip` with Trusted Proxy CIDR Validator
- **Location**: `http/data.go:107`
- **Description**: Replace unmaintained `realip` package with an internal trusted proxy evaluator that checks client remote IP against configured trusted CIDRs (e.g. `127.0.0.1/8`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) before trusting `X-Forwarded-For` or `X-Real-IP`.
- **Risk / Impact**: Protects audit logging and IP-based access rules from spoofing.
- **Remediation Effort**: 2 SP (1 developer day).

---

### Medium Priority Backlog

#### TD-07: Plan Migration from Abandoned `asdine/storm` to Native `bbolt`
- **Location**: `go.mod:6`, `storage/bolt/`
- **Description**: Design and implement a typed serialization layer directly on `bbolt` using protobuf or JSON, eliminating the unmaintained Storm ORM reflection layer while preserving existing bucket binary layouts.
- **Risk / Impact**: Removes stagnant dependency and enables transparent index management.
- **Remediation Effort**: 5 SP (3 developer days).

#### TD-08: Eliminate TypeScript Exclusions for Vue Components
- **Location**: `frontend/tsconfig.app.json:9-15`
- **Description**: Refactor `Shell.vue` and the 6 prompt components (`Copy.vue`, `Move.vue`, `Delete.vue`, `FileList.vue`, `Rename.vue`, `Share.vue`) from Options API to `<script setup lang="ts">`, removing them from the tsconfig exclusion list.
- **Risk / Impact**: Restores 100% compiler coverage across the frontend.
- **Remediation Effort**: 5 SP (3 developer days).

#### TD-09: Eliminate TypeScript `any` Boundaries Across Frontend API
- **Location**: `frontend/src/api/files.ts`, `frontend/src/types/global.d.ts`
- **Description**: Replace all `: any` and `as any` annotations with typed interfaces (`ResourceItem`, `ApiMethod`, `MoveCopyPayload`, `DownloadFormat`).
- **Risk / Impact**: Prevents silent runtime type mismatches.
- **Remediation Effort**: 3 SP (1.5 developer days).

#### TD-10: Consolidate Byte Formatting Utilities
- **Location**: `frontend/package.json:39`, `frontend/src/components/Sidebar.vue:137,183`
- **Description**: Remove `pretty-bytes` dependency. Migrate `Sidebar.vue` to use `filesize` from `frontend/src/utils/index.ts`.
- **Risk / Impact**: Reduces frontend bundle size and ensures uniform byte unit display.
- **Remediation Effort**: 1 SP (0.5 developer days).

#### TD-11: Implement Memory-Bounded Streaming Image Decoders
- **Location**: `img/service.go:178`
- **Description**: Replace `imaging.Decode` with streaming decoders that downscale during IDCT decoding, enforcing a strict $128\text{ MB}$ maximum memory budget per slot.
- **Risk / Impact**: Prevents OOM crashes on small VPS instances.
- **Remediation Effort**: 3 SP (1.5 developer days).

#### TD-12: Enable `"noUncheckedIndexedAccess"` in TypeScript Config
- **Location**: `frontend/tsconfig.app.json:27`
- **Description**: Enable `"noUncheckedIndexedAccess": true` and resolve all compiler errors with proper guard checks.
- **Risk / Impact**: Prevents runtime `undefined` property access errors.
- **Remediation Effort**: 3 SP (2 developer days).

---

### Low Priority Backlog

#### TD-13: Refactor Obsolete `monkey` Closure Alias
- **Location**: `http/http.go:41-43`
- **Description**: Rename `monkey` to `wrapHandler` or inline `handle(fn, prefix, store, server)`.
- **Remediation Effort**: 1 SP (0.5 developer days).

#### TD-14: Update Legacy Redis Cache Key Prefix
- **Location**: `http/upload_cache_redis.go:39`
- **Description**: Update key prefix from `filebrowser:upload:` to `nyx:upload:`.
- **Remediation Effort**: 1 SP (0.25 developer days).

#### TD-15: Correct File Permissions in Disk Cache
- **Location**: `diskcache/file_cache.go:40,44`
- **Description**: Change file creation mode from `0700` to `0644`.
- **Remediation Effort**: 1 SP (0.25 developer days).

#### TD-16: Clean Unused Direct Dependency in `go.mod`
- **Location**: `go.mod:30`
- **Description**: Move `golang.org/x/image` to test dependencies or remove if unneeded.
- **Remediation Effort**: 1 SP (0.25 developer days).

#### TD-17: Consolidate Overlapping YAML Parsers
- **Location**: `go.mod:33,77,83`
- **Description**: Standardize dependencies on `gopkg.in/yaml.v3`.
- **Remediation Effort**: 1 SP (0.5 developer days).

---

## 6. Verification & Regression Safeguards

To verify that technical debt remediation proceeds without breaking existing contracts:
1. **Backend Verification Suite**:
   ```bash
   go test -count=1 ./...
   go test -race -count=1 ./...
   go vet ./...
   ```
2. **Frontend Verification Suite**:
   ```bash
   pnpm --dir frontend run typecheck
   pnpm --dir frontend run lint
   pnpm --dir frontend test
   pnpm --dir frontend run build
   ```
3. **Database Upgrade Safeguards**:
   - Verify that any storage layer changes preserve compatibility with existing `nyx.db` and legacy `filebrowser.db` files.
