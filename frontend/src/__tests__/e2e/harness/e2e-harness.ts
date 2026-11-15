/**
 * Hermetic E2E Test Harness for Nyx Browser Phase 02 UI Redesign
 *
 * Provides Node-compatible DOM/window/event stubs, virtual filesystem fixtures,
 * API interceptors, and reference contract implementations for:
 * - Reactive selection (useFileSelection contract)
 * - Natural alphanumeric sort with folders-first (useFileSort contract)
 * - Context menu coordinate clamping
 * - Lucide icon name translation map (65 mappings)
 * - Design tokens & CSS parsing helpers
 */

import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { ref, computed, type Ref, type ComputedRef } from "vue";

/* =========================================================================
 * 1. Virtual File System Fixtures
 * ========================================================================= */

export interface MockResource {
  path: string;
  name: string;
  size: number;
  modified: string;
  isDir: boolean;
  type?: string;
  items?: MockResource[];
}

export function createDefaultTestFS(): MockResource {
  return {
    path: "/files/",
    name: "files",
    size: 0,
    modified: "2026-11-15T12:00:00Z",
    isDir: true,
    items: [
      {
        path: "/files/Documents",
        name: "Documents",
        size: 0,
        modified: "2026-11-14T10:00:00Z",
        isDir: true,
        items: [
          {
            path: "/files/Documents/Architecture.pdf",
            name: "Architecture.pdf",
            size: 2516582, // 2.4 MB
            modified: "2026-11-15T09:12:00Z",
            isDir: false,
            type: "pdf",
          },
        ],
      },
      {
        path: "/files/Media",
        name: "Media",
        size: 0,
        modified: "2026-11-14T11:00:00Z",
        isDir: true,
        items: [
          {
            path: "/files/Media/Floorplan_v2.png",
            name: "Floorplan_v2.png",
            size: 5033164, // 4.8 MB
            modified: "2026-11-15T14:00:00Z",
            isDir: false,
            type: "image",
          },
          {
            path: "/files/Media/Demo_Recording.mp4",
            name: "Demo_Recording.mp4",
            size: 148897792, // 142.0 MB
            modified: "2026-11-10T16:45:00Z",
            isDir: false,
            type: "video",
          },
        ],
      },
      {
        path: "/files/README.md",
        name: "README.md",
        size: 1024,
        modified: "2026-11-15T08:00:00Z",
        isDir: false,
        type: "text",
      },
    ],
  };
}

export function createBoundaryTestFS(): {
  emptyDir: MockResource;
  singleItemDir: MockResource;
  specialCharsItems: MockResource[];
  longNameItem: MockResource;
} {
  return {
    emptyDir: {
      path: "/files/empty/",
      name: "empty",
      size: 0,
      modified: "2026-11-15T10:00:00Z",
      isDir: true,
      items: [],
    },
    singleItemDir: {
      path: "/files/single/",
      name: "single",
      size: 0,
      modified: "2026-11-15T10:00:00Z",
      isDir: true,
      items: [
        {
          path: "/files/single/solo.txt",
          name: "solo.txt",
          size: 2048,
          modified: "2026-11-15T10:05:00Z",
          isDir: false,
          type: "text",
        },
      ],
    },
    specialCharsItems: [
      {
        path: "/files/special/🎉_final_report_v2.pdf",
        name: "🎉_final_report_v2.pdf",
        size: 1048576,
        modified: "2026-11-15T11:00:00Z",
        isDir: false,
        type: "pdf",
      },
      {
        path: "/files/special/مستند_مهم.docx",
        name: "مستند_مهم.docx",
        size: 512000,
        modified: "2026-11-15T11:01:00Z",
        isDir: false,
        type: "doc",
      },
      {
        path: "/files/special/テスト_画像.png",
        name: "テスト_画像.png",
        size: 2097152,
        modified: "2026-11-15T11:02:00Z",
        isDir: false,
        type: "image",
      },
      {
        path: "/files/special/file #1 & 2 [draft] (copy).txt",
        name: "file #1 & 2 [draft] (copy).txt",
        size: 4096,
        modified: "2026-11-15T11:03:00Z",
        isDir: false,
        type: "text",
      },
    ],
    longNameItem: {
      path: `/files/${"a".repeat(251)}.png`,
      name: `${"a".repeat(251)}.png`,
      size: 8192,
      modified: "2026-11-15T11:04:00Z",
      isDir: false,
      type: "image",
    },
  };
}

