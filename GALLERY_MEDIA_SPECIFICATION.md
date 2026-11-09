# Nyx Gallery & Media Processing Architecture Specification

> **Document Version**: 2.0.0  
> **Repository**: `github.com/nyxlumen/nyx-browser/v2`  
> **Target Subsystems**: Media Decoding, Thumbnail Downscaling, Video Poster Generation, FFmpeg Integration, EXIF/IPTC/XMP Extraction, Multi-Tier Caching  
> **Status**: Production Architecture & Design Reference  
> **Branch**: `nyx_foundation_phase_one`  

---

## 1. Executive Summary

Nyx Browser requires a modern, memory-safe, hardware-accelerated media pipeline capable of powering the high-density visual media workflows of Nyx Gallery. An in-depth audit of the existing preview subsystem (`img/service.go`, `diskcache/file_cache.go`, `http/preview.go`) reveals critical bottlenecks and reliability hazards:
1. **Format Stagnation**: Only 5 legacy formats (JPEG, PNG, GIF, TIFF, BMP) are supported. Modern formats (**WebP, AVIF, HEIC, JXL, SVG, RAW**) are missing.
2. **Bandwidth Penalty on Unsupported Formats**: When a user browses folders containing WebP, AVIF, or HEIC files, `http/preview.go:90-92` falls back to serving the **full-resolution uncompressed raw file over HTTP**, causing massive bandwidth spikes and slow rendering.
3. **Decompression Memory Bomb Vulnerability**: Pure Go `imaging.Decode` allocates full $10,000 \times 10,000 \times 4\text{ bytes} = 400\text{ MB}$ RGBA bitmaps in system RAM per worker slot. An 8-worker burst can allocate over 3.2 GB of RAM, triggering kernel Out-Of-Memory (`oom-killer`) process termination on small VPS instances.
4. **Complete Absence of Video Thumbnails**: All video formats (MP4, MKV, WebM, MOV, AVI) return HTTP 501 (`StatusNotImplemented`), displaying generic blank slate icons.
5. **Disk Cache Storage & Memory Leaks**: Cached files are created with Unix mode `0700` (`-rwx------`), fine-grained mutexes in `scopedLocks` permanently leak memory, cache reads in `Load()` lack locks (causing read-write races), and directories lack recursive thumbnail cleanup on deletion.

This specification defines the authoritative architecture for the Nyx Gallery media pipeline:
- **Expanded Format Matrix**: WebP, AVIF, HEIC, JXL, SVG, and RAW previews.
- **Bounded Memory-Safe Downscaling**: Streaming decoders with strict memory budgets ($128\text{ MB}$ per slot, $12,000 \times 12,000$ hard limit).
- **Video Poster Extraction Pipeline**: Multi-stage heuristic keyframe extraction for MP4, MKV, WebM, MOV, and AVI containers.
- **Sandboxed CLI FFmpeg Architecture**: Out-of-process isolation via `exec.CommandContext` with `-protocol_whitelist "file,pipe"`, strict timeouts, and concurrency throttling.
- **Unified EXIF / Video Metadata Engine**: Deep extraction of camera settings, GPS geohashing, IPTC/XMP tags, and video stream metadata.
- **Multi-Tier Caching Hierarchy**: Tier 1 (128 MB RAM LRU), Tier 2 (persistent quota-managed on-disk cache with striped locks and mode 0644), and Tier 3 (HTTP 304 ETags).

---

## 2. Current Image Processing Baseline Audit

