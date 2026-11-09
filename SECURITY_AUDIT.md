# Nyx Browser Security Audit & Vulnerability Assessment Report

**Audit Date**: November 2026  
**Auditor**: Teamwork Worker M3 (QA / Security Specialist)  
**Target Repository**: `github.com/nyxlumen/nyx-browser/v2` (Nyx Foundation: Phase 01)  
**Scope**: Backend API authorization, authentication mechanisms, filesystem scoping (`files.ScopedFs`), upload handling, command execution runner, share storage, and container deployment.

---

## Executive Summary

As part of Milestone 3 of the Nyx Foundation campaign, a comprehensive security audit of Nyx Browser was conducted. The audit combined static code analysis, threat modeling, boundary fuzzing, and regression testing across all 19 Go packages and HTTP API handlers.

The review verified that Nyx Browser features robust baseline security architectures—notably the `files.ScopedFs` confinement layer, strict JWT algorithm pinning (`HS256`), and image processing memory bounding. However, two demonstrable defects were identified and patched:
1. **NYX-SEC-001 (Medium/High - Patched)**: Missing or blank proxy header handling in `ProxyAuth` allowed unauthenticated requests to auto-provision empty-username accounts and obtain valid JWT authentication tokens.
2. **NYX-SEC-002 (Low/Medium - Patched)**: Slice mutation during range iteration in `share/storage.go` skipped consecutive expired share links, causing stale links to leak to callers and remain in storage.

Additionally, key architectural hardening considerations were evaluated and documented for operational deployment (reverse proxy header hygiene and command runner shell expansion).

---

## Findings Matrix

| Finding ID | Title | Component | Severity | Status |
|---|---|---|---|---|
| **NYX-SEC-001** | Missing/Blank Proxy Header Auto-Provisioning | `auth/proxy.go` | **Medium/High (CVSS 7.3)** | **FIXED** (with regression tests) |
| **NYX-SEC-002** | Slice Mutation Range Skip in Share Expiration | `share/storage.go` | **Low/Medium (CVSS 4.3)** | **FIXED** (with regression tests) |
| **NYX-SEC-003** | Reverse Proxy Header Spoofing via Direct Socket Exposure | `auth/proxy.go`, `http/auth.go` | **Medium (CVSS 6.5)** | **DOCUMENTED** (Operational Remediation) |
| **NYX-SEC-004** | Hook Command Runner Shell Variable Expansion | `runner/runner.go` | **Medium (CVSS 6.0)** | **DOCUMENTED** (Architectural Remediation) |
| **NYX-SEC-005** | Symlink Directory Traversal Confinement | `files/scoped.go` | Informational (Safe) | **AUDITED & VERIFIED** |
| **NYX-SEC-006** | JWT Token Expiration & Algorithm Confusion Protection | `http/auth.go` | Informational (Safe) | **AUDITED & VERIFIED** |
| **NYX-SEC-007** | Image Previews Decompression Bomb & Resource Bounding | `img/service.go` | Informational (Safe) | **AUDITED & VERIFIED** |

---

## Detailed Findings & Remediations

### NYX-SEC-001: Missing/Blank Proxy Header Auto-Provisioning (PATCHED)

- **Severity**: Medium / High (CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L — Base Score 7.3)
- **Affected File**: `auth/proxy.go`
- **Description**:  
  When Nyx Browser was configured to use the `proxy` authentication method (`settings.AuthMethod = "proxy"`), `ProxyAuth.Auth()` read the username from the configured header:
  ```go
  username := r.Header.Get(a.Header)
  user, err := usr.Get(srv.Root, srv.FollowExternalSymlinks, username)
  if errors.Is(err, fberrors.ErrNotExist) {
      return a.createUser(usr, setting, srv, username)
  }
  ```
  If an unauthenticated HTTP request was dispatched without the proxy header (or with an empty header string), `r.Header.Get(a.Header)` returned `""`. Because no user with an empty username existed, `usr.Get(..., "")` returned `ErrNotExist`. The handler subsequently called `a.createUser(usr, setting, srv, "")`, automatically provisioning a user account with a blank username in the database and returning a valid JWT session token to the unauthenticated caller.
- **Reproduction**:
  1. Configure Nyx with `auth.method: "proxy"` and header `X-Remote-User`.
  2. Send `GET /api/login` without `X-Remote-User` header.
  3. Pre-fix response: HTTP 200 OK with a newly minted JWT token and an empty user record created in BoltDB.
