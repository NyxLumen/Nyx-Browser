<template>
  <Transition name="nyx-drawer">
    <div
      v-if="uploadStore.totalBytes > 0"
      class="upload-drawer"
      role="status"
      aria-label="Upload Progress"
    >
      <div class="upload-drawer__progress-bar">
        <div
          class="upload-drawer__progress-fill"
          :style="{ width: percent + '%' }"
        />
      </div>

      <div class="upload-drawer__content">
        <div class="upload-drawer__info">
          <span class="upload-drawer__spinner"></span>
          <div class="upload-drawer__text">
            <span class="upload-drawer__title">
              UPLOADING {{ uploadStore.pendingUploadCount }} {{ uploadStore.pendingUploadCount === 1 ? 'ITEM' : 'ITEMS' }}
            </span>
            <span class="upload-drawer__metrics">
              {{ percent }}% &bull; {{ humanSent }} / {{ humanTotal }}
            </span>
          </div>
        </div>

        <button
          class="upload-drawer__abort"
          @click="uploadStore.abort"
          :title="$t('buttons.cancel')"
          aria-label="Cancel uploads"
        >
          <NyxIcon name="close" size="16" />
          <span class="upload-drawer__abort-text">{{ $t("buttons.cancel") }}</span>
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useUploadStore } from "@/stores/upload";
import NyxIcon from "@/components/ui/NyxIcon.vue";
import prettyBytes from "pretty-bytes";

const uploadStore = useUploadStore();

const percent = computed(() => {
  if (!uploadStore.totalBytes) return 0;
  return Math.min(
    100,
    Math.round((uploadStore.sentBytes / uploadStore.totalBytes) * 100)
  );
});

const humanSent = computed(() =>
  prettyBytes(uploadStore.sentBytes, { binary: true })
);

const humanTotal = computed(() =>
  prettyBytes(uploadStore.totalBytes, { binary: true })
);
</script>

<style scoped>
.upload-drawer {
  position: fixed;
  bottom: 0;
  right: 24px;
  width: 360px;
  max-width: calc(100vw - 48px);
  background-color: var(--nyx-surface-1);
  border: 1px solid var(--nyx-border-hairline);
  border-bottom: none;
  border-radius: var(--nyx-radius-sm) var(--nyx-radius-sm) 0 0;
  box-shadow: 0 -8px 24px rgba(0, 0, 0, 0.7);
  z-index: 1000;
  overflow: hidden;
}

.upload-drawer__progress-bar {
  width: 100%;
  height: 2px;
  background-color: var(--nyx-surface-3);
  overflow: hidden;
}

.upload-drawer__progress-fill {
  height: 100%;
  background-color: var(--nyx-accent-red);
  transition: width 0.2s ease;
}

.upload-drawer__content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  gap: 12px;
}

.upload-drawer__info {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.upload-drawer__spinner {
  width: 12px;
  height: 12px;
  border: 2px solid rgba(215, 25, 33, 0.3);
  border-top-color: var(--nyx-accent-red);
  border-radius: 50%;
  animation: nyx-spin 0.8s linear infinite;
  flex-shrink: 0;
}

.upload-drawer__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.upload-drawer__title {
  font-family: var(--nyx-font-mono);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: var(--nyx-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.upload-drawer__metrics {
  font-family: var(--nyx-font-mono);
  font-size: 11px;
  color: var(--nyx-text-muted);
}

.upload-drawer__abort {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: transparent;
  border: 1px solid var(--nyx-border-hairline);
  border-radius: var(--nyx-radius-xs);
  color: var(--nyx-text-secondary);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.upload-drawer__abort:hover {
  background-color: var(--nyx-accent-red-subtle);
  border-color: rgba(215, 25, 33, 0.4);
  color: var(--nyx-accent-red);
}

.upload-drawer__abort-text {
  font-size: 11px;
}

@keyframes nyx-spin {
  to {
    transform: rotate(360deg);
  }
}

.nyx-drawer-enter-active,
.nyx-drawer-leave-active {
  transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

.nyx-drawer-enter-from,
.nyx-drawer-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
