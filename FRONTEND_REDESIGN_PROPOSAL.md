# Nyx Browser Frontend Redesign Proposal & Architecture Specification

> **Document Version**: 2.0.0  
> **Repository**: `github.com/nyxlumen/nyx-browser/v2`  
> **Target Subsystem**: `frontend/` (Vue 3, TypeScript, Pinia, Vue Router, Vite, Styling, Ergonomics)  
> **Status**: Production Architecture & Design Reference  
> **Branch**: `nyx_foundation_phase_one`  

---

## 1. Executive Summary

Nyx Browser requires an independent, production-grade, high-end visual and architectural identity. An in-depth audit of `frontend/` reveals a legacy Vue 2-to-Vue 3 migration that stalled midway:
- **18 components remain on the legacy Options API**, while 36 use `<script setup lang="ts">`.
- **`DropdownModal.vue` contains an antipattern dual-script block** (`<script setup>` and `<script export default>`).
- **Styling relies on 18 loose vanilla CSS files** in `frontend/src/css/` with Google Material Design 2 color tokens (`--background: rgb(20, 29, 36)`).
- **The navigation sidebar is coupled to `LayoutStore.prompts`**, popping up as a full-screen blocking modal overlay rather than a persistent desktop command rail.
- **Boot sequence triggers a blinding white flash** due to `#loading { background: #fff; }` hardcoded in `frontend/public/index.html`.
- **Media previewing is constrained to a single basic modal** with zero support for masonry feeds, video hover scrubbing, or filmstrip navigation.

This proposal provides a concrete, non-breaking architectural blueprint to transition Nyx Browser into a minimalist, OLED dark-first, responsive PWA-first media browsing experience. The proposal includes:
1. **Design Token System**: OLED black (`#090a0f`), deep slate surfaces, crisp 1px borders, electric indigo/cyan accents, Geist/Inter typography, and tree-shakeable Lucide SVG iconography.
2. **Responsive PWA-First Layout Shell**: Persistent 64px/240px collapsible command rail on desktop, thumb-zone bottom navigation bar on mobile with safe-area insets, and a global `Cmd+K` command palette.
3. **Fluid Media Listing & Previewer**: Responsive table, bento card grid, and masonry gallery engine, paired with a hardware-accelerated media lightbox featuring a filmstrip scrubber and EXIF inspector.
4. **Zero-Duplicate-DB Nyx Gallery Integration**: Seamless `/gallery/:path*` route and in-situ toggle running directly on the existing Go BoltDB backend and `fileStore` without duplicating database tables or file indexes.
5. **4-Phase Migration Roadmap**: Strict incremental rollout preserving existing API contracts, runtime window shims (`window.Nyx`, `window.FileBrowser`), and test suites.

---

## 2. Current State Architectural Assessment

### 2.1 Project Configuration & Dependency Stack

Inspection of `frontend/package.json`, `frontend/vite.config.ts`, and `frontend/postcss.config.cjs` reveals:
- **Core Runtime**: Vue `3.5.17`, Vue Router `5.0.0`, Pinia `3.0.4`, Dayjs `1.11.13`, Vue-i18n `11.1.10`, Vite `8.0.0`, TypeScript `5.9.3`, Node `>=24.0.0`, pnpm `10.33.4`.
- **Styling Pipeline**:
  - `postcss.config.cjs:1-5`: Only `autoprefixer` is configured.
  - **Tailwind CSS is NOT installed**. Styling is driven entirely by vanilla CSS (`normalize.css` and 18 internal stylesheets in `frontend/src/css/`).
  - Icons are provided by the `material-icons` web font package (`material-icons: ^1.13.14`) via text ligatures.
