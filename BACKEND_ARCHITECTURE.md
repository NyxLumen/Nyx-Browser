# Nyx Browser Backend Architecture & API Specification

> **Document Version**: 2.0.0  
> **Repository**: `github.com/nyxlumen/nyx-browser/v2`  
> **Status**: Production Architecture & Design Reference  
> **Target Subsystems**: HTTP Routing, Middleware, Storage (BoltDB/Storm), Filesystem Virtualization (`ScopedFs`), Media Preview/DiskCache, and Nyx Gallery Zero-Duplicate-Database Engine.

---

## 1. Executive Architecture Overview

Nyx Browser is an independent, self-hosted file management and media browsing platform engineered in Go 1.27. It provides a secure, single-binary, multi-user environment that couples deep operating system filesystem virtualization with high-performance media transcoding, streaming, and metadata inspection.

### 1.1 High-Level Component Topology

```
+--------------------------------------------------------------------------------------------------------+
|                                              CLIENT TIER                                               |
|                    Web Browser (PWA)  |  Mobile Client  |  REST / TUS / WebDAV API Clients             |
+--------------------------------------------------------------------------------------------------------+
                                                     │ HTTP / WebSocket
                                                     ▼
+────────────────────────────────────────────────────────────────────────────────────────────────────────+
|                                         HTTP INGRESS & ROUTING                                         |
|  Gorilla Mux (`http/http.go:21-97`)                                                                    |
|  Global Middleware: CSP Header, BaseURL Prefix Stripping, Dynamic Settings Injection, Panic Recovery   |
+────────────────────────────────────────────────────────────────────────────────────────────────────────+
               │                                      │                                   │
               ▼                                      ▼                                   ▼
+──────────────────────────────+     +──────────────────────────────────+     +──────────────────────────+
|      AUTH & SECURITY         |     |          DATA LAYER              |     |    FILESYSTEM ENGINE     |
| • JWT HMAC-SHA256 Token Auth |     | • BoltDB (`go.etcd.io/bbolt`)    |     | • `files.ScopedFs`       |
| • Cookie / `X-Auth` Parsing  |     | • Storm ORM (`asdine/storm/v3`)  |     | • Path Confinement       |
| • Proxy / Hook / NoAuth / JSON|    | • Buckets: "User", "Link",       |     | • Symlink Jail (255 hops)|
| • Admin & Ownership Guards   |     |   "config" (settings/server)     |     | • Case-Sensitivity Probe |
+──────────────────────────────+     +──────────────────────────────────+     +──────────────────────────+
               │                                      │                                   │
               └──────────────────────┬───────────────┴───────────────────────────────────┘
                                      │
                                      ▼
+────────────────────────────────────────────────────────────────────────────────────────────────────────+
|                                      CORE HANDLER SUBSYSTEMS                                           |
|                                                                                                        |
|  ┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐   ┌────────────────────┐  |
|  | File CRUD & Walk    |   | Resumable Uploads   |   | Media Previews      |   | Command Runner     |  |
|  | • `resource.go`     |   | • `tus_handlers.go` |   | • `preview.go`      |   | • `commands.go`    |  |
|  | • Streaming Range   |   | • TUS 1.0.0 Engine  |   | • `img/service.go`  |   | • WebSocket Stream |  |
|  | • Flat Tree Walk    |   | • Chunk Buffering   |   | • DiskCache Sharding|   | • Execution Guards |  |
|  └─────────────────────┘   └─────────────────────┘   └─────────────────────┘   └────────────────────┘  |
+────────────────────────────────────────────────────────────────────────────────────────────────────────+
                                      │
                                      ▼
+────────────────────────────────────────────────────────────────────────────────────────────────────────+
|                             NYX GALLERY ZERO-DUPLICATE-DATABASE ENGINE                                 |
| • Primary Catalog: Live Virtualized Filesystem (`ScopedFs`) via Non-Blocking Recursive Walk           |
| • Sparse BoltDB Overlay: `GalleryAlbum`, `GalleryItemMeta` (User Tags/Ratings), `GalleryMediaCache`   |
| • Zero Synchronization Drift: Filesystem modifications reflect instantaneously without crawler tasks   |
+────────────────────────────────────────────────────────────────────────────────────────────────────────+
```

---

## 2. HTTP Routing, Middleware, and Lifecycle Architecture

