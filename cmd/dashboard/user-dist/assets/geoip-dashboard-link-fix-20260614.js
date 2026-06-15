(function () {
  if (window.__geoipDashboardLinkFixInstalled) return;
  window.__geoipDashboardLinkFixInstalled = true;

  function dashboardLoginUrl() {
    return new URL("/dashboard/login", window.location.origin).href;
  }

  function isDashboardEntryUrl(url) {
    return url.origin === window.location.origin && (url.pathname === "/dashboard" || url.pathname === "/dashboard/");
  }

  function isDashboardEntry(link) {
    if (!link || !link.href) return false;

    try {
      var url = new URL(link.href, window.location.href);
      return isDashboardEntryUrl(url);
    } catch (_e) {
      return false;
    }
  }

  function redirectToLogin() {
    window.location.assign(dashboardLoginUrl());
  }

  function redirectIfDashboardEntry() {
    try {
      if (isDashboardEntryUrl(new URL(window.location.href))) {
        window.location.replace(dashboardLoginUrl());
      }
    } catch (_e) {}
  }

  function handleNavigationEvent(event) {
    var link = event.target && event.target.closest && event.target.closest("a[href]");

    if (!isDashboardEntry(link)) return;

    event.preventDefault();
    event.stopPropagation();
    redirectToLogin();
  }

  function wrapHistoryMethod(methodName) {
    var nativeMethod = window.history && window.history[methodName];
    if (typeof nativeMethod !== "function") return;

    window.history[methodName] = function (_state, _title, url) {
      var result = nativeMethod.apply(this, arguments);

      if (url !== undefined) {
        try {
          var nextUrl = new URL(url, window.location.href);
          if (isDashboardEntryUrl(nextUrl)) {
            window.location.replace(dashboardLoginUrl());
          }
        } catch (_e) {}
      }

      return result;
    };
  }

  document.addEventListener("pointerdown", handleNavigationEvent, true);
  document.addEventListener("touchend", handleNavigationEvent, true);
  document.addEventListener("click", handleNavigationEvent, true);
  window.addEventListener("popstate", redirectIfDashboardEntry);
  wrapHistoryMethod("pushState");
  wrapHistoryMethod("replaceState");
  redirectIfDashboardEntry();
})();