- **Media & Utility Libraries**: `video.js: ^8.23.3` (with hotkeys and mobile UI plugins), `ace-builds: ^1.43.2`, `dompurify: ^3.2.6`, `epubjs: ^0.3.93`, `utif: ^3.1.0`, `tus-js-client: ^4.3.1`, `vue-toastification: ^2.0.0-rc.5`, `vue-lazyload: ^3.0.0`.
- **Quality Baseline**: `vitest: ^4.1.0` with 5 test files (29 unit tests passing), `vue-tsc: ^3.1.3` (strict TypeScript validation), `eslint: ^10.0.0`.

### 2.2 Options API vs Composition API Breakdown

The frontend codebase is split between modern Vue 3 `<script setup lang="ts">` and legacy Options API:

| Category | Total Files | Composition API (`<script setup lang="ts">`) | Options API (`export default { ... }`) | Mixed / Dual Script |
|---|---|---|---|---|
| **Views** (`src/views/`) | 12 | 12 (100%) | 0 (0%) | 0 |
| **Header Components** (`src/components/header/`) | 2 | 2 (100%) | 0 (0%) | 0 |
| **File Components** (`src/components/files/`) | 4 | 4 (100%) | 0 (0%) | 0 |
| **Settings Components** (`src/components/settings/`) | 7 | 3 (43%) | 4 (57%) | 0 |
| **Prompt Modals** (`src/components/prompts/`) | 22 | 11 (50%) | 11 (50%) | 0 |
| **Core Layout / Shell** (`src/components/`) | 8 | 4 (50%) | 3 (37.5%) | 1 (12.5%) |
| **Total Components** | **55** | **36 (65.5%)** | **18 (32.7%)** | **1 (1.8%)** |

#### Concrete Legacy Code Citations:
1. **`frontend/src/components/Sidebar.vue:141-223`**:
   Uses legacy Options API with `mapState` and `mapActions`:
   ```javascript
   // Sidebar.vue:151-157
   computed: {
     ...mapState(useAuthStore, ["user", "isLoggedIn"]),
     ...mapState(useFileStore, ["isFiles", "reload"]),
     ...mapState(useLayoutStore, ["currentPromptName"]),
     active() {
       return this.currentPromptName === "sidebar";
     },
   }
   ```
2. **`frontend/src/components/Shell.vue:56-65`**:
   Uses Options API with `mapState(useLayoutStore, ["showShell"])` and direct DOM manipulation for drag resizing.
3. **`frontend/src/components/ProgressBar.vue:53-70`**:
   Uses Options API with local property mutation.
4. **`frontend/src/components/DropdownModal.vue:1,73`**:
   Contains **dual script blocks** (`<script setup lang="ts">` at line 1 and `<script>` with `export default` at line 73), an antipattern that creates maintenance overhead and typecheck friction.
5. **11 Prompt Modals on Options API**:
   - `src/components/prompts/DiscardEditorChanges.vue:45`
   - `src/components/prompts/Rename.vue:52`
   - `src/components/prompts/Delete.vue:42`
   - `src/components/prompts/Replace.vue:48`
   - `src/components/prompts/Share.vue:152`
   - `src/components/prompts/Copy.vue:67`
   - `src/components/prompts/ShareDelete.vue:34`
   - `src/components/prompts/Help.vue:41`
   - `src/components/prompts/FileList.vue:37`
   - `src/components/prompts/Move.vue:69`
   - `src/components/prompts/Info.vue:110`
6. **4 Settings Sub-Components on Options API**:
   - `src/components/settings/Languages.vue:12`
   - `src/components/settings/Permissions.vue:48`
   - `src/components/settings/Commands.vue:12`
   - `src/components/settings/Rules.vue:36`

### 2.3 State Management Audit (`frontend/src/stores/`)

