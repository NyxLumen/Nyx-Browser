# Security Policy

## Supported Versions

Nyx Browser is actively maintained. Security patches and hardening improvements are applied to the active release stream.

| Version | Supported |
| ------- | --------- |
| Phase 01 / v2.x | :white_check_mark: |
| Legacy File Browser (< 2.0) | :x: |

## Security Hardening History

Recent security hardening and vulnerability remediations in Nyx Browser include:

- **NYX-SEC-001**: Reverse proxy authentication bypass remediation in `auth/proxy.go`, properly validating trusted header sources and preventing unauthenticated spoofing.
- **NYX-SEC-002**: Share link expiration slice mutation remediation in `share/storage.go`, eliminating race conditions and unintended persistence of expired public shares.
- **Filesystem Sandboxing**: Strict scoped filesystem path traversal guards and symlink confinement in `files/scoped.go`.

## Reporting a Vulnerability

If you discover a security vulnerability in Nyx Browser, please report it responsibly:

- Submit a private vulnerability report via [GitHub Security Advisories](https://github.com/nyxlumen/nyx-browser/security/advisories) or repository issue intake at [https://github.com/nyxlumen/nyx-browser/security](https://github.com/nyxlumen/nyx-browser/security).
- Please do not disclose vulnerabilities publicly until a patch has been coordinated and released.

Please include in your report:

- The commit SHA or release version where the vulnerability was observed
- A plaintext proof of concept (no binaries)
- Clear, step-by-step instructions to reproduce
- Impact assessment and suggested remediation, if available

Security reports are reviewed promptly, and verified vulnerabilities will receive a security advisory and coordinated release.