```
+─────────────────────────────────────────────────────────────────────────────────────────+
|                           CURRENT IMAGE PROCESSING PIPELINE                             |
+─────────────────────────────────────────────────────────────────────────────────────────+
|                                                                                         |
|   HTTP Request: GET /api/preview/{thumb|big}/{path}                                     |
|         │                                                                               |
|         ▼                                                                               |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ Format Check (`img/service.go:42-51`)                                           │   |
|   │ Formats: JPEG, PNG, GIF, TIFF, BMP ONLY                                         │   |
|   │ ── Unsupported Format (WebP, AVIF, HEIC)? ──► Fallback to `rawFileHandler`      │   |
|   │                                                (Streams 20MB raw file over HTTP)|   |
|   └──────────────────────────────────────┬──────────────────────────────────────────┘   |
|                                          │ Supported                                    |
|                                          ▼                                              |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ EXIF Thumbnail Fast-Path (`img/service.go:218-259`)                             │   |
|   │ IF QualityLow AND JPEG: Extract IFD1 EXIF thumbnail (1ms) ── Hit? ──► Return    │   |
|   └──────────────────────────────────────┬──────────────────────────────────────────┘   |
|                                          │ Miss / QualityMedium                         |
|                                          ▼                                              |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ Decompression & Resizing (`img/service.go:178`)                                 │   |
|   │ • `imaging.Decode()`: Decodes entire image into uncompressed `image.NRGBA`      │   |
|   │ • Memory: 10,000 x 10,000 x 4 bytes = 400 MB RAM per slot                       │   |
|   │ • Concurrency: Semaphore pool (CLI `--imageProcessors`, default 4 workers)      │   |
|   │ • 4 workers x 400 MB = 1.6 GB RAM allocation spike                              │   |
|   │ • Resample: Lanczos / Box / NearestNeighbor                                     │   |
|   │ • Encode: JPEG (QualityLow = 40, QualityMedium = 75, QualityHigh = 85)          │   |
|   └──────────────────────────────────────┬──────────────────────────────────────────┘   |
|                                          │                                              |
|                                          ▼                                              |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ Disk Storage (`diskcache/file_cache.go`)                                        │   |
|   │ • Key: sha1(RealPath + ModTime + Size)                                          │   |
|   │ • Path: hash[0:1]/hash[1:3]/hash                                                │   |
|   │ • BUG: Mode 0700 (executable bit set on cached JPEGs)                           │   |
|   │ • BUG: `scopedLocks.locks` map never evicts mutexes (unbounded heap leak)        │   |
|   │ • BUG: `Load()` lacks lock acquisition (data race with concurrent Store)        │   |
|   │ • BUG: No eviction policy (disk grows unbounded until volume full)              │   |
|   └─────────────────────────────────────────────────────────────────────────────────┘   |
+─────────────────────────────────────────────────────────────────────────────────────────+
```

### 2.1 Supported Formats vs Missing Modern Formats

- **Supported (`img/service.go:42-51`)**: `jpeg`, `png`, `gif`, `tiff`, `bmp`.
- **Missing Modern Formats**:
  - `webp` (`.webp`): Standard format for modern web imagery, supported natively by 98% of web browsers.
  - `avif` (`.avif`): High-efficiency AV1 image format providing 50% better compression than JPEG.
  - `heic` / `heif` (`.heic`, `.heif`): Default photo container for iOS (iPhone 7+) and modern Android devices.
  - `jxl` (`.jxl`): Next-generation JPEG XL standard.
  - `svg` (`.svg`): Vector graphics requiring safe rasterization to prevent stored XSS.
  - `raw` (`.dng`, `.cr2`, `.nef`, `.arw`, `.orf`): Professional DSLR / mirrorless camera RAW formats.

### 2.2 Memory Allocation Profile & Decompression Spikes

`img/service.go:26-29` defines:
```go
const (
    MaxImageWidth  = 10000
    MaxImageHeight = 10000
)
```
In `img/service.go:178`:
```go
img, err := imaging.Decode(wrappedReader, imaging.AutoOrientation(true))
```
`imaging.Decode` uses pure Go decoders to decode the entire image into an uncompressed `image.NRGBA` buffer in system RAM. For an image at maximum allowed dimensions ($10,000 \times 10,000$ pixels), the decompressed buffer consumes:
$$10,000 \times 10,000 \times 4\text{ bytes (RGBA)} = 400,000,000\text{ bytes } \approx 400\text{ MB RAM}$$
Because concurrency is governed by `semaphore.New(workers)` (`img/service.go:38`), on an 8-core machine where `workers = 8`, concurrent requests for high-resolution images can allocate up to $8 \times 400\text{ MB} = 3.2\text{ GB RAM}$, risking Linux kernel Out-Of-Memory (`oom-killer`) termination on resource-constrained home servers and VPS instances.

### 2.3 Fallback Penalty for Unsupported Formats

