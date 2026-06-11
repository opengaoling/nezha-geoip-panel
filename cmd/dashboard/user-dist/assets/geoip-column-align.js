(function () {
  const root = document.documentElement;
  const containerSelector = ".server-inline-list > section > div > div:last-child";
  const minWidths = [70, 58, 78, 86, 86, 70, 70, 82, 82];
  const styleId = "geoip-column-align-runtime-css";
  let frame = 0;
  let measurer = null;

  function isMobile() {
    return root.classList.contains("geoip-mobile-ua");
  }

  function ensureStyle() {
    if (document.getElementById(styleId)) return;
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
#root .server-inline-list {
  width: max-content !important;
  min-width: 100% !important;
  max-width: none !important;
  overflow: visible !important;
  scrollbar-width: auto !important;
}
#root .server-inline-list::-webkit-scrollbar {
  display: none !important;
}
#root .server-inline-list > section,
#root .server-inline-list > div {
  width: max-content !important;
  min-width: 100% !important;
  max-width: none !important;
}
#root .server-inline-list > section > div > div:last-child > section:first-child {
  display: grid !important;
  grid-template-columns:
    var(--geoip-col-1,70px)
    var(--geoip-col-2,58px)
    var(--geoip-col-3,78px)
    var(--geoip-col-4,86px)
    var(--geoip-col-5,86px)
    var(--geoip-col-6,70px)
    var(--geoip-col-7,70px)
    var(--geoip-col-8,82px)
    var(--geoip-col-9,82px) !important;
  gap: 0 12px !important;
  align-items: center !important;
  width: max-content !important;
}
#root .server-inline-list > section > div > div:last-child > section:first-child > div {
  width: 100% !important;
  min-width: 0 !important;
  max-width: none !important;
  overflow: hidden !important;
}
#root .server-inline-list > section > div > div:last-child > section:not(:first-child) {
  display: flex !important;
  grid-template-columns: none !important;
  width: auto !important;
  min-width: 0 !important;
  max-width: none !important;
  flex-wrap: wrap !important;
  gap: 4px !important;
}
.geoip-mobile-ua #root .server-inline-list,
.geoip-mobile-ua #root .server-inline-list > section,
.geoip-mobile-ua #root .server-inline-list > div,
.geoip-mobile-ua #root .server-inline-list > section > div,
.geoip-mobile-ua #root .server-inline-list > section > div > div:last-child,
.geoip-mobile-ua #root .server-inline-list > div > div:last-child {
  width: 100% !important;
  min-width: 0 !important;
  max-width: 100% !important;
}

.geoip-mobile-ua #root .server-inline-list > section > div > div:last-child > section:first-child,
.geoip-mobile-ua #root .server-inline-list > div > div:last-child > section:first-child {
  width: 100% !important;
  min-width: 0 !important;
  max-width: 100% !important;
  display: grid !important;
  grid-template-columns: repeat(3, minmax(0, 1fr)) !important;
  gap: 8px 10px !important;
}
`;
    document.head.appendChild(style);
  }

  function metricRows() {
    return Array.from(document.querySelectorAll(containerSelector))
      .map((container) => container.firstElementChild)
      .filter((row) => row && row.tagName === "SECTION" && row.children.length >= minWidths.length);
  }

  function ensureMeasurer() {
    if (measurer) return measurer;
    measurer = document.createElement("div");
    measurer.setAttribute("aria-hidden", "true");
    measurer.style.cssText = [
      "position:absolute",
      "left:-10000px",
      "top:0",
      "visibility:hidden",
      "pointer-events:none",
      "contain:layout style",
      "white-space:nowrap",
      "z-index:-1",
    ].join(";");
    document.body.appendChild(measurer);
    return measurer;
  }

  function measureCell(cell) {
    const clone = cell.cloneNode(true);
    clone.style.setProperty("width", "max-content", "important");
    clone.style.setProperty("min-width", "max-content", "important");
    clone.style.setProperty("max-width", "none", "important");
    clone.style.setProperty("overflow", "visible", "important");
    clone.style.setProperty("text-overflow", "clip", "important");
    clone.style.setProperty("white-space", "nowrap", "important");

    clone.querySelectorAll("*").forEach((element) => {
      element.style.setProperty("max-width", "none", "important");
      element.style.setProperty("overflow", "visible", "important");
      element.style.setProperty("text-overflow", "clip", "important");
      element.style.setProperty("white-space", "nowrap", "important");
    });

    const host = ensureMeasurer();
    host.appendChild(clone);
    const width = Math.ceil(clone.getBoundingClientRect().width);
    clone.remove();
    return width;
  }

  function applyWidths() {
    frame = 0;
    if (isMobile()) return;
    ensureStyle();
    const rows = metricRows();
    if (!rows.length) return;

    const widths = minWidths.slice();
    for (const row of rows) {
      const cells = Array.from(row.children);
      cells.forEach((cell, index) => {
        if (index >= widths.length) return;
        widths[index] = Math.max(widths[index], measureCell(cell));
      });
    }

    widths.forEach((width, index) => {
      root.style.setProperty(`--geoip-col-${index + 1}`, `${width}px`);
    });
  }

  function schedule() {
    if (isMobile()) {
      frame = 0;
      minWidths.forEach((_, index) => root.style.removeProperty(`--geoip-col-${index + 1}`));
      return;
    }
    if (frame) return;
    frame = requestAnimationFrame(applyWidths);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", schedule, { once: true });
  } else {
    schedule();
  }

  window.addEventListener("resize", schedule, { passive: true });
  new MutationObserver(schedule).observe(document.body, {
    childList: true,
    subtree: true,
  });
})();
