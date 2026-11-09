# Docker Build Reproducibility & Dependency Investigation: `JSON.sh`

**Milestone**: NYX FOUNDATION Phase 01.1 — Requirement R5  
**Document Author**: Teamwork Engineering  
**Target Repository**: `Nyx-Browser`  
**Date**: November 9, 2026  

---

## 1. Executive Summary

This investigation evaluates the container build dependency on `JSON.sh` in the minimal Alpine/BusyBox container build (`Dockerfile` and `docker/alpine/healthcheck.sh`). We examine:
1. The architectural role of `JSON.sh` in the Nyx container ecosystem.
2. Whether "offline Docker builds" are an actual project requirement.
3. The licensing, maintenance status, and code provenance of `dominictarr/JSON.sh`.
4. The engineering trade-offs of vendoring versus remote pinned fetching.

**Core Recommendation**: In alignment with the project directive to *"prefer the simplest maintainable solution; do not vendor code unnecessarily"*, Nyx Browser should **retain the pinned remote fetch** (`0d5e5c77365f63809bf6e77ef44a1f34b0e05840`) in `Dockerfile`. Vendoring is unnecessary repository bloat because the Docker build process inherently requires active network connectivity to pull base images and Alpine package repositories, while the remote script is cryptographically pinned by commit SHA to an upstream repository that has been frozen since December 2018. Furthermore, `docker/alpine/healthcheck.sh` provides deterministic short-circuit evaluation via `NYX_PORT`, `FB_PORT`, and `FILEBROWSER_PORT` overrides.

---

## 2. Background: Role of `JSON.sh` in Nyx Container Builds

Across the Nyx Browser codebase, `JSON.sh` is referenced in exactly two locations:

1. **`Dockerfile:7`**:
   ```dockerfile
   # Stage 1: Fetcher
   FROM alpine:3.23 AS fetcher
   RUN apk update && \
       apk --no-cache add ca-certificates mailcap tini-static && \
       wget -O /JSON.sh https://raw.githubusercontent.com/dominictarr/JSON.sh/0d5e5c77365f63809bf6e77ef44a1f34b0e05840/JSON.sh
   ...
   # Stage 2: Runtime
   FROM busybox:1.37.0-musl
   ...
   COPY --from=fetcher /JSON.sh /JSON.sh
   ```
2. **`docker/alpine/healthcheck.sh:5-7`**:
   ```sh
   PORT=${NYX_PORT:-${FB_PORT:-${FILEBROWSER_PORT:-$(cat /config/settings.json | sh /JSON.sh | grep '\["port"\]' | awk '{print $2}')}}}
   ADDRESS=${NYX_ADDRESS:-${FB_ADDRESS:-${FILEBROWSER_ADDRESS:-$(cat /config/settings.json | sh /JSON.sh | grep '\["address"\]' | awk '{print $2}' | sed 's/"//g')}}}
   ADDRESS=${ADDRESS:-localhost}
   ```

### Why `JSON.sh` is Needed in the BusyBox Image
The primary runtime image (`Dockerfile`) uses `busybox:1.37.0-musl` to maintain a minimal container attack surface and footprint (~15 MB). BusyBox provides standard POSIX utilities (`cat`, `awk`, `grep`, `sed`), but deliberately excludes `jq` or Python runtimes.

During container health checks, if port and address environment variables are not specified, the healthcheck script must read the listening address and port from `/config/settings.json`. `JSON.sh` transforms JSON structures into newline-delimited key-value paths (e.g., `["port"] 80`), enabling standard shell utilities to extract settings without compiling or packaging additional C binaries.

By contrast, the secondary `Dockerfile.s6` image builds on LinuxServer's Alpine base (`ghcr.io/linuxserver/baseimage-alpine:3.23`), which includes Alpine's package manager (`apk`). It installs native `jq` (`apk add jq`) and uses `docker/common/healthcheck.sh` (`jq -r .port /config/settings.json`), never invoking or referencing `JSON.sh`.

---

## 3. Analysis of Offline Build Feasibility

An audit observation previously raised whether remote fetching of `JSON.sh` during `docker build` prevents offline builds.

Our empirical investigation reveals:
1. **Base Image Dependencies**:
   - `Dockerfile` requires fetching `alpine:3.23` and `busybox:1.37.0-musl` from Docker Hub registry.
   - `Dockerfile.s6` requires fetching `ghcr.io/linuxserver/baseimage-alpine:3.23` from GitHub Container Registry.
2. **Alpine Package Dependencies**:
   - `Dockerfile:5` executes `apk update && apk --no-cache add ca-certificates mailcap tini-static`. This strictly requires network connectivity to `dl-cdn.alpinelinux.org`.
   - `Dockerfile.s6:3-4` executes `apk update && apk --no-cache add ca-certificates mailcap jq libcap`. This also requires live network access.