Inspection of the Pinia stores reveals several structural flaws:
- **`stores/auth.ts:5-46`**: Uses Pinia Options syntax (`defineStore("auth", { state, getters, actions })`). State contains `user: IUser | null`, `jwt: string`, and `logoutTimer: number | null`.
- **`stores/file.ts:3-65`**: Uses Pinia Options syntax.
  - Manages `req: Resource | null`, `oldReq: Resource | null`, `selected: number[]`, `multiple: boolean`, `isFiles: boolean`, `preselect: string | null`.
  - **Numerical Index Flaw**: `selected` is an array of numerical item indices (`number[]`). When files are deleted, renamed, or dynamically filtered in memory, numerical indices point to the wrong files.
- **`stores/layout.ts:5-86`**:
  - Manages `loading: boolean`, `prompts: PopupProps[]`, and `showShell: boolean | null`.
  - **Sidebar Modal Antipattern**: Modals, alert dialogs, prompts, AND the navigation sidebar are all funneled into `prompts: PopupProps[]`. Opening the sidebar executes `layoutStore.showHover("sidebar")`. On desktop monitors, this treats the main navigation like an emergency modal, popping up an overlay and dimming the screen instead of displaying a persistent command rail.
- **`stores/upload.ts:17-199`**: Uses Pinia Setup syntax (`defineStore("upload", () => { ... })`). Well-structured with concurrency bounds (`UPLOADS_LIMIT = 5`), `rawProgress: markRaw(...)` to avoid reactive thrashing, and TUS chunked upload management.

### 2.4 Styling & Visual Design Flaws

1. **Design Tokens & Palette (`_variables.css:1-59`)**:
   - Light theme is default (`:root`). Dark theme is scoped under `:root.dark`.
   - Dark theme palette values:
     - `--background`: `rgb(20, 29, 36)` (File Browser greenish/cyan slate)
     - `--surfacePrimary`: `rgb(32, 41, 47)`
     - `--surfaceSecondary`: `rgb(58, 65, 71)`
     - Accent colors: `--blue: #2196f3`, `--red: #f44336` (standard Google Material 2 palette).
   - Lacks true OLED blacks, lacks subtle multi-tiered dark surfaces, and lacks modern border transparency variables.
2. **Flash of White on Boot (`frontend/public/index.html:87-98`)**:
   - The initial loading splash screen hardcodes:
     ```css
     #loading {
       position: fixed;
       top: 0; left: 0; width: 100%; height: 100%;
       background: #fff; /* Blinding white flash before Vue mounts */
       z-index: 9999;
     }
     ```
   - When a user prefers dark mode, opening or refreshing the app flashes a pure white screen before `App.vue:20` reads local storage and sets `:root.dark`.
3. **Icon System (`listing-icons.css:1-254`, `fonts.css:1-246`)**:
   - Icons rely on the `material-icons` font loaded via font ligatures and CSS pseudo-element injection:
     ```css
     .file-icons [data-type="image"] i::before { content: "image"; }
     .file-icons [data-type="video"] i::before { content: "movie"; }
     ```
   - This prevents SVG animation, prevents independent stroke-width tuning, prevents tree-shaking, and causes text flash or broken icons if webfonts fail to load.
4. **Layout Shell & Breakpoints (`header.css:1-280`, `mobile.css:1-200`, `listing.css:1-284`)**:
   - Hardcoded top header `height: 4em` with fixed positioning.
   - Breakpoints hardcoded at `1024px` and `736px` (legacy iPhone 6 Plus breakpoint).
   - `#listing.mosaic` forces `width: calc(33% - 1em);` regardless of screen width, breaking ultra-wide monitors and tablet screens.
   - `#listing.mosaic.gallery` simply sets image height to `12em` with `object-fit: cover`, cropping photos indiscriminately without masonry, aspect ratio preservation, or video scrubbing.

---

## 3. Minimalist Dark-First Design System

The redesign transitions Nyx Browser from the dated Material Design 2 look to an ultra-refined, minimalist, OLED dark aesthetic reminiscent of Linear, Raycast, and Vercel.

### 3.1 Semantic Color Token Hierarchy

