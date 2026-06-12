(function () {
  function clampPercent(value) {
    var number = Number(value);
    if (!Number.isFinite(number)) return 0;
    return Math.max(0, Math.min(100, number));
  }

  function valueFromIndicator(indicator) {
    if (!indicator) return 0;
    var transform = indicator.style && indicator.style.transform ? indicator.style.transform : "";
    var match = transform.match(/translateX\(-?([0-9.]+)%\)/);
    if (!match) return 0;
    return clampPercent(100 - Number(match[1]));
  }

  function updateProgressBars(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var bars = [];
    if (scope.matches && scope.matches('[aria-label="Server Usage Bar"]')) {
      bars.push(scope);
    }
    scope.querySelectorAll('[aria-label="Server Usage Bar"]').forEach(function (bar) {
      bars.push(bar);
    });
    bars.forEach(function (bar) {
      var indicator = bar.firstElementChild;
      var value = bar.hasAttribute("aria-valuenow")
        ? clampPercent(bar.getAttribute("aria-valuenow"))
        : valueFromIndicator(indicator);
      var nextTransform = "translateX(-" + (100 - value) + "%)";
      bar.style.setProperty("--geoip-progress-value", String(value));
      if (
        indicator &&
        (indicator.style.transform !== nextTransform ||
          indicator.style.getPropertyPriority("transform") !== "important")
      ) {
        indicator.style.setProperty("transform", nextTransform, "important");
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
      attributeFilter: ["aria-valuenow", "class", "style"]
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