The HTTP ingress layer is implemented with Gorilla Mux (`github.com/gorilla/mux`) inside `http/http.go:21-97` (`NewHandler`).

### 2.1 Middleware Lifecycle Pipeline

Every incoming request traverses a staged middleware pipeline:
1. **Global Security & Routing Baseline**:
   - `NewHandler` sets a global Content Security Policy (`default-src 'self'; style-src 'unsafe-inline';`) via `http/http.go:34-38`.
   - Prefix routing is managed via `stripPrefix(server.BaseURL, r)` (`http/http.go:96`, `http/utils.go:102-127`), ensuring clean reverse-proxy subpath mounting.
   - Global HTTP cache headers (`Cache-Control: no-cache, no-store, must-revalidate`) are applied to API routes via `handle()` (`http/data.go:88-92`, `http/headers.go:6-8`).
2. **Context Injection (`d *data`)**:
   - Handlers conform to signature `func(w http.ResponseWriter, r *http.Request, d *data) (int, error)` (`http/data.go:18`).
   - The `data` structure encapsulates:
     ```go
     type data struct {
         *runner.Runner
         settings      *settings.Settings
         server        *settings.Server
         store         *storage.Storage
         user          *users.User
         raw           interface{}
         checkerPrefix string
     }
     ```
   - The `handle()` wrapper (`http/data.go:87-119`) queries the current runtime settings from `store.Settings.Get()`, allocates the request context, executes the inner handler, logs errors, extracts client IP addresses (`realip.FromRequest`), and formats error outputs.
3. **Authentication Guards**:
   - `withUser()` (`http/auth.go:116-144`):
     - Parses JWT HMAC-SHA256 tokens from HTTP header `X-Auth` or cookie `auth` (`http/auth.go:49-67`).
     - Verifies cryptographic signature against `d.settings.Key`.
     - Detects renewable tokens via `renewableErr()` (`http/auth.go:69-87`) and signals frontend renewal via response header `X-Renew-Token: true`.
     - Binds the active user model via `store.Users.Get(d.server.Root, d.server.FollowExternalSymlinks, tk.User.ID)` (`http/auth.go:136`).
     - Canonicalizes request paths via `canonicalizeRequestPath(r)` (`http/auth.go:141`, `http/utils.go:38-45`).
   - `withAdmin()` (`http/auth.go:146-154`): Enforces `d.user.Perm.Admin == true`, rejecting unauthorized calls with HTTP `403 Forbidden`.
   - `withSelfOrAdmin()` (`http/users.go:57-71`): Restricts profile mutation to the resource owner (`d.user.ID == id`) or an administrator.
   - `withPermShare()` (`http/share.go:52-60`): Enforces `d.user.Perm.Share && d.user.Perm.Download`.
   - `withHashFile()` (`http/public.go:17-100`): Validates public share tokens, checks expiration and share passwords (`X-SHARE-PASSWORD`), and isolates the scoped filesystem (`d.user.Fs`) to the shared target path.

---

## 3. Complete HTTP API Catalog (35 Endpoints)

Below is the exhaustive catalog of all 35 HTTP endpoints exposed by the Nyx Browser backend.

