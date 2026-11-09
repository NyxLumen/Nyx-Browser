# Nyx Browser Deployment & Operations Guide

Welcome to the **Nyx Browser** Deployment & Operations Guide. This guide covers clean installations, Docker deployments, migration from legacy File Browser, permission models, security best practices, and production configurations.

---

## 1. System Architecture & Requirements

Nyx Browser is designed as a single, lightweight binary with embedded web assets, backed by an embedded database.

### Core Requirements
- **OS**: Linux (amd64, arm64, armv7), macOS (Apple Silicon / Intel), Windows (amd64).
- **RAM**: Minimum 64 MB (128 MB recommended; additional RAM recommended if preview downscaling large photo libraries).
- **Disk**: Minimal disk footprint for the binary (~30 MB). Dedicated storage for data files (`/srv`) and database (`/database`).
- **Network**: Single listening HTTP/HTTPS port (default `:80` in container, `:8080` in standalone CLI).

---

## 2. Clean Installation

### Option A: Standalone Binary (Linux / macOS / Windows)

1. **Download or Build the Binary**:
   Compile the binary using Go 1.27+:
   ```bash
   go build -o nyx main.go
   sudo mv nyx /usr/local/bin/
   ```

2. **Initialize Configuration and Database**:
   Create a dedicated system user and directories:
   ```bash
   sudo useradd -r -s /bin/false -U nyx
   sudo mkdir -p /etc/nyx /var/lib/nyx /srv/files
   sudo chown -R nyx:nyx /var/lib/nyx /srv/files /etc/nyx
   ```

3. **Initialize Database**:
   ```bash
   sudo -u nyx nyx config init -d /var/lib/nyx/nyx.db
   sudo -u nyx nyx users add admin <SecurePassword> --perm.admin -d /var/lib/nyx/nyx.db
   ```

4. **Create Configuration File (`/etc/nyx/.nyx.yaml`)**:
   ```yaml
   port: 8080
   address: "127.0.0.1"
   root: "/srv/files"
   database: "/var/lib/nyx/nyx.db"
   log: "stdout"
   ```

5. **Create Systemd Service (`/etc/systemd/system/nyx.service`)**:
   ```ini
   [Unit]
   Description=Nyx Browser File Management Platform
   After=network.target

   [Service]
   User=nyx
   Group=nyx
   ExecStart=/usr/local/bin/nyx -c /etc/nyx/.nyx.yaml
   Restart=on-failure
   RestartSec=5s
   AmbientCapabilities=CAP_NET_BIND_SERVICE
   NoNewPrivileges=true
   ProtectSystem=full
   ProtectHome=true

   [Install]
   WantedBy=multi-user.target
   ```

   Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now nyx
   ```

---

### Option B: Docker Container

Nyx Browser provides two official Docker container flavours:
- **Standard (Alpine/BusyBox + Tini)**: Minimal, rootless, non-root user `user` (UID 1000, GID 1000).
- **S6-Overlay (`Dockerfile.s6`)**: For dynamic UID/GID environments (compatible with LinuxServer.io conventions).

#### Quick Run
```bash
docker run -d \
  --name nyx \
  -p 8080:80 \
  -v /local/storage:/srv \
  -v /local/data:/database \
  -v /local/config:/config \
  nyxlumen/nyx-browser:latest
