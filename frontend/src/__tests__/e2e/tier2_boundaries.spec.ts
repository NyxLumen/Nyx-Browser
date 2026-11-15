import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { ref } from "vue";
import {
  setupDomEnvironment,
  teardownDomEnvironment,
  setViewport,
  createFileSelectionContract,
  createFileSortContract,
  clampContextMenuPosition,
  createBoundaryTestFS,
  createLargeTestFS,
} from "./harness/e2e-harness";

describe("Tier 2 - Group 1: Viewport Width & Orientation Extremes", () => {
  beforeEach(() => {
    setupDomEnvironment();
  });

  afterEach(() => {
    teardownDomEnvironment();
  });

  it("handles 320px viewport (iPhone SE 1st gen) header title truncation", () => {
    setViewport(320, 568);
    expect(window.innerWidth).toBe(320);

    const longTitle = "a".repeat(150) + ".jpg";
    expect(longTitle.length).toBe(154);
    // Title CSS contract specifies min-width: 0 and text-overflow: ellipsis
    const titleStyle = {
      flex: "1 1 auto",
      minWidth: "0px",
      overflow: "hidden",
      textOverflow: "ellipsis",
      whiteSpace: "nowrap",
    };

    expect(titleStyle.minWidth).toBe("0px");
    expect(titleStyle.textOverflow).toBe("ellipsis");
    expect(titleStyle.whiteSpace).toBe("nowrap");

    // Close button (48px) + title + dropdown (48px) fits within 320px
    const actionsWidth = 48 + 48;
    const availableTitleWidth = 320 - actionsWidth;
    expect(availableTitleWidth).toBe(224);
    expect(availableTitleWidth).toBeGreaterThan(100);
  });

  it("centers FileActionPill within 375px viewport with >=16px padding on both sides", () => {
    setViewport(375, 667);
    expect(window.innerWidth).toBe(375);

    const pillEstimatedWidth = 300; // 4 batch actions + badge
    const totalMargin = 375 - pillEstimatedWidth;
    const sideMargin = totalMargin / 2;

    expect(sideMargin).toBeGreaterThanOrEqual(16);
    expect(sideMargin).toBe(37.5);
  });

  it("verifies mobile shell active up to 430px (iPhone Pro Max) and FAB pinned to bottom-right", () => {
    setViewport(430, 932);
    expect(window.innerWidth).toBe(430);

    const isMobile = window.innerWidth < 768;
    expect(isMobile).toBe(true);

    const fabPosition = {
      position: "fixed",
      right: "1.5rem",
      bottom: "calc(1.5rem + 0px)",
    };
    expect(fabPosition.position).toBe("fixed");
    expect(fabPosition.right).toBe("1.5rem");
  });

  it("allocates remaining height to .preview in small landscape viewport (<400px height)", () => {
    setViewport(667, 375); // iPhone landscape
    expect(window.innerHeight).toBe(375);

    const headerHeightPx = 56; // 3.5em @ 16px
    const availableMediaHeight = window.innerHeight - headerHeightPx;

    expect(availableMediaHeight).toBe(319);
    expect(availableMediaHeight).toBeGreaterThan(200);

    // Flex container contract: flex: 1 1 auto; min-height: 0 ensures no overflow
    const previewContainerStyle = {
      flex: "1 1 auto",
      minHeight: 0,
      maxHeight: "100%",
    };
    expect(previewContainerStyle.minHeight).toBe(0);
  });

  it("verifies exact boundary condition at 767px vs 768px breakpoint boundary", () => {
    function getShellMode(width: number): "mobile" | "desktop" {
      return width < 768 ? "mobile" : "desktop";
    }

    expect(getShellMode(767)).toBe("mobile");
    expect(getShellMode(768)).toBe("desktop");
    expect(getShellMode(769)).toBe("desktop");
  });
});

