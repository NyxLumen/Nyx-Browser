<template>
  <button
    v-if="canUpload && isFilesRoute"
    class="mobile-upload-fab"
    @click="triggerUpload"
    :title="$t('buttons.upload')"
    aria-label="Upload File"
  >
    <NyxIcon name="plus" size="22" stroke-width="2" class="mobile-upload-fab__icon" />
  </button>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import NyxIcon from "@/components/ui/NyxIcon.vue";

const authStore = useAuthStore();
const route = useRoute();

const canUpload = computed(() => authStore.user?.perm.create ?? true);
const isFilesRoute = computed(() => route.path.startsWith("/files"));

const triggerUpload = () => {
  const input = document.getElementById("upload-input") as HTMLInputElement | null;
  if (input) {
    input.click();
  }
};
</script>

<style scoped>
.mobile-upload-fab {
  display: none;
}

@media (max-width: 1023px) {
  .mobile-upload-fab {
    display: flex;
    align-items: center;
    justify-content: center;
    position: fixed;
    right: 20px;
    bottom: calc(24px + env(safe-area-inset-bottom, 0px));
    width: 52px;
    height: 52px;
    border-radius: var(--nyx-radius-sm);
    background-color: var(--nyx-surface-2);
    border: 1px solid var(--nyx-accent-red);
    color: var(--nyx-accent-red);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.7);
    z-index: 900;
    cursor: pointer;
    transition: transform 0.15s ease, background-color 0.15s ease;
  }

  .mobile-upload-fab:active {
    transform: scale(0.94);
    background-color: var(--nyx-surface-3);
  }

  .mobile-upload-fab__icon {
    flex-shrink: 0;
  }
}
</style>