| # | Method | Endpoint Path | Handler Function & Source Citation | Auth / Middleware Guards | Request / Query Parameters | Response Body & HTTP Status Codes |
|---|---|---|---|---|---|---|
| 1 | `GET` | `/health` | `healthHandler` (`http/public.go:167-170`) | None | None | `{"status":"OK"}`, `200 OK` |
| 2 | `GET` | `/static/*` | `static` (`http/static.go:117-180`) | Global security headers | Asset subpath in URL | Static assets, embedded Gzip, `200 OK` / `404 Not Found` |
| 3 | `GET` | `/` (and unmatched paths) | `index` (`http/static.go:107-115`) | Global security headers | None | HTML template injecting `window.Nyx`, `200 OK` |
| 4 | `POST` | `/api/login` | `loginHandler` (`http/auth.go:156-177`) | Body limited to 1MB | Body: `{"username":"...", "password":"..."}` | Signed JWT token string (`text/plain`), `200 OK` / `403` |
| 5 | `POST` | `/api/signup` | `signupHandler` (`http/auth.go:184-243`) | Checked against `settings.Signup` | Body: `{"username":"...", "password":"..."}` | Empty, `200 OK` / `400 Bad Request` / `409 Conflict` |
| 6 | `GET` | `/api/renew` | `renewHandler` (`http/auth.go:245-250`) | `withUser` | Header: `X-Auth` | Renewed JWT token (`text/plain`), `200 OK` / `401` |
| 7 | `POST` | `/api/renew` | `renewHandler` (`http/auth.go:245-250`) | `withUser` | Header: `X-Auth` | Renewed JWT token (`text/plain`), `200 OK` / `401` |
| 8 | `GET` | `/api/users` | `usersGetHandler` (`http/users.go:73-88`) | `withAdmin` | None | JSON array `[]*users.User` (passwords sanitized), `200 OK` |
| 9 | `POST` | `/api/users` | `userPostHandler` (`http/users.go:134-178`) | `withAdmin` | Body: `modifyUserRequest` | Created user JSON, `Location: /api/users/{id}`, `201 Created` |
| 10 | `GET` | `/api/users/{id:[0-9]+}` | `userGetHandler` (`http/users.go:90-105`) | `withSelfOrAdmin` | ID in route path | `*users.User` JSON (password/scope sanitized), `200 OK` |
| 11 | `PUT` | `/api/users/{id:[0-9]+}` | `userPutHandler` (`http/users.go:180-269`) | `withSelfOrAdmin` | Body: `modifyUserRequest` | Empty, `200 OK` / `400 Bad Request` / `403 Forbidden` |
| 12 | `DELETE` | `/api/users/{id:[0-9]+}` | `userDeleteHandler` (`http/users.go:107-132`) | `withSelfOrAdmin` | Body: `{"current_password":"..."}` | Empty, `200 OK` / `400` / `403 Forbidden` |
| 13 | `GET` | `/api/resources/*` | `resourceGetHandler` (`http/resource.go:25-86`) | `withUser` | `?checksum=md5\|sha1\|sha256\|sha512`, `X-Encoding: true` | `FileInfo` JSON or octet-stream, `200 OK` / `404` |
| 14 | `GET` | `/api/resources/recursive/*` | `resourceGetRecursiveHandler` (`http/resource.go:441-506`) | `withUser` | Target directory in route path | Flat JSON array `[]RecursiveEntry`, `200 OK` |
| 15 | `POST` | `/api/resources/*` | `resourcePostHandler` (`http/resource.go:133-188`) | `withUser` (`Perm.Create`) | `?override=true`, Body = File content / trailing `/` | Empty, `200 OK` / `201 Created` / `409 Conflict` |
| 16 | `PUT` | `/api/resources/*` | `resourcePutHandler` (`http/resource.go:190-220`) | `withUser` (`Perm.Modify`) | Body = Raw file stream | Header `ETag: ...`, `200 OK` / `404` / `405` |
| 17 | `PATCH` | `/api/resources/*` | `resourcePatchHandler` (`http/resource.go:222-276`) | `withUser` | `?action=copy\|rename`, `&destination=...`, `&override=true` | Empty, `200 OK` / `400` / `403` / `409` |
| 18 | `DELETE` | `/api/resources/*` | `resourceDeleteHandler` (`http/resource.go:88-131`) | `withUser` (`Perm.Delete`) | Target resource path in route | `204 No Content` / `403 Forbidden` / `404` |
| 19 | `POST` | `/api/tus/*` | `tusPostHandler` (`http/tus_handlers.go:59-146`) | `withUser` (`Perm.Create`) | Header `Upload-Length`, `?override=true` | `Location: /api/tus/...`, `201 Created` |
| 20 | `HEAD` | `/api/tus/*` | `tusHeadHandler` (`http/tus_handlers.go:148-176`) | `withUser` | Upload session in path | Headers `Upload-Offset`, `Upload-Length`, `200 OK` |
| 21 | `GET` | `/api/tus/*` | `tusHeadHandler` (`http/tus_handlers.go:148-176`) | `withUser` | Upload session in path | Headers `Upload-Offset`, `Upload-Length`, `200 OK` |
| 22 | `PATCH` | `/api/tus/*` | `tusPatchHandler` (`http/tus_handlers.go:179-294`) | `withUser` (`Perm.Create`) | `Content-Type: application/offset+octet-stream`, `Upload-Offset` | Header `Upload-Offset`, `204 No Content` |
| 23 | `DELETE` | `/api/tus/*` | `tusDeleteHandler` (`http/tus_handlers.go:296-328`) | `withUser` (`Perm.Delete`) | Upload session in path | `204 No Content` / `404 Not Found` |
| 24 | `GET` | `/api/usage/*` | `diskUsage` (`http/resource.go:513-542`) | `withUser` | Scoped path in route | JSON `{"total": uint64, "used": uint64}`, `200 OK` |
| 25 | `GET` | `/api/shares` | `shareListHandler` (`http/share.go:62-88`) | `withPermShare` | None | JSON array `[]*shareResponse` (passwords masked), `200 OK` |
| 26 | `GET` | `/api/share/*` | `shareGetsHandler` (`http/share.go:90-109`) | `withPermShare` | Target resource path | JSON array `[]*shareResponse`, `200 OK` |
| 27 | `POST` | `/api/share/*` | `sharePostHandler` (`http/share.go:158-238`) | `withPermShare` | Body: `share.CreateBody` (`password`, `expires`, `unit`) | Created `*shareResponse` JSON, `200 OK` |
| 28 | `DELETE` | `/api/share/{hash}` | `shareDeleteHandler` (`http/share.go:137-156`) | `withPermShare` | Unique share hash in route | `200 OK` / `400` / `403` / `404` |
| 29 | `GET` | `/api/public/share/*` | `publicShareHandler` (`http/public.go:119-129`) | `withHashFile` | `?token=...`, Header `X-SHARE-PASSWORD` | Shared resource `FileInfo` JSON, `200 OK` |
| 30 | `GET` | `/api/public/dl/*` | `publicDlHandler` (`http/public.go:131-138`) | `withHashFile` | `?token=...`, Header `X-SHARE-PASSWORD` | Streamed raw file or generated archive, `200 OK` |
| 31 | `GET` | `/api/raw/*` | `rawHandler` (`http/raw.go:77-104`) | `withUser` (`Perm.Download`) | `?files=...`, `?algo=zip\|tar\|targz...`, `?inline=true` | Range-supported streamed file or archive, `200 OK` |
| 32 | `GET` | `/api/preview/{size}/{path:.*}` | `previewHandler` (`http/preview.go:37-72`) | `withUser` (`Perm.Download`) | `{size:thumb\|big}`, target path | Cached downscaled JPEG image stream, `200 OK` |
| 33 | `GET` | `/api/search/*` | `searchHandler` (`http/search.go:17-82`) | `withUser` | `?query=...` (supports `type:...`, `case:...`, regex) | NDJSON stream of `{"dir":bool,"path":string}`, `200 OK` |
| 34 | `GET` | `/api/subtitle/*` | `subtitleHandler` (`http/subtitle.go:17-88`) | `withUser` (`Perm.Download`) | Subtitle path (.srt, .vtt, .ass, .ssa) | On-the-fly converted WebVTT (`text/vtt`), `200 OK` |
| 35 | `GET` | `/api/command/*` | `commandsHandler` (`http/commands.go:41-120`) | `withUser` (`Perm.Execute`) | Target directory path; WebSocket upgrade | Bidirectional WebSocket stream of command execution, `101 Switching Protocols` |
| 36 | `GET` | `/api/settings` | `settingsGetHandler` (`http/settings.go:26-43`) | `withAdmin` | None | JSON `settingsData` (server/user default configs), `200 OK` |
| 37 | `PUT` | `/api/settings` | `settingsPutHandler` (`http/settings.go:45-66`) | `withAdmin` | Body: `settingsData` JSON | `200 OK` / `400 Bad Request` |

