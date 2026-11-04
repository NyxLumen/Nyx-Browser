# Nyx Browser --- Project Context & Development Brief

> **Purpose:** This document is the source of truth for AI assistants,
> coding agents, and contributors working on Nyx Browser. Read it before
> proposing changes or modifying the repository.

## 1. What Nyx Browser Is

**Nyx Browser is an independent, self-hosted file management and media
browsing application built from a fork of the archived File Browser
project.**

Upstream File Browser provided a useful, working foundation for
filesystem access, authentication, permissions, storage, and file
operations. Nyx Browser is intended to evolve that foundation into a
polished, modern product with its own identity, interface, and
capabilities.

-   **Fork:** https://github.com/NyxLumen/Nyx-Browser
-   **Upstream:** https://github.com/filebrowser/filebrowser
-   **Product name:** Nyx Browser
-   **Access model:** Local network only
-   **Product direction:** PWA-first
-   **Core experiences:** A practical file manager and a separate visual
    gallery for images and videos

Nyx is not intended to be a superficial rebrand of File Browser. It
should become a distinct product while retaining and extending the
proven functionality of the existing codebase.

## 2. Product Vision

Build a dependable local-network file hub that makes it easy to browse,
organize, and view files across devices.

A user should be able to run Nyx on a machine or home server, access it
from another device on the same local network, manage files normally,
and browse photos and videos in a dedicated gallery experience.

The product should combine:

1.  **A capable file manager** for everyday file operations and detailed
    file information.
2.  **A beautiful media gallery** for visually browsing images and
    videos.
3.  **A reliable self-hosted backend** that manages files, permissions,
    authentication, previews, and media processing.
4.  **A convenient local-network experience** with server discovery and
    installable PWA support.

The product must remain useful as a real file manager. Visual design
must never come at the expense of functionality or information.

## 3. Non-Negotiable Product Decisions

These decisions have already been made. Do not reopen them without an
explicit request.

  -----------------------------------------------------------------------
  Area                                Decision
  ----------------------------------- -----------------------------------
  Access                              Local network only

  Platform direction                  PWA-first

  Media gallery                       Images and videos together

  File manager                        Remains a full, practical file
                                      manager

  Gallery                             A separate view, not a replacement
                                      for the file manager

  Backend                             Extend the existing File Browser
                                      backend where practical

  Storage                             Continue using the existing
                                      filesystem and storage
                                      infrastructure

  Media data                          Reuse existing resource/file data;
                                      do not create a duplicate media
                                      database without a demonstrated
                                      need

  UI foundation                       Minimal dark interface

  Glass effects                       Selective frosted glass accents
                                      only
  -----------------------------------------------------------------------

## 4. Visual and UI Direction

### Overall aesthetic

Nyx should be **minimal dark first, with selective frosted glass**.

The interface should feel like polished desktop software: calm, precise,
modern, and functional. It should not look like a generic futuristic
dashboard or a UI made entirely of translucent cards.

The visual hierarchy is:

1.  Minimal dark surfaces form the foundation.
2.  Clear typography and spacing make information easy to scan.
3.  Subtle borders and restrained contrast define structure.
4.  Frosted glass is used selectively for floating or layered UI.
5.  Motion is purposeful and restrained.

### Visual principles

-   Use deep charcoal and near-black backgrounds rather than making
    every surface pure black.
-   Use slightly lighter dark surfaces to establish hierarchy.
-   Prefer thin, low-contrast borders over heavy outlines.
-   Use a clean, modern sans-serif type system.
-   Use one restrained accent color consistently.
-   Keep icons consistent, simple, and preferably line-based.
-   Use subtle shadows and depth only where they clarify layering.
-   Keep animations smooth and purposeful. Avoid gratuitous motion.
-   Preserve comfortable information density. Do not waste space merely
    to make the UI look sparse.
-   Ensure readable contrast and accessible interaction states.

### Where frosted glass can be used

Use glass sparingly for elements that genuinely sit above other content:

-   Floating command palettes.
-   Modal dialogs and selected overlays.
-   Context menus and dropdowns.
-   Media viewer controls.
-   Small floating toolbars.
-   Potentially subtle sidebar treatments, if they improve the
    composition.

### Where frosted glass should not be used

-   Every file row or folder card.
-   Every button.
-   Every section or content background.
-   Every nested container.
-   Large areas where blur reduces text or metadata legibility.

**Avoid the "everything is glass" look.** Glass is an accent, not the
design system.

### Distinction between the two main experiences

-   **File Manager:** Information-first. Prioritize names, extensions,
    sizes, modification dates, folders, selection, sorting, and actions.
-   **Nyx Gallery:** Image-first. Prioritize thumbnails, visual
    browsing, media filters, and an immersive viewer.