In `http/preview.go:90-92`:
```go
format, err := imgSvc.FormatFromExtension(file.Extension)
if errors.Is(err, img.ErrUnsupportedFormat) || format == img.FormatGif {
    return rawFileHandler(w, r, file)
}
```
For WebP, AVIF, and HEIC files, `FormatFromExtension` returns `ErrUnsupportedFormat`. The preview endpoint falls back to serving the **full-resolution, uncompressed raw file over HTTP**. Opening a gallery directory with 50 WebP photos ($20\text{ MB}$ each) forces the client browser to download $1\text{ GB}$ of data instead of thumbnails.

---

## 3. Target Modern Media Pipeline Architecture

```
+─────────────────────────────────────────────────────────────────────────────────────────+
|                                NYX MEDIA PIPELINE ARCHITECTURE                          |
+─────────────────────────────────────────────────────────────────────────────────────────+
|                                                                                         |
|   HTTP Request: GET /api/preview/{size}/{path}                                          |
|         │                                                                               |
|         ▼                                                                               |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ TIER 1: In-Memory RAM LRU Cache (hashicorp/golang-lru/v2)                       │   |
|   │ Key: sha256(RealPath + ":" + Size) ── Hit? ──► HTTP 200 (304 if ETag matches)   │   |
|   └──────────────────────────────────────┬──────────────────────────────────────────┘   |
|                                          │ Miss                                         |
|                                          ▼                                              |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ TIER 2: Persistent On-Disk Cache (diskcache/ with LRU Eviction & Quota)         │   |
|   │ Check Key Index (BoltDB / In-Memory Index) ── Hit? ──► Promote to RAM & Return  │   |
|   └──────────────────────────────────────┬──────────────────────────────────────────┘   |
|                                          │ Miss                                         |
|                                          ▼                                              |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ MEDIA DISPATCHER (Semaphore Pool: GOMAXPROCS * 2)                               │   |
|   └───────────────┬─────────────────────────────────────────────────┬───────────────┘   |
|                   │                                                 │                   |
|         [Image MIME Type]                                 [Video MIME Type]             |
|                   │                                                 │                   |
|                   ▼                                                 ▼                   |
|   ┌───────────────────────────────┐                 ┌───────────────────────────────┐   |
|   │ IMAGE PIPELINE                │                 │ VIDEO PIPELINE (FFmpeg/Probe) │   |
|   │ - JPEG/PNG/WebP/TIFF/BMP/AVIF │                 │ - Probe duration & streams    │   |
|   │ - Check EXIF IFD1 thumbnail   │                 │ - Extract attached cover art  │   |
|   │ - Streaming shrink-on-decode  │                 │ - Optimal keyframe seek (10%) │   |
|   │ - Auto-orient via EXIF        │                 │ - CLI fork/exec with sandbox  │   |
|   │ - Lanczos downscaling         │                 │ - Fallback: pure Go / SVG     │   |
|   └───────────────┬───────────────┘                 └───────────────┬───────────────┘   |
|                   │                                                 │                   |
|                   └───────────────────────┬─────────────────────────┘                   |
|                                           ▼                                             |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ OUTPUT ENCODING & CACHE WRITE                                                   │   |
|   │ - Encode: WebP (lossy q=82) or JPEG (q=85)                                      │   |
|   │ - Write Tier 2 Disk Cache (atomic rename, mode 0644)                            │   |
|   │ - Promote to Tier 1 RAM LRU                                                     │   |
|   │ - Response: Content-Type: image/webp, Cache-Control, ETag                        │   |
|   └─────────────────────────────────────────────────────────────────────────────────┘   |
+─────────────────────────────────────────────────────────────────────────────────────────+
```

---

## 4. Modern Format Expansion & Memory-Safe Downscaling

### 4.1 Expanded Format Matrix

1. **WebP (`golang.org/x/image/webp`)**:
   - Native pure Go decoding.
   - Handles VP8 lossy, VP8L lossless, and extended containers.
2. **AVIF (`github.com/gen2brain/avif` or FFmpeg pipe)**:
   - Modern AV1 intra-frame image codec.
   - Extracts 10-bit and 12-bit HDR profiles and maps to 8-bit sRGB thumbnail color space.
3. **HEIC / HEIF (`libheif` via FFmpeg pipe or `github.com/adrium/goheif`)**:
   - Extracts master image item from ISOBMFF box container (`ftypheic`, `ftypmif1`).
   - Automatically handles NCLX color profiles and EXIF rotation tags.
