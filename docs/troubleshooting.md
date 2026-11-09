# Troubleshooting

## Session Timeout

By default, user sessions expire after **2 hours**. If you are uploading large files over slower connections, you may need to increase this timeout to prevent sessions from expiring mid-upload. You can configure the session timeout using the `tokenExpirationTime` setting.

You can set this option at runtime using the `--tokenExpirationTime` CLI flag, the primary environment variable `NYX_TOKEN_EXPIRATION_TIME` (with legacy fallback to `FB_TOKEN_EXPIRATION_TIME` and `FILEBROWSER_TOKEN_EXPIRATION_TIME`), or in your configuration file. If you want to persist this setting into the database, use [`nyx config set`](cli/nyx-config-set.md):

```sh
nyx config set --tokenExpirationTime "12h"
```

Valid duration formats include `"2h"`, `"30m"`, `"24h"`, or combinations like `"2h30m"`.