export function createLargeTestFS(count = 1000): MockResource[] {
  const items: MockResource[] = [];
  for (let i = 0; i < count; i++) {
    const isFolder = i % 10 === 0;
    items.push({
      path: `/files/item_${i}`,
      name: isFolder ? `Folder_${i}` : `file_${i}.dat`,
      size: isFolder ? 0 : (i + 1) * 1024,
      modified: new Date(Date.now() - i * 60000).toISOString(),
      isDir: isFolder,
      type: isFolder ? "dir" : "data",
    });
  }
  return items;
}

/* =========================================================================
 * 2. DOM & Window Hermetic Stubs (Node-Compatible)
 * ========================================================================= */

interface StoredGlobals {
  window?: unknown;
  document?: unknown;
  localStorage?: unknown;
}

let storedGlobals: StoredGlobals = {};

class MockLocalStorage {
  private store: Map<string, string> = new Map();
  public errorMode = false;

  public getItem(key: string): string | null {
    if (this.errorMode) {
      throw new Error("SecurityError: The operation is insecure.");
    }
    return this.store.get(key) ?? null;
  }

  public setItem(key: string, value: string): void {
    if (this.errorMode) {
      throw new Error("SecurityError: The operation is insecure.");
    }
    this.store.set(key, String(value));
  }

  public removeItem(key: string): void {
    if (this.errorMode) {
      throw new Error("SecurityError: The operation is insecure.");
    }
    this.store.delete(key);
  }

  public clear(): void {
    if (this.errorMode) {
      throw new Error("SecurityError: The operation is insecure.");
    }
    this.store.clear();
  }

  public get length(): number {
    return this.store.size;
  }
}

export function setupDomEnvironment(initialState?: {
  innerWidth?: number;
  innerHeight?: number;
  localStorageEnabled?: boolean;
}): {
  window: any;
  document: any;
  localStorage: MockLocalStorage;
} {
  const mockStorage = new MockLocalStorage();
  if (initialState?.localStorageEnabled === false) {
    mockStorage.errorMode = true;
  }

  const listeners = new Map<string, Set<(e: any) => void>>();

  const mockWindow: any = {
    innerWidth: initialState?.innerWidth ?? 1280,
    innerHeight: initialState?.innerHeight ?? 800,
    localStorage: mockStorage,
    sessionStorage: new MockLocalStorage(),
    addEventListener: (event: string, cb: (e: any) => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set());
      listeners.get(event)!.add(cb);
    },
    removeEventListener: (event: string, cb: (e: any) => void) => {
      listeners.get(event)?.delete(cb);
    },
    dispatchEvent: (event: any) => {
      const type = event.type || event;
      const cbs = listeners.get(type);
      if (cbs) {
        cbs.forEach((cb) => cb(event));
      }
      return true;
    },
    matchMedia: (query: string) => ({
      matches: query.includes("768") ? mockWindow.innerWidth < 768 : false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => true,
    }),
    requestAnimationFrame: (cb: (time: number) => void) =>
      setTimeout(() => cb(Date.now()), 0),
    cancelAnimationFrame: (id: any) => clearTimeout(id),
  };

  const documentListeners = new Map<string, Set<(e: any) => void>>();

  const mockDocument: any = {
    body: {
      tagName: "BODY",
      classList: {
        contains: () => false,
        add: () => {},
        remove: () => {},
      },
      appendChild: (el: any) => el,
      removeChild: (el: any) => el,
      dispatchEvent: (event: any) => {
        const type = event.type || event;
        documentListeners.get(type)?.forEach((cb) => cb(event));
        return true;
      },
    },
    createElement: (tag: string) => {
      const elListeners = new Map<string, Set<(e: any) => void>>();
      return {
        tagName: tag.toUpperCase(),
        style: {},
        classList: {
          contains: () => false,
          add: () => {},
          remove: () => {},
        },
        setAttribute: () => {},
        getAttribute: () => null,
        appendChild: (child: any) => child,
        removeChild: (child: any) => child,
        addEventListener: (event: string, cb: (e: any) => void) => {
          if (!elListeners.has(event)) elListeners.set(event, new Set());
          elListeners.get(event)!.add(cb);
        },
        removeEventListener: (event: string, cb: (e: any) => void) => {
          elListeners.get(event)?.delete(cb);
        },
        dispatchEvent: (event: any) => {
          const type = event.type || event;
          elListeners.get(type)?.forEach((cb) => cb(event));
          return true;
        },
      };
    },
    addEventListener: (event: string, cb: (e: any) => void) => {
      if (!documentListeners.has(event))
        documentListeners.set(event, new Set());
      documentListeners.get(event)!.add(cb);
    },
    removeEventListener: (event: string, cb: (e: any) => void) => {
      documentListeners.get(event)?.delete(cb);
    },
    dispatchEvent: (event: any) => {
      const type = event.type || event;
      documentListeners.get(type)?.forEach((cb) => cb(event));
      return true;
    },
  };

  storedGlobals = {
    window: (globalThis as any).window,
    document: (globalThis as any).document,
    localStorage: (globalThis as any).localStorage,
  };

  (globalThis as any).window = mockWindow;
  (globalThis as any).document = mockDocument;
  (globalThis as any).localStorage = mockStorage;

  return {
    window: mockWindow,
    document: mockDocument,
    localStorage: mockStorage,
  };
}

