<template>
  <div class="file-list-table" data-clear-on-click="true">
    <!-- Table Column Header -->
    <div class="file-list-table__header-row">
      <div
        class="file-list-table__col file-list-table__col--name"
        :class="{ 'file-list-table__col--active': nameSorted }"
        role="button"
        tabindex="0"
        @click="$emit('sort', 'name')"
        :title="$t('files.sortByName')"
        :aria-label="$t('files.sortByName')"
      >
        <span class="file-list-table__header-label">{{ $t("files.name") }}</span>
        <NyxIcon
          :name="nameIcon === 'arrow_upward' ? 'arrow_up' : 'arrow_down'"
          size="14"
          class="file-list-table__sort-icon"
        />
      </div>

      <div
        class="file-list-table__col file-list-table__col--size"
        :class="{ 'file-list-table__col--active': sizeSorted }"
        role="button"
        tabindex="0"
        @click="$emit('sort', 'size')"
        :title="$t('files.sortBySize')"
        :aria-label="$t('files.sortBySize')"
      >
        <span class="file-list-table__header-label">{{ $t("files.size") }}</span>
        <NyxIcon
          :name="sizeIcon === 'arrow_upward' ? 'arrow_up' : 'arrow_down'"
          size="14"
          class="file-list-table__sort-icon"
        />
      </div>

      <div
        class="file-list-table__col file-list-table__col--modified"
        :class="{ 'file-list-table__col--active': modifiedSorted }"
        role="button"
        tabindex="0"
        @click="$emit('sort', 'modified')"
        :title="$t('files.sortByLastModified')"
        :aria-label="$t('files.sortByLastModified')"
      >
        <span class="file-list-table__header-label">{{ $t("files.lastModified") }}</span>
        <NyxIcon
          :name="modifiedIcon === 'arrow_upward' ? 'arrow_up' : 'arrow_down'"
          size="14"
          class="file-list-table__sort-icon"
        />
      </div>
    </div>

    <!-- Folders Group -->
    <div v-if="dirs.length > 0" class="file-list-table__group" data-clear-on-click="true">
      <div class="file-list-table__section-header" data-clear-on-click="true">
        <span class="file-list-table__section-tag">// FOLDERS ({{ dirs.length }})</span>
      </div>
      <div
        class="file-list-table__rows"
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

    <!-- Files Group -->
    <div v-if="files.length > 0" class="file-list-table__group" data-clear-on-click="true">
      <div class="file-list-table__section-header" data-clear-on-click="true">
        <span class="file-list-table__section-tag">// FILES ({{ files.length }})</span>
      </div>
      <div
        class="file-list-table__rows"
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
import NyxIcon from "@/components/ui/NyxIcon.vue";
import { Base64 } from "js-base64";

defineProps<{
  dirs: Resource[];
  files: Resource[];
  nameSorted: boolean;
  sizeSorted: boolean;
  modifiedSorted: boolean;
  nameIcon: string;
  sizeIcon: string;
  modifiedIcon: string;
  readOnly?: boolean;
}>();

defineEmits<{
  (e: "sort", by: "name" | "size" | "modified"): void;
  (e: "contextmenu", event: MouseEvent): void;
}>();

const base64 = (name: string) => Base64.encodeURI(name);
</script>

<style scoped>
.file-list-table {
  width: 100%;
  display: flex;
  flex-direction: column;
}

.file-list-table__header-row {
  display: flex;
  align-items: center;
  height: 38px;
  padding: 0 12px;
  background-color: var(--nyx-surface-1);
  border-bottom: 1px solid var(--nyx-border-hairline);
  user-select: none;
  font-family: var(--nyx-font-mono);
  font-size: 11px;
  letter-spacing: 0.05em;
  color: var(--nyx-text-muted);
}

.file-list-table__col {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: color 0.15s ease;
}

.file-list-table__col:hover {
  color: var(--nyx-text-primary);
}

.file-list-table__col--active {
  color: var(--nyx-accent-red);
}

