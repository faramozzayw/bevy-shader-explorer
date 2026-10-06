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

      const shaderPage = select.closest(".shader-page");
      const pathParts = location.pathname.split("/").filter(Boolean);
      const currentVersion = pathParts[0] === packageName ? pathParts[1] : "";
      const shaderTail = currentVersion ? pathParts.slice(2).join("/") : "";
      const existingLinks = new Map([...select.options].map((option) => [option.textContent, option.value]));

      const resolveOption = async (version) => {
        const option = document.createElement("option");
        option.value = "/" + version.url;
        option.textContent = version.label;
        option.selected = currentVersion === version.label;
        if (shaderPage && shaderTail) {
          const existing = existingLinks.get(version.label);
          if (existing && !existing.endsWith("/index.html")) {
            option.value = existing;
          } else if (version.label === currentVersion) {
            option.value = location.pathname;
          } else {
            const candidate = `/${packageName}/${version.label}/${shaderTail}`;
            try {
              const response = await fetch(candidate, { method: "HEAD", cache: "no-store" });
              if (response.ok) option.value = candidate;
            } catch {
              // Keep the package-index fallback when the candidate is absent.
            }
          }
        }
        return option;
      };

      Promise.all(versions.map(resolveOption)).then((options) => {
        select.replaceChildren(...options);
      });
    });
  });
})();