export function teardownDomEnvironment(): void {
  if ("window" in storedGlobals)
    (globalThis as any).window = storedGlobals.window;
  if ("document" in storedGlobals)
    (globalThis as any).document = storedGlobals.document;
  if ("localStorage" in storedGlobals)
    (globalThis as any).localStorage = storedGlobals.localStorage;
  storedGlobals = {};
}

export function setViewport(width: number, height: number): void {
  if ((globalThis as any).window) {
    (globalThis as any).window.innerWidth = width;
    (globalThis as any).window.innerHeight = height;
    (globalThis as any).window.dispatchEvent({ type: "resize" });
  }
}

export function dispatchKeyboardEvent(
  key: string,
  code?: string,
  options?: {
    ctrlKey?: boolean;
    shiftKey?: boolean;
    metaKey?: boolean;
    target?: { tagName?: string };
  }
): {
  type: string;
  key: string;
  code: string;
  ctrlKey: boolean;
  shiftKey: boolean;
  metaKey: boolean;
  target: { tagName: string };
  defaultPrevented: boolean;
  preventDefault: () => void;
  stopPropagation: () => void;
} {
  const event = {
    type: "keydown",
    key,
    code: code || key,
    ctrlKey: !!options?.ctrlKey,
    shiftKey: !!options?.shiftKey,
    metaKey: !!options?.metaKey,
    target: options?.target || { tagName: "BODY" },
    defaultPrevented: false,
    preventDefault: () => {
      event.defaultPrevented = true;
    },
    stopPropagation: () => {},
  };

  if ((globalThis as any).window) {
    (globalThis as any).window.dispatchEvent(event);
  }
  return event;
}

export function dispatchContextMenu(
  clientX: number,
  clientY: number,
  target?: any
): {
  type: string;
  clientX: number;
  clientY: number;
  target: any;
  defaultPrevented: boolean;
  preventDefault: () => void;
} {
  const event = {
    type: "contextmenu",
    clientX,
    clientY,
    target: target || (globalThis as any).document?.body,
    defaultPrevented: false,
    preventDefault: () => {
      event.defaultPrevented = true;
    },
  };
  return event;
}

export function dispatchTouchEvent(
  type: "touchstart" | "touchmove" | "touchend",
  touches: Array<{ clientX: number; clientY: number }>,
  target?: any
): {
  type: string;
  touches: Array<{ clientX: number; clientY: number }>;
  target: any;
  defaultPrevented: boolean;
  propagationStopped: boolean;
  preventDefault: () => void;
  stopPropagation: () => void;
} {
  const event = {
    type,
    touches,
    target: target || (globalThis as any).document?.body,
    defaultPrevented: false,
    propagationStopped: false,
    preventDefault: () => {
      event.defaultPrevented = true;
    },
    stopPropagation: () => {
      event.propagationStopped = true;
    },
  };
  return event;
}

/* =========================================================================
 * 3. Reference Contracts: Selection, Sort, Clamping, and Icons
 * ========================================================================= */

/**
 * useFileSelection contract implementation & validator
 */
export interface FileSelectionContract {
  selected: Ref<number[]>;
  selectedCount: ComputedRef<number>;
  isSelected: (idx: number) => boolean;
  select: (idx: number) => void;
  toggle: (idx: number) => void;
  rangeSelect: (start: number, end: number) => void;
  selectAll: (total?: number) => void;
  clear: () => void;
}