4. **RAW Containers (DNG, CR2, NEF, ARW, ORF)**:
   - Uses `dsoprea/go-exif/v3` to extract full-resolution embedded preview JPEGs stored by camera firmware in IFD0/IFD1 tags.
   - Eliminates the need for expensive CPU demosaicing (dcraw/libraw).
5. **SVG Rasterization**:
   - Sanitized via XML decoder to strip `<script>` tags, event handlers, and external entity references (`XXE`).
   - Rasterized into high-DPI thumbnail bitmaps using `github.com/tdewolff/canvas`.

### 4.2 Memory-Safe Downscaling Architecture

To eliminate the $400\text{ MB}$ decompression spike:
1. **Shrink-on-Decode**:
   - For JPEG: Inspect dimensions via `image.DecodeConfig`. Compute scale power of 2 ($1/2, 1/4, 1/8$). Use IDCT downscaling during DCT coefficient decompression, reducing intermediate RAM allocation by $75\%\text{–}93\%$.
2. **Memory Budgets & Guardrails**:
   - `MaxInputDimension = 12000` (rejects malicious decompression bombs).
   - Hard memory budget: $128\text{ MB}$ maximum decompression allocation per slot.
   - If dimension exceeds memory budget, downscaling aborts with HTTP 413 (`StatusRequestEntityTooLarge`).
3. **Thumbnail Output Encoding**:
   - Default thumbnail format: **WebP** (`Quality = 82`), achieving 30–45% smaller file sizes than baseline JPEG at equivalent visual fidelity.
   - Automatic Content-Negotiation fallback: If client `Accept` header omits `image/webp`, encode as JPEG (`Quality = 80`).

---

## 5. Video Poster Generation Architecture

### 5.1 Supported Containers & Codecs

- **Containers**: MP4 (`.mp4`, `.m4v`), Matroska (`.mkv`), WebM (`.webm`), QuickTime (`.mov`), Audio Video Interleave (`.avi`).
- **Codecs**: H.264 (AVC), H.265 (HEVC), VP8, VP9, AV1, ProRes.

### 5.2 Three-Stage Extraction Pipeline

```
+─────────────────────────────────────────────────────────────────────────────+
|                    3-STAGE VIDEO POSTER EXTRACTION                          |
+─────────────────────────────────────────────────────────────────────────────+
|                                                                             |
|   Video File Input (MP4, MKV, WebM, MOV, AVI)                               |
|         │                                                                   |
|         ▼                                                                   |
|   ┌─────────────────────────────────────────────────────────────────────┐   |
|   │ STAGE 1: Embedded Cover Art Probe (< 15 ms)                         │   |
|   │ • Check container metadata for `disposition:attached_pic`           │   |
|   │ • If present: Extract embedded album art directly via stream copy   │   |
|   │   Command: ffmpeg -i [FILE] -map 0:v:1 -c copy -f image2 -          │   |
|   │ • Found? ──► Encode WebP Thumbnail & Return                         │   |
|   └──────────────────────────────────┬──────────────────────────────────┘   |
|                                      │ No attached picture                  |
|                                      ▼                                      |
|   ┌─────────────────────────────────────────────────────────────────────┐   |
|   │ STAGE 2: Heuristic Keyframe Seek (< 120 ms)                         │   |
|   │ • Probe duration via ffprobe:                                       │   |
|   │   - Duration <= 3s : target t = 0.5s                                │   |
|   │   - 3s < Duration <= 30s : target t = 2.0s or 15% duration          │   |
|   │   - Duration > 30s : target t = min(60.0s, duration * 0.10)         │   |
|   │ • Fast input seek before input flag (-ss [t] -i [FILE]):            │   |
|   │   Seeks keyframe in container index without decoding GOP frames!    │   |
|   │ • Scale & extract single frame:                                     │   |
|   │   -vf "scale=1080:-2:force_original_aspect_ratio=decrease"          │   |
|   │ • Success? ──► Encode WebP Thumbnail & Return                       │   |
|   └──────────────────────────────────┬──────────────────────────────────┘   |
|                                      │ Failure / Broken Index               |
|                                      ▼                                      |
|   ┌─────────────────────────────────────────────────────────────────────┐   |
|   │ STAGE 3: Fallback Degradation                                       │   |
|   │ • Retry extraction at timestamp t = 0.0s (first frame)              │   |
|   │ • If FFmpeg unavailable or video is audio-only:                     │   |
|   │   Generate dynamic SVG poster badge displaying video container type,│   |
|   │   dimensions, duration, and filename.                               │   |
|   └─────────────────────────────────────────────────────────────────────┘   |
+─────────────────────────────────────────────────────────────────────────────+
```

