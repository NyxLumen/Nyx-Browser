<template>
  <div class="file-list-grid" data-clear-on-click="true">
    <!-- Folders Section -->
    <div v-if="dirs.length > 0" class="file-list-grid__group" data-clear-on-click="true">
      <div class="file-list-grid__section-header" data-clear-on-click="true">
        <span class="file-list-grid__section-tag">// FOLDERS ({{ dirs.length }})</span>
      </div>
      <div
        class="file-list-grid__grid"
        data-clear-on-click="true"
        @contextmenu="$emit('contextmenu', $event)"
      >
        <ListingItem
          v-for="item in dirs"
          :key="base64(item.name)"
          :index="item.index"
          :name="item.name"
          :isDir="item.isDir"
          :url="item.url"
          :modified="item.modified"
          :type="item.type"
          :size="item.size"
          :path="item.path"
          :readOnly="readOnly"
        />
      </div>
    </div>

    <!-- Files Section -->
    <div v-if="files.length > 0" class="file-list-grid__group" data-clear-on-click="true">
      <div class="file-list-grid__section-header" data-clear-on-click="true">
        <span class="file-list-grid__section-tag">// FILES ({{ files.length }})</span>
      </div>
      <div
        class="file-list-grid__grid"
        data-clear-on-click="true"
        @contextmenu="$emit('contextmenu', $event)"
      >
        <ListingItem
          v-for="item in files"
          :key="base64(item.name)"
          :index="item.index"
          :name="item.name"
          :isDir="item.isDir"
          :url="item.url"
          :modified="item.modified"
          :type="item.type"
          :size="item.size"
          :path="item.path"
          :readOnly="readOnly"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import ListingItem from "@/components/files/ListingItem.vue";
import { Base64 } from "js-base64";

defineProps<{
  dirs: Resource[];
  files: Resource[];
  readOnly?: boolean;
}>();

defineEmits<{
  (e: "contextmenu", event: MouseEvent): void;
}>();

const base64 = (name: string) => Base64.encodeURI(name);
</script>

<style scoped>
.file-list-grid {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.file-list-grid__group {
  display: flex;
  flex-direction: column;
}

.file-list-grid__section-header {
  padding: 8px 12px 6px 12px;
}

.file-list-grid__section-tag {
  font-family: var(--nyx-font-mono);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--nyx-text-muted);
}

.file-list-grid__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
  padding: 4px;
  width: 100%;
  box-sizing: border-box;
}

/* Explicit geometric card styling for ListingItem */
.file-list-grid__grid :deep(.item) {
  width: 100% !important;
  max-width: 100% !important;
  min-height: 130px !important;
  margin: 0 !important;
  padding: 14px 12px !important;
  display: flex !important;
  flex-direction: column !important;
  align-items: center !important;
  justify-content: center !important;
  background-color: var(--nyx-surface-1) !important;
  border: 1px solid var(--nyx-border-hairline) !important;
  border-radius: var(--nyx-radius-sm) !important;
  box-shadow: none !important;
  cursor: pointer !important;
  box-sizing: border-box !important;
  transition: all 0.15s ease !important;
}

.file-list-grid__grid :deep(.item:hover) {
  background-color: var(--nyx-surface-2) !important;
  border-color: var(--nyx-border-strong) !important;
}

.file-list-grid__grid :deep(.item[aria-selected="true"]) {
  background-color: var(--nyx-surface-selected) !important;
  border-color: var(--nyx-accent-red) !important;
  box-shadow: 0 0 12px var(--nyx-accent-red-glow) !important;
}

/* Icon / Thumbnail Box */
.file-list-grid__grid :deep(.item > div:first-of-type) {
  width: 100% !important;
  height: 64px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  overflow: hidden !important;
  margin-bottom: 8px !important;
  flex-shrink: 0 !important;
}

.file-list-grid__grid :deep(.item > div:first-of-type i) {
  font-size: 40px !important;
  line-height: 1 !important;
  margin: 0 !important;
  color: var(--nyx-text-secondary) !important;
}

.file-list-grid__grid :deep(.item > div:first-of-type img) {
  max-width: 100% !important;
  max-height: 100% !important;
  width: auto !important;
  height: auto !important;
  object-fit: contain !important;
  border-radius: var(--nyx-radius-xs) !important;
  margin: 0 !important;
}

/* Text Container */
.file-list-grid__grid :deep(.item > div:last-of-type) {
  width: 100% !important;
  display: flex !important;
  flex-direction: column !important;
  align-items: center !important;
  text-align: center !important;
  overflow: hidden !important;
  min-width: 0 !important;
}

/* Name */
.file-list-grid__grid :deep(.item .name) {
  width: 100% !important;
  font-size: 13px !important;
  font-weight: 500 !important;
  color: var(--nyx-text-primary) !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  white-space: nowrap !important;
  line-height: 1.4 !important;
  text-align: center !important;
  display: block !important;
}

/* Size */
.file-list-grid__grid :deep(.item .size) {
  width: 100% !important;
  font-family: var(--nyx-font-mono) !important;
  font-size: 11px !important;
  color: var(--nyx-text-muted) !important;
  margin-top: 4px !important;
  text-align: center !important;
  display: block !important;
}

/* Modified date hidden in grid cards for clean minimal look */
.file-list-grid__grid :deep(.item .modified) {
  display: none !important;
}

@media (max-width: 600px) {
  .file-list-grid__grid {
    grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
    gap: 8px;
  }
}
</style>
