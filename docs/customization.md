# Customization

You can customize the styles, branding, and icons of your Nyx Browser instance to give it a personalized appearance.

## Custom Branding

You can configure Nyx Browser to use your own branding. This includes the following:

- **Name**: The name of the instance that shows up in the tab title, login pages, and UI header.
- **Disable External Links**: Disables external links in the interface.
- **Disable Used Percentage**: Disables the disk usage percentage indicator on the sidebar.
- **Branding Folder**: Directory which can contain two items:
  - `custom.css`, containing a global stylesheet applied to all users.
  - `img`, a directory containing replacements for default logotypes and icons.

This can be configured by an administrator under **Settings → Global Settings**, or directly using the [CLI](cli/nyx-config-set.md):

```sh
nyx config set --branding.name "My Brand" \
  --branding.files "/abs/path/to/my/dir" \
  --branding.disableExternal
```

> [!NOTE] 
>
> When using Docker, ensure that you mount a volume with the `branding` directory into the container.

### Custom Icons

To replace the default logotypes and favicons, create an `img` directory under the branding directory:

```
img/
  logo.svg
  icons/
    favicon.ico
    favicon.svg
    (...)
```

Different versions of favicons exist in multiple sizes. You can use the [Real Favicon Generator](https://realfavicongenerator.net/) to generate these assets from your source image.

> [!NOTE]
>
> Icons are cached aggressively by the browser; clear your browser cache if changes do not appear immediately.