- **Remediation Applied**:
  `auth/proxy.go` was updated to explicitly validate that `a.Header` is configured and that `strings.TrimSpace(r.Header.Get(a.Header))` is non-empty. If either condition fails, `ProxyAuth.Auth()` immediately aborts with `os.ErrPermission`, returning HTTP 403 Forbidden without provisioning any user.
  ```go
  if a.Header == "" {
      return nil, os.ErrPermission
  }
  username := strings.TrimSpace(r.Header.Get(a.Header))
  if username == "" {
      return nil, os.ErrPermission
  }
  ```
- **Regression Verification**:
  Automated tests in `auth/proxy_test.go` (`TestProxyAuthRejectsMissingOrEmptyHeader`) verify that missing headers, empty strings, whitespace-only strings, and empty header configurations return `os.ErrPermission` and create 0 users.

---

### NYX-SEC-002: Slice Mutation Range Skip in Share Expiration (PATCHED)

- **Severity**: Low / Medium (CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N — Base Score 4.3)
- **Affected File**: `share/storage.go`
- **Description**:  
  In `share/storage.go`, the methods `All()`, `FindByUserID()`, and `Gets()` checked each link's expiration timestamp and attempted to delete expired links:
  ```go
  for i, link := range links {
      if link.Expire != 0 && link.Expire <= time.Now().Unix() {
          if err := s.Delete(link.Hash); err != nil {
              return nil, err
          }
          links = append(links[:i], links[i+1:]...)
      }
  }
  return links, nil
  ```
  In Go, iterating over a slice with `for i, link := range links` evaluates the slice boundary once. Reassigning `links = append(links[:i], links[i+1:]...)` shrinks the slice in-place, but the loop counter `i` increments on the next iteration. As a result, if two consecutive expired share links existed in the slice:
  - Link 0 was deleted.
  - Link 1 shifted to index 0.
  - On the next iteration, the loop inspected index 1, completely skipping Link 1.
  Consequently, expired share links were not purged from storage and were returned in the active share links list.
- **Reproduction**:
  1. Insert three share links: two consecutive expired links (`exp1`, `exp2`) and one valid link (`valid`).
  2. Invoke `store.All()`, `store.FindByUserID()`, or `store.Gets()`.
  3. Pre-fix output: returned `[exp2, valid]`, leaving `exp2` unpurged in the database.
- **Remediation Applied**:
  `share/storage.go` was updated to construct a fresh `active` slice, filtering expired links safely without in-place slice mutation:
  ```go
  active := make([]*Link, 0, len(links))
  for _, link := range links {
      if link.Expire != 0 && link.Expire <= time.Now().Unix() {
          if err := s.Delete(link.Hash); err != nil {
              return nil, err
          }
          continue
      }
      active = append(active, link)
  }
  return active, nil
  ```
- **Regression Verification**:
  Automated unit tests in `share/share_test.go` (`TestStorage_ConsecutiveExpiredLinks`) verify that consecutive expired links are purged and never returned.

---

### NYX-SEC-003: Reverse Proxy Header Spoofing Risk (OPERATIONAL HARDENING)

- **Severity**: Medium (CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N — Base Score 6.5)
- **Affected Subsystem**: `auth/proxy.go`, `http/auth.go`
- **Description**:  
  When using `proxy` authentication, Nyx trusts the upstream HTTP header (e.g. `X-Forwarded-User` or `Remote-User`) specified in its settings. If the Nyx service is bound to `0.0.0.0` or otherwise reachable directly by clients (bypassing the authenticating reverse proxy), an attacker can forge this header and impersonate any user.
- **Architectural Analysis**:  
  Nyx does not maintain an embedded CIDR whitelist for proxy IP validation; it delegates network perimeter isolation to the deployment topology.
- **Operational Remediation**:
  1. **Bind to Localhost or Private Docker Network**:
     Always bind Nyx to `127.0.0.1` or deploy within an isolated Docker network where only the reverse proxy (e.g., Nginx, Caddy, Traefik, Authelia) can communicate with port 80.
  2. **Header Sanitization**:
     Ensure the reverse proxy unconditionally strips incoming client-supplied `X-Forwarded-User` and `Remote-User` headers before forwarding requests.

---

### NYX-SEC-004: Hook Command Runner Shell Variable Expansion (ARCHITECTURAL RECOMMENDATION)

