(function () {
  function clampPercent(value) {
    var number = Number(value);
    if (!Number.isFinite(number)) return 0;
    return Math.max(0, Math.min(100, number));
  }

  function updateProgressBars(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var bars = [];
    if (scope.matches && scope.matches('[aria-label="Server Usage Bar"][aria-valuenow]')) {
      bars.push(scope);
    }
    scope.querySelectorAll('[aria-label="Server Usage Bar"][aria-valuenow]').forEach(function (bar) {
      bars.push(bar);
    });
    bars.forEach(function (bar) {
      var value = clampPercent(bar.getAttribute("aria-valuenow"));
      var indicator = bar.firstElementChild;
      bar.style.setProperty("--geoip-progress-value", String(value));
      if (indicator) {
        indicator.style.setProperty("transform", "translateX(-" + (100 - value) + "%)", "important");
      }
    });
  }

  function alignDesktopMetrics() {
    var html = document.documentElement;
    if (html.classList.contains("geoip-mobile-ua") || window.innerWidth < 768) return;
    var offlineTitle = document.querySelector("#root .server-overview > div:nth-child(3) p:first-child");
    var firstRow = document.querySelector("#root .server-card-list > *");
    if (!offlineTitle || !firstRow) return;
    var rowStyle = window.getComputedStyle(firstRow);
    var rowLeft = firstRow.getBoundingClientRect().left;
    var titleLeft = offlineTitle.getBoundingClientRect().left;
    var rowPaddingLeft = parseFloat(rowStyle.paddingLeft) || 0;
    var rowColumnGap = parseFloat(rowStyle.columnGap) || parseFloat(rowStyle.gap) || 0;
    var nameWidth = Math.round(titleLeft - rowLeft - rowPaddingLeft - rowColumnGap);
    if (nameWidth >= 220 && nameWidth <= 900) {
      html.style.setProperty("--geoip-desktop-name-col", nameWidth + "px");
    } else {
      html.style.setProperty("--geoip-desktop-name-col", "292px");
    }
  }

  function run(root) {
    updateProgressBars(root);
    alignDesktopMetrics();
  }

  function boot() {
    var root = document.getElementById("root") || document.body;
    run(document);
    [80, 240, 600, 1200, 2400].forEach(function (delay) {
      window.setTimeout(function () {
        run(document);
      }, delay);
    });
    window.addEventListener("resize", function () {
      window.requestAnimationFrame(function () {
        alignDesktopMetrics();
      });
    }, { passive: true });
    new MutationObserver(function (mutations) {
      for (var i = 0; i < mutations.length; i += 1) {
        run(mutations[i].target);
      }
    }).observe(root, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["aria-valuenow", "class"]
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
