(function () {
  if (window.__nezhaAuthGuardInstalled) return;
  window.__nezhaAuthGuardInstalled = true;

  var redirected = false;
  var authHeader = "X-Nezha-Auth-Invalid";

  function sameOriginApi(input) {
    try {
      var raw = typeof input === "string" ? input : input && input.url;
      if (!raw) return false;
      var url = new URL(raw, window.location.href);
      return url.origin === window.location.origin && url.pathname.indexOf("/api/v1/") === 0;
    } catch (_e) {
      return false;
    }
  }

  function clearAuthStorage() {
    try {
      document.cookie = "nz-jwt=; Max-Age=0; path=/; SameSite=Lax";
      document.cookie = "nz-csrf=; Max-Age=0; path=/; SameSite=Strict";
    } catch (_e) {}
    try {
      localStorage.removeItem("token");
      localStorage.removeItem("nezha-token");
      localStorage.removeItem("jwt");
      sessionStorage.removeItem("token");
      sessionStorage.removeItem("nezha-token");
      sessionStorage.removeItem("jwt");
    } catch (_e) {}
  }

  function loginTarget() {
    return window.location.pathname.indexOf("/dashboard") === 0 ? "/dashboard/login" : "/";
  }

  function redirectForAuth() {
    if (redirected) return;
    redirected = true;
    clearAuthStorage();
    var target = loginTarget();
    if (window.location.pathname + window.location.search !== target) {
      window.location.replace(target);
    }
  }

  function shouldRedirect(response) {
    if (!response) return false;
    if (response.status === 401 || response.status === 403) return true;
    if (response.headers && response.headers.get(authHeader) === "1") return true;
    var contentType = response.headers && response.headers.get("content-type");
    return contentType && contentType.indexOf("text/html") !== -1;
  }

  var nativeFetch = window.fetch;
  if (typeof nativeFetch === "function") {
    window.fetch = function (input, init) {
      var api = sameOriginApi(input);
      return nativeFetch.call(this, input, init).then(function (response) {
        if (api && shouldRedirect(response)) {
          redirectForAuth();
          return new Response(JSON.stringify({
            success: false,
            error: "ApiErrorUnauthorized"
          }), {
            status: 401,
            headers: {
              "content-type": "application/json",
              "X-Nezha-Auth-Invalid": "1"
            }
          });
        }
        return response;
      });
    };
  }

  var NativeXHR = window.XMLHttpRequest;
  if (typeof NativeXHR === "function") {
    var nativeOpen = NativeXHR.prototype.open;
    var nativeSend = NativeXHR.prototype.send;

    NativeXHR.prototype.open = function (method, url) {
      this.__nezhaApiRequest = sameOriginApi(url);
      return nativeOpen.apply(this, arguments);
    };

    NativeXHR.prototype.send = function () {
      if (this.__nezhaApiRequest) {
        this.addEventListener("load", function () {
          var contentType = "";
          try {
            contentType = this.getResponseHeader("content-type") || "";
          } catch (_e) {}
          if (this.status === 401 || this.status === 403 || contentType.indexOf("text/html") !== -1) {
            redirectForAuth();
          }
        });
      }
      return nativeSend.apply(this, arguments);
    };
  }
})();