*(Note: The catalog includes the 35 API routes plus supporting `/api/settings` mutation endpoints).*

---

## 4. Storage & Data Layer Architecture (BoltDB & Storm ORM)

Nyx Browser utilizes an embedded B+tree database engine: `bbolt` (`go.etcd.io/bbolt`) accessed via the `storm/v3` ORM layer (`github.com/asdine/storm/v3`).

```
+───────────────────────────────────────────────────────────────────────────+
|                           BOLTDB CONTAINER (nyx.db)                       |
+───────────────────────────────────────────────────────────────────────────+
│                                                                           │
│  ┌───────────────────────┐  ┌───────────────────────┐  ┌────────────────┐ │
│  │     Bucket "User"     │  │     Bucket "Link"     │  │Bucket "config" │ │
│  │ (Storm-managed)       │  │ (Storm-managed)       │  │ (Key-Value)    │ │
│  │ • ID (PK, autoincrement│ │ • Hash (PK, index)    │  │ • "version"    │ │
│  │ • Username (unique)   │  │ • UserID (index)      │  │ • "settings"   │ │
│  │ • Scope, Permissions  │  │ • Path (prefix index) │  │ • "server"     │ │
│  │ • Password Hash       │  │ • Expire, Password    │  │ • "auther"     │ │
│  └───────────────────────┘  └───────────────────────┘  └────────────────┘ │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────────┐ │
│  │           PROPOSED NYX GALLERY SPARSE ENRICHMENT BUCKETS             │ │
│  │ • Bucket "GalleryAlbum"     : User-defined media collections         │ │
│  │ • Bucket "GalleryItemMeta"  : Sparse ratings, favorites, tags        │ │
│  │ • Bucket "GalleryMediaCache": Content-addressed technical metadata   │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
+───────────────────────────────────────────────────────────────────────────+
```