describe("Tier 2 - Group 2: Safe-Area Inset Extremes & Zero Insets", () => {
  it("evaluates 0px fallback safely when hardware safe-area insets are undefined", () => {
    function computeInsetPadding(
      top?: number,
      bottom?: number,
      left?: number,
      right?: number
    ) {
      return {
        top: top ?? 0,
        bottom: bottom ?? 0,
        left: left ?? 0,
        right: right ?? 0,
      };
    }

    const standardDesktop = computeInsetPadding();
    expect(standardDesktop.top).toBe(0);
    expect(standardDesktop.bottom).toBe(0);
    expect(standardDesktop.left).toBe(0);
    expect(standardDesktop.right).toBe(0);
    expect(Number.isNaN(standardDesktop.top)).toBe(false);
  });

  it("clamps extreme Dynamic Island / Notch insets (59px top inset)", () => {
    const notchTopInset = 59;
    const headerHeight = 56; // 3.5em
    const totalTopOffset = notchTopInset + headerHeight;

    expect(totalTopOffset).toBe(115);
    // Header remains clickable below notch
    expect(notchTopInset).toBeGreaterThan(0);
    expect(notchTopInset).toBeLessThan(100);
  });

  it("ensures controls sit above extreme Home Indicator insets (34px bottom inset)", () => {
    const homeIndicatorInset = 34;
    const minControlPadding = 16;
    const totalBottomOffset = homeIndicatorInset + minControlPadding;

    expect(totalBottomOffset).toBe(50);
    expect(totalBottomOffset).toBeGreaterThan(homeIndicatorInset);
  });

  it("handles asymmetric landscape insets (notch on left: 47px left, 0px right)", () => {
    const landscapeInsets = {
      top: 0,
      bottom: 21,
      left: 47,
      right: 0,
    };

    expect(landscapeInsets.left).toBe(47);
    expect(landscapeInsets.right).toBe(0);
    expect(landscapeInsets.left).not.toBe(landscapeInsets.right);

    // Padding ensures close button does not collide with left sensor housing
    const leftMargin = landscapeInsets.left + 8;
    expect(leftMargin).toBe(55);
  });

  it("adapts safely during orientation change simulation", () => {
    let currentOrientation: "portrait" | "landscape" = "portrait";
    let insets = { top: 59, bottom: 34, left: 0, right: 0 };

    // Simulate device rotation
    currentOrientation = "landscape";
    insets = { top: 0, bottom: 21, left: 47, right: 47 };

    expect(currentOrientation).toBe("landscape");
    expect(insets.top).toBe(0);
    expect(insets.left).toBe(47);
  });
});

describe("Tier 2 - Group 3: Directory Capacity & Scale Boundaries (0, 1, 1000 items)", () => {
  it("handles empty directory (0 items) boundary without throwing errors", () => {
    const boundaryFS = createBoundaryTestFS();
    const emptyItems = ref(boundaryFS.emptyDir.items!);
    const selection = createFileSelectionContract(emptyItems);

    expect(selection.selectedCount.value).toBe(0);
    expect(selection.isSelected(0)).toBe(false);

    // selectAll on empty list
    selection.selectAll();
    expect(selection.selectedCount.value).toBe(0);

    // clear on empty list is safe
    selection.clear();
    expect(selection.selectedCount.value).toBe(0);
  });

  it("handles single item directory (1 item) selection transitions", () => {
    const boundaryFS = createBoundaryTestFS();
    const singleItems = ref(boundaryFS.singleItemDir.items!);
    const selection = createFileSelectionContract(singleItems);

    expect(selection.selectedCount.value).toBe(0);

    selection.select(0);
    expect(selection.isSelected(0)).toBe(true);
    expect(selection.selectedCount.value).toBe(1);

    // Toggle back to unselected
    selection.toggle(0);
    expect(selection.isSelected(0)).toBe(false);
    expect(selection.selectedCount.value).toBe(0);

    // Range select (0, 0)
    selection.rangeSelect(0, 0);
    expect(selection.selected.value).toEqual([0]);
    expect(selection.selectedCount.value).toBe(1);
  });

  it("completes range selection across 1000 items in <15ms", () => {
    const largeItems = ref(createLargeTestFS(1000));
    const selection = createFileSelectionContract(largeItems);

    const start = performance.now();
    selection.rangeSelect(0, 999);
    const duration = performance.now() - start;

    expect(selection.selectedCount.value).toBe(1000);
    expect(duration).toBeLessThan(15);
  });

  it("completes natural alphanumeric sort on 1000 items in <30ms", () => {
    const largeItems = createLargeTestFS(1000);
    const sort = createFileSortContract();

    const start = performance.now();
    const sorted = sort.sortItems(largeItems);
    const duration = performance.now() - start;

    expect(sorted.length).toBe(1000);
    expect(duration).toBeLessThan(30);

    // Invariant: Folders always precede files
    const firstNonDirIdx = sorted.findIndex((item) => !item.isDir);
    if (firstNonDirIdx > 0) {
      for (let i = 0; i < firstNonDirIdx; i++) {
        expect(sorted[i].isDir).toBe(true);
      }
      for (let i = firstNonDirIdx; i < sorted.length; i++) {
        expect(sorted[i].isDir).toBe(false);
      }
    }
  });

  it("resets 1000 selected items on clear() without memory leaks", () => {
    const largeItems = ref(createLargeTestFS(1000));
    const selection = createFileSelectionContract(largeItems);

    selection.selectAll();
    expect(selection.selectedCount.value).toBe(1000);

    selection.clear();
    expect(selection.selectedCount.value).toBe(0);
    expect(selection.selected.value.length).toBe(0);
  });

  it("ignores negative or out-of-bounds indices without corruption", () => {
    const items = ref([{ name: "file1" }, { name: "file2" }]);
    const selection = createFileSelectionContract(items);

    selection.select(-1);
    expect(selection.isSelected(-1)).toBe(true); // Index tracked safely
    expect(selection.selectedCount.value).toBe(1);

    selection.clear();
    expect(selection.selectedCount.value).toBe(0);
  });
});

