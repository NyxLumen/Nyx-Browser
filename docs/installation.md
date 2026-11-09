# Installation

Nyx Browser is a single binary and can be used as a standalone executable. It is also available as a Docker container image. Installation and first-time setup are straightforward across platforms.

## Binary

You can download the compiled `nyx` binary from the [releases page](https://github.com/NyxLumen/Nyx-Browser/releases) or build from source.

### Standalone Execution

Once downloaded or compiled, execute `nyx` directly:

```sh
# Run Nyx serving files from /path/to/your/files
nyx -r /path/to/your/files
```

Nyx Browser is now up and running. Read the ["First Boot"](#first-boot) section for more information.

## Docker

Nyx Browser is available as two container configurations: a [bare Alpine image](#bare-alpine-image) and an [S6 Overlay image](#s6-overlay-image).

### Bare Alpine Image

```sh
docker run \
    -v nyx_data:/srv \
    -v nyx_database:/database \
    -v nyx_config:/config \
    -p 8080:80 \
    nyxlumen/nyx-browser:latest
```

Where `nyx_data`, `nyx_database`, and `nyx_config` are Docker [volumes](https://docs.docker.com/engine/storage/volumes/) storing application data, database, and configuration respectively. The default configuration and database will be automatically initialized.

The default user running inside the container has UID 1000 and GID 1000. If you wish to run the container with a different user, consult Docker's [user documentation](https://docs.docker.com/engine/containers/run/#user).

> [!NOTE]
>
> When using [bind mounts](https://docs.docker.com/engine/storage/bind-mounts/) (mounting a host directory into the container), ensure that the host directories have appropriate **permissions**. The host directories must be readable and writable by the container user (UID/GID 1000). You can use [`chown`](https://linux.die.net/man/1/chown) to set ownership.

Nyx Browser is now up and running. Read the ["First Boot"](#first-boot) section for more information.

### S6 Overlay Image

The `s6` image leverages the [s6-overlay](https://github.com/just-containers/s6-overlay) system for process supervision and customizable UID/GID mapping:

```shell
docker run \
    -v /path/to/srv:/srv \
    -v /path/to/database:/database \
    -v /path/to/config:/config \
    -e PUID=$(id -u) \
    -e PGID=$(id -g) \
    -p 8080:80 \
    nyxlumen/nyx-browser:s6
```

Where:
- `/path/to/srv` contains the files root directory for Nyx Browser
- `/path/to/config` contains a `settings.json` file
- `/path/to/database` contains a `nyx.db` file (with fallback to `filebrowser.db`)

Both `settings.json` and `nyx.db` will automatically be initialized if they do not exist.

Nyx Browser is now up and running. Read the ["First Boot"](#first-boot) section for more information.

## First Boot

On first boot, Nyx Browser automatically bootstraps a database (`./nyx.db` by default) containing configuration and user data. The address where your instance is running and the randomly generated password for the `admin` account will be displayed in the console logs.

> [!WARNING]
>
> The automatically generated password for `admin` is only displayed once in the logs on initial setup.

To customize your installation and configure security settings, check [`config init`](cli/nyx-config-init.md) and [`config set`](cli/nyx-config-set.md).

For production deployment and network hardening, see the [deployment](deployment.md) guide.