---

## 6. Architectural Evaluation of FFmpeg/FFprobe Integration Models

| Criterion | Option A: CLI Sub-Process (`exec.CommandContext`) | Option B: Embedded CGO (`go-astiav`) | Option C: Pure Go / WebAssembly (`wazero`) | Option D: Pluggable Microservice |
|---|---|---|---|---|
| **Crash Isolation** | **Complete** (segfault in FFmpeg does not affect Nyx) | **Zero** (`SIGSEGV` terminates Go runtime immediately) | **Complete** (isolated Wasm memory) | **Complete** (isolated process/network) |
| **Go Cross-Compilation** | **Preserved** (`CGO_ENABLED=0`, static binary) | **Broken** (requires multiarch C toolchains & headers) | **Preserved** (`CGO_ENABLED=0`) | **Preserved** (pure Go client) |
| **Memory Safety & Leaks** | **Safe** (OS reclaims memory on process exit) | **Hazardous** (manual `av_frame_free`, leak risks) | **Safe** (bounded Wasm linear memory) | **Safe** (managed by worker process) |
| **Hardware Acceleration** | **Yes** (VA-API, NVDEC, QSV via system ffmpeg) | **Yes** (direct API integration) | **No** (Wasm lacks direct GPU/DRM access) | **Yes** (worker can access GPU devices) |
| **Performance / Latency** | Low overhead ($15\text{–}30\text{ ms}$ fork/exec) | Lowest latency ($2\text{–}5\text{ ms}$) | Poor ($5\times\text{–}10\times$ slower decoding) | Moderate (network/IPC serialization) |
| **Deployment Complexity** | Low (bundle binary in Docker; optional on host) | High (shared library dependencies) | Low (single static binary) | High (requires multi-container compose) |
| **Codec Coverage** | **Universal** (all ffmpeg formats) | **Universal** | Severely limited (MP4/H.264 only) | **Universal** |

### 6.1 Recommended Architecture: Option A (Sandboxed CLI Execution)

Nyx adopts **Option A (CLI Sub-Process Execution)** with security sandboxing and resource isolation:
- **Sandbox Protocol Whitelist**: `-protocol_whitelist "file,pipe"` ensures FFmpeg cannot access network URLs, HTTP streams, or local loopback sockets during media processing.
- **Strict Execution Timeout**: Wrapped in `context.WithTimeout(ctx, 15*time.Second)` to terminate hanging processes.
- **Bounded Concurrency Semaphore**: Throttled by `semaphore.NewWeighted(int64(max(2, runtime.NumCPU()/2)))`.
- **Output Stream Bounding**: Stdout piped through `io.LimitReader(stdout, 20 << 20)` ($20\text{ MB}$ limit) to prevent memory allocation bombs from corrupt streams.

---

## 7. Media Metadata Extraction Architecture

### 7.1 Unified Metadata Extraction Pipeline

```
+─────────────────────────────────────────────────────────────────────────────+
|                         UNIFIED METADATA PIPELINE                           |
+──────────────────────────────────────┬──────────────────────────────────────+
|            IMAGE ASSETS              |             VIDEO ASSETS             |
+──────────────────────────────────────┼──────────────────────────────────────+
| • EXIF Tags:                         | • Container Metadata:                |
|   - Make, Model, Lens                |   - Format name, Duration (seconds)  |
|   - Focal Length, F-Number, ISO      |   - Bitrate, Creation timestamp      |
|   - Shutter Speed, Orientation       | • Video Stream Metadata:             |
|   - DateTimeOriginal, Timezone Offset|   - Codec (H264, HEVC, VP9, AV1)     |
| • GPS Geolocation:                   |   - Width, Height, Aspect Ratio      |
|   - Latitude, Longitude, Altitude    |   - Frame Rate (r_frame_rate)        |
|   - Geohash (for geographic search)  |   - Rotation tag (90°, 180°, 270°)   |
| • IPTC & XMP Metadata:               | • Audio Stream Metadata:             |
|   - Copyright, Creator, Headline     |   - Codec, Sample Rate, Channels     |
|   - Tags, Keywords, User Rating      | • Subtitle Streams:                  |
|                                      |   - Language tags, Formats (SRT, ASS)|
+──────────────────────────────────────┴──────────────────────────────────────+
```

