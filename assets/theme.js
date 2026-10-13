(() => {
  const root = document.documentElement;
  const button = document.getElementById("theme-toggle");
  if (!button) return;

  const themeColor = document.querySelector('meta[name="theme-color"]');

  function applyTheme(theme, persist) {
    root.dataset.theme = theme;
    if (persist) {
      try {
        localStorage.setItem("bevy-shader-theme", theme);
      } catch (_) {}
    }
    const dark = theme === "dark";
    button.setAttribute("aria-label", dark ? "Switch to light theme" : "Switch to dark theme");
    button.setAttribute("aria-pressed", String(dark));
    button.title = dark ? "Switch to light theme" : "Switch to dark theme";
    button.querySelector(".theme-toggle-icon").textContent = dark ? "☀" : "☾";
    if (themeColor) themeColor.content = dark ? "#1c1b18" : "#c9c4a8";
  }

  applyTheme(root.dataset.theme === "dark" ? "dark" : "light", false);
  button.addEventListener("click", () => {
    applyTheme(root.dataset.theme === "dark" ? "light" : "dark", true);
  });
})();
