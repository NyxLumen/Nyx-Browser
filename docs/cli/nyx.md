# nyx

A stylish web-based file and media browser

## Synopsis

Nyx Browser CLI lets you create the database to use with Nyx Browser,
manage your users and all the configurations without accessing the
web interface.

If you've never run Nyx Browser, you'll need to have a database for
it. Don't worry: you don't need to setup a separate database server.
We're using Bolt DB which is a single file database and all managed
by ourselves.

For this command, all flags are available as environmental variables,
except for "--config", which specifies the configuration file to use.
The environment variables are prefixed by "NYX_" (with fallback to legacy "FB_"
and "FILEBROWSER_") followed by the flag name in UPPER_SNAKE_CASE. For example, the flag
"--disablePreviewResize" is available as NYX_DISABLE_PREVIEW_RESIZE (or FB_DISABLE_PREVIEW_RESIZE / FILEBROWSER_DISABLE_PREVIEW_RESIZE).

If "--config" is not specified, Nyx Browser will look for a configuration
file named .nyx.{json, toml, yaml, yml} (with fallback to .filebrowser.{...}) in the following directories:

- ./
- $HOME/
- /etc/nyx/
- /etc/filebrowser/

**Note:** Only the options listed below can be set via the config file or
environment variables. Other configuration options live exclusively in the
database and so they must be set by the "config set" or "config
import" commands.

The precedence of the configuration values are as follows:

- Flags
- NYX_ environment variables
- FB_ environment variables
- FILEBROWSER_ environment variables
- Configuration file (.nyx.*, fallback .filebrowser.*)
- Database values
- Defaults

Also, if the database path doesn't exist, Nyx Browser will enter into
the quick setup mode and a new database will be bootstrapped and a new
user created with the credentials from options "username" and "password".

```
nyx [flags]
```

## Options

```
  -a, --address string                 address to listen on (default "127.0.0.1")
  -b, --baseURL string                 base url
      --cacheDir string                file cache directory (disabled if empty)
  -t, --cert string                    tls certificate
  -c, --config string                  config file path
  -d, --database string                database path (default "./nyx.db")
      --disableExec                    disables Command Runner feature (default true)
      --disableImageResolutionCalc     disables image resolution calculation by reading image files
      --disablePreviewResize           disable resize of image previews
      --disableThumbnails              disable image thumbnails
      --disableTypeDetectionByHeader   disables type detection by reading file headers
      --followExternalSymlinks         follow symlinks whose target is outside the user scope (unsafe)
  -h, --help                           help for nyx
      --imageProcessors int            image processors count (default 4)
  -k, --key string                     tls key
  -l, --log string                     log output (default "stdout")
      --noauth                         use the noauth auther when using quick setup
      --password string                hashed password for the first user when using quick setup
  -p, --port string                    port to listen on (default "8080")
      --redisCacheUrl string           redis cache URL (for multi-instance deployments), e.g. redis://user:pass@host:port
  -r, --root string                    root to prepend to relative paths (default ".")
      --socket string                  socket to listen to (cannot be used with address, port, cert nor key flags)
      --socketPerm uint32              unix socket file permissions (default 438)
      --tokenExpirationTime string     user session timeout (default "2h")
      --username string                username for the first user when using quick setup (default "admin")
```

## See Also

* [nyx cmds](nyx-cmds.md)	 - Command runner management utility
* [nyx completion](nyx-completion.md)	 - Generate the autocompletion script for the specified shell
* [nyx config](nyx-config.md)	 - Configuration management utility
* [nyx hash](nyx-hash.md)	 - Hashes a password
* [nyx rules](nyx-rules.md)	 - Rules management utility
* [nyx users](nyx-users.md)	 - Users management utility
* [nyx version](nyx-version.md)	 - Print the version number

