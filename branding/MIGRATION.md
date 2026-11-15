# nyxbrowser: logo and branding migration

You are working in the nyxbrowser repo, a fork of filebrowser/filebrowser redesigned as a gallery/media-first file browser. The new brand assets live in this folder (`brand/` once copied into the repo). Replace the old File Browser branding everywhere, converting file formats and sizes where necessary. The UI is **dark mode only**.

## Brand tokens

| Token  | Value     | Use |
|--------|-----------|-----|
| ink    | `#0A0A0A` | app background, icon tile, `theme_color`, `background_color` |
| fg     | `#F2F0EA` | dots, text |
| accent | `#E5252A` | the single red dot, never anything else |

The mark ("Dot Moon") is a dot-matrix crescent plus one red dot. The wordmark is also dots (no font needed). Do not edit the SVG geometry, add gradients, shadows, outlines or extra colours, and do not recolour the red dot (except the monochrome pinned-tab icon).

## Assets in this folder

| File | What it is | Use it for |
|------|------------|------------|
| `logo-lockup.svg` / `.png` | mark + `nyxbrowser` wordmark, transparent, light dots | header, login screen, README banner (dark backgrounds only) |
| `logo-mark.svg` / `.png` | mark alone, transparent, light dots | sidebar, compact header, anywhere only a square logo fits |
| `icon.svg` | rounded black tile with mark | `favicon.svg`, "any" PWA icon source |
| `icon-square.svg` | full-bleed black tile with mark | apple-touch-icon, maskable icon, mstile source |
| `favicon.ico`, `favicon-16x16.png`, `favicon-32x32.png`, `favicon-48x48.png` | favicons | `<link rel="icon">` set |
| `apple-touch-icon.png` | 180x180 full-bleed | `<link rel="apple-touch-icon">` |
| `icon-192.png`, `icon-512.png` | rounded, purpose `any` | web manifest |
| `icon-maskable-512.png` | full-bleed, mark inside safe zone | web manifest, purpose `maskable` |
| `mstile-150x150.png` | Windows tile | `browserconfig.xml` |
| `safari-pinned-tab.svg` | monochrome mark | `<link rel="mask-icon">` |
| `og-image.svg` / `.png` | 1200x630 social card | `og:image` / `twitter:image` |

## Steps

### 1. Inventory first, then report before editing

Search the whole repo (excluding `node_modules`, `.git`, build output) for every place the old branding lives, and list what was found with file paths. Cover at least:

- Image files: logo, favicon, icons of any size, mstile, safari-pinned-tab, og/social images, README and docs screenshots that show the old logo.
- Code and templates: `index.html` and Go templates, `<link rel="icon|apple-touch-icon|mask-icon|manifest">`, `<meta name="theme-color|msapplication-*">`, `og:` tags, the web manifest (it may be generated in Go rather than a static file), `browserconfig.xml`.
- Frontend source: imports or `<img>` references to the old logo, inline SVGs, base64 `data:` URIs, CSS `background-image`, any logo or colour settings in the Vue/JS code and the settings UI ("branding" options, custom name, custom logo).
- Text: user-facing "File Browser" / "filebrowser" strings in the UI, page titles, i18n/locale files, README, docs, Dockerfile labels, package metadata.

Likely locations, to be verified and not trusted: `frontend/public/img/logo.svg`, `frontend/public/img/icons/`, `frontend/public/index.html`, the manifest handler under `http/`, and the frontend constants file. Report anything surprising, such as embedded or generated copies of the assets.

### 2. Replace assets, matching what already exists

- For every old image, check its format and pixel size, then replace it with the closest asset above under the **same filename and path**, so no references break. Dark-mode-only means any old light-theme or alternate logo variants can be removed or pointed at the same new file.
- If a size or format is needed that is not here (for example a 144x144 or WebP), generate it from the SVG sources (`rsvg-convert`, ImageMagick or `sharp`). Never upscale a PNG. Always rasterise from `icon.svg` / `icon-square.svg` / `logo-mark.svg`.
- In-app logos must be SVG, not PNG. Use `logo-lockup.svg` where there is horizontal room and `logo-mark.svg` where the slot is square. Size them with CSS height only and never stretch.
- Do not hand-edit generated or embedded output (built frontend `dist`, embedded asset bundles). Edit the source and rebuild.

### 3. Wire everything up

- Head tags: `<link rel="icon" href="/favicon.svg" type="image/svg+xml">` (from `icon.svg`), the `.ico`, 32x32 and 16x16 PNGs, `apple-touch-icon`, `mask-icon` (color `#E5252A`).
- Web manifest: name and short_name `nyxbrowser`, icons 192 and 512 (`any`) plus 512 maskable, `theme_color` and `background_color` `#0A0A0A`.
- `browserconfig.xml` / `msapplication-TileColor`: `#0A0A0A`.
- Social tags: `og:image`, `twitter:image` pointing at `og-image.png`, `og:title` / `og:site_name` `nyxbrowser`.
- If the frontend has a light/dark theme switch, make sure no light-theme logo path is left behind.

### 4. Rename user-facing text

Change visible "File Browser" / "filebrowser" strings to `nyxbrowser` (UI, titles, README headings, docs prose, manifest). Leave these alone unless asked: Go module paths and import paths, package names, the CLI binary name, config keys, API routes, env var names. **Keep the LICENSE and keep attribution to the original filebrowser project in the README**, because the upstream license requires it.

### 5. Verify

- Build the frontend and the Go binary, run the server, then confirm in the browser: tab icon, login page, main header, PWA install metadata, and that the network tab shows no 404s for old logo paths.
- Re-run the inventory search and confirm no references to the old logo/icon files remain, and list any remaining "File Browser" strings you deliberately kept.
- Final report: files added, files replaced, files deleted, strings changed, and anything skipped with the reason.

## Rules

- Make changes in small commits grouped by concern (assets, head/manifest, in-app logo, text).
- If anything is ambiguous (for example a custom-branding feature whose intent is unclear), stop and ask rather than guessing.