- **Severity**: Medium (CVSS:3.1/AV:L/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H — Base Score 6.0)
- **Affected Subsystem**: `runner/runner.go`
- **Description**:  
  The command runner allows administrators to define command hooks (e.g. `before_upload`, `after_save`). In `runner/runner.go`, when `Settings.Shell` is configured (e.g. `["sh", "-c"]`), the command string is expanded via `os.Expand` before execution:
  ```go
  for i, arg := range command {
      if i == 0 { continue }
      command[i] = os.Expand(arg, envMapping)
  }
  ```
  If an administrator configures a shell hook containing raw string interpolation (e.g. `echo $FILE`), and a file is uploaded with shell metacharacters in its name (e.g. `foo; rm -rf /; bar.txt`), the string expansion creates a shell command injection risk.
- **Remediation & Best Practices**:
  1. `runner.go` also injects these values directly into the subprocess environment (`FILE`, `SCOPE`, `TRIGGER`, `USERNAME`, `DESTINATION`).
  2. Administrators configuring hook scripts in `Settings.Commands` should use shell environment variables quoted (e.g. `sh -c 'process "$FILE"'`) or invoke standalone scripts (`/usr/local/bin/hook.sh`) rather than inline string expansion.
  3. Non-admin users are strictly prevented from modifying runner commands or executing arbitrary binaries (`user.Perm.Execute` is default false and restricted).

---

### NYX-SEC-005: Filesystem Confinement & Dangling Symlink Escape Defense (VERIFIED)

- **Status**: Audited and Verified
- **Component**: `files/scoped.go`, `files/file.go`
- **Audit Findings**:
  Nyx Browser enforces filesystem boundaries through `files.ScopedFs`. The implementation wraps `afero.BasePathFs` and implements a rigorous `within()` check on all operations (`Open`, `Create`, `OpenFile`, `Stat`, `Remove`, `RemoveAll`, `Rename`, `Mkdir`, `Chmod`):
  - Normalizes root and resolved paths using `filepath.EvalSymlinks`.
  - Enforces prefix matching with trailing path separators, preventing sibling directory escapes (e.g., `/srvother` accessing `/srv`).
  - Resolves dangling symlinks iteratively (up to `maxSymlinkHops = 255`) relative to their real containing directories to prevent dangling-symlink write escapes.
  - Regression coverage verified in `files/file_test.go` and `http/resource_test.go`.

---

### NYX-SEC-006: JWT Session Handling & Cryptographic Security (VERIFIED)

- **Status**: Audited and Verified
- **Component**: `http/auth.go`, `settings/settings.go`
- **Audit Findings**:
  - **Key Generation**: 512-bit (64-byte) cryptographic random keys generated via `crypto/rand` and stored securely in BoltDB.
  - **Algorithm Enforcement**: The JWT parser explicitly pins allowed methods to `HS256` (`jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})`), preventing `none`-algorithm and RS/HS confusion attacks.
  - **Expiration**: `jwt.WithExpirationRequired()` is enforced. Tokens expire after 2 hours (or configured duration).
  - **Revocation / Invalidation**: User updates flag `X-Renew-Token: true` when `IssuedAt` precedes the user's `LastUpdate` timestamp.
  - Regression coverage verified in `http/api_security_test.go`.

---

### NYX-SEC-007: Image Previews Resource Bounding (VERIFIED)

- **Status**: Audited and Verified
- **Component**: `img/service.go`
- **Audit Findings**:
  - Image dimension limits are strictly enforced: `MaxImageWidth = 10000`, `MaxImageHeight = 10000`. Images exceeding these dimensions immediately fail with `ErrImageTooLarge` before memory allocation for decoding.
  - Worker semaphore limits concurrent image downscaling pipelines to avoid CPU/memory starvation.
  - Corrupt or malicious image payloads fail gracefully with `ErrUnsupportedFormat`.
  - Regression coverage verified in `img/service_test.go`.

---

## Security Verification Checklist

- [x] All 19 Go packages compile and pass automated tests: `go test -count=1 ./...`
- [x] Go race detector reports 0 data races: `go test -race ./...`
- [x] Path traversal boundary escapes blocked by `files.ScopedFs`
- [x] Non-admin privilege escalation via API blocked (`PUT /api/users/{self}`)
- [x] Missing/empty proxy auth headers rejected (`auth/proxy.go`)
- [x] Consecutive expired share link deletion verified without slice mutation bugs
- [x] JWT expiration, algorithm pinning, and signature verification covered by tests
