<template>
  <div
    class="context-menu"
    ref="contextMenu"
    v-show="show"
    :style="{
      top: `${props.pos.y}px`,
      left: `${left}px`,
    }"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted } from "vue";

const emit = defineEmits(["hide"]);
const props = defineProps<{ show: boolean; pos: { x: number; y: number } }>();
const contextMenu = ref<HTMLElement | null>(null);

const left = computed(() => {
  return Math.min(
    props.pos.x,
    window.innerWidth - (contextMenu.value?.clientWidth ?? 0)
  );
});

const hideContextMenu = () => {
  emit("hide");
};

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === "Escape") {
    hideContextMenu();
  }
};

watch(
  () => props.show,
  (val) => {
    if (val) {
      document.addEventListener("click", hideContextMenu);
      document.addEventListener("keydown", handleKeydown);
    } else {
      document.removeEventListener("click", hideContextMenu);
      document.removeEventListener("keydown", handleKeydown);
    }
  }
);

onUnmounted(() => {
  document.removeEventListener("click", hideContextMenu);
  document.removeEventListener("keydown", handleKeydown);
});
</script>
