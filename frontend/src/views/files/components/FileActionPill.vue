<template>
  <Transition name="nyx-pill">
    <div
      v-if="selectedCount > 0"
      class="file-action-pill"
      role="toolbar"
      aria-label="Batch file actions"
    >
      <div class="file-action-pill__badge">
        <span class="file-action-pill__dot"></span>
        <span class="file-action-pill__count">{{ selectedCount }} SELECTED</span>
      </div>

      <div class="file-action-pill__divider"></div>

      <div class="file-action-pill__actions">
        <button
          v-if="canShare"
          class="file-action-pill__btn"
          @click="$emit('share')"
          :title="$t('buttons.share')"
          aria-label="Share"
        >
          <NyxIcon name="share" size="16" />
        </button>

        <button
          v-if="canRename"
          class="file-action-pill__btn"
          @click="$emit('rename')"
          :title="$t('buttons.rename')"
          aria-label="Rename"
        >
          <NyxIcon name="edit" size="16" />
        </button>

        <button
          v-if="canCopy"
          class="file-action-pill__btn"
          @click="$emit('copy')"
          :title="$t('buttons.copyFile')"
          aria-label="Copy"
        >
          <NyxIcon name="copy" size="16" />
        </button>

        <button
          v-if="canMove"
          class="file-action-pill__btn"
          @click="$emit('move')"
          :title="$t('buttons.moveFile')"
          aria-label="Move"
        >
          <NyxIcon name="forward" size="16" />
        </button>

        <button
          v-if="canDownload"
          class="file-action-pill__btn"
          @click="$emit('download')"
          :title="$t('buttons.download')"
          aria-label="Download"
        >
          <NyxIcon name="download" size="16" />
        </button>

        <button
          v-if="canDelete"
          class="file-action-pill__btn file-action-pill__btn--danger"
          @click="$emit('delete')"
          :title="$t('buttons.delete')"
          aria-label="Delete"
        >
          <NyxIcon name="trash" size="16" />
        </button>
      </div>

      <div class="file-action-pill__divider"></div>

      <button
        class="file-action-pill__close"
        @click="$emit('clear')"
        :title="$t('buttons.clear')"
        aria-label="Clear selection"
      >
        <NyxIcon name="close" size="16" />
      </button>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useAuthStore } from "@/stores/auth";
import NyxIcon from "@/components/ui/NyxIcon.vue";

const props = defineProps<{
  selectedCount: number;
}>();

defineEmits<{
  (e: "share"): void;
  (e: "rename"): void;
  (e: "copy"): void;
  (e: "move"): void;
  (e: "download"): void;
  (e: "delete"): void;
  (e: "clear"): void;
}>();

const authStore = useAuthStore();

const canShare = computed(
  () =>
    props.selectedCount === 1 &&
    authStore.user?.perm.share &&
    authStore.user?.perm.download
);

const canRename = computed(
  () => props.selectedCount === 1 && authStore.user?.perm.rename
);

const canCopy = computed(
  () => props.selectedCount > 0 && authStore.user?.perm.create
);

const canMove = computed(
  () => props.selectedCount > 0 && authStore.user?.perm.rename
);

const canDownload = computed(
  () => props.selectedCount > 0 && authStore.user?.perm.download
);

const canDelete = computed(
  () => props.selectedCount > 0 && authStore.user?.perm.delete
);
</script>

<style scoped>
.file-action-pill {
  position: fixed;
  bottom: calc(28px + env(safe-area-inset-bottom, 0px));
  left: 50%;
  transform: translateX(-50%);
  z-index: 950;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 12px;
  background-color: var(--nyx-glass-bg);
  backdrop-filter: var(--nyx-glass-blur);
  -webkit-backdrop-filter: var(--nyx-glass-blur);
  border: var(--nyx-glass-border);
  border-radius: 9999px;
  box-shadow: var(--nyx-glass-shadow);
  user-select: none;
}

.file-action-pill__badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-left: 4px;
}

.file-action-pill__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--nyx-accent-red);
  box-shadow: 0 0 6px var(--nyx-accent-red-glow);
}

.file-action-pill__count {
  font-family: var(--nyx-font-mono);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--nyx-text-primary);
  white-space: nowrap;
}

.file-action-pill__divider {
  width: 1px;
  height: 18px;
  background-color: var(--nyx-border-hairline);
}

.file-action-pill__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.file-action-pill__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--nyx-radius-sm);
  background: transparent;
  border: 1px solid transparent;
  color: var(--nyx-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.file-action-pill__btn:hover {
  background-color: var(--nyx-surface-3);
  color: var(--nyx-text-primary);
}

.file-action-pill__btn--danger:hover {
  background-color: var(--nyx-accent-red-subtle);
  color: var(--nyx-accent-red);
  border-color: rgba(215, 25, 33, 0.4);
}

.file-action-pill__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: transparent;
  border: none;
  color: var(--nyx-text-muted);
  cursor: pointer;
  transition: all 0.15s ease;
}

.file-action-pill__close:hover {
  background-color: var(--nyx-surface-3);
  color: var(--nyx-text-primary);
}

/* Animations */
.nyx-pill-enter-active,
.nyx-pill-leave-active {
  transition: opacity 0.2s cubic-bezier(0.16, 1, 0.3, 1), transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.nyx-pill-enter-from,
.nyx-pill-leave-to {
  opacity: 0;
  transform: translate(-50%, 16px);
}
</style>