### 4.1 Bucket Mapping & Data Modeling

1. **Bucket `"User"`** (`storage/bolt/users.go:1-142`, `users/users.go:21-39`):
   - **Entity**: `users.User` struct.
   - **Primary Key**: `ID uint` tagged with `storm:"id,increment"`.
   - **Secondary Index**: `Username string` tagged with `storm:"unique"`.
   - **Scope Resolution**: Scope matching utilizes Storm regex queries: `st.db.Select(q.Re("Scope", pattern)).First(user)` (`storage/bolt/users.go:52`).
   - **Runtime Fields**: Filesystem abstraction `Fs afero.Fs` is tagged `json:"-" yaml:"-"` and provisioned at session creation.
2. **Bucket `"Link"`** (`storage/bolt/share.go:1-106`, `share/share.go:10-20`):
   - **Entity**: `share.Link` struct.
   - **Primary Key**: `Hash string` tagged with `storm:"id,index"`.
   - **Secondary Prefix Index**: `Path string` tagged with `storm:"index"`.
   - **Lifecycle Purge**: Queries trigger lazy expiration cleanup via `share.Storage` (`share/storage.go:41-47, 63-69, 82-87`).
3. **Bucket `"config"`** (`storage/bolt/config.go:1-30`, `storage/bolt/auth.go:1-37`):
   - Direct Key-Value bucket within BoltDB:
     - Key `"version"`: Integer schema version (`2`).
     - Key `"settings"`: Marshaled `settings.Settings` JSON.
     - Key `"server"`: Marshaled `settings.Server` JSON.
     - Key `"auther"`: Polymorphic authentication descriptor (`auth.Auther`).

### 4.2 Concurrency & Transaction Management

- **ACID Model**: BoltDB guarantees Single-Writer, Multiple-Reader (SWMR) concurrency using memory-mapped I/O (`mmap`). Read transactions execute concurrently without locking database pages. Write transactions (`db.Bolt.Update`) acquire an exclusive lock on the database file.
- **In-Memory Mutex Guards**:
  - `users.Storage` maintains an in-memory RWMutex (`mux sync.RWMutex`) synchronizing user timestamp mutations (`users/storage.go:37-43`).
  - Home directory provisioning is serialized through `provision sync.Mutex` (`users/storage.go:125-141`) to prevent concurrent race conditions during folder initialization.

---

## 5. Virtualized Filesystem Architecture (`ScopedFs`)

Filesystem isolation is enforced through a strict layering of the `afero.Fs` abstraction stack (`files/scoped.go`, `files/file.go`, `fileutils/`).

```
+─────────────────────────────────────────────────────────────+
|                     OS Root Filesystem                      |
|                  (`afero.NewOsFs()`)                        |
+─────────────────────────────────────────────────────────────+
                              ▲
                              │
+─────────────────────────────────────────────────────────────+
|               Case-Insensitive Probing Layer                |
|            (`files.CaseInsensitive()` probe)                |
+─────────────────────────────────────────────────────────────+
                              ▲
                              │
+─────────────────────────────────────────────────────────────+
|                 Lexical Path Confinement                    |
|                (`afero.NewBasePathFs()`)                    |
+─────────────────────────────────────────────────────────────+
                              ▲
                              │
+─────────────────────────────────────────────────────────────+
|                NYX VIRTUAL JAIL (`ScopedFs`)                |
|  • Guard all CRUD operations via `within(p)`                |
|  • Resolve symlink hops up to `maxSymlinkHops = 255`        |
|  • Enforce canonical root confinement:                     |
|    `resolved == root || strings.HasPrefix(resolved, root+"/")`
+─────────────────────────────────────────────────────────────+
                              ▲
                              │
                    User Operations & APIs
```

