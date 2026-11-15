import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";

// Load CSS and HTML files
const stylesCssPath = resolve(__dirname, "../styles.css");
const stylesCss = readFileSync(stylesCssPath, "utf8");

// Strip comments and normalize whitespace
const cleanStylesCss = stylesCss.replace(/\/\*[\s\S]*?\*\//g, "");
const normalizedStylesCss = cleanStylesCss.replace(/\s+/g, " ");

// Helper to extract a declaration block for a given selector pattern
function getBlock(css: string, selectorPattern: string): string | null {
  const match = css.match(new RegExp(selectorPattern + "\\s*\\{([^}]+)\\}"));
  return match ? match[1].trim() : null;
}

describe("P2-M0: Viewer Viewport & Safe-Area Constraints (#previewer)", () => {
  const previewerBlock = getBlock(normalizedStylesCss, "#previewer");

  it("declares #previewer container with dynamic viewport cascade (100vh followed by 100dvh)", () => {
    expect(previewerBlock).not.toBeNull();
    const block = previewerBlock!;

    expect(block).toContain("height: 100vh;");
    expect(block).toContain("height: 100dvh;");

    const vhIndex = block.indexOf("height: 100vh;");
    const dvhIndex = block.indexOf("height: 100dvh;");

    expect(vhIndex).toBeGreaterThan(-1);
    expect(dvhIndex).toBeGreaterThan(-1);
    // Strict cascade order: fallback 100vh must appear BEFORE dynamic 100dvh
    expect(vhIndex).toBeLessThan(dvhIndex);

    // Ensure fixed height: 100% is replaced
    expect(block).not.toMatch(/height:\s*100%;/);
  });

  it("declares hardware safe-area insets with fallback values and border-box sizing", () => {
    expect(previewerBlock).not.toBeNull();
    const block = previewerBlock!;

    expect(block).toMatch(
      /padding-top:\s*env\(safe-area-inset-top(,\s*0px)?\);/
    );
    expect(block).toMatch(
      /padding-bottom:\s*env\(safe-area-inset-bottom(,\s*0px)?\);/
    );
    expect(block).toMatch(
      /padding-left:\s*env\(safe-area-inset-left(,\s*0px)?\);/
    );
    expect(block).toMatch(
      /padding-right:\s*env\(safe-area-inset-right(,\s*0px)?\);/
    );

    expect(block).toContain("box-sizing: border-box;");
  });

  it("establishes fixed positioning, background color, full width, and hidden overflow", () => {
    expect(previewerBlock).not.toBeNull();
    const block = previewerBlock!;

    expect(block).toContain("position: fixed;");
    expect(block).toMatch(/inset:\s*0;|top:\s*0;\s*left:\s*0;/);
    expect(block).toContain("width: 100%;");
    expect(block).toContain("overflow: hidden;");
    expect(block).toContain("background-color: #000000;");
    expect(block).toContain("z-index: 99999;");
  });
});

describe("P2-M0: Flexbox Layout Flow & Clamping", () => {
  it("configures #previewer as a flex column container", () => {
    const previewerBlock = getBlock(normalizedStylesCss, "#previewer");
    expect(previewerBlock).not.toBeNull();
    expect(previewerBlock).toContain("display: flex;");
    expect(previewerBlock).toContain("flex-direction: column;");
  });

  it("positions #previewer header in normal flex flow with auto dimensions and smooth visibility transitions", () => {
    const headerBlock = getBlock(normalizedStylesCss, "#previewer\\s+header");
    expect(headerBlock).not.toBeNull();
    const block = headerBlock!;

    // Must be in flex flow (relative, not fixed overlay)
    expect(block).toContain("position: relative;");
    expect(block).toContain("flex: 0 0 auto;");
    expect(block).toContain("height: 3.5em;");
    expect(block).toContain("width: 100%;");
    expect(block).toContain("display: flex;");
    expect(block).toContain("align-items: center;");
    expect(block).toContain("box-sizing: border-box;");
    expect(block).toContain(
      "transition: opacity 0.2s ease, visibility 0.2s ease;"
    );
  });

  it("defines header--hidden state to smoothly fade out controls without layout jumping", () => {
    const hiddenBlock = getBlock(
      normalizedStylesCss,
      "#previewer\\s+header\\.header--hidden"
    );
    expect(hiddenBlock).not.toBeNull();
    const block = hiddenBlock!;
    expect(block).toContain("opacity: 0;");
    expect(block).toContain("visibility: hidden;");
    expect(block).toContain("pointer-events: none;");
  });

  it("configures .preview media container with flex clamping (flex: 1 1 auto; min-height: 0)", () => {
    const previewBlock = getBlock(
      normalizedStylesCss,
      "#previewer\\s+\\.preview"
    );
    expect(previewBlock).not.toBeNull();
    const block = previewBlock!;

    // Flex clamping prevents media from expanding outside viewport bounds
    expect(block).toContain("flex: 1 1 auto;");
    expect(block).toContain("min-height: 0;");
    expect(block).toContain("height: 100%;");
    expect(block).toContain("display: flex;");
    expect(block).toContain("align-items: center;");
    expect(block).toContain("justify-content: center;");
    expect(block).toContain("overflow: hidden;");
  });

  it("constrains media elements (pre, video, img) within preview boundaries without aspect ratio distortion", () => {
    // Check media containment rules
    expect(normalizedStylesCss).toMatch(/#previewer\s+\.preview\s+video/);
    expect(normalizedStylesCss).toMatch(/#previewer\s+\.preview\s+img/);
    expect(normalizedStylesCss).toMatch(/max-height:\s*100%;/);
    expect(normalizedStylesCss).toMatch(/object-fit:\s*contain;/);
  });

  it("anchors Video.js control bar at bottom within container", () => {
    const vjsControlBlock = getBlock(
      normalizedStylesCss,
      "#previewer\\s+\\.preview\\s+\\.video-js\\s+\\.vjs-control-bar"
    );
    expect(vjsControlBlock).not.toBeNull();
    expect(vjsControlBlock).toContain("bottom: 0;");
  });

  it("removes legacy padding-top: 4em from #previewer .pdf", () => {
    expect(normalizedStylesCss).not.toMatch(
      /#previewer\s+\.pdf\s*\{[^}]*padding-top:\s*4em;/
    );
    expect(normalizedStylesCss).toContain(
      "#previewer .pdf { width: 100%; height: 100%; }"
    );
  });
});

describe("P2-M0: Header Title & Mobile Dropdown Action Polish", () => {
  it("truncates header title with min-width: 0 and ellipsis to prevent pushing actions off-screen", () => {
    const titleBlock = getBlock(
      normalizedStylesCss,
      "#previewer\\s+header\\s*>\\s*title"
    );
    expect(titleBlock).not.toBeNull();
    const block = titleBlock!;

    // min-width: 0 is mandatory for flex items to shrink below content size
    expect(block).toContain("min-width: 0;");
    expect(block).toContain("text-overflow: ellipsis;");
    expect(block).toContain("overflow: hidden;");
    expect(block).toContain("white-space: nowrap;");
    expect(block).toContain("flex: 1 1 auto;");
  });

  it("hides header action text by default but restores display: inline-block in mobile dropdown", () => {
    const defaultSpanBlock = getBlock(
      normalizedStylesCss,
      "#previewer\\s+header\\s+\\.action\\s+span"
    );
    const dropdownSpanBlock = getBlock(
      normalizedStylesCss,
      "#previewer\\s+header\\s+#dropdown\\s+\\.action\\s+span"
    );

    expect(defaultSpanBlock).not.toBeNull();
    expect(defaultSpanBlock).toContain("display: none;");

    expect(dropdownSpanBlock).not.toBeNull();
    expect(dropdownSpanBlock).toContain("display: inline-block;");

    // Ensure dropdown override appears after default span hiding rule in cascade
    const defaultSpanIndex = normalizedStylesCss.indexOf(
      "#previewer header .action span"
    );
    const dropdownSpanIndex = normalizedStylesCss.indexOf(
      "#previewer header #dropdown .action span"
    );
    expect(defaultSpanIndex).toBeGreaterThan(-1);
    expect(dropdownSpanIndex).toBeGreaterThan(-1);
    expect(defaultSpanIndex).toBeLessThan(dropdownSpanIndex);
  });
});

describe("P2-M0: Viewport Meta Tag in HTML Entry Files", () => {
  const indexHtmlPath = resolve(__dirname, "../../../index.html");
  const publicIndexHtmlPath = resolve(__dirname, "../../../public/index.html");

  it("configures viewport-fit=cover in frontend/index.html (Vite entry)", () => {
    expect(existsSync(indexHtmlPath)).toBe(true);
    const html = readFileSync(indexHtmlPath, "utf8");
    const viewportMetaMatch = html.match(
      /<meta[^>]*name=["']viewport["'][^>]*content=["']([^"']+)["']/i
    );
    expect(viewportMetaMatch).not.toBeNull();
    const content = viewportMetaMatch![1];
    expect(content).toContain("width=device-width");
    expect(content).toContain("viewport-fit=cover");
  });

  it("configures viewport-fit=cover in frontend/public/index.html (Production/Go template entry)", () => {
    expect(existsSync(publicIndexHtmlPath)).toBe(true);
    const html = readFileSync(publicIndexHtmlPath, "utf8");
    const viewportMetaMatch = html.match(
      /<meta[^>]*name=["']viewport["'][^>]*content=["']([^"']+)["']/i
    );
    expect(viewportMetaMatch).not.toBeNull();
    const content = viewportMetaMatch![1];
    expect(content).toContain("width=device-width");
    expect(content).toContain("viewport-fit=cover");
  });
});

describe("P2-M0: Touch & Scrubber Event Propagation Integrity", () => {
  const previewVuePath = resolve(__dirname, "../../views/files/Preview.vue");

  it("does not stop touchmove event propagation on root #previewer to unblock video scrubbing", () => {
    expect(existsSync(previewVuePath)).toBe(true);
    const vueContent = readFileSync(previewVuePath, "utf8");

    // Preview.vue root container should not use @touchmove.prevent.stop
    // If .stop is used on touchmove, document-level touch listeners attached by video.js are blocked
    expect(vueContent).not.toMatch(/@touchmove\.prevent\.stop/);
  });
});