```
+─────────────────────────────────────────────────────────────────────────────+
|                       NYX DESIGN TOKEN ARCHITECTURE                         |
+───────────────────┬──────────────────────────────┬──────────────────────────+
| Token Name        | Value / Hex / RGBA           | Semantic Role            |
+───────────────────┼──────────────────────────────┼──────────────────────────+
| --nyx-bg-root     | #090a0f (OLED Deep Void)     | App background & canvas  |
| --nyx-bg-surface  | #0f121a (Deep Slate Canvas)  | Main layout shell        |
| --nyx-surface-1   | #141824 (Card / Container)   | Listing rows, sidebar    |
| --nyx-surface-2   | #1c2233 (Elevated Surface)   | Dropdowns, popovers, nav |
| --nyx-surface-3   | #262e45 (Active / Selected)  | Selected item, active tab|
| --nyx-border-subtle| rgba(255, 255, 255, 0.06)   | Structural splitters     |
| --nyx-border-crisp| rgba(255, 255, 255, 0.12)    | Card borders, inputs     |
| --nyx-border-focus| #6366f1 (Electric Indigo)    | Keyboard focus rings     |
| --nyx-accent-glow | rgba(99, 102, 241, 0.15)     | Ambient focus glow       |
| --nyx-accent-cyan | #06b6d4 (Cyan Accent)        | Media / video badges     |
| --nyx-text-bright | #ffffff (100% White)         | Headings, active labels  |
| --nyx-text-body   | #94a3b8 (Slate 400)          | File names, metadata     |
| --nyx-text-muted  | #64748b (Slate 500)          | Sizes, dates, disabled   |
+───────────────────┴──────────────────────────────┴──────────────────────────+
```

### 3.2 CSS Variables Specification (`frontend/src/css/tokens.css`)

```css
:root {
  /* Default dark-mode-first token set */
  --bg-app: #090a0f;
  --bg-canvas: #0b0d14;
  --surface-primary: #131620;
  --surface-secondary: #1a1e2e;
  --surface-tertiary: #222738;
  --surface-hover: rgba(255, 255, 255, 0.04);
  --surface-active: rgba(99, 102, 241, 0.12);

  --border-hairline: rgba(255, 255, 255, 0.06);
  --border-subtle: rgba(255, 255, 255, 0.10);
  --border-strong: rgba(255, 255, 255, 0.18);
  --border-focus: #6366f1;

  --accent-primary: #6366f1;       /* Electric Indigo */
  --accent-hover: #4f46e5;
  --accent-cyan: #06b6d4;          /* Gallery Video Accent */
  --accent-emerald: #10b981;       /* Upload Success Accent */
  --accent-rose: #f43f5e;          /* Destructive / Delete */

  --text-primary: #f8fafc;
  --text-secondary: #94a3b8;
  --text-tertiary: #64748b;
  --text-inverted: #090a0f;

  --font-sans: "Geist", "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --font-mono: "JetBrains Mono", "Fira Code", monospace;

  --radius-xs: 4px;
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;
  --radius-full: 9999px;

  --shadow-card: 0 1px 3px 0 rgba(0, 0, 0, 0.4), 0 1px 2px -1px rgba(0, 0, 0, 0.4);
  --shadow-overlay: 0 10px 25px -5px rgba(0, 0, 0, 0.8), 0 8px 10px -6px rgba(0, 0, 0, 0.8);
  --shadow-glow: 0 0 15px -3px rgba(99, 102, 241, 0.35);

  --backdrop-blur: blur(16px);
}
```

### 3.3 Eliminating the White Boot Flash (`index.html`)

Fix `frontend/public/index.html:87-98` by making `#loading` OLED dark by default:
```html
<style>
  html, body { background-color: #090a0f; color: #f8fafc; }
  #loading {
    position: fixed;
    top: 0; left: 0; width: 100%; height: 100%;
    background: #090a0f;
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  #loading.done { opacity: 0; pointer-events: none; transition: opacity 0.2s ease; }
</style>
```