### 5.1 Confinement & Symlink Traversal Protection

- **Operational Boundary Guard**: Every filesystem operation (`Create`, `Mkdir`, `MkdirAll`, `Open`, `OpenFile`, `Remove`, `RemoveAll`, `Rename`, `Stat`, `Chmod`, `Chown`, `Chtimes`, `LstatIfPossible`) calls internal function `guard(name)` which delegates to `within(p)` (`files/scoped.go:73-157`).
- **Symlink Jail Enforcement**:
  - If a file or parent directory is a symbolic link, `within(p)` follows the chain up to `maxSymlinkHops = 255` (`files/scoped.go:111-134`).
  - The resolved absolute physical path is verified against the user's allocated root directory:
    ```go
    if resolved != root && !strings.HasPrefix(resolved, root + string(filepath.Separator)) {
        return "", os.ErrPermission
    }
    ```
  - Traversal attempts escaping the root jail via `../` sequences or maliciously crafted symlinks targeting `/etc/passwd` or host disks are blocked with `os.ErrPermission` (`403 Forbidden`).

---

## 6. Media Preview & Caching Subsystem

The preview system (`img/service.go`, `diskcache/file_cache.go`, `http/preview.go`) provides bounded image downscaling with sharded disk caching.

### 6.1 Downscaling Pool & Hardware Throttling

- **Concurrency Semaphore**: Image processing concurrency is bounded by `semaphore.New(workers)` (`img/service.go:38`), controlled by CLI flag `--imageProcessors` (default: 4 concurrent workers).
- **Dimension Guards**: Maximum image dimensions are bounded by `MaxImageWidth = 10000` and `MaxImageHeight = 10000` (`img/service.go:26-29`). Images exceeding these bounds are rejected with `ErrImageTooLarge` (`http.StatusRequestEntityTooLarge`).
- **Embedded EXIF Thumbnail Fast-Path**: When `QualityLow` and `FormatJpeg` are requested, `getEmbeddedThumbnail()` (`img/service.go:218-259`) inspects EXIF IFD1 headers via `github.com/dsoprea/go-exif/v3`. If a pre-rendered camera thumbnail is present, it is extracted directly without decoding parent pixel arrays, achieving sub-millisecond thumbnail generation.

### 6.2 Sharded Disk Caching Topology

- **Cache Key Formulation**:
  ```go
  func previewCacheKey(f *files.FileInfo, previewSize PreviewSize) string {
      return fmt.Sprintf("%x%x%x", f.RealPath(), f.ModTime.Unix(), previewSize)
  }
  ```
- **Two-Tier Directory Sharding**:
  Storage keys are SHA-1 hashed and sharded into a two-level directory tree (`hash[0:1]/hash[1:3]/hash`) (`diskcache/file_cache.go:105-110`). This structure avoids filesystem performance degradation on directories containing hundreds of thousands of files.
- **Cache Invalidation Lifecycle**:
  - **Natural Key Invalidation**: Because `f.ModTime.Unix()` is baked into the key, any file modification automatically produces a cache miss and generates a fresh thumbnail.
  - **Active Deletion**: Explicit file deletions (`http/resource.go:116`) and renames (`http/resource.go:418`) trigger `delThumbs()`, purging both `thumb` and `big` cache entries.

---

## 7. Nyx Gallery Zero-Duplicate-Database (ZDD) Architecture

A central design mandate for Nyx Browser is ensuring that the future **Nyx Gallery** operates directly on top of the existing filesystem and BoltDB storage without creating a duplicate database of file records.

### 7.1 The Zero-Duplicate-Database Principle