Both experiences must share the same design language and feel like parts
of one product.

## 5. Core Product Experiences

### A. File Manager

The file manager remains a first-class feature and must preserve useful
functionality from upstream.

Expected capabilities include:

-   Directory and folder navigation.
-   File and folder names and extensions.
-   File sizes and modification dates.
-   Sorting by name, size, and modified date.
-   List and grid-style browsing.
-   File and folder selection, including multi-selection.
-   Upload and download.
-   Rename, move, copy, and delete where supported by the existing
    backend.
-   Context menus and useful keyboard interactions.
-   File previews and editing capabilities already present upstream,
    where applicable.
-   Responsive layouts for desktop, tablet, and mobile.

Do not remove metadata or standard file-management actions to make the
interface more visual.

### B. Nyx Gallery

Nyx Gallery is a **separate browsing experience**, not a replacement for
the file manager.

It should provide:

-   A dedicated Gallery view.
-   Mixed image and video browsing.
-   Filters for All, Photos, and Videos.
-   Folder navigation and access to media in directories.
-   A clean visual thumbnail grid.
-   Metadata overlays where useful.
-   Sorting by name, size, and modification date.
-   A full-screen lightbox/media viewer.
-   Previous/next media navigation.
-   Video playback using the existing playback foundation where
    practical.
-   Lazy loading and progressive rendering for large collections.
-   Video poster thumbnails generated by the media pipeline.

The initial implementation should reuse the existing directory/resource
API and file records. Gallery is a different presentation of the same
underlying files, not a second storage system.

Folders should remain navigable. The UI may separate folders from media
items visually, provided navigation remains obvious.

## 6. Technical Foundation and Existing Architecture

Nyx is based on an existing Go backend and Vue frontend. The first
instinct should be to understand and extend the current architecture,
not replace it.

Known technology foundation:

-   **Backend:** Go
-   **Frontend:** Vue 3 + TypeScript + Vite
-   **Storage/database foundation:** Existing filesystem and BoltDB
    infrastructure
-   **Media playback:** Existing Video.js-based player
-   **Image previews:** Existing image preview service and cache

The repository has existing file/resource APIs, authentication,
permissions, previews, and file operations. Preserve their contracts
unless a deliberate, documented change is necessary.

### Existing areas worth understanding

The following paths were identified during an earlier repository review.
Confirm their current contents before making changes:

-   `main.go` --- application entry point.
-   `http/http.go` --- HTTP handler and route registration.
-   `http/preview.go` --- existing image preview handler and cache
    integration.
-   `img/service.go` --- image processing service.
-   `frontend/src/views/Files.vue` --- file/directory route-level view.
-   `frontend/src/views/files/FileListing.vue` --- file listing and
    view-mode logic.
-   `frontend/src/components/files/ListingItem.vue` --- file/folder
    listing item.
-   `frontend/src/views/files/Preview.vue` --- file preview experience.
-   `frontend/src/components/files/VideoPlayer.vue` --- video playback.
-   `frontend/src/api/files.ts` --- frontend file/resource API.
-   `frontend/src/stores/auth.ts` --- authentication state.
-   `frontend/src/utils/constants.ts` --- frontend constants and default
    branding.
-   `frontend/public/index.html` --- HTML entry and existing dynamic
    manifest foundation.

Repository structure and implementation details can change. Inspect the
live repository before relying on this list.

### Existing preview behavior

The existing preview service handles images. It has thumbnail and larger
preview sizes, with caching based on file information and preview size.
Video poster generation is not currently provided by the existing video
player.

Do not break the existing image preview route or its API contract when
adding video posters. Prefer a service interface and additive endpoint
or carefully versioned extension.

## 7. Development Roadmap --- Official Order

The order below is intentional. Follow it unless the project owner
explicitly changes it.

### Phase 01 --- Foundation and Identity

Establish Nyx as an independent, buildable project while preserving
upstream functionality.

Tasks:

-   Audit the repository and understand the existing architecture.
-   Establish baseline Go and frontend builds and tests.
-   Rename upstream branding to Nyx Browser.
-   Rename the Go module path to a Nyx-owned path and update internal
    imports.
-   Update frontend package metadata, titles, icons, logos, and manifest
    identity.
-   Preserve the upstream license and required attribution.
-   Document the architecture, key APIs, and extension points.
-   Identify inherited security concerns before release.

**Completion goal:** Nyx has its own project identity, builds
successfully, and retains the original file-management foundation.

### Phase 02 --- Complete UI Redesign

Redesign the application around the minimal-dark visual direction.

Tasks:

-   Define design tokens for colors, typography, spacing, borders,
    radii, and elevation.
-   Establish shared components and interaction patterns.
-   Redesign application shell, navigation, sidebar, breadcrumbs,
    toolbars, dialogs, and menus.
