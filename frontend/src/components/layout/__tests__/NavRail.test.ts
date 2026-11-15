import { describe, expect, it, beforeEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";

// Mock constants and api before importing layout/NavRail
vi.mock("@/utils/constants", () => ({
  disableUsedPercentage: false,
  noAuth: false,
  loginPage: false,
  logoutPage: "/login",
  logoURL: "/logo.svg",
  logoLockupURL: "/img/logo-lockup.svg",
  logoMarkURL: "/img/logo-mark.svg",
}));
vi.mock("@/utils/auth", () => ({
  logout: vi.fn(),
}));
vi.mock("@/api", () => ({
  files: {
    usage: vi.fn().mockResolvedValue({ used: 1024, total: 2048 }),
  },
}));
vi.mock("vue-router", () => ({
  useRoute: () => ({ path: "/files" }),
  useRouter: () => ({ push: vi.fn() }),
}));
vi.mock("vue-i18n", async (importOriginal) => {
  const actual = await importOriginal<typeof import("vue-i18n")>();
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  };
});

import { useLayoutStore } from "@/stores/layout";
import NavRail from "../NavRail.vue";
import MobileUploadFab from "../MobileUploadFab.vue";

describe("Milestone P2-M2: Application Shell & NavRail", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    // Provide a simple mock localStorage for node test environment
    const storage: Record<string, string> = {};
    globalThis.localStorage = {
      getItem: (key: string) => storage[key] ?? null,
      setItem: (key: string, val: string) => {
        storage[key] = val;
      },
      removeItem: (key: string) => {
        delete storage[key];
      },
      clear: () => {
        for (const k in storage) delete storage[k];
      },
      key: () => null,
      length: 0,
    };
    (globalThis as any).window = {};
  });

  it("exports valid Vue components for NavRail and MobileUploadFab", () => {
    expect(NavRail).toBeDefined();
    expect(typeof NavRail).toBe("object");
    expect(NavRail.__name).toBe("NavRail");

    expect(MobileUploadFab).toBeDefined();
    expect(typeof MobileUploadFab).toBe("object");
    expect(MobileUploadFab.__name).toBe("MobileUploadFab");
  });

  it("initializes rail state and allows toggling with localStorage persistence", () => {
    const layout = useLayoutStore();

    // Default is expanded (true)
    expect(layout.isRailExpanded).toBe(true);

    // Toggle to collapsed
    layout.toggleRail();
    expect(layout.isRailExpanded).toBe(false);
    expect(localStorage.getItem("nyx_nav_rail_expanded")).toBe("false");

    // Toggle back to expanded
    layout.toggleRail();
    expect(layout.isRailExpanded).toBe(true);
    expect(localStorage.getItem("nyx_nav_rail_expanded")).toBe("true");
  });

  it("handles mobile drawer navigation state properly", () => {
    const layout = useLayoutStore();

    expect(layout.isMobileNavOpen).toBe(false);

    layout.toggleMobileNav();
    expect(layout.isMobileNavOpen).toBe(true);

    layout.closeMobileNav();
    expect(layout.isMobileNavOpen).toBe(false);
  });
});
