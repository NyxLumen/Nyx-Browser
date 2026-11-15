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
}

@media (max-width: 600px) {
  .file-list-grid__grid {
    grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
    gap: 8px;
  }
}
</style>