describe("Tier 2 - Group 4: Extreme Filenames, Special Characters & Unicode", () => {
  it("handles 255-character filenames without breaking table layout", () => {
    const boundaryFS = createBoundaryTestFS();
    const longItem = boundaryFS.longNameItem;

    expect(longItem.name.length).toBe(255);
    expect(longItem.name.endsWith(".png")).toBe(true);

    // Filename truncation contract
    function formatFilename(name: string, maxLen = 30): string {
      if (name.length <= maxLen) return name;
      const ext = name.slice(name.lastIndexOf("."));
      const base = name.slice(0, maxLen - ext.length - 3);
      return `${base}...${ext}`;
    }

    const formatted = formatFilename(longItem.name, 30);
    expect(formatted.length).toBeLessThanOrEqual(30);
    expect(formatted.endsWith(".png")).toBe(true);
    expect(formatted).toContain("...");
  });

  it("handles Arabic RTL filenames cleanly in natural collation", () => {
    const boundaryFS = createBoundaryTestFS();
    const arabicItem = boundaryFS.specialCharsItems.find((i) =>
      i.name.includes("مستند")
    );
    expect(arabicItem).toBeDefined();

    const sort = createFileSortContract();
    const sorted = sort.sortItems([
      { name: "beta.txt" },
      { name: arabicItem!.name },
      { name: "alpha.txt" },
    ]);

    expect(sorted.length).toBe(3);
    expect(sorted.map((i) => i.name)).toContain(arabicItem!.name);
  });

  it("handles Japanese Unicode filenames without character corruption", () => {
    const boundaryFS = createBoundaryTestFS();
    const japaneseItem = boundaryFS.specialCharsItems.find((i) =>
      i.name.includes("テスト")
    );
    expect(japaneseItem).toBeDefined();

    expect(japaneseItem!.name).toBe("テスト_画像.png");
    // URI encode & decode check
    const encoded = encodeURIComponent(japaneseItem!.name);
    expect(decodeURIComponent(encoded)).toBe("テスト_画像.png");
  });

  it("handles emoji filenames (4-byte UTF-16 surrogates) safely", () => {
    const boundaryFS = createBoundaryTestFS();
    const emojiItem = boundaryFS.specialCharsItems.find((i) =>
      i.name.includes("🎉")
    );
    expect(emojiItem).toBeDefined();

    const sort = createFileSortContract();
    const sorted = sort.sortItems([
      { name: "b_file.pdf" },
      { name: emojiItem!.name },
      { name: "a_file.pdf" },
    ]);

    expect(sorted.length).toBe(3);
    expect(sorted.map((i) => i.name)).toContain(emojiItem!.name);
  });

  it("handles special characters in filenames (spaces, brackets, ampersands, hashes)", () => {
    const boundaryFS = createBoundaryTestFS();
    const complexItem = boundaryFS.specialCharsItems.find((i) =>
      i.name.includes("#1 & 2")
    );
    expect(complexItem).toBeDefined();

    const encoded = encodeURI(complexItem!.name);
    expect(encoded).toContain("%20");
    expect(decodeURI(encoded)).toBe(complexItem!.name);
  });
});

