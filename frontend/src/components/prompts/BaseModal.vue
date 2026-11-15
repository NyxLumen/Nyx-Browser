<template>
  <div id="modal-background" @click="backgroundClick">
    <div ref="modalContainer" class="modal-container">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";

const emit = defineEmits(["closed"]);

const modalContainer = ref<HTMLElement | null>(null);

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === "Escape") {
    event.stopImmediatePropagation();
    emit("closed");
  }
};

onMounted(() => {
  const element = document.querySelector("#focus-prompt") as HTMLElement | null;
  if (element) {
    element.focus();
  } else if (modalContainer.value) {
    modalContainer.value.focus();
  }

  window.addEventListener("keydown", handleKeyDown);
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleKeyDown);
});

const backgroundClick = (event: Event) => {
  const target = event.target as HTMLElement;
  if (target.id === "modal-background") {
    emit("closed");
  }
};
</script>

<style scoped>
#modal-background {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 10000;
  animation: ease-out 150ms opacity-enter;
  padding: env(safe-area-inset-top, 16px) env(safe-area-inset-right, 16px) env(safe-area-inset-bottom, 16px) env(safe-area-inset-left, 16px);
}

.modal-container {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  max-width: 100%;
}

@keyframes opacity-enter {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
</style>