```

---

## 3. Docker Compose Production Setups

### A. Standalone Production Compose (`compose.yaml`)

```yaml
services:
  nyx:
    container_name: nyx
    image: nyxlumen/nyx-browser:latest
    restart: unless-stopped
    user: "1000:1000"
    ports:
      - "127.0.0.1:8080:80"
    volumes:
      - ./data/srv:/srv
      - ./data/database:/database
      - ./data/config:/config
    environment:
      - NYX_PORT=80
      - NYX_ADDRESS=0.0.0.0
      - NYX_DATABASE=/database/nyx.db
      - NYX_ROOT=/srv
    healthcheck:
      test: ["CMD", "/healthcheck.sh"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s
```

### B. High-Concurrency Setup with Redis Upload Cache & Caddy TLS Proxy

When running multi-chunk TUS uploads or multiple replicas, configure Redis for upload session tracking:

```yaml
services:
  caddy:
    image: caddy:2-alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - nyx

  nyx:
    container_name: nyx
    image: nyxlumen/nyx-browser:latest
    restart: unless-stopped
    user: "1000:1000"
    expose:
      - "80"
    volumes:
      - ./storage:/srv
      - ./database:/database
      - ./config:/config
    environment:
      - NYX_REDIS_CACHE_URL=redis://default:SecureRedisPass@redis:6379
      - NYX_DATABASE=/database/nyx.db
      - NYX_ROOT=/srv
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: ["redis-server", "--requirepass", "SecureRedisPass"]
    volumes:
      - redis_data:/data

volumes:
  caddy_data:
  caddy_config:
  redis_data:
```

#### Example `Caddyfile`:
```caddy
files.example.com {
    encode zstd gzip

    # Reverse proxy to Nyx
    reverse_proxy nyx:80 {
        # Strip spoofed proxy auth headers from untrusted clients
        header_up -X-Forwarded-User
        header_up -Remote-User
    }
}
```

---

## 4. Upgrade & Migration from Legacy File Browser

Nyx Browser is built with seamless, non-destructive backward compatibility for File Browser installations.

### 1. Database Compatibility
- **Default Database**: Nyx uses `./nyx.db` by default.
- **Automatic Fallback**: If `./nyx.db` does not exist and `./filebrowser.db` is present in the target directory, Nyx opens `./filebrowser.db` directly.
- **Zero Silent Mutation**: Nyx never renames, converts, or destroys an existing `filebrowser.db`.
- **Migration**: To migrate to the new standard name without data loss, simply copy the database:
  ```bash
  cp /path/to/filebrowser.db /path/to/nyx.db
  ```

### 2. Configuration Precedence
Nyx checks configuration sources in strict order:
1. **CLI Flags** (e.g. `--port 8080`)
2. **`NYX_` Environment Variables** (e.g. `NYX_PORT=8080`)
3. **`FB_` Legacy Environment Variables** (e.g. `FB_PORT=8080`) — used if `NYX_` is unset
4. **`.nyx.{json,yaml,yml,toml}` Config Files**
5. **`.filebrowser.{json,yaml,yml,toml}` Legacy Config Files** — used if `.nyx.*` is absent
6. **Database Stored Settings**
7. **Built-in Defaults**

### 3. Binary & Symlink Compatibility
- The primary binary is named `nyx`.
- In Docker containers, `/bin/filebrowser` is symlinked to `/bin/nyx`, ensuring legacy scripts calling `filebrowser` continue to work without modification.

---

## 5. Permissions & Security Best Practices

### 1. Filesystem Permissions
Ensure storage and database folders are owned by the user running Nyx:
```bash
# Host directories for container UID 1000:
sudo chown -R 1000:1000 ./data/srv ./data/database ./data/config
chmod 750 ./data/database
chmod 640 ./data/database/nyx.db
```

### 2. Scoped Filesystem & External Symlinks
- By default, Nyx wraps all user filesystems in `files.ScopedFs`. Any symbolic link pointing outside the user's scope is strictly forbidden from reads, writes, stats, and deletions.
- **Do not enable `--follow-external-symlinks`** in multi-tenant or untrusted environments. Only enable it if you intentionally require symlinks to resolve outside the server root in a single-user trusted environment.

### 3. Command Execution (`runner/`)
- Command execution is disabled by default (`enableExec: false`).
- Only administrators can configure commands in `Settings`.
- Regular users cannot run commands unless explicitly granted the `Execute` permission bit and assigned allowed commands in `user.Commands`.
- When using command hooks, prefer referencing environment variables (`"$FILE"`, `"$SCOPE"`) in scripts rather than inline shell expansion.

### 4. Reverse Proxy Header Hygiene
When configuring `proxy` authentication (`auth.method: "proxy"`):
- Always bind Nyx to `127.0.0.1` or deploy inside an internal Docker network.
- Ensure your reverse proxy (Nginx, Caddy, Traefik) clears incoming client headers before injecting the authenticated user header (`X-Forwarded-User` or `Remote-User`).

---

## 6. Healthchecks & Operational Monitoring

Nyx exposes a lightweight healthcheck endpoint at `/health` returning HTTP 200 OK.

- **Curl check**:
  ```bash
  curl -f http://127.0.0.1:8080/health || exit 1
  ```
- **Container healthcheck**:
  The official container includes `/healthcheck.sh`, which automatically detects configured ports and addresses from environment variables or `/config/settings.json`.