### 3.4 Icon System Migration: Lucide SVG (`lucide-vue-next`)

Replace `material-icons` web font ligatures with tree-shakeable, stroke-calibrated Lucide SVG icons (`lucide-vue-next`):
- `FileText`, `Folder`, `Image`, `Video`, `Music`, `Code`, `Archive`, `Database`, `MoreVertical`, `Search`, `Grid`, `List`, `LayoutGrid`, `Share2`, `Download`, `Trash2`, `UploadCloud`, `Terminal`, `Settings`, `Sun`, `Moon`.
- **Benefits**: Crisp 1.5px vector rendering on Retina/OLED displays, elimination of font layout shifts, SVG animation capabilities, and zero external font asset dependencies.

---

## 4. Responsive PWA-First Shell Architecture

### 4.1 Desktop Layout (>= 1024px): Collapsible Persistent Navigation Rail

```
+─────────────────────────────────────────────────────────────────────────────+
| DESKTOP SHELL (>= 1024px)                                                   |
+──────┬──────────────────────────────────────────────────────────────────────+
| RAIL | TOP COMMAND BAR: Breadcrumbs > Context Actions > Cmd+K Search > Auth |
| (64p +──────────────────────────────────────────────────────────────────────+
|  or  | VIEW AREA (Table / Bento / Masonry Gallery)                          |
| 240p)|                                                                      |
| [Nyx]|  [Filter Bar: All | Photos | Videos | Docs]  [Sort: Name / Date]     |
|  📁  |  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐            |
| Files|  │ Photo_1   │ │ Photo_2   │ │ Video_1   │ │ Photo_3   │            |
|  🖼️  |  │ Aspect 4:3│ │ Aspect 1:1│ │ [▶ 01:24] │ │ Aspect 16:│            |
| Gllry|  └───────────┘ └───────────┘ └───────────┘ └───────────┘            |
|  ⚡  |                                                                      |
| Cmds |                                                                      |
|  ⚙️  |                                                                      |
| Sett +──────────────────────────────────────────────────────────────────────+
| [Sto]| POP-OUT FLOATING SHELL (Toggleable at bottom or drawer)              |
+──────┴──────────────────────────────────────────────────────────────────────+
```

1. **Persistent Navigation Rail (`NavRail.vue`)**:
   - Replaces the blocking modal drawer in `Sidebar.vue`.
   - **Collapsed Mode (64px width)**: High-density icon column with floating tooltips.
   - **Expanded Mode (240px width)**: Shows labels, folder shortcuts, storage usage progress bar, and user profile card.
   - Rail state persisted in `localStorage('nyx_nav_rail_expanded')`.
2. **Top Command Bar (`TopCommandBar.vue`)**:
   - Interactive path breadcrumbs with copy-path on click and dropdown folder switcher.
   - Universal search trigger: `Cmd+K` / `Ctrl+K`.
   - Batch selection actions toolbar (appears smoothly via slide-down animation when `fileStore.selectedCount > 0`).

### 4.2 Mobile & PWA Layout (< 768px): Thumb-Zone Ergonomics

```
+─────────────────────────────────────────────────+
| MOBILE / PWA VIEW (< 768px)                     |
+─────────────────────────────────────────────────+
| Top App Bar: [Nyx] /Current/Folder/Path [Search]|
+─────────────────────────────────────────────────+
| Content Viewport (Fluid Masonry / Touch List)   |
|                                                 |
|                                                 |
|                                                 |
| (Pull-to-refresh enabled)                       |
+─────────────────────────────────────────────────+
| FLOATING BATCH ACTION PILL (When selected > 0)  |
| [ 3 selected  │ Share │ Copy │ Move │ Delete ]  |
+─────────────────────────────────────────────────+
| BOTTOM NAVIGATION BAR (Thumb Ergonomics)        |
| ┌─────────┬─────────┬─────────┬─────────┬─────┐ |
| │  📁     │  🖼️     │  🔍     │  ⚡     │  ⚙️ │ |
| │ Files   │ Gallery │ Search  │ Actions │ Set │ |
| └─────────┴─────────┴─────────┴─────────┴─────┘ |
| Safe Area Inset: env(safe-area-inset-bottom)    |
+─────────────────────────────────────────────────+
```