export function createFileSelectionContract(
  itemsRef?: Ref<any[]>
): FileSelectionContract {
  const selected = ref<number[]>([]);
  const selectedCount = computed(() => selected.value.length);

  function isSelected(idx: number): boolean {
    return selected.value.includes(idx);
  }

  function select(idx: number): void {
    if (!selected.value.includes(idx)) {
      selected.value = [...selected.value, idx].sort((a, b) => a - b);
    }
  }

  function toggle(idx: number): void {
    if (selected.value.includes(idx)) {
      selected.value = selected.value.filter((i) => i !== idx);
    } else {
      select(idx);
    }
  }

  function rangeSelect(start: number, end: number): void {
    const min = Math.max(0, Math.min(start, end));
    const max = Math.max(start, end);
    const newIndices: number[] = [];
    for (let i = min; i <= max; i++) {
      newIndices.push(i);
    }
    selected.value = Array.from(
      new Set([...selected.value, ...newIndices])
    ).sort((a, b) => a - b);
  }

  function selectAll(total?: number): void {
    const count = total ?? (itemsRef ? itemsRef.value.length : 0);
    const all: number[] = [];
    for (let i = 0; i < count; i++) {
      all.push(i);
    }
    selected.value = all;
  }

  function clear(): void {
    selected.value = [];
  }

  return {
    selected,
    selectedCount,
    isSelected,
    select,
    toggle,
    rangeSelect,
    selectAll,
    clear,
  };
}

/**
 * useFileSort contract implementation & validator
 */
export interface FileSortContract {
  sortKey: Ref<"name" | "size" | "modified">;
  sortAsc: Ref<boolean>;
  setSort: (key: "name" | "size" | "modified") => void;
  sortItems: <
    T extends {
      isDir?: boolean;
      name: string;
      size?: number;
      modified?: string;
    },
  >(
    items: T[]
  ) => T[];
}

export function createFileSortContract(): FileSortContract {
  const sortKey = ref<"name" | "size" | "modified">("name");
  const sortAsc = ref<boolean>(true);

  function setSort(key: "name" | "size" | "modified"): void {
    if (sortKey.value === key) {
      sortAsc.value = !sortAsc.value;
    } else {
      sortKey.value = key;
      sortAsc.value = true;
    }
  }

  const collator = new Intl.Collator(undefined, {
    numeric: true,
    sensitivity: "base",
  });

  function sortItems<
    T extends {
      isDir?: boolean;
      name: string;
      size?: number;
      modified?: string;
    },
  >(items: T[]): T[] {
    const copy = [...items];
    copy.sort((a, b) => {
      // 1. Folders first invariant: isDir: true always precedes isDir: false
      const aDir = !!a.isDir;
      const bDir = !!b.isDir;
      if (aDir !== bDir) {
        return aDir ? -1 : 1;
      }

      // 2. Collation by selected key
      let result = 0;
      if (sortKey.value === "name") {
        result = collator.compare(a.name || "", b.name || "");
      } else if (sortKey.value === "size") {
        const aSize = a.size ?? 0;
        const bSize = b.size ?? 0;
        result = aSize - bSize;
      } else if (sortKey.value === "modified") {
        const aTime = a.modified ? new Date(a.modified).getTime() : 0;
        const bTime = b.modified ? new Date(b.modified).getTime() : 0;
        result = aTime - bTime;
      }

      return sortAsc.value ? result : -result;
    });
    return copy;
  }

  return { sortKey, sortAsc, setSort, sortItems };
}

/**
 * Context menu boundary coordinate clamping calculator
 */
export function clampContextMenuPosition(
  x: number,
  y: number,
  menuW: number,
  menuH: number,
  winW: number,
  winH: number
): { x: number; y: number } {
  const margin = 8;
  const clampedX = Math.max(margin, Math.min(x, winW - menuW - margin));
  const clampedY = Math.max(margin, Math.min(y, winH - menuH - margin));
  return { x: clampedX, y: clampedY };
}

/**
 * Authoritative 65-entry Legacy Material -> Lucide Icon Mapping Dictionary
 */