3. **Project Requirements**:
   - Neither `PROJECT.md`, `NYX-PROJECT-CONTEXT.md`, nor `DEPLOYMENT.md` specifies offline Docker builds as a requirement. Container images are compiled in automated CI environments (`.github/workflows/ci.yaml`) with full internet access.
   - Removing the `wget /JSON.sh` command would **not** enable offline builds, as the preceding `apk update` fails immediately without an active network connection or local package mirror.

**Conclusion**: "Offline Docker builds" are neither an existing capability nor a project requirement.

---

## 4. Provenance, Licensing & Maintenance Status

Direct analysis of upstream `dominictarr/JSON.sh` at commit `0d5e5c77365f63809bf6e77ef44a1f34b0e05840`:

- **Repository**: `https://github.com/dominictarr/JSON.sh`
- **Pinned Commit**: `0d5e5c77365f63809bf6e77ef44a1f34b0e05840` (December 23, 2018)
- **Commit Status**: This commit is the exact `HEAD` of the upstream `master` branch. The repository has been dormant and frozen since late 2018.
- **Footprint**: A single, dependency-free POSIX shell script consisting of **208 lines** (4,809 bytes, ~4.8 KB).
- **Licensing**: Upstream is dual-licensed under:
  1. **Apache License 2.0** (`LICENSE.APACHE2`)
  2. **MIT License** (`LICENSE.MIT`)
- **Compatibility**: Both Apache 2.0 and MIT licenses are completely compatible with Nyx Browser's Apache License 2.0.

Because the script is immutable, small, and frozen, there is no upstream churn or breaking changes to anticipate.

---

## 5. Architectural Comparison: Remote Fetch vs. Vendoring

| Criteria | Remote Fetch (Current Implementation) | Vendored (`docker/alpine/JSON.sh`) |
| :--- | :--- | :--- |
| **Simplicity** | High: No files checked into git repository | Moderate: Requires tracked asset in repository |
| **Reproducibility** | Cryptographically pinned to commit `0d5e5c7...` | Stored directly in git tree |
| **Build Network Need** | Requires access to `raw.githubusercontent.com` | Eliminates GitHub raw fetch; still needs Alpine mirror |
| **Repository Clutter** | Zero repository footprint | Adds 208 lines (~4.8 KB) of third-party shell script |
| **Maintenance** | None required | None required (upstream is frozen) |
| **Project Directive** | Fully satisfies *"do not vendor code unnecessarily"* | Deviates from preference against unnecessary vendoring |

---

## 6. Environment Variable Short-Circuit in Healthchecks

In Phase 01.1 Milestone 1, `docker/alpine/healthcheck.sh` was enhanced to support canonical `NYX_` variables and backward-compatible fallbacks:

```sh
PORT=${NYX_PORT:-${FB_PORT:-${FILEBROWSER_PORT:-$(cat /config/settings.json | sh /JSON.sh | grep '\["port"\]' | awk '{print $2}')}}}
ADDRESS=${NYX_ADDRESS:-${FB_ADDRESS:-${FILEBROWSER_ADDRESS:-$(cat /config/settings.json | sh /JSON.sh | grep '\["address"\]' | awk '{print $2}' | sed 's/"//g')}}}
```

Due to POSIX shell parameter expansion (`${VAR:-DEFAULT}`):
- If `NYX_PORT`, `FB_PORT`, or `FILEBROWSER_PORT` is set, the default expansion command `$(cat ... | sh /JSON.sh ...)` is **never executed**.
- If `NYX_ADDRESS`, `FB_ADDRESS`, or `FILEBROWSER_ADDRESS` is set, `JSON.sh` is bypassed for address resolution.
- `JSON.sh` is only invoked when running with file-based `settings.json` without container environment variable overrides.

---

## 7. Final Recommendation & Implementation Guidance

1. **Retain Pinned Remote Fetch**:
   Keep `Dockerfile` fetching `JSON.sh` pinned to commit `0d5e5c77365f63809bf6e77ef44a1f34b0e05840`. This satisfies Requirement R5:
   - Avoids unnecessary vendoring.
   - Pinned commit SHA guarantees reproducible container builds.
   - Acknowledges that base image and `apk` updates already mandate network access.

2. **Network Troubleshooting Guidance**:
   If local Docker daemon environments encounter IPv4 DNS timeouts connecting to `raw.githubusercontent.com` on Docker bridge networks, builders can pass `--network=host` or configure daemon DNS (`/etc/docker/daemon.json`), which also resolves Alpine package mirror queries.

3. **Vendoring Contingency**:
   If the maintainers decide at a future milestone that complete disconnection from GitHub domain infrastructure during container assembly is desirable, vendoring `JSON.sh` to `docker/alpine/JSON.sh` is legally straightforward under Apache 2.0 with minimal code size (~4.8 KB). However, for Phase 01.1, the simplest and cleanest maintainable solution is to keep the pinned remote fetch.
