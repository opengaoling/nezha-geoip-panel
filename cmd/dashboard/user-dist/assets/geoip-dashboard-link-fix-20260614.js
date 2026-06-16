(function () {
  if (window.__geoipDashboardLinkFixInstalled) return;
  window.__geoipDashboardLinkFixInstalled = true;

  var nativeFetch = typeof window.fetch === "function" ? window.fetch.bind(window) : null;
  var authState = null;
  var authCheckPromise = null;

  function sameOriginUrl(path) {
    return new URL(path, window.location.origin).href;
  }

  function dashboardUrl() {
    return sameOriginUrl("/dashboard");
  }

  function dashboardLoginUrl() {
    return sameOriginUrl("/dashboard/login");
  }

  function isDashboardEntryUrl(url) {
    return url.origin === window.location.origin && (
      url.pathname === "/dashboard" ||
      url.pathname === "/dashboard/" ||
      url.pathname === "/dashboard/login"
    );
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

  function currentLanguage() {
    try {
      return (localStorage.getItem("language") || document.documentElement.lang || navigator.language || "").toLowerCase();
    } catch (_e) {
      return (navigator.language || "").toLowerCase();
    }
  }

  function labelForState(isAuthed) {
    var lang = currentLanguage();
    if (lang.indexOf("zh-tw") === 0 || lang.indexOf("zh-hk") === 0) {
      return isAuthed ? "管理後台" : "登錄";
    }
    if (lang.indexOf("zh") === 0) {
      return isAuthed ? "后台管理" : "登录";
    }
    return isAuthed ? "Dashboard" : "Login";
  }

  function isAuthedProfilePayload(payload) {
    if (!payload || typeof payload !== "object") return false;
    if (payload.error) return false;
    if (payload.success === false) return false;
    return Boolean(payload.data || payload.user || payload.id || payload.username);
  }

  function checkAuth() {
    if (authCheckPromise) return authCheckPromise;
    if (!nativeFetch) {
      authState = false;
      updateDashboardLinks();
      return Promise.resolve(false);
    }

    authCheckPromise = nativeFetch("/api/v1/profile", {
      credentials: "same-origin",
      cache: "no-store",
      headers: { Accept: "application/json" }
    }).then(function (response) {
      if (!response || !response.ok) return false;
      return response.clone().json().then(isAuthedProfilePayload, function () {
        return true;
      });
    }).catch(function () {
      return false;
    }).then(function (isAuthed) {
      authState = Boolean(isAuthed);
      updateDashboardLinks();
      return authState;
    }).finally(function () {
      authCheckPromise = null;
    });

    return authCheckPromise;
  }

  function targetUrlForState(isAuthed) {
    return isAuthed ? dashboardUrl() : dashboardLoginUrl();
  }

  function setLinkText(link, text) {
    var textTargets = [];
    for (var i = 0; i < link.childNodes.length; i += 1) {
      if (link.childNodes[i].nodeType === Node.TEXT_NODE && link.childNodes[i].nodeValue.trim()) {
        textTargets.push(link.childNodes[i]);
      }
    }

    if (textTargets.length) {
      textTargets[textTargets.length - 1].nodeValue = text;
      return;
    }

    var label = link.querySelector("span:not(.sr-only), p, div");
    if (label && label.textContent && /dashboard|login|后台|後台|登录|登錄|管理/i.test(label.textContent)) {
      label.textContent = text;
      return;
    }

    link.textContent = text;
  }

  function updateDashboardLinks() {
    if (authState === null) return;

    var links = document.querySelectorAll("a[href]");
    for (var i = 0; i < links.length; i += 1) {
      var link = links[i];
      if (!isDashboardEntry(link)) continue;

      link.href = targetUrlForState(authState);
      setLinkText(link, labelForState(authState));
      link.setAttribute("data-geoip-dashboard-entry", authState ? "dashboard" : "login");
    }
  }

  function navigateForAuth(isAuthed) {
    window.location.assign(targetUrlForState(isAuthed));
  }

  function handleNavigationEvent(event) {
    var link = event.target && event.target.closest && event.target.closest("a[href]");

    if (!isDashboardEntry(link)) return;

    event.preventDefault();
    event.stopPropagation();
    if (authState !== null) {
      navigateForAuth(authState);
      return;
    }

    checkAuth().then(navigateForAuth);
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
            checkAuth().then(function (isAuthed) {
              if (!isAuthed && nextUrl.pathname !== "/dashboard/login") {
                window.location.replace(dashboardLoginUrl());
              }
            });
          }
        } catch (_e) {}
      }

      return result;
    };
  }

  document.addEventListener("pointerdown", handleNavigationEvent, true);
  document.addEventListener("touchend", handleNavigationEvent, true);
  document.addEventListener("click", handleNavigationEvent, true);
  wrapHistoryMethod("pushState");
  wrapHistoryMethod("replaceState");

  new MutationObserver(updateDashboardLinks).observe(document.documentElement, {
    childList: true,
    subtree: true
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", checkAuth, { once: true });
  } else {
    checkAuth();
  }
})();
