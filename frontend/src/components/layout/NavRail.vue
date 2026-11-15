<template>
  <div>
    <!-- Mobile Backdrop Overlay -->
    <div
      v-if="isMobile && isMobileOpen"
      class="nav-rail__backdrop"
      @click="closeNav"
    />

    <!-- Navigation Rail Element -->
    <aside
      :class="[
        'nav-rail',
        {
          'nav-rail--collapsed': !isMobile && !layoutStore.isRailExpanded,
          'nav-rail--expanded': !isMobile && layoutStore.isRailExpanded,
          'nav-rail--mobile-open': isMobile && isMobileOpen,
        },
      ]"
      aria-label="Application Navigation"
    >
      <!-- Rail Header / Brand -->
      <div class="nav-rail__header">
        <div class="nav-rail__brand" @click="toFiles">
          <img
            v-if="!layoutStore.isRailExpanded && !isMobile"
            :src="logoMarkURL"
            class="nav-rail__logo-mark"
            alt="nyxbrowser"
          />
          <img
            v-else
            :src="logoLockupURL"
            class="nav-rail__logo-lockup"
            alt="nyxbrowser"
          />
        </div>
      </div>

      <!-- Navigation Content -->
      <div class="nav-rail__nav">
        <!-- Files Link -->
        <button
          :class="['nav-rail__item', { 'nav-rail__item--active': isFilesActive }]"
          @click="toFiles"
          :title="$t('sidebar.myFiles')"
          aria-label="Files"
        >
          <NyxIcon name="folder" size="20" class="nav-rail__icon" />
          <span v-if="layoutStore.isRailExpanded || isMobile" class="nav-rail__label">
            {{ $t("sidebar.myFiles") }}
          </span>
        </button>

        <!-- New Folder & New File (if permitted) -->
        <div v-if="authStore.user?.perm.create" class="nav-rail__section">
          <button
            class="nav-rail__item"
            @click="openNewDir"
            :title="$t('sidebar.newFolder')"
            aria-label="New Folder"
          >
            <NyxIcon name="add" size="20" class="nav-rail__icon" />
            <span v-if="layoutStore.isRailExpanded || isMobile" class="nav-rail__label">
              {{ $t("sidebar.newFolder") }}
            </span>
          </button>
          <button
            class="nav-rail__item"
            @click="openNewFile"
            :title="$t('sidebar.newFile')"
            aria-label="New File"
          >
            <NyxIcon name="insert_drive_file" size="20" class="nav-rail__icon" />
            <span v-if="layoutStore.isRailExpanded || isMobile" class="nav-rail__label">
              {{ $t("sidebar.newFile") }}
            </span>
          </button>
        </div>

        <!-- Settings Links -->
        <div v-if="authStore.isLoggedIn" class="nav-rail__section">
          <button
            :class="['nav-rail__item', { 'nav-rail__item--active': isSettingsActive }]"
            @click="toSettings"
            :title="$t('sidebar.settings')"
            aria-label="Settings"
          >
            <NyxIcon name="settings" size="20" class="nav-rail__icon" />
            <span v-if="layoutStore.isRailExpanded || isMobile" class="nav-rail__label">
              {{ $t("sidebar.settings") }}
            </span>
          </button>
        </div>
      </div>

      <!-- Storage Quota Gauge -->
      <div
        v-if="fileStore.isFiles && !disableUsedPercentage && (layoutStore.isRailExpanded || isMobile)"
        class="nav-rail__storage"
      >
        <div class="nav-rail__storage-header">
          <span class="nav-rail__storage-title">STORAGE</span>
          <span class="nav-rail__storage-numbers">{{ usage.used }} / {{ usage.total }}</span>
        </div>
        <div class="nav-rail__storage-bar">
          <div
            class="nav-rail__storage-fill"
            :style="{ width: usage.usedPercentage + '%' }"
          />
        </div>
      </div>

      <!-- Rail Footer / User Profile & Collapse Toggle -->
      <div class="nav-rail__footer">
        <!-- User profile snippet (if logged in) -->
        <div v-if="authStore.isLoggedIn" class="nav-rail__user" @click="toProfile">
          <NyxIcon name="person" size="18" class="nav-rail__user-icon" />
          <span v-if="layoutStore.isRailExpanded || isMobile" class="nav-rail__username">
            {{ authStore.user?.username }}
          </span>
          <button
            v-if="canLogout"
            class="nav-rail__logout"
            @click.stop="handleLogout"
            :title="$t('sidebar.logout')"
            aria-label="Logout"
          >
            <NyxIcon name="exit_to_app" size="16" />
          </button>
        </div>

        <!-- Collapse/Expand Toggle Button (Desktop only) -->
        <button
          v-if="!isMobile"
          class="nav-rail__toggle"
          @click="layoutStore.toggleRail"
          :title="layoutStore.isRailExpanded ? 'Collapse Navigation' : 'Expand Navigation'"
          aria-label="Toggle Navigation Rail"
        >
          <NyxIcon
            :name="layoutStore.isRailExpanded ? 'chevron_left' : 'chevron_right'"
            size="18"
          />
          <span v-if="layoutStore.isRailExpanded" class="nav-rail__toggle-text">
            Collapse
          </span>
        </button>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";
