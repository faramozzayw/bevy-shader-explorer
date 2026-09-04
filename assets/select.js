(() => {
  const selects = document.querySelectorAll(".version-select");
  const packagesByName = fetch("/public/package-versions.json")
    .then((response) => (response.ok ? response.json() : {}))
    .catch(() => ({}));

  selects.forEach((select) => {
    const packageName = select.dataset.package;
    if (!packageName) return;

    packagesByName.then((packages) => {
      const versions = (packages[packageName] || []).slice().sort((a, b) =>
        b.label.localeCompare(a.label, undefined, { numeric: true }),
      );
      if (versions.length === 0) return;

      select.replaceChildren(...versions.map((version) => {
        const option = document.createElement("option");
        option.value = "/" + version.url;
        option.textContent = version.label;
        // A shader page has the package version followed by the shader path,
        // so compare the version segment instead of the complete URL.
        const packagePrefix = "/" + packageName + "/";
        const currentVersion = location.pathname.startsWith(packagePrefix)
          ? location.pathname.slice(packagePrefix.length).split("/")[0]
          : "";
        option.selected = currentVersion === version.label;
        return option;
      }));
    });
  });
})();
