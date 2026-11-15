import { computed, ref } from "vue";
import { useFileStore } from "@/stores/file";
import { useAuthStore } from "@/stores/auth";
import { users } from "@/api";

export type SortBy = "name" | "size" | "modified";

export function useFileSort() {
  const fileStore = useFileStore();
  const authStore = useAuthStore();

  const currentSortBy = computed<SortBy>(() => {
    return (fileStore.req?.sorting.by as SortBy) || "name";
  });

  const isAsc = computed<boolean>(() => {
    return fileStore.req?.sorting.asc ?? true;
  });

  const nameSorted = computed(() => currentSortBy.value === "name");
  const sizeSorted = computed(() => currentSortBy.value === "size");
  const modifiedSorted = computed(() => currentSortBy.value === "modified");

  const nameIcon = computed(() => {
    if (nameSorted.value && !isAsc.value) {
      return "arrow_upward";
    }
    return "arrow_downward";
  });

  const sizeIcon = computed(() => {
    if (sizeSorted.value && isAsc.value) {
      return "arrow_downward";
    }
    return "arrow_upward";
  });

  const modifiedIcon = computed(() => {
    if (modifiedSorted.value && isAsc.value) {
      return "arrow_downward";
    }
    return "arrow_upward";
  });

  const naturalCompare = (a: string, b: string): number => {
    return a.localeCompare(b, undefined, {
      numeric: true,
      sensitivity: "base",
    });
  };

  const sortItems = <T extends { name: string; size?: number; modified?: string; isDir?: boolean }>(
    items: T[],
    by: SortBy = currentSortBy.value,
    asc: boolean = isAsc.value
  ): T[] => {
    const sorted = [...items];
    sorted.sort((a, b) => {
      // Folders always sort before files if mixed
      if (a.isDir !== b.isDir) {
        return a.isDir ? -1 : 1;
      }

      let cmp = 0;
      if (by === "name") {
        cmp = naturalCompare(a.name, b.name);
      } else if (by === "size") {
        const sizeA = a.size ?? 0;
        const sizeB = b.size ?? 0;
        cmp = sizeA - sizeB;
      } else if (by === "modified") {
        const dateA = a.modified ? new Date(a.modified).getTime() : 0;
        const dateB = b.modified ? new Date(b.modified).getTime() : 0;
        cmp = dateA - dateB;
      }

      return asc ? cmp : -cmp;
    });
    return sorted;
  };

  const toggleSort = async (by: SortBy) => {
    let nextAsc = true;
    if (by === currentSortBy.value) {
      nextAsc = !isAsc.value;
    } else {
      // Sensible defaults when switching columns
      nextAsc = by === "name";
    }

    try {
      if (authStore.user?.id) {
        await users.update(
          { id: authStore.user.id, sorting: { by, asc: nextAsc } },
          ["sorting"]
        );
      }
    } catch {}

    fileStore.reload = true;
  };

  return {
    currentSortBy,
    isAsc,
    nameSorted,
    sizeSorted,
    modifiedSorted,
    nameIcon,
    sizeIcon,
    modifiedIcon,
    sortItems,
    toggleSort,
    naturalCompare,
  };
}
