# Nyx Browser

<p align="center">
  <img src="./branding/logo-lockup.svg" width="400" alt="nyxbrowser" />
</p>

Nyx Browser is an independent, self-hosted file management and media browsing platform built from a hardened Go and Vue 3 foundation. Designed for local network environments and home servers, it pairs a full-featured, desktop-grade file manager with an upcoming dedicated visual media gallery for photos and videos, wrapped in a minimal dark aesthetic.

## Quickstart

### Single Binary

Download the compiled `nyx` binary for your platform or build from source, then start serving files:

```bash
# Run Nyx serving files from /path/to/files on port 8080 with default database ./nyx.db
./nyx -r /path/to/files -p 8080 -d ./nyx.db
```

On first boot, Nyx automatically bootstraps `./nyx.db` and outputs an initial administrative login credential to the console.

### Docker Container

Run Nyx Browser using Docker:

```bash
docker run -d \
  --name nyx \
  -p 8080:80 \
  -v /path/to/files:/srv \
  -v /path/to/nyx.db:/nyx.db \
  nyxlumen/nyx-browser:latest
```

### Configuration

Nyx Browser supports flexible configuration via CLI flags, environment variables, config files, and the internal Bolt database. Configuration evaluation follows a strict precedence hierarchy:

1. **CLI Flags** (e.g. `--port`, `--database`, `--root`)
2. **`NYX_*` Environment Variables** (e.g. `NYX_PORT`, `NYX_DATABASE`, `NYX_ROOT`)
3. **`FB_*` Legacy Environment Variables** (e.g. `FB_PORT`, `FB_DATABASE`)
4. **`FILEBROWSER_*` Compatibility Environment Variables**
5. **Configuration Files**: Discovered in `./`, `$HOME/`, `/etc/nyx/`, or `/etc/filebrowser/` as `.nyx.{yaml,yml,json,toml}` (with fallback to `.filebrowser.{...}`)
6. **Database Values & Defaults**: Default database `./nyx.db` (with non-destructive fallback to `./filebrowser.db`)

## Security

Nyx Browser is actively maintained with ongoing security hardening:
- **Proxy Authentication Hardening (`NYX-SEC-001`)**: Patched reverse proxy header validation to prevent unauthenticated bypass.
- **Share Link Expiration (`NYX-SEC-002`)**: Patched expiration handling to eliminate race conditions during share query mutations.
- **Filesystem Confinement**: Strict scoped filesystem guards enforcing path traversal boundaries and symlink confinement.
- **Command Runner**: The command runner feature is disabled by default (`--disable-exec=true`).
- **Vulnerability Reporting**: For security concerns, please consult [SECURITY.md](SECURITY.md) and report privately via GitHub Security Advisories at [https://github.com/nyxlumen/nyx-browser/security](https://github.com/nyxlumen/nyx-browser/security).

## Documentation

Full documentation lives in the [`docs`](docs) directory:
- [Installation Guide](docs/installation.md)
- [Customization](docs/customization.md)
- [Deployment](docs/deployment.md)
- [Troubleshooting](docs/troubleshooting.md)
- [CLI Reference](docs/cli/nyx.md)
- [Contributing & Build Guide](CONTRIBUTING.md)

## License & Attribution

Nyx Browser is licensed under the [Apache License 2.0](LICENSE).  
Copyright 2026 Nyx Lumen and contributors.

This product is derived from File Browser (https://github.com/filebrowser/filebrowser), licensed under the Apache License, Version 2.0. Copyright 2018 File Browser Contributors. See [NOTICE](NOTICE) for complete attribution and historical copyright details.