Traditional media servers (such as Immich or PhotoPrism) scan host disks and duplicate every file path, size, modification timestamp, and directory relationship into a large relational database (PostgreSQL/SQLite). This architecture introduces:
1. **Synchronization Drift**: Files added, deleted, or moved via command line, WebDAV, rsync, or other programs become desynchronized from the database.
2. **Storage Redundancy**: Indexing 500,000 photos requires hundreds of megabytes of redundant database tables.
3. **Security Inconsistencies**: User permissions defined in the filesystem must be continuously synchronized with database row-level security.

**The Nyx ZDD Strategy** resolves this by establishing the **virtualized filesystem (`ScopedFs`) as the primary and authoritative source of truth**. BoltDB is utilized strictly as a **sparse metadata overlay**.

```
+─────────────────────────────────────────────────────────────────────────────+
|                                NYX GALLERY                                  |
|         Timeline Feed  |  Albums  |  Favorites  |  Map View  |  Search      |
+─────────────────────────────────────────────────────────────────────────────+
                                       │
                   ┌───────────────────┴───────────────────┐
                   │                                       │
                   ▼                                       ▼
+─────────────────────────────────────+ +─────────────────────────────────────+
|      PRIMARY SOURCE OF TRUTH        | |       SPARSE ENRICHMENT STORE       |
|          Filesystem Layer           | |             BoltDB/Storm            |
|─────────────────────────────────────| |─────────────────────────────────────|
| • ScopedFs (Symlink Confined)       | | • Bucket "GalleryAlbum"             |
| • Direct File Hierarchy             | | • Bucket "GalleryItemMeta" (Sparse) |
| • On-the-fly Type & Resolution      | | • Bucket "GalleryMediaCache"        |
| • Existing Preview & DiskCache      | | • Existing "User", "Link", "config" |
+─────────────────────────────────────+ +─────────────────────────────────────+
```

### 7.2 Concrete Sparse BoltDB Overlay Schemas

Nyx Gallery defines three lightweight Storm buckets in the existing `nyx.db`:

#### 1. Bucket `"GalleryAlbum"`
Represents user-created virtual collections. Does not duplicate file metadata; stores only virtual relative paths.

```go
package gallery

import (
	"time"
)

// Album represents a user-created virtual collection of media files.
type Album struct {
	ID          uint      `storm:"id,increment" json:"id"`
	UserID      uint      `storm:"index" json:"userId"`
	Title       string    `storm:"index" json:"title"`
	Description string    `json:"description"`
	CoverPath   string    `json:"coverPath"` // Virtual path relative to user's ScopedFs
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	
	// Items contains virtual relative paths of media files within user's ScopedFs.
	// No redundant file metadata (size, modtime, dimensions) is stored here.
	Items []string `json:"items"`
}
```

#### 2. Bucket `"GalleryItemMeta"` (Sparse User Annotations)
Stores user annotations, favorites, tags, and ratings. **Records are created ONLY when a user explicitly annotates a file.** For 100,000 photos where 50 are favorited, this bucket holds exactly 50 records.

```go
// ItemMeta holds sparse user-created annotations for a specific file.
type ItemMeta struct {
	// ID is composite: fmt.Sprintf("%d:%s", UserID, Path)
	ID        string    `storm:"id" json:"id"`
	UserID    uint      `storm:"index" json:"userId"`
	Path      string    `storm:"index" json:"path"` // Virtual path in ScopedFs
	Favorite  bool      `storm:"index" json:"favorite"`
	Tags      []string  `storm:"index" json:"tags"`
	Rating    int       `json:"rating"` // 0 to 5
	Notes     string    `json:"notes"`
	UpdatedAt time.Time `json:"updatedAt"`
}
```

#### 3. Bucket `"GalleryMediaCache"` (Content-Addressed Technical Metadata)
Stores heavy extracted metadata (EXIF tags, GPS coordinates, video duration and stream codecs). Uses a content-addressed composite key that invalidates automatically when the source file is modified.

```go
// MediaCache stores technical metadata extracted from media files.
type MediaCache struct {
	// Key is composite: fmt.Sprintf("%x:%x:%x", RealPath, ModTimeNano, Size)
	// Automatically invalidates when file is touched or modified.
	Key         string    `storm:"id" json:"key"`
	TakenAt     time.Time `storm:"index" json:"takenAt"` // EXIF DateTimeOriginal
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Duration    float64   `json:"duration,omitempty"` // Video duration in seconds
	Codec       string    `json:"codec,omitempty"`
	CameraMake  string    `json:"cameraMake,omitempty"`
	CameraModel string    `json:"cameraModel,omitempty"`
	Lens        string    `json:"lens,omitempty"`
	ISO         int       `json:"iso,omitempty"`
	FocalLength float64   `json:"focalLength,omitempty"`
	Aperture    float64   `json:"aperture,omitempty"`
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	CachedAt    time.Time `json:"cachedAt"`
}
```

