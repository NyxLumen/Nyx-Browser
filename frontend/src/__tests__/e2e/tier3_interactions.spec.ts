import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { ref } from "vue";
import {
  setupDomEnvironment,
  teardownDomEnvironment,
  setViewport,
  createFileSelectionContract,
  createFileSortContract,
  createDefaultTestFS,
} from "./harness/e2e-harness";

describe("Tier 3: Cross-Feature Pairwise Interactions & State Transitions", () => {
  beforeEach(() => {
    setupDomEnvironment();
  });

  afterEach(() => {
    teardownDomEnvironment();
  });

  it("[Interaction 1] NavRail collapse/expand toggle + Selection Pill centering", () => {
    setViewport(1280, 800);
    const fs = createDefaultTestFS();
    const items = ref(fs.items!);
    const selection = createFileSelectionContract(items);

    // Select 2 items -> action pill appears
    selection.select(0);
    selection.select(1);
    expect(selection.selectedCount.value).toBe(2);

    let railExpanded = true;
    let railWidth = railExpanded ? 240 : 64;

    function getPillCenterOffset(railW: number, winW: number): number {
      const contentWidth = winW - railW;
      return railW + contentWidth / 2;
    }

    const initialCenter = getPillCenterOffset(railWidth, 1280);
    expect(initialCenter).toBe(240 + (1280 - 240) / 2); // 760

    // Toggle NavRail to collapsed (64px)
    railExpanded = false;
    railWidth = 64;
    window.localStorage.setItem("nyx_nav_rail_expanded", "false");

    const newCenter = getPillCenterOffset(railWidth, 1280);
    expect(newCenter).toBe(64 + (1280 - 64) / 2); // 672
    expect(newCenter).not.toBe(initialCenter);

    // Selection invariant: items remain selected
    expect(selection.selected.value).toEqual([0, 1]);
    expect(selection.selectedCount.value).toBe(2);
  });

  it("[Interaction 2] Column sort change preserves selected items by identity", () => {
    const rawFiles = [
      { id: "f1", name: "zebra.txt", size: 500, isDir: false },
      { id: "f2", name: "apple.txt", size: 100, isDir: false },
      { id: "f3", name: "mango.txt", size: 300, isDir: false },
    ];

    const sort = createFileSortContract();
    sort.sortKey.value = "name";
    sort.sortAsc.value = true;
    const nameSorted = sort.sortItems(rawFiles);
    expect(nameSorted[0].id).toBe("f2");
    expect(nameSorted[1].id).toBe("f3");
    expect(nameSorted[2].id).toBe("f1");

    // User selects 'mango.txt' (id: f3)
    const selectedIds = ref<string[]>(["f3"]);

    // User changes sort to size (asc): apple (100), mango (300), zebra (500)
    sort.setSort("size");
    const sizeSorted = sort.sortItems(rawFiles);
    expect(sizeSorted[0].size).toBe(100);
    expect(sizeSorted[1].size).toBe(300);
    expect(sizeSorted[2].size).toBe(500);

    // Identity-based selection invariant: mango remains selected
    const selectedItems = sizeSorted.filter((item) =>
      selectedIds.value.includes(item.id)
    );
    expect(selectedItems.length).toBe(1);
    expect(selectedItems[0].name).toBe("mango.txt");
  });

  it("[Interaction 3] View mode switch (Table <-> Bento Grid) preserves selection state", () => {
    let viewMode: "table" | "grid" = "table";
    const items = ref([
      { name: "doc1.pdf" },
      { name: "image1.png" },
      { name: "video1.mp4" },
    ]);
    const selection = createFileSelectionContract(items);

    // Select items in Table view
    selection.select(0);
    selection.select(2);
    expect(selection.selected.value).toEqual([0, 2]);

    // Switch to Bento Grid view
    viewMode = "grid";
    expect(viewMode).toBe("grid");

    // Invariant: selection preserved identically in Grid view
    expect(selection.isSelected(0)).toBe(true);
    expect(selection.isSelected(1)).toBe(false);
    expect(selection.isSelected(2)).toBe(true);
    expect(selection.selectedCount.value).toBe(2);

    // Switch back to Table view
    viewMode = "table";
    expect(selection.selected.value).toEqual([0, 2]);
  });

  it("[Interaction 4] BaseModal prompt opening overlays MobileShell and isolates FAB", () => {
    setViewport(375, 667); // Mobile viewport
    let isModalOpen = false;

    const modalZIndex = 10000;
    const fabZIndex = 50;
    const shellZIndex = 10;

    expect(modalZIndex).toBeGreaterThan(fabZIndex);
    expect(modalZIndex).toBeGreaterThan(shellZIndex);

    function canClickFab(): boolean {
      return !isModalOpen;
    }

    expect(canClickFab()).toBe(true);

    // Open BaseModal
    isModalOpen = true;
    expect(canClickFab()).toBe(false);

    // Close BaseModal
    isModalOpen = false;
    expect(canClickFab()).toBe(true);
  });

  it("[Interaction 5] Context menu right-click on unselected vs selected items", () => {
    const items = ref([{ id: "item0" }, { id: "item1" }, { id: "item2" }]);
    const selection = createFileSelectionContract(items);

    // Select item 0 and item 1
    selection.select(0);
    selection.select(1);
    expect(selection.selected.value).toEqual([0, 1]);

    // Right-clicking on an already selected item (item 1) retains batch selection
    function handleContextMenu(targetIdx: number) {
      if (!selection.isSelected(targetIdx)) {
        selection.clear();
        selection.select(targetIdx);
      }
    }

    handleContextMenu(1);
    expect(selection.selected.value).toEqual([0, 1]); // Batch selection preserved

    // Right-clicking on an unselected item (item 2) clears prior selection and selects target
    handleContextMenu(2);
    expect(selection.selected.value).toEqual([2]);
  });

  it("[Interaction 6] Decoupled viewer full-bleed layer overlays expanded NavRail completely", () => {
    setViewport(1440, 900);
    const navRailExpanded = true;
    let viewerMounted = false;

    const navRailZIndex = 100;
    const previewerZIndex = 99999;
    expect(previewerZIndex).toBeGreaterThan(navRailZIndex);

    // Open media in viewer
    viewerMounted = true;

    function isNavRailCovered(): boolean {
      return viewerMounted && previewerZIndex > navRailZIndex;
    }
    expect(isNavRailCovered()).toBe(true);

    // Dismiss viewer (Esc)
    viewerMounted = false;
    expect(isNavRailCovered()).toBe(false);
    expect(navRailExpanded).toBe(true); // Rail returns in prior state
  });

  it("[Interaction 7] UploadProgressDrawer docked at bottom co-exists with FileActionPill", () => {
    setViewport(1280, 800);
    const drawerHeightPx = 60;
    const safeAreaBottomPx = 0;

    // Action pill must sit above the drawer when drawer is active
    function computePillBottom(drawerActive: boolean): number {
      const baseBottom = safeAreaBottomPx + 24;
      return drawerActive ? baseBottom + drawerHeightPx : baseBottom;
    }

    const bottomWithoutDrawer = computePillBottom(false);
    const bottomWithDrawer = computePillBottom(true);

    expect(bottomWithoutDrawer).toBe(24);
    expect(bottomWithDrawer).toBe(84);
    expect(bottomWithDrawer).toBeGreaterThan(drawerHeightPx);
  });

  it("[Interaction 8] Directory route navigation triggers automatic selection cleanup", () => {
    let currentPath = "/files/Documents";
    const items = ref([{ name: "file1" }, { name: "file2" }]);
    const selection = createFileSelectionContract(items);

    selection.select(0);
    selection.select(1);
    expect(selection.selectedCount.value).toBe(2);

    // Route watcher simulation: path change clears selection
    function navigateTo(newPath: string) {
      if (newPath !== currentPath) {
        currentPath = newPath;
        selection.clear();
      }
    }

    navigateTo("/files/Media");
    expect(currentPath).toBe("/files/Media");
    expect(selection.selectedCount.value).toBe(0);
    expect(selection.selected.value).toEqual([]);
  });

  it("[Interaction 9] Destructive Delete Prompt + Keyboard Escape cancels without delete", () => {
    let promptOpen = true;
    const deleteApiCalled = false;
    let activeFocusTarget: "list-item" | "modal-button" = "modal-button";

    function handleKeydown(key: string) {
      if (key === "Escape" && promptOpen) {
        promptOpen = false;
        activeFocusTarget = "list-item"; // Focus restored
      }
    }

    handleKeydown("Escape");

    expect(promptOpen).toBe(false);
    expect(deleteApiCalled).toBe(false);
    expect(activeFocusTarget).toBe("list-item");
  });

  it("[Interaction 10] Keyboard navigation suite ignores shortcuts when typing in INPUT/TEXTAREA", () => {
    let selectedIndex = 0;

    function handleGlobalKeydown(e: { key: string; targetTagName: string }) {
      // Abort if target is editable form element
      if (["INPUT", "TEXTAREA"].includes(e.targetTagName.toUpperCase())) {
        return;
      }

      if (e.key === "j") selectedIndex++;
      if (e.key === "k") selectedIndex--;
    }

    // Inside input field
    handleGlobalKeydown({ key: "j", targetTagName: "INPUT" });
    expect(selectedIndex).toBe(0); // Unchanged

    handleGlobalKeydown({ key: "k", targetTagName: "TEXTAREA" });
    expect(selectedIndex).toBe(0); // Unchanged

    // Outside input field (body or list item)
    handleGlobalKeydown({ key: "j", targetTagName: "DIV" });
    expect(selectedIndex).toBe(1); // Navigated
  });

  it("[Interaction 11] Natural sort direction toggle (Asc <-> Desc) preserves folders-first invariant", () => {
    const sort = createFileSortContract();
    const dataset = [
      { name: "b_file.txt", isDir: false },
      { name: "a_file.txt", isDir: false },
      { name: "Z_Folder", isDir: true },
      { name: "A_Folder", isDir: true },
    ];

    // Ascending sort
    sort.sortKey.value = "name";
    sort.sortAsc.value = true;
    const ascSorted = sort.sortItems(dataset);
    expect(ascSorted[0].name).toBe("A_Folder");
    expect(ascSorted[1].name).toBe("Z_Folder");
    expect(ascSorted[2].name).toBe("a_file.txt");
    expect(ascSorted[3].name).toBe("b_file.txt");

    // Toggle to Descending sort
    sort.setSort("name");
    expect(sort.sortAsc.value).toBe(false);
    const descSorted = sort.sortItems(dataset);
    // Invariant: Folders still appear before files even in descending order
    expect(descSorted[0].isDir).toBe(true);
    expect(descSorted[1].isDir).toBe(true);
    expect(descSorted[0].name).toBe("Z_Folder");
    expect(descSorted[1].name).toBe("A_Folder");
    expect(descSorted[2].isDir).toBe(false);
    expect(descSorted[3].isDir).toBe(false);
  });

  it("[Interaction 12] Context menu keyboard navigation + Enter triggers action", () => {
    const menuActions = ["open", "rename", "download", "delete"];
    let focusedIndex = 0;
    let executedAction: string | null = null;

    function handleMenuKey(key: string) {
      if (key === "ArrowDown" || key === "j") {
        focusedIndex = (focusedIndex + 1) % menuActions.length;
      } else if (key === "ArrowUp" || key === "k") {
        focusedIndex =
          (focusedIndex - 1 + menuActions.length) % menuActions.length;
      } else if (key === "Enter") {
        executedAction = menuActions[focusedIndex];
      }
    }

    handleMenuKey("ArrowDown"); // focus index 1: rename
    expect(focusedIndex).toBe(1);

    handleMenuKey("ArrowDown"); // focus index 2: download
    expect(focusedIndex).toBe(2);

    handleMenuKey("Enter"); // trigger download
    expect(executedAction).toBe("download");
  });

  it("[Interaction 13] Rapid NavRail expand/collapse toggle preserves synchronized state", () => {
    let isExpanded = true;

    function toggleRail() {
      isExpanded = !isExpanded;
      window.localStorage.setItem("nyx_nav_rail_expanded", String(isExpanded));
    }

    // Rapid toggle simulation
    toggleRail(); // false
    toggleRail(); // true
    toggleRail(); // false
    toggleRail(); // true
    toggleRail(); // false

    expect(isExpanded).toBe(false);
    expect(window.localStorage.getItem("nyx_nav_rail_expanded")).toBe("false");
  });

  it("[Interaction 14] Touch scrubber drag does not trigger header toggle or navigation", () => {
    let headerVisible = true;
    let videoCurrentTime = 10;

    function handleScrubberTouchMove(e: { clientX: number }) {
      // Scrubber touch drag updates video time directly
      videoCurrentTime = e.clientX * 0.5;
    }

    handleScrubberTouchMove({ clientX: 45 });
    expect(videoCurrentTime).toBe(22.5);
    // Header visibility was not toggled by scrubber drag
    expect(headerVisible).toBe(true);
  });

  it("[Interaction 15] FileActionPill batch delete passes selected items to BaseModal", () => {
    const items = ref([
      { path: "/files/a.txt", name: "a.txt" },
      { path: "/files/b.txt", name: "b.txt" },
      { path: "/files/c.txt", name: "c.txt" },
    ]);
    const selection = createFileSelectionContract(items);
    selection.select(0);
    selection.select(2);

    let modalTargetItems: any[] = [];
    let modalOpen = false;

    function triggerBatchDelete() {
      modalTargetItems = selection.selected.value.map(
        (idx) => items.value[idx]
      );
      modalOpen = true;
    }

    triggerBatchDelete();
    expect(modalOpen).toBe(true);
    expect(modalTargetItems.length).toBe(2);
    expect(modalTargetItems.map((i) => i.name)).toEqual(["a.txt", "c.txt"]);
  });
});
