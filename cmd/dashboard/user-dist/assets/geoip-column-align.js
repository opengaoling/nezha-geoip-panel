(function () {
  const root = document.documentElement;
  const selector = ".server-inline-list > section > div > div:last-child > section";
  const minWidths = [70, 58, 78, 86, 86, 70, 70, 82, 82];
  const measuringClass = "geoip-column-align-measuring";
  let frame = 0;

  function applyWidths() {
    frame = 0;
    const rows = Array.from(document.querySelectorAll(selector));
    if (!rows.length) return;

    root.classList.add(measuringClass);
    const widths = minWidths.slice();
    try {
      for (const row of rows) {
        const cells = Array.from(row.children);
        cells.forEach((cell, index) => {
          if (index >= widths.length) return;
          widths[index] = Math.max(widths[index], Math.ceil(cell.scrollWidth));
        });
      }
    } finally {
      root.classList.remove(measuringClass);
    }

    widths.forEach((width, index) => {
      root.style.setProperty(`--geoip-col-${index + 1}`, `${width}px`);
    });
  }

  function schedule() {
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
    characterData: true,
  });
})();