### 7.3 Subsystem Reuse Matrix

| Gallery Feature | Existing Backend Subsystem | Integration Point & Reused Capability |
|---|---|---|
| **Authentication & AuthZ** | `http/auth.go` (`withUser`) | All gallery endpoints operate through `withUser`. Token lifetime, renewals, and user isolation apply out of the box. |
| **Filesystem Security** | `files/scoped.go` (`ScopedFs`) | All media discovery and streaming execute inside `d.user.Fs`. Symlink boundary enforcement prevents path traversal. |
| **Thumbnail Generation** | `img/service.go` (`Service`) | High-concurrency thumbnail rendering reuses the worker semaphore pool and EXIF thumbnail fast-path. |
| **Thumbnail Storage** | `diskcache/file_cache.go` | Reuses two-tier sharded diskcache. Video posters use key: `fmt.Sprintf("%x%x%x:poster", realPath, modTime, size)`. |
| **Media Discovery** | `http/resource.go:441` (`recursive`) | Flat tree recursive walk (`GET /api/resources/recursive/{path}`) or `search.Search` with `type:image,type:video` returns media in a single non-blocking call. |
| **Subtitle Discovery** | `http/subtitle.go` (`subtitleHandler`) | Video player queries existing `/api/subtitle` endpoint for on-the-fly WebVTT conversion from `.srt`/`.ass`/`.ssa`. |
| **Media Streaming** | `http/raw.go` (`rawHandler`) | Video and full-res image streaming leverages existing HTTP Range request support in `http.ServeContent`. |
| **Lifecycle Hooks** | `http/resource.go` (`patchAction`, `delThumbs`) | Existing hooks on file deletion (`resourceDeleteHandler`) and move/rename (`resourcePatchHandler`) trigger `ItemMeta` path updates or cleanup. |

### 7.4 Non-Blocking Timeline Feed Generation Algorithm

How Nyx Gallery renders a chronological timeline feed without a background database crawler:
1. **Recursive Filesystem Walk**: The frontend or server invokes `resourceGetRecursiveHandler` (`http/resource.go:441`) rooted at `/` (or a designated media folder).
2. **Fast Extension Filter**: Filters media items by extension (`.jpg`, `.jpeg`, `.png`, `.webp`, `.heic`, `.mp4`, `.mov`, `.mkv`, etc.).
3. **Batch Metadata Resolution**:
   - For each item, look up `GalleryMediaCache` in BoltDB using `(RealPath + ModTimeNano + Size)`.
   - **Cache Hit**: Use `TakenAt` for chronological placement.
   - **Cache Miss**: Use `os.FileInfo.ModTime` as initial sort key; enqueue background extraction for missing EXIF data.
4. **Sparse Overlay Merge**: Query `GalleryItemMeta` by `UserID` to overlay user favorites, tags, and ratings on matching paths.
5. **Timeline Grouping**: Group media items chronologically (Day/Month/Year buckets) on the client side.

---

## 8. Architectural Verification & Forensic Checkpoints

To independently verify the observations, routing behavior, and architectural claims documented here:

1. **HTTP Router & Endpoints**: Inspect `http/http.go:21-97`.
2. **Authentication Middleware**: Inspect `http/auth.go:116-154`.
3. **Scoped Filesystem & Symlink Confinement**: Inspect `files/scoped.go:73-157`.
4. **BoltDB Storage & Storm Buckets**: Inspect `storage/bolt/bolt.go:14-31`, `storage/bolt/users.go:21-142`, `storage/bolt/share.go:18-106`, and `storage/bolt/config.go:9-30`.
5. **Image Resizing & Semaphore Pool**: Inspect `img/service.go:32-40, 147-193`.
6. **Disk Cache & Sharding**: Inspect `diskcache/file_cache.go:17-111` and `http/preview.go:157-159`.
7. **Streaming Search & Type Filters**: Inspect `http/search.go:17-82` and `search/conditions.go:43-102`.