import NyxIcon from "@/components/ui/NyxIcon.vue";
import { files as api } from "@/api";
import prettyBytes from "pretty-bytes";
import * as auth from "@/utils/auth";
import {
  disableUsedPercentage,
  noAuth,
  loginPage,
  logoutPage,
  logoLockupURL,
  logoMarkURL,
} from "@/utils/constants";

const authStore = useAuthStore();
const fileStore = useFileStore();
const layoutStore = useLayoutStore();
const route = useRoute();
const router = useRouter();

const isMobile = ref(false);
const usage = reactive({ used: "0 B", total: "0 B", usedPercentage: 0 });
let usageAbortController = new AbortController();

const isMobileOpen = computed(
  () => layoutStore.isMobileNavOpen || layoutStore.currentPromptName === "sidebar"
);

const closeNav = () => {
  layoutStore.closeMobileNav();
  if (layoutStore.currentPromptName === "sidebar") {
    layoutStore.closeHovers();
  }
};

const canLogout = computed(
  () => !noAuth && (loginPage || logoutPage !== "/login")
);

const isFilesActive = computed(() => route.path.startsWith("/files"));
const isSettingsActive = computed(() => route.path.startsWith("/settings"));

const checkMobile = () => {
  if (typeof window !== "undefined") {
    isMobile.value = window.innerWidth < 1024;
  }
};

onMounted(() => {
  checkMobile();
  window.addEventListener("resize", checkMobile);
  if (route.path.includes("/files")) {
    fetchUsage();
  }
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", checkMobile);
  usageAbortController.abort();
});

watch(
  () => route.path,
  (newPath) => {
    if (newPath.includes("/files")) {
      fetchUsage();
    }
    if (isMobile.value) {
      closeNav();
    }
  }
);

const fetchUsage = async () => {
  if (disableUsedPercentage) return;
  const path = route.path.endsWith("/") ? route.path : route.path + "/";
  try {
    usageAbortController.abort();
    usageAbortController = new AbortController();
    const res = await api.usage(path, usageAbortController.signal);
    usage.used = prettyBytes(res.used, { binary: true });
    usage.total = prettyBytes(res.total, { binary: true });
    usage.usedPercentage = Math.round((res.used / res.total) * 100);
  } catch {}
};

const toFiles = () => {
  router.push({ path: "/files" });
  closeNav();
};

const toSettings = () => {
  if (authStore.user?.perm.admin) {
    router.push({ path: "/settings/global" });
  } else {
    router.push({ path: "/settings/profile" });
  }
  closeNav();
};

const toProfile = () => {
  router.push({ path: "/settings/profile" });
  closeNav();
};

const openNewDir = () => {
  layoutStore.showHover("newDir");
  closeNav();
};

const openNewFile = () => {
  layoutStore.showHover("newFile");
  closeNav();
};

const handleLogout = () => {
  auth.logout();
};
</script>

<style scoped>
.nav-rail {
  position: fixed;
  top: 0;
  left: 0;
  height: 100vh;
  height: 100dvh;
  background-color: var(--nyx-surface-1);
  border-right: 1px solid var(--nyx-border-hairline);
  z-index: 1000;
  display: flex;
  flex-direction: column;
  transition: width 0.2s cubic-bezier(0.16, 1, 0.3, 1), transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  box-sizing: border-box;
  padding-top: env(safe-area-inset-top, 0px);
  padding-bottom: env(safe-area-inset-bottom, 0px);
  user-select: none;
}