1. **Bottom Navigation Bar (`MobileTabBar.vue`)**:
   - Replaces the cramped mobile hamburger dropdown.
   - Direct one-thumb access to: **Files**, **Gallery**, **Search**, **Quick Actions (+)**, and **Settings**.
   - Built with `env(safe-area-inset-bottom)` padding for seamless iOS home indicator and Android gesture bar integration.
2. **Bottom Sheet Drawers (`BottomSheet.vue`)**:
   - Replaces desktop-centric modal prompts (`BaseModal.vue`) on mobile devices.
   - Gesture swipe-to-dismiss, touch draggable, and native haptic feedback triggers.
3. **Pull-to-Refresh**:
   - Native touch pull down on mobile triggers `fileStore.reload = true`.

### 4.3 Global Command Palette (`CommandPalette.vue`)

Accessible via `Cmd+K` (macOS) or `Ctrl+K` (Linux/Windows):
- **Fuzzy Search**: Instantly searches current directory and recent files without navigating away.
- **Navigation Jumps**: Direct jumps to `/Photos`, `/Documents`, `/Downloads`, Settings, or User Management.
- **Quick Commands**: `New Folder` (`Cmd+Shift+N`), `Upload Files` (`Cmd+U`), `Toggle Gallery View` (`Cmd+G`), `Toggle Terminal` (`Ctrl+\``), `Download Selected` (`Cmd+S`).

---

## 5. Fluid Media Listing & Lightbox Previewer

### 5.1 Multi-Layout Engine Architecture

The monolithic `FileListing.vue` (1,133 lines) will be modularized into discrete, high-performance view renderers:

```
frontend/src/views/files/
├── FileListing.vue           # Container view (routing, selection, event dispatch)
├── views/
│   ├── TableView.vue         # Dense sortable table (file size, permissions, date)
│   ├── BentoGridView.vue     # Modern square/bento grid with file type tags
│   └── MasonryGalleryView.vue# Justified aspect-ratio media feed for photos/videos
```

### 5.2 Masonry Gallery Engine Specification

```
+─────────────────────────────────────────────────────────────────────────────+
| MASONRY GALLERY ENGINE                                                      |
+─────────────────────────────────────────────────────────────────────────────+
| [Header Bar: Group by Month | Filter: All (128) | Photos (110) | Videos (18)]|
+─────────────────────────────────────────────────────────────────────────────+
| 📅 OCTOBER 2026                                                             |
| ┌───────────────────┐ ┌─────────┐ ┌───────────────────────────────────────┐ |
| │                   │ │         │ │ 📹 04:12 [Hover: Scrub preview]       │ |
| │ Photo (4:3)       │ │ (1:1)   │ │ Video (16:9)                          │ |
| │ EXIF: ISO 100     │ │         │ │ 4K 60fps                              │ |
| └───────────────────┘ └─────────┘ └───────────────────────────────────────┘ |
| ┌───────────────┐ ┌───────────────────────┐ ┌─────────────────────────────┐ |
| │ Portrait(9:16)│ │ Panorama (21:9)       │ │ Photo (3:2)                 │ |
| └───────────────┘ └───────────────────────┘ └─────────────────────────────┘ |
+─────────────────────────────────────────────────────────────────────────────+
```

