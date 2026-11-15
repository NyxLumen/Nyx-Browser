<template>
  <button @click="action" :aria-label="label" :title="label" class="action">
    <NyxIcon v-if="icon" :name="icon" />
    <span>{{ label }}</span>
    <span v-if="counter && counter > 0" class="counter">{{ counter }}</span>
  </button>
</template>

<script setup lang="ts">
import { useLayoutStore } from "@/stores/layout";
import NyxIcon from "@/components/ui/NyxIcon.vue";

const props = defineProps<{
  icon?: string;
  label?: string;
  counter?: number;
  show?: string;
}>();

const emit = defineEmits<{
  (e: "action"): any;
}>();

const layoutStore = useLayoutStore();

const action = () => {
  if (props.show) {
    layoutStore.showHover(props.show);
  }

  emit("action");
};
</script>
