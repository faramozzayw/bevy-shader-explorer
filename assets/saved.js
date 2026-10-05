(() => {
  const storageKey = "bevy-shader-explorer:saved-shaders";

  const localPath = (value) => {
    try {
      const url = new URL(value, window.location.href);
      return url.pathname + url.search + url.hash;
    } catch (_error) {
      return value;
    }
  };

  const readSaved = () => {
    try {
      const value = JSON.parse(localStorage.getItem(storageKey) || "[]");
      return Array.isArray(value)
        ? value.map((item) => ({ ...item, url: localPath(item.url) }))
        : [];
    } catch (_error) {
      return [];
    }
  };

  const writeSaved = (items) => localStorage.setItem(storageKey, JSON.stringify(items));

  const saveButton = document.querySelector("[data-save-shader]");
  if (saveButton) {
    const entry = {
      url: window.location.pathname + window.location.search + window.location.hash,
      name: saveButton.dataset.saveName || document.title,
      packageName: saveButton.dataset.savePackage || "",
      version: saveButton.dataset.saveVersion || "",
    };
    const updateButton = () => {
      const saved = readSaved().some((item) => item.url === entry.url);
      saveButton.textContent = saved ? "Saved" : "Save";
      saveButton.classList.toggle("is-saved", saved);
      saveButton.setAttribute("aria-pressed", String(saved));
    };
    saveButton.addEventListener("click", () => {
      const saved = readSaved();
      const index = saved.findIndex((item) => item.url === entry.url);
      if (index >= 0) saved.splice(index, 1);
      else saved.unshift(entry);
      writeSaved(saved);
      updateButton();
    });
    updateButton();
  }

  const section = document.querySelector("[data-saved-list]");
  if (!section) return;

  const savedSection = document.getElementById("saved-shaders");
  const render = () => {
    const saved = readSaved();
    savedSection.hidden = saved.length === 0;
    section.replaceChildren();
    saved.forEach((item) => {
      const row = document.createElement("article");
      row.className = "saved-shader-row";
      const link = document.createElement("a");
      link.href = item.url;
      link.className = "saved-shader-link";
      link.textContent = item.name;
      const meta = document.createElement("span");
      meta.textContent = `${item.packageName} ${item.version}`.trim();
      const remove = document.createElement("button");
      remove.type = "button";
      remove.className = "saved-remove-button";
      remove.textContent = "Remove";
      remove.addEventListener("click", () => {
        writeSaved(readSaved().filter((savedItem) => savedItem.url !== item.url));
        render();
      });
      row.append(link, meta, remove);
      section.append(row);
    });
  };

  document.querySelector("[data-saved-clear]")?.addEventListener("click", () => {
    writeSaved([]);
    render();
  });
  render();
})();