- **Aspect Ratio Preservation**: Instead of cropping thumbnails into uniform square tiles (`listing.css:122: height: 12em; width: 100%`), the Masonry engine calculates natural aspect ratios via flex row justification (`flex-grow: aspect-ratio`) or CSS Grid subgrids.
- **Hover Video Scrubbing**: Hovering over a video thumbnail scrubs keyframes or loops a lightweight 3-second micro-preview.
- **Virtualization**: Uses `@vueuse/core` `useVirtualList` or `vue-virtual-scroller` to effortlessly handle folders with 10,000+ images without DOM degradation or memory leaks.

### 5.3 Immersive Media Previewer (`MediaPreviewModal.vue`)

Overhaul of `frontend/src/views/files/Preview.vue` into a studio-grade lightbox:
1. **Interactive Canvas**: Pan, smooth pinch-to-zoom (up to 500%), and double-tap zoom for high-resolution images.
2. **Bottom Filmstrip Scrubber**: Shows adjacent 12 thumbnails along the bottom edge for rapid scrubbing across media sets.
3. **EXIF & Media Metadata Inspector Drawer** (Collapsible side panel):
   - Camera model, lens, focal length, aperture ($f$), shutter speed ($1/s$), ISO.
   - GPS coordinate map pin (OpenStreetMap embed without third-party tracking).
   - Audio/Video codec (`H.264`, `AV1`, `HEVC`), bitrate, audio channels, color space (`BT.709`, `BT.2020`).

---

## 6. Zero-Duplicate-DB Nyx Gallery View Integration

### 6.1 Architectural Integration Strategy

Nyx Gallery integrates directly into the existing Go BoltDB backend and `fileStore` without duplicating database tables, file indexes, or crawler tasks:

```
+─────────────────────────────────────────────────────────────────────────────+
| ZERO-DUPLICATE-DATABASE MEDIA ARCHITECTURE                                  |
+─────────────────────────────────────────────────────────────────────────────+
|                         FRONTEND ROUTER & STORE                             |
|                                                                             |
|    Route: /files/:path* ──┐               ┌──> Route: /gallery/:path*       |
|    (File Manager Mode)    │               │    (Media Gallery Mode)         |
|                           ▼               ▼                                 |
|                   ┌───────────────────────────────┐                         |
|                   │      useFileStore (Pinia)     │                         |
|                   │  - req: Resource              │                         |
|                   │  - mediaItems: computed()     │                         |
|                   │  - albums / timeline grouping │                         |
|                   └───────────────┬───────────────┘                         |
|                                   │                                         |
+───────────────────────────────────┼─────────────────────────────────────────+
|                                   │ HTTP REST API                           |
|                                   ▼                                         |
|                   ┌───────────────────────────────┐                         |
|                   │     Go Backend (Existing)     │                         |
|                   │  GET /api/resources/:path*    │                         |
|                   │  GET /api/preview/:path*      │                         |
|                   └───────────────┬───────────────┘                         |
|                                   │                                         |
+───────────────────────────────────┼─────────────────────────────────────────+
|                                   ▼                                         |
|                      ┌─────────────────────────┐                            |
|                      │   BoltDB & ScopedFs     │                            |
|                      │  (No Duplicate Tables)  │                            |
|                      └─────────────────────────┘                            |
+─────────────────────────────────────────────────────────────────────────────+
```

### 6.2 Router Integration (`router/index.ts`)

Add `/gallery` as a primary route sharing the exact same `Layout` shell and auth guards:
```typescript
{
  path: "/gallery",
  component: AppShell,
  meta: { requiresAuth: true },
  children: [
    {
      path: ":path*",
      name: "Gallery",
      component: () => import("@/views/Gallery.vue"),
      props: (route) => ({
        path: route.params.path || "",
        viewMode: "gallery",
      }),
    },
  ],
},
```

### 6.3 Unified In-Situ Toggle on `/files`

In `FileListing.vue`:
- If current directory contains `> 40%` media items (`type === 'image' || type === 'video'`), a subtle pill appears in the top bar: **"Switch to Gallery View (Cmd+G)"**.
- Clicking the pill toggles `viewMode = 'gallery'`. The URL updates from `/files/Photos` to `/gallery/Photos` (or query param `?view=gallery`) with zero page reload, zero re-fetching, and instant rendering from the already loaded `fileStore.req` cache.