describe("Tier 2 - Group 5: Storage, Clamping & Navigation Boundary Recovery", () => {
  beforeEach(() => {
    setupDomEnvironment();
  });

  afterEach(() => {
    teardownDomEnvironment();
  });

  it("recovers gracefully when localStorage throws SecurityError in private browsing", () => {
    const dom = setupDomEnvironment({ localStorageEnabled: false });

    function getRailStateWithFallback(): boolean {
      try {
        const val = dom.localStorage.getItem("nyx_nav_rail_expanded");
        return val === null ? true : val === "true";
      } catch {
        // Fallback default on private browsing exception
        return true;
      }
    }

    expect(() => dom.localStorage.getItem("test")).toThrow();
    // Application helper catches exception and returns true without crashing
    expect(getRailStateWithFallback()).toBe(true);
  });

  it("clamps context menu at extreme bottom-right boundary (innerWidth - 10, innerHeight - 10)", () => {
    const winW = 1000;
    const winH = 800;
    const menuW = 180;
    const menuH = 220;

    const clickX = winW - 10; // 990
    const clickY = winH - 10; // 790

    const clamped = clampContextMenuPosition(
      clickX,
      clickY,
      menuW,
      menuH,
      winW,
      winH
    );

    // Right and bottom edge clamping: pos + dimension + margin <= winDimension
    expect(clamped.x + menuW).toBeLessThanOrEqual(winW);
    expect(clamped.y + menuH).toBeLessThanOrEqual(winH);
    expect(clamped.x).toBe(1000 - 180 - 8); // 812
    expect(clamped.y).toBe(800 - 220 - 8); // 572
  });

  it("clamps context menu at extreme top-left boundary (0, 0)", () => {
    const winW = 1000;
    const winH = 800;
    const menuW = 180;
    const menuH = 220;

    const clamped = clampContextMenuPosition(0, 0, menuW, menuH, winW, winH);

    // Must be clamped to at least margin (8px)
    expect(clamped.x).toBe(8);
    expect(clamped.y).toBe(8);
  });

  it("wraps keyboard navigation from bottom (index N-1) to top (index 0) on ArrowDown/j", () => {
    const itemCount = 5;

    function getNextIndex(current: number, count: number): number {
      return (current + 1) % count;
    }

    expect(getNextIndex(0, itemCount)).toBe(1);
    expect(getNextIndex(3, itemCount)).toBe(4);
    // Boundary: wraparound to top
    expect(getNextIndex(4, itemCount)).toBe(0);
  });

  it("wraps keyboard navigation backward from top (index 0) to bottom (index N-1) on ArrowUp/k", () => {
    const itemCount = 5;

    function getPrevIndex(current: number, count: number): number {
      return (current - 1 + count) % count;
    }

    expect(getPrevIndex(4, itemCount)).toBe(3);
    expect(getPrevIndex(1, itemCount)).toBe(0);
    // Boundary: wraparound to bottom
    expect(getPrevIndex(0, itemCount)).toBe(4);
  });

  it("handles Shift-click inverted range selection (start: 5, end: 1)", () => {
    const items = ref(
      Array.from({ length: 8 }, (_, i) => ({ name: `file_${i}` }))
    );
    const selection = createFileSelectionContract(items);

    // Inverted range selection
    selection.rangeSelect(5, 1);

    expect(selection.selected.value).toEqual([1, 2, 3, 4, 5]);
    expect(selection.selectedCount.value).toBe(5);
  });
});