-   Redesign file and folder list/grid presentations.
-   Improve responsive behavior across desktop, tablet, and mobile.
-   Preserve file metadata, sorting, selection, permissions, and file
    actions.
-   Add selective frosted-glass treatments only where they improve
    layering.
-   Audit accessibility, contrast, focus, hover, active, disabled, and
    loading states.

**Completion goal:** The complete file manager feels like a coherent,
polished Nyx product without losing functionality.

### Phase 03 --- Nyx Gallery

Build the separate mixed-media gallery on top of the redesigned system.

Tasks:

-   Add a dedicated Gallery view and navigation entry.
-   Reuse existing file/resource records and APIs.
-   Support images and videos together.
-   Add All, Photos, and Videos filters.
-   Add folder navigation.
-   Build the visual grid and media items.
-   Add metadata overlays and sorting.
-   Add a full-screen lightbox/viewer and previous/next navigation.
-   Implement progressive rendering or virtualization where appropriate.
-   Ensure large directories do not create an excessive number of DOM
    nodes.

**Completion goal:** Users can browse photos and videos in a dedicated
visual experience while retaining the normal file manager.

### Phase 04 --- Media Processing Pipeline

Build the backend capabilities needed to power fast, scalable media
browsing.

Tasks:

-   Introduce a media processing service with a testable interface.
-   Integrate FFmpeg for on-demand video poster generation.
-   Consider FFprobe for duration and other useful metadata.
-   Reuse the existing image preview pipeline where appropriate.
-   Implement thumbnail and poster caching.
-   Invalidate cached results when the underlying file changes.
-   Generate previews lazily as media enters the viewport or is
    requested.
-   Limit processing concurrency, execution time, dimensions, and
    resource use.
-   Avoid shell invocation and unsafe command construction.
-   Handle missing FFmpeg gracefully with a fallback.
-   Test malformed, oversized, and unsupported media.

**Completion goal:** Nyx Gallery can load media previews efficiently
without overwhelming the server.

### Phase 05 --- Local Network Discovery

After the core file and media experience is established, make Nyx
instances discoverable on a local network.

Tasks:

-   Give every installation a persistent UUID.
-   Advertise Nyx instances using mDNS/DNS-SD.
-   Discover other Nyx instances on the LAN.
-   Provide stable `.local` hostnames where supported.
-   Add minimal discovery API endpoints.
-   Build a server discovery and connection interface.
-   Add QR pairing and manual address fallback.
-   Ensure authentication always happens against the selected server.
-   Test networks where multicast/mDNS is blocked.

Important constraints:

-   Browser JavaScript cannot freely scan arbitrary LAN IP ranges or
    browse mDNS multicast services.
-   Discovery should be implemented by the Go backend.
-   Discovery endpoints should expose only minimal, non-sensitive
    metadata.
-   Do not implement arbitrary port scanning.
-   Local-only access does not remove the need for authentication and
    authorization.

**Completion goal:** Users can find and connect to Nyx servers on their
local network without manually tracking IP addresses.

### Phase 06 --- PWA Experience

Make Nyx installable and convenient across supported devices.

Tasks:

-   Complete the PWA manifest.
-   Add and register a service worker.
-   Cache only the static application shell and versioned assets by
    default.
-   Add install guidance and supported installation flows.
-   Remember previously connected servers.
-   Add connection and reconnection states.
-   Provide clear errors when a server is unreachable.
-   Test on Android and desktop browsers.
-   Document local HTTPS and certificate setup.

Important technical constraint:

Service workers and full PWA functionality require a secure context,
normally HTTPS or localhost. A plain HTTP LAN IP address does not
provide the same capabilities. Local HTTPS and certificate trust must be
tested rather than assumed to work automatically.

Do not silently cache private files, file listings, authentication
tokens, uploads, downloads, or media in the service worker.

**Completion goal:** Nyx is installable and provides a reliable app-like
experience on supported devices.

### Phase 07 --- Security, Performance, and Release

Harden the complete application before treating it as production-ready.

Tasks:

-   Audit inherited authentication and authorization behavior.
-   Review upstream command execution and session revocation concerns.
-   Keep command execution disabled unless there is a separately
    reviewed and explicitly approved design.
-   Implement or verify session revocation.
-   Verify permissions for every file operation.
-   Rate-limit and minimize discovery responses.
-   Restrict media processing resources and execution time.
-   Test large directories and large media collections.
-   Test mobile and desktop file operations.
-   Test multiple Nyx instances on one network.
-   Improve Docker and self-hosted deployment documentation.
-   Write installation, backup, upgrade, and troubleshooting guides.
-   Prepare release builds and a clear versioning strategy.

**Completion goal:** A stable, secure, documented self-hosted release.

## 8. Architecture Principles

