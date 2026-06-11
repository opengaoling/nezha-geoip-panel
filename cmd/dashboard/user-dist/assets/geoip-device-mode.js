(function () {
  const root = document.documentElement;
  const uaData = navigator.userAgentData;
  const ua = navigator.userAgent || navigator.vendor || "";
  const isIpadDesktopUa = /\bMacintosh\b/.test(ua) && navigator.maxTouchPoints > 1;
  const mobileByUa =
    uaData && typeof uaData.mobile === "boolean"
      ? uaData.mobile
      : isIpadDesktopUa ||
        /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Windows Phone|Mobi|Mobile/i.test(ua);

  root.classList.toggle("geoip-mobile-ua", mobileByUa);
  root.classList.toggle("geoip-desktop-ua", !mobileByUa);
  root.dataset.geoipDevice = mobileByUa ? "mobile" : "desktop";
})();
