(() => {
  const toc = document.querySelector(".shader-toc");
  if (toc && window.matchMedia("(max-width: 700px)").matches) {
    toc.open = false;
  }

  const toggle = document.getElementById("interactive-shader-defs");
  const options = document.getElementById("shader-def-options");
  const declarations = [...document.querySelectorAll("[data-shader-defs]")];
  const sourceLines = [...document.querySelectorAll(".source-line[data-shader-defs]")];
  const requirementsFor = (declaration) => {
    try {
      const requirements = JSON.parse(declaration.dataset.shaderDefs || "[]");
      return Array.isArray(requirements) ? requirements : [];
    } catch (_error) {
      return [];
    }
  };

  const shaderDefOptions = [...document.querySelectorAll(".shader-def-option")].filter(
    (option) => option.querySelector("input[type=checkbox][value]"),
  );

  const url = new URL(window.location.href);
  const urlDefinitions = new Set((url.searchParams.get("shader-defs") || "").split(",").filter(Boolean));
  const hasURLDefinitions = url.searchParams.has("shader-defs");

  const clearTocHighlights = () => {
    document.querySelectorAll(".shader-toc a.shader-def-highlight").forEach((link) => {
      link.classList.remove("shader-def-highlight");
    });
  };

  const conditionContains = (condition, name) =>
    condition?.clauses?.some((clause) => clause.some((term) => term.name === name));

  const highlightTocForDefinition = (name) => {
    clearTocHighlights();
    const sectionIds = new Set();
    declarations.forEach((declaration) => {
      const requirements = requirementsFor(declaration);
      if (requirements.some((requirement) => conditionContains(requirement.condition, name))) {
        const section = declaration.closest("section[id]");
        if (section) sectionIds.add(section.id);
      }
    });
    document.querySelectorAll(".shader-toc a").forEach((link) => {
      if (sectionIds.has(link.getAttribute("href")?.slice(1))) {
        link.classList.add("shader-def-highlight");
      }
    });
  };

  shaderDefOptions.forEach((option) => {
    const input = option.querySelector("input[type=checkbox]");
    if (!input) return;
    const highlight = () => highlightTocForDefinition(input.value);
    option.addEventListener("mouseenter", highlight);
    option.addEventListener("focusin", highlight);
    option.addEventListener("mouseleave", clearTocHighlights);
    option.addEventListener("focusout", clearTocHighlights);
  });

  const syncURL = () => {
    const next = new URL(window.location.href);
    if (!toggle.checked) {
      next.searchParams.delete("shader-defs");
    } else {
      const enabled = [...options.querySelectorAll("input[type=checkbox][value]:checked")]
        .map((input) => input.value)
        .sort();
      next.searchParams.set("shader-defs", enabled.join(","));
    }
    window.history.replaceState({}, "", next);
  };

  document.querySelectorAll('a[title="View source"], a.source-back-button').forEach((link) => {
    link.addEventListener("click", () => {
      const current = new URL(window.location.href);
      const target = new URL(link.href, window.location.href);
      const state = current.searchParams.get("shader-defs");
      if (state) target.searchParams.set("shader-defs", state);
      link.href = target.toString();
    });
  });

  if (!toggle || !options) return;

  const conditionEnabled = (condition, enabled) =>
    condition.clauses.some((clause) =>
      clause.every((term) => (term.negated ? !enabled.has(term.name) : enabled.has(term.name))),
    );

  const requirementCondition = (requirement) => requirement.condition || { clauses: [] };

  const updateVisibility = () => {
    const enabled = new Set(
      [...options.querySelectorAll("input[type=checkbox][value]:checked")].map((input) => input.value),
    );
    declarations.forEach((declaration) => {
      const visible = requirementsFor(declaration).every((requirement) =>
        requirement.branch === "if"
          ? conditionEnabled(requirementCondition(requirement), enabled)
          : !conditionEnabled(requirementCondition(requirement), enabled),
      );
      declaration.classList.toggle("shader-def-inactive", toggle.checked && !visible);
    });
    sourceLines.forEach((line) => {
      const visible = requirementsFor(line).every((requirement) =>
        requirement.branch === "if"
          ? conditionEnabled(requirementCondition(requirement), enabled)
          : !conditionEnabled(requirementCondition(requirement), enabled),
      );
      line.classList.toggle("source-line-inactive", toggle.checked && !visible);
    });
  };

  toggle.addEventListener("change", () => {
    options.hidden = !toggle.checked;
    updateVisibility();
    syncURL();
  });
  options.addEventListener("change", () => {
    updateVisibility();
    syncURL();
  });
  shaderDefOptions.forEach((option) => {
    const input = option.querySelector("input[type=checkbox][value]");
    input?.addEventListener("change", () => {
      updateVisibility();
      syncURL();
    });
  });
  if (hasURLDefinitions) {
    toggle.checked = true;
    options.hidden = false;
    options.querySelectorAll("input[type=checkbox][value]").forEach((input) => {
      input.checked = urlDefinitions.has(input.value);
    });
  }
  options.hidden = !toggle.checked;
  updateVisibility();
})();