export const ICON_MAPPINGS: Record<string, string> = {
  close: "X",
  clear: "X",
  cancel: "X",
  search: "Search",
  menu: "Menu",
  more_vert: "MoreVertical",
  more_horiz: "MoreHorizontal",
  mode_edit: "Pencil",
  edit_note: "FileEdit",
  delete: "Trash2",
  file_download: "Download",
  download: "Download",
  file_upload: "Upload",
  upload: "Upload",
  content_copy: "Copy",
  copy: "Copy",
  forward: "ArrowRight",
  move: "ArrowRight",
  content_cut: "Scissors",
  content_paste: "Clipboard",
  content_paste_go: "ClipboardCopy",
  check_circle: "CheckCircle",
  done_all: "CheckCheck",
  chevron_left: "ChevronLeft",
  chevron_right: "ChevronRight",
  keyboard_arrow_right: "ChevronRight",
  arrow_upward: "ArrowUp",
  arrow_downward: "ArrowDown",
  arrow_drop_down: "ChevronDown",
  folder: "Folder",
  create_new_folder: "FolderPlus",
  folder_copy: "FolderSync",
  folder_zip: "FolderArchive",
  insert_drive_file: "File",
  description: "FileText",
  note_add: "FilePlus",
  image: "Image",
  movie: "Film",
  volume_up: "Volume2",
  code: "Code",
  settings_applications: "Settings",
  settings: "Settings",
  person: "User",
  person_add: "UserPlus",
  exit_to_app: "LogOut",
  logout: "LogOut",
  open_in_new: "ExternalLink",
  photo_size_select_large: "Maximize2",
  hd: "Tv",
  view_module: "LayoutGrid",
  grid_view: "Grid",
  view_list: "List",
  slideshow: "Presentation",
  border_all: "Table",
  web_asset: "Terminal",
  archive: "Package",
  album: "Disc",
  font_download: "Type",
  link_off: "Unlink",
  sentiment_dissatisfied: "Frown",
  feedback: "HelpCircle",
  info: "Info",
  error: "AlertTriangle",
  add: "Plus",
  remove: "Minus",
  save: "Save",
  share: "Share2",
  undo: "Undo",
  replay: "RotateCcw",
  home: "Home",
  checklist: "ListChecks",
};

/**
 * Authoritative 21 Nothing-inspired Design Tokens Dictionary
 */
export const EXPECTED_DESIGN_TOKENS: Record<string, string> = {
  // Surfaces
  "--nyx-bg-canvas": "#000000",
  "--nyx-bg-chassis": "#080808",
  "--nyx-surface-1": "#101010",
  "--nyx-surface-2": "#161616",
  "--nyx-surface-3": "#222222",
  "--nyx-surface-selected": "#1a1213",

  // Borders
  "--nyx-border-hairline": "rgba(255, 255, 255, 0.06)",
  "--nyx-border-subtle": "rgba(255, 255, 255, 0.10)",
  "--nyx-border-strong": "rgba(255, 255, 255, 0.18)",
  "--nyx-border-focus": "#ffffff",
  "--nyx-border-selected": "#d71921",

  // Typography
  "--nyx-text-primary": "#ffffff",
  "--nyx-text-secondary": "#a0a0a0",
  "--nyx-text-muted": "#666666",
  "--nyx-text-disabled": "#3a3a3a",

  // Technical Red
  "--nyx-accent-red": "#d71921",
  "--nyx-accent-red-hover": "#b8141b",
  "--nyx-accent-red-subtle": "rgba(215, 25, 33, 0.12)",
  "--nyx-accent-red-glow": "rgba(215, 25, 33, 0.35)",

  // Radii
  "--nyx-radius-xs": "2px",
  "--nyx-radius-sm": "4px",
  "--nyx-radius-md": "6px",
  "--nyx-radius-lg": "8px",

  // Frosted Glass
  "--nyx-glass-bg": "rgba(14, 14, 14, 0.88)",
  "--nyx-glass-blur": "blur(16px)",
  "--nyx-glass-border": "1px solid rgba(255, 255, 255, 0.12)",
};

/* =========================================================================
 * 4. CSS Inspection & Parsing Utilities
 * ========================================================================= */

export function loadAndCleanCss(relativePath: string): {
  raw: string;
  clean: string;
  normalized: string;
  exists: boolean;
} {
  const fullPath = resolve(__dirname, relativePath);
  if (!existsSync(fullPath)) {
    return { raw: "", clean: "", normalized: "", exists: false };
  }
  const raw = readFileSync(fullPath, "utf8");
  const clean = raw.replace(/\/\*[\s\S]*?\*\//g, "");
  const normalized = clean.replace(/\s+/g, " ");
  return { raw, clean, normalized, exists: true };
}

export function getCssDeclarationBlock(
  css: string,
  selectorPattern: string
): string | null {
  const match = css.match(new RegExp(selectorPattern + "\\s*\\{([^}]+)\\}"));
  return match ? match[1].trim() : null;
}

export function parseCssVariables(css: string): Record<string, string> {
  const vars: Record<string, string> = {};
  const regex = /(--[\w-]+)\s*:\s*([^;]+);/g;
  let match: RegExpExecArray | null;
  while ((match = regex.exec(css)) !== null) {
    vars[match[1].trim()] = match[2].trim();
  }
  return vars;
}
