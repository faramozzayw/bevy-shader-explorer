const select = document.querySelector(".version-select");
if (select) {
  const packageName = select.dataset.package;
  if (packageName) {
    fetch("/public/package-versions.json")
      .then((response) => response.ok ? response.json() : {})
      .then((packages) => {
        const versions = (packages[packageName] || []).slice().sort((a, b) =>
          b.label.localeCompare(a.label, undefined, { numeric: true })
        );
        if (versions.length > 0) {
          select.replaceChildren(...versions.map((version) => {
            const option = document.createElement("option");
            option.value = "/" + version.url;
            option.textContent = version.label;
            option.selected = location.pathname === option.value;
            return option;
          }));
        }
      })
      .catch(() => {});
  }
}
