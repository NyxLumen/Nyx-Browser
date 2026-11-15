import { describe, expect, it } from "vitest";
import NyxIcon from "../NyxIcon.vue";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

describe("NyxIcon Component (Lucide Line Icon System)", () => {
  it("defines icon component with correct default props", () => {
    expect(NyxIcon).toBeDefined();
    expect(NyxIcon.props).toBeDefined();
    // Default prop definitions
    expect(NyxIcon.props.name).toBeDefined();
  });

  it("exports a valid Vue component", () => {
    expect(typeof NyxIcon).toBe("object");
    expect(NyxIcon.name || NyxIcon.__name).toBe("NyxIcon");
  });
});

describe("Design Tokens (tokens.css)", () => {
  const tokensCssPath = resolve(__dirname, "../../../css/tokens.css");
  const tokensCss = readFileSync(tokensCssPath, "utf8");

  it("defines Nothing-inspired monochromatic dark surfaces", () => {
    expect(tokensCss).toContain("--nyx-bg-canvas: #000000;");
    expect(tokensCss).toContain("--nyx-bg-chassis: #080808;");
    expect(tokensCss).toContain("--nyx-surface-1: #101010;");
    expect(tokensCss).toContain("--nyx-surface-2: #161616;");
    expect(tokensCss).toContain("--nyx-surface-3: #222222;");
    expect(tokensCss).toContain("--nyx-surface-selected: #1a1213;");
  });

  it("defines the surgical Nothing technical red accent (#d71921)", () => {
    expect(tokensCss).toContain("--nyx-accent-red: #d71921;");
    expect(tokensCss).toContain("--nyx-border-selected: #d71921;");
  });

  it("defines crisp 1px borders and typography", () => {
    expect(tokensCss).toContain("--nyx-border-hairline: rgba(255, 255, 255, 0.06);");
    expect(tokensCss).toContain("--nyx-border-subtle: rgba(255, 255, 255, 0.10);");
    expect(tokensCss).toContain("--nyx-font-sans:");
    expect(tokensCss).toContain("--nyx-font-mono:");
  });

  it("bridges legacy File Browser variables to Nothing tokens", () => {
    expect(tokensCss).toContain("--background: var(--nyx-bg-canvas);");
    expect(tokensCss).toContain("--surfacePrimary: var(--nyx-surface-1);");
    expect(tokensCss).toContain("--surfaceSecondary: var(--nyx-surface-2);");
    expect(tokensCss).toContain("--red: var(--nyx-accent-red);");
  });
});