.file-list-table__col--name {
  flex: 1 1 50%;
  min-width: 0;
  padding-left: 36px;
}

.file-list-table__col--size {
  flex: 0 0 120px;
  justify-content: flex-end;
  padding-right: 16px;
}

.file-list-table__col--modified {
  flex: 0 0 160px;
  justify-content: flex-end;
}

.file-list-table__sort-icon {
  opacity: 0.8;
}

.file-list-table__group {
  display: flex;
  flex-direction: column;
  margin-top: 12px;
}

.file-list-table__section-header {
  padding: 8px 12px 4px 12px;
}

.file-list-table__section-tag {
  font-family: var(--nyx-font-mono);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--nyx-text-muted);
}

.file-list-table__rows {
  display: flex;
  flex-direction: column;
  width: 100%;
}

/* Explicit high-density row styling for ListingItem */
.file-list-table__rows :deep(.item) {
  width: 100% !important;
  max-width: 100% !important;
  height: 44px !important;
  margin: 0 !important;
  padding: 0 12px !important;
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  border: none !important;
  border-bottom: 1px solid var(--nyx-border-hairline) !important;
  background-color: transparent !important;
  box-shadow: none !important;
  border-radius: 0 !important;
  box-sizing: border-box !important;
}

.file-list-table__rows :deep(.item:hover) {
  background-color: var(--nyx-surface-2) !important;
}

.file-list-table__rows :deep(.item[aria-selected="true"]) {
  background-color: var(--nyx-surface-selected) !important;
  border-left: 2px solid var(--nyx-accent-red) !important;
}

.file-list-table__rows :deep(.item > div:first-of-type) {
  width: 36px !important;
  height: 36px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: flex-start !important;
  flex-shrink: 0 !important;
}

.file-list-table__rows :deep(.item > div:first-of-type i) {
  font-size: 20px !important;
  line-height: 1 !important;
  margin: 0 !important;
  color: var(--nyx-text-secondary) !important;
}

.file-list-table__rows :deep(.item > div:first-of-type img) {
  width: 24px !important;
  height: 24px !important;
  border-radius: var(--nyx-radius-xs) !important;
  object-fit: cover !important;
  margin: 0 !important;
}

.file-list-table__rows :deep(.item > div:last-of-type) {
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  flex: 1 1 auto !important;
  width: calc(100% - 36px) !important;
  min-width: 0 !important;
  overflow: visible !important;
}

.file-list-table__rows :deep(.item .name) {
  flex: 1 1 50% !important;
  min-width: 0 !important;
  font-size: 13px !important;
  font-weight: 500 !important;
  color: var(--nyx-text-primary) !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  white-space: nowrap !important;
  padding-right: 16px !important;
  display: block !important;
}

.file-list-table__rows :deep(.item .size) {
  flex: 0 0 120px !important;
  text-align: right !important;
  font-family: var(--nyx-font-mono) !important;
  font-size: 12px !important;
  font-feature-settings: "tnum" !important;
  color: var(--nyx-text-secondary) !important;
  padding-right: 16px !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  white-space: nowrap !important;
  display: block !important;
}

.file-list-table__rows :deep(.item .modified) {
  flex: 0 0 160px !important;
  text-align: right !important;
  font-family: var(--nyx-font-mono) !important;
  font-size: 12px !important;
  font-feature-settings: "tnum" !important;
  color: var(--nyx-text-muted) !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  white-space: nowrap !important;
  display: block !important;
}

@media (max-width: 736px) {
  .file-list-table__col--size {
    display: none;
  }
  .file-list-table__col--name {
    flex: 1 1 65%;
  }
  .file-list-table__col--modified {
    flex: 0 0 110px;
  }

  .file-list-table__rows :deep(.item .size) {
    display: none !important;
  }
  .file-list-table__rows :deep(.item .name) {
    flex: 1 1 65% !important;
  }
  .file-list-table__rows :deep(.item .modified) {
    flex: 0 0 110px !important;
  }
}
</style>
