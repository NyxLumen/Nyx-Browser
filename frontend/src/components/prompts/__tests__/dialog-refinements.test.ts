import { describe, expect, it } from "vitest";
import BaseModal from "../BaseModal.vue";
import ContextMenu from "@/components/ContextMenu.vue";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

describe("Milestone P2-M4: Dialogs, Context Menu & Refinements", () => {
  it("exports valid BaseModal and ContextMenu components", () => {
    expect(BaseModal).toBeDefined();
    expect(typeof BaseModal).toBe("object");

    expect(ContextMenu).toBeDefined();
    expect(typeof ContextMenu).toBe("object");
  });

  it("applies Nothing dark aesthetic and 1px borders to floating cards", () => {
    const dashboardCss = readFileSync(
      resolve(__dirname, "../../../css/dashboard.css"),
      "utf8"
    );
    expect(dashboardCss).toContain(".card.floating");
    expect(dashboardCss).toContain("background: var(--nyx-surface-1);");
    expect(dashboardCss).toContain("border: 1px solid var(--nyx-border-strong);");
    expect(dashboardCss).toContain("font-family: var(--nyx-font-mono);");
  });

  it("applies frosted glass tokens to context menu", () => {
    const contextMenuCss = readFileSync(
      resolve(__dirname, "../../../css/context-menu.css"),
      "utf8"
    );
    expect(contextMenuCss).toContain("background: var(--nyx-glass-bg);");
    expect(contextMenuCss).toContain("backdrop-filter: var(--nyx-glass-blur);");
    expect(contextMenuCss).toContain("border: var(--nyx-glass-border);");
    expect(contextMenuCss).toContain("box-shadow: var(--nyx-glass-shadow);");
  });

  it("styles buttons with Nothing technical red and crisp geometry", () => {
    const buttonsCss = readFileSync(
      resolve(__dirname, "../../../css/_buttons.css"),
      "utf8"
    );
    expect(buttonsCss).toContain("background: var(--nyx-accent-red);");
    expect(buttonsCss).toContain(":focus-visible");
    expect(buttonsCss).toContain(".button--danger");
  });
});