### 7.2 REST API Metadata Endpoint

Nyx introduces a dedicated metadata inspection endpoint:
- **Route**: `GET /api/resources/metadata/{path:.*}`
- **Response Format**:
  ```json
  {
    "type": "image",
    "mime": "image/jpeg",
    "width": 6000,
    "height": 4000,
    "aspectRatio": 1.5,
    "takenAt": "2026-10-14T15:22:31Z",
    "exif": {
      "cameraMake": "Sony",
      "cameraModel": "ILCE-7M4",
      "lens": "FE 24-70mm F2.8 GM II",
      "focalLength": 35.0,
      "fNumber": 2.8,
      "iso": 100,
      "exposureTime": "1/250"
    },
    "geo": {
      "latitude": 37.7749,
      "longitude": -122.4194,
      "altitude": 16.5
    }
  }
  ```

---

## 8. Multi-Tier Caching & Invalidation Architecture

```
+─────────────────────────────────────────────────────────────────────────────+
|                       MULTI-TIER CACHE ARCHITECTURE                         |
+─────────────────────────────────────────────────────────────────────────────+
|                                                                             |
|  TIER 1: In-Memory RAM LRU (`hashicorp/golang-lru/v2`)                      |
|  • Size: 128 MB (holds ~8,000 WebP thumbnails)                              |
|  • Key: sha256(RealPath + ":" + Size)                                       |
|  • Zero-GC: Stores compressed binary byte slices directly                   |
|                                                                             |
|  TIER 2: Persistent On-Disk Cache (`diskcache/`)                            |
|  • Quota: Configurable `NYX_CACHE_DIR_MAX_SIZE` (default 2 GB)              |
|  • Layout: Two-level SHA-1 directory sharding (`hash[0:1]/hash[1:3]/hash`)  |
|  • Permissions: Fixed mode 0644 (replaces buggy 0700 executable mode)       |
|  • Concurrency: 256-striped fixed mutex pool (eliminates memory leak)       |
|  • Eviction: Background worker purges LRU files from 90% down to 75% quota  |
|                                                                             |
|  TIER 3: HTTP Client & Reverse Proxy Caching                                |
|  • Headers:                                                                 |
|    - `Cache-Control: private, max-age=86400, stale-while-revalidate=3600`    |
|    - `ETag: W/"<sha256(RealPath:ModTime:Size)>"`                            |
|  • Conditional Requests: If client `If-None-Match` matches, return HTTP 304 |
|                                                                             |
|  DECOUPLED INVALIDATION HOOKS                                               |
|  • PUT /api/resources: Pass fileCache to resourcePutHandler; call delThumbs |
|  • PATCH /api/tus: Invalidate cached thumbnails on chunked upload complete  |
|  • DELETE /api/resources: Recursive tree purge for deleted directories      |
|  • PATCH /api/resources (rename/move): Invalidate or re-key cache entries   |
+─────────────────────────────────────────────────────────────────────────────+
```

---

## 9. Verification & Acceptance Checkpoints

1. **Memory Allocation Guardrail**:
   - Verify that requests for images $> 12,000 \times 12,000$ return HTTP 413.
   - Verify that decompression memory is capped at $128\text{ MB}$ per slot.
2. **Video Poster Generation**:
   - Verify that requesting `/api/preview/thumb/sample.mp4` returns a valid WebP poster image (HTTP 200) instead of HTTP 501.
3. **Disk Cache Permissions & Memory Leak Fix**:
   - Inspect files in `diskcache/`; verify Unix mode is `0644`.
   - Verify that repeated requests to unique keys do not grow memory in `scopedLocks`.
4. **ETag & 304 Not Modified**:
   - Send `If-None-Match` with preview ETag; verify response is HTTP 304 with 0-byte body.
5. **Decoupled Invalidation on PUT and Directory DELETE**:
   - Overwrite an existing image via `PUT /api/resources/photo.jpg`. Verify cached thumbnail is purged.
   - Delete a parent folder containing images via `DELETE /api/resources/album`. Verify all child thumbnails are removed from `diskcache/`.
