(function () {
  var metricClasses = ["geoip-mobile-cpu", "geoip-mobile-mem", "geoip-mobile-stg"];

  function percentFromText(cell) {
    var textNode = cell.querySelector("div:not([aria-label])");
    if (!textNode) return null;
    var match = textNode.textContent.match(/([0-9]+(?:\.[0-9]+)?)\s*%/);
    if (!match) return null;
    var value = Number(match[1]);
    if (!Number.isFinite(value)) return null;
    return Math.max(0, Math.min(100, value));
  }

  function normalizeUsageBar(cell) {
    var bar = cell.querySelector('[aria-label="Server Usage Bar"]');
    if (!bar) return;
    if (bar.parentElement !== cell) {
      cell.appendChild(bar);
    }
    var value = percentFromText(cell);
    if (value == null) return;
    bar.setAttribute("data-value", value.toFixed(2));
    bar.setAttribute("data-max", "100");
    var indicator = bar.firstElementChild;
    if (indicator) {
      indicator.style.transform = "translateX(-" + (100 - value).toFixed(4) + "%)";
    }
  }

  function normalizeUsageBars(root) {
    metricClasses.forEach(function (className) {
      root.querySelectorAll("." + className).forEach(normalizeUsageBar);
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
    var rowWidth = firstRow.getBoundingClientRect().width;
    var titleLeft = offlineTitle.getBoundingClientRect().left;
    var rowPaddingLeft = parseFloat(rowStyle.paddingLeft) || 0;
    var rowPaddingRight = parseFloat(rowStyle.paddingRight) || 0;
    var rowColumnGap = parseFloat(rowStyle.columnGap) || parseFloat(rowStyle.gap) || 0;
    var metricsWidth = 740;
    var actionCell = firstRow.children && firstRow.children.length > 2 ? firstRow.children[2] : null;
    var actionWidth = actionCell ? actionCell.getBoundingClientRect().width : 0;
    var availableNameWidth = Math.floor(rowWidth - rowPaddingLeft - rowPaddingRight - metricsWidth - actionWidth - rowColumnGap * 2);
    var targetNameWidth = Math.round(titleLeft - rowLeft - rowPaddingLeft - rowColumnGap);
    var nameWidth = Math.max(180, Math.min(targetNameWidth, availableNameWidth, 340));
    html.style.setProperty("--geoip-desktop-metrics-width", metricsWidth + "px");
    if (nameWidth >= 180 && nameWidth <= 340) {
      html.style.setProperty("--geoip-desktop-name-col", nameWidth + "px");
    } else {
      html.style.setProperty("--geoip-desktop-name-col", "292px");
    }
  }

  function run(root) {
    alignDesktopMetrics();
    normalizeUsageBars(root || document);
  }

  function boot() {
    var root = document.getElementById("root") || document.body;
    var scheduled = false;
    function schedule(target) {
      if (scheduled) return;
      scheduled = true;
      window.requestAnimationFrame(function () {
        scheduled = false;
        run(target || document);
      });
    }
    schedule(document);
    [80, 240, 600, 1200, 2400].forEach(function (delay) {
      window.setTimeout(function () {
        schedule(document);
      }, delay);
    });
    window.addEventListener("resize", function () {
      schedule(document);
    }, { passive: true });
    new MutationObserver(function (mutations) {
      var target = document;
      for (var i = 0; i < mutations.length; i += 1) {
        target = mutations[i].target || document;
        break;
      }
      schedule(target);
    }).observe(root, {
      childList: true,
      subtree: true,
      attributes: false
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