.nav-rail--collapsed {
  width: 64px;
}

.nav-rail--expanded {
  width: 240px;
}

/* Header & Brand */
.nav-rail__header {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-bottom: 1px solid var(--nyx-border-hairline);
  flex-shrink: 0;
}

.nav-rail__brand {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  width: 100%;
}

.nav-rail--collapsed .nav-rail__brand {
  justify-content: center;
}

.nav-rail__logo-mark {
  height: 28px;
  width: auto;
  display: block;
  flex-shrink: 0;
}

.nav-rail__logo-lockup {
  height: 28px;
  width: auto;
  max-width: 180px;
  display: block;
  flex-shrink: 0;
}

/* Nav items */
.nav-rail__nav {
  flex: 1 1 auto;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 12px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-rail__section {
  padding-top: 8px;
  margin-top: 8px;
  border-top: 1px solid var(--nyx-border-hairline);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-rail__item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  height: 40px;
  padding: 0 12px;
  border-radius: var(--nyx-radius-sm);
  background: transparent;
  border: 1px solid transparent;
  color: var(--nyx-text-secondary);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease;
  white-space: nowrap;
}

.nav-rail__item:hover {
  background-color: var(--nyx-surface-3);
  color: var(--nyx-text-primary);
}

.nav-rail__item--active {
  background-color: var(--nyx-surface-selected);
  border-color: rgba(215, 25, 33, 0.4);
  color: var(--nyx-text-primary);
}

.nav-rail__item--active .nav-rail__icon {
  color: var(--nyx-accent-red);
}

.nav-rail__icon {
  flex-shrink: 0;
}

.nav-rail__label {
  font-size: 13px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Storage Gauge */
.nav-rail__storage {
  padding: 12px 16px;
  border-top: 1px solid var(--nyx-border-hairline);
  flex-shrink: 0;
}

.nav-rail__storage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  font-family: var(--nyx-font-mono);
  font-size: 11px;
}

.nav-rail__storage-title {
  color: var(--nyx-text-muted);
  letter-spacing: 0.05em;
}

.nav-rail__storage-numbers {
  color: var(--nyx-text-secondary);
}

.nav-rail__storage-bar {
  width: 100%;
  height: 3px;
  background-color: var(--nyx-surface-3);
  border-radius: var(--nyx-radius-xs);
  overflow: hidden;
}

.nav-rail__storage-fill {
  height: 100%;
  background-color: var(--nyx-accent-red);
  transition: width 0.3s ease;
}

/* Footer / Toggle */
.nav-rail__footer {
  padding: 8px;
  border-top: 1px solid var(--nyx-border-hairline);
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex-shrink: 0;
}

.nav-rail__user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--nyx-radius-sm);
  color: var(--nyx-text-secondary);
  cursor: pointer;
}

.nav-rail__user:hover {
  background-color: var(--nyx-surface-3);
  color: var(--nyx-text-primary);
}

.nav-rail__username {
  font-size: 12px;
  font-family: var(--nyx-font-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1 1 auto;
}

.nav-rail__logout {
  background: transparent;
  border: none;
  color: var(--nyx-text-muted);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--nyx-radius-xs);
}

.nav-rail__logout:hover {
  color: var(--nyx-accent-red);
}

.nav-rail__toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 36px;
  padding: 0 10px;
  border-radius: var(--nyx-radius-sm);
  background: transparent;
  border: none;
  color: var(--nyx-text-muted);
  cursor: pointer;
  font-size: 12px;
  transition: color 0.15s ease, background 0.15s ease;
}

.nav-rail__toggle:hover {
  background-color: var(--nyx-surface-3);
  color: var(--nyx-text-primary);
}

/* Mobile responsive drawer rules */
@media (max-width: 1023px) {
  .nav-rail {
    width: 260px;
    transform: translateX(-100%);
    box-shadow: none;
    z-index: 10001;
  }

  .nav-rail--mobile-open {
    transform: translateX(0);
    box-shadow: 0 0 24px rgba(0, 0, 0, 0.85);
  }

  .nav-rail__backdrop {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.7);
    backdrop-filter: blur(4px);
    z-index: 10000;
  }
}
</style>