---

## 7. Four-Phase Non-Breaking Migration Roadmap

To ensure continuous delivery, unbroken test suites, and zero regressions, the implementation is structured across 4 incremental phases:

```
+─────────────────────────────────────────────────────────────────────────────+
|                    4-PHASE REFACTORING ROADMAP                              |
+─────────────────────────────────────────────────────────────────────────────+
| PHASE 1: Design Tokens, Styling Pipeline & White Flash Fix                  |
| • Install Tailwind CSS v4 / configure PostCSS tokens                       |
| • Implement `tokens.css` with OLED dark variables                           |
| • Fix `#loading` white boot flash in `public/index.html`                    |
| • Install `lucide-vue-next` and create Lucide icon adapter component        |
| • Verification: vitest tests pass, production build succeeds                |
+─────────────────────────────────────────────────────────────────────────────+
| PHASE 2: Composition API & Pinia Modernization                              |
| • Refactor 18 Options API components to `<script setup lang="ts">`         |
| • Fix dual-script block in `DropdownModal.vue`                              |
| • Refactor `fileStore.selected` from index[] to path/URL identifiers        |
| • Decouple `Sidebar.vue` from `LayoutStore.prompts` modal stack             |
| • Verification: `vue-tsc` typecheck 100% clean, all prompt dialogs work     |
+─────────────────────────────────────────────────────────────────────────────+
| PHASE 3: PWA-First Layout Shell & Command Palette                           |
| • Implement `NavRail.vue` (persistent desktop collapsible rail 64px/240px)  |
| • Implement `MobileTabBar.vue` (mobile bottom navigation bar)               |
| • Implement `CommandPalette.vue` (`Cmd+K` fuzzy search & shortcuts)         |
| • Implement unified keyboard navigation composable (`useKeyNavigation`)     |
| • Verification: responsive layout testing across 320px, 768px, 1920px, 4K   |
+─────────────────────────────────────────────────────────────────────────────+
| PHASE 4: Media Engine & Nyx Gallery Integration                             |
| • Modularize `FileListing.vue` into TableView, BentoGrid, and MasonryGallery|
| • Implement aspect-ratio justified masonry layout                           |
| • Integrate `/gallery/:path*` route and in-situ gallery view mode toggle    |
| • Overhaul `Preview.vue` into `MediaPreviewModal.vue` with filmstrip & EXIF |
| • Verification: full E2E manual media test, 0 duplicate DB records verified |
+─────────────────────────────────────────────────────────────────────────────+
```

---

## 8. Verification & Acceptance Checkpoints

1. **White Boot Flash Elimination**:
   - Inspect `frontend/public/index.html:87-157`. Verify `#loading` background is `#090a0f`. Reload browser in dark mode; verify zero white frames appear before app initialization.
2. **Options API Deprecation**:
   - Verify `grep -rn "export default {" frontend/src/components frontend/src/views` returns 0 matches (100% `<script setup lang="ts">`).
3. **Vitest Regression Safeguard**:
   - Run `pnpm --dir frontend test`. Verify `src/css/__tests__/mobile.test.ts` passes or is cleanly updated alongside the refactored mobile style rules.
4. **PWA Standalone & Bottom Bar Verification**:
   - In mobile view (<768px), verify bottom navigation bar renders within safe-area bounds (`env(safe-area-inset-bottom)`).
   - In desktop view (>=1024px), verify persistent 64px/240px navigation rail without full-screen modal dimming.
5. **Zero-Duplicate-Database Verification**:
   - Navigate to `/gallery/`. Verify that HTTP network tab issues requests strictly to `/api/resources/*` and `/api/preview/*` with zero duplicate database collections or separate tables.