These principles should guide implementation decisions.

### Extend before replacing

The existing backend already provides valuable functionality. Prefer
additive changes and focused refactors over wholesale rewrites.

### One source of truth for files

The file manager and gallery should use the same underlying
file/resource model and existing APIs. Do not create a duplicate media
database unless a measured requirement justifies it.

### Separate presentation from data and services

Do not turn a single large Vue component into a collection of unrelated
conditional branches. Build focused views, components, and services.

Possible frontend organization as the product evolves:

``` text
frontend/src/
  api/
  components/
    files/
    gallery/
    layout/
    ui/
  services/
    discovery.ts
    pwa.ts
  stores/
  views/
    files/
      FileListing.vue
      GalleryView.vue
      Preview.vue
      Editor.vue
    ServerDiscovery.vue
```

Possible backend additions as needed:

``` text
discovery/
  service.go
  mdns.go
  identity.go
  discovery_test.go

media/
  service.go
  video.go
  metadata.go
  cache.go
  service_test.go

http/
  discovery.go
```

This is a proposed direction, not a mandate to create every file
immediately. Follow existing project conventions where they are sound.

### Preserve API compatibility

Do not casually change existing routes or response shapes. If an API
change is required, document the reason, affected callers, migration
path, and tests.

### Keep expensive work bounded

Media processing, discovery, and large-directory rendering must have
explicit limits and graceful failure behavior.

### Make changes reviewable

Prefer small, coherent commits. Avoid mixing branding cleanup, backend
redesign, UI overhaul, and media processing into one enormous change.

## 9. Security and Operational Constraints

-   Nyx is local-network-only. Do not add cloud relay or remote-access
    infrastructure.
-   Do not add telemetry or cloud file indexing.
-   Do not expose filesystem paths, usernames, or sensitive server
    details through discovery.
-   Each server must authenticate users independently.
-   Retain and verify existing permission checks.
-   Run the server with the least filesystem privileges practical.
-   Do not execute user-controlled shell commands.
-   Use safe process invocation for FFmpeg, with fixed argument
    construction, timeouts, concurrency limits, and input validation.
-   Treat local network access as a boundary, not as a substitute for
    security.
-   Preserve the upstream license and attribution obligations.

## 10. Development Workflow and Agent Instructions

Any AI assistant or coding agent working on Nyx should:

1.  Read this document first.
2.  Inspect the current repository and relevant files before editing.
3.  Confirm the active branch and working tree state.
4.  Understand existing code paths before proposing replacements.
5.  Follow the official phase order.
6.  Avoid implementing future-phase features prematurely unless needed
    for a small, explicitly documented technical validation.
7.  Preserve existing functionality unless the task explicitly changes
    it.
8.  Explain the intended files and behavior before a broad refactor.
9.  Make focused changes and add or update tests.
10. Run relevant tests, type checks, linting, and builds.
11. Report exactly what changed, what was tested, and what remains
    incomplete.
12. Never claim code was changed or tests passed unless that actually
    happened.

### Baseline commands

Use the repository's current scripts and version requirements. The
previously identified baseline commands are:

``` bash
cd frontend
pnpm install
pnpm run typecheck
pnpm run lint
pnpm run build

cd ..
go test ./...
```

The repository was previously identified as using Go 1.25, Node 24+, and
pnpm 10. Verify current requirements in the repository before installing
or upgrading toolchains.

## 11. Definition of Success

Nyx Browser is successful when:

-   It is clearly an independent product, not merely File Browser with a
    new logo.
-   The standard file manager remains powerful and practical.
-   The Gallery offers an excellent mixed image/video browsing
    experience.
-   Media previews are fast, cached, and resource-bounded.
-   The interface is minimal, dark, readable, and polished.
-   Frosted glass is used selectively rather than everywhere.
-   Users can eventually discover and connect to local Nyx instances.
-   The PWA experience works reliably on supported devices and secure
    origins.
-   Security, documentation, and deployment are good enough for real
    self-hosted use.

## 12. Current Priority

**The immediate development priority is Phase 01: Foundation and
Identity.**

After that, follow this exact sequence:

1.  Foundation and identity.
2.  Complete UI redesign.
3.  Nyx Gallery.
4.  Media processing pipeline.
5.  Local network discovery.
6.  PWA experience.
7.  Security, performance, and release.

Do not skip ahead and build the entire discovery/PWA system before the
core file manager and gallery are established.

------------------------------------------------------------------------

**Short version:** Nyx Browser is a minimal-dark, self-hosted,
local-network file manager with a separate immersive mixed-media
gallery. It is built by evolving the archived File Browser codebase, not
throwing away its working backend. The priority is to establish the
project, redesign the UI, build Gallery, create the media pipeline, and
only then tackle discovery, PWA infrastructure, and release hardening.
