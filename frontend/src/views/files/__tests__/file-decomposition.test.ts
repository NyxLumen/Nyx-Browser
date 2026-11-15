import { beforeEach, describe, expect, it, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";

vi.mock("@/utils/constants", () => ({
  enableThumbs: false,
  enableExec: false,
  logoURL: "/logo.svg",
  baseURL: "",
}));
vi.mock("@/router", () => ({ default: {} }));
vi.mock("@/api", () => ({
  users: { update: vi.fn().mockResolvedValue({}) },
  files: { usage: vi.fn() },
}));
vi.mock("vue-router", () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ path: "/files" }),
}));
vi.mock("vue-i18n", async (importOriginal) => {
  const actual = await importOriginal<typeof import("vue-i18n")>();
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  };
});

import { useFileSelection } from "../composables/useFileSelection";
import { useFileSort } from "../composables/useFileSort";
import FileListTable from "../views/FileListTable.vue";
import FileListGrid from "../views/FileListGrid.vue";
import FileActionPill from "../components/FileActionPill.vue";
import UploadProgressDrawer from "../components/UploadProgressDrawer.vue";

describe("Milestone P2-M3: File Manager Decomposition", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  describe("Component definitions", () => {
    it("exports valid Vue SFC components for views and widgets", () => {
      expect(FileListTable).toBeDefined();
      expect(FileListGrid).toBeDefined();
      expect(FileActionPill).toBeDefined();
      expect(UploadProgressDrawer).toBeDefined();
    });
  });

  describe("useFileSelection composable", () => {
    it("handles single and multi-selection correctly", () => {
      const selection = useFileSelection();

      expect(selection.selectedCount.value).toBe(0);

      // Single select
      selection.select(3);
      expect(selection.selected.value).toEqual([3]);
      expect(selection.isSelected(3)).toBe(true);
      expect(selection.isSelected(2)).toBe(false);

      // Multi select toggle
      selection.select(5, true);
      expect(selection.selected.value).toEqual([3, 5]);

      selection.select(3, true);
      expect(selection.selected.value).toEqual([5]);
    });

    it("handles selectAll, clearSelection, and invertSelection", () => {
      const selection = useFileSelection();
      const allIndices = [0, 1, 2, 3, 4];

      selection.selectAll(allIndices);
      expect(selection.selected.value).toEqual([0, 1, 2, 3, 4]);
      expect(selection.selectedCount.value).toBe(5);

      selection.clearSelection();
      expect(selection.selected.value).toEqual([]);
      expect(selection.selectedCount.value).toBe(0);

      // Invert
      selection.select(1);
      selection.invertSelection(allIndices);
      expect(selection.selected.value).toEqual([0, 2, 3, 4]);
    });

    it("handles shift-click range selection", () => {
      const selection = useFileSelection();
      const allIndices = [0, 1, 2, 3, 4, 5];

      selection.select(1);
      selection.select(4, false, true, allIndices);
      expect(selection.selected.value).toEqual([1, 2, 3, 4]);
    });
  });

  describe("useFileSort composable", () => {
    it("sorts items by natural alphanumeric order, with folders first", () => {
      const { sortItems } = useFileSort();

      const items = [
        { name: "file10.txt", isDir: false },
        { name: "folderB", isDir: true },
        { name: "file2.txt", isDir: false },
        { name: "folderA", isDir: true },
        { name: "file1.txt", isDir: false },
      ];

      const sorted = sortItems(items, "name", true);

      // Folders first in natural order
      expect(sorted[0].name).toBe("folderA");
      expect(sorted[1].name).toBe("folderB");
      // Files in natural order (file1, file2, file10)
      expect(sorted[2].name).toBe("file1.txt");
      expect(sorted[3].name).toBe("file2.txt");
      expect(sorted[4].name).toBe("file10.txt");
    });

    it("sorts by size and modified date", () => {
      const { sortItems } = useFileSort();

      const items = [
        { name: "b.txt", size: 500, modified: "2026-01-02", isDir: false },
        { name: "a.txt", size: 100, modified: "2026-01-01", isDir: false },
        { name: "c.txt", size: 1000, modified: "2026-01-03", isDir: false },
      ];

      const sortedBySize = sortItems(items, "size", true);
      expect(sortedBySize.map((i) => i.name)).toEqual(["a.txt", "b.txt", "c.txt"]);

      const sortedByDateDesc = sortItems(items, "modified", false);
      expect(sortedByDateDesc.map((i) => i.name)).toEqual(["c.txt", "b.txt", "a.txt"]);
    });
  });
});
