import { computed, ref } from "vue";
import { useFileStore } from "@/stores/file";

export function useFileSelection() {
  const fileStore = useFileStore();
  const lastSelectedIndex = ref<number | null>(null);

  const selected = computed({
    get: () => fileStore.selected,
    set: (val: number[]) => {
      fileStore.selected = val;
    },
  });

  const selectedCount = computed(() => fileStore.selectedCount);
  const isMultiple = computed(() => fileStore.multiple);

  const isSelected = (index: number) => fileStore.selected.includes(index);

  const select = (
    index: number,
    isMulti = false,
    isRange = false,
    allIndices: number[] = []
  ) => {
    if (isRange && lastSelectedIndex.value !== null && allIndices.length > 0) {
      const start = Math.min(lastSelectedIndex.value, index);
      const end = Math.max(lastSelectedIndex.value, index);
      const rangeIndices = allIndices.filter((i) => i >= start && i <= end);
      const newSelected = new Set([...fileStore.selected, ...rangeIndices]);
      fileStore.selected = Array.from(newSelected);
      lastSelectedIndex.value = index;
      return;
    }

    if (isMulti || fileStore.multiple) {
      const idx = fileStore.selected.indexOf(index);
      if (idx !== -1) {
        fileStore.selected.splice(idx, 1);
        if (fileStore.selected.length === 0) {
          lastSelectedIndex.value = null;
        }
      } else {
        fileStore.selected.push(index);
        lastSelectedIndex.value = index;
      }
      return;
    }

    fileStore.selected = [index];
    lastSelectedIndex.value = index;
  };

  const selectAll = (indices: number[]) => {
    fileStore.selected = [...indices];
    lastSelectedIndex.value =
      indices.length > 0 ? indices[indices.length - 1] : null;
  };

  const clearSelection = () => {
    fileStore.selected = [];
    lastSelectedIndex.value = null;
  };

  const invertSelection = (allIndices: number[]) => {
    const current = new Set(fileStore.selected);
    fileStore.selected = allIndices.filter((i) => !current.has(i));
    lastSelectedIndex.value = null;
  };

  const toggleMultiple = () => {
    fileStore.toggleMultiple();
  };

  return {
    selected,
    selectedCount,
    isMultiple,
    isSelected,
    select,
    selectAll,
    clearSelection,
    invertSelection,
    toggleMultiple,
    lastSelectedIndex,
  };
}
