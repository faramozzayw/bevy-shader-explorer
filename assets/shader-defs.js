(() => {
  const toc = document.querySelector(".shader-toc");
  if (toc && window.matchMedia("(max-width: 700px)").matches) {
    toc.open = false;
  }

  const toggle = document.getElementById("interactive-shader-defs");
  const options = document.getElementById("shader-def-options");
  const declarations = [...document.querySelectorAll("[data-shader-defs]")];
  const requirementsFor = (declaration) => {
    try {
      const requirements = JSON.parse(declaration.dataset.shaderDefs || "[]");
      return Array.isArray(requirements) ? requirements : [];
    } catch (_error) {
      return [];
    }
  };

  const shaderDefOptions = [...document.querySelectorAll(".shader-def-option")];

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

  if (!toggle || !options) return;

  const conditionEnabled = (condition, enabled) =>
    condition.clauses.some((clause) =>
      clause.every((term) => (term.negated ? !enabled.has(term.name) : enabled.has(term.name))),
    );

  const requirementCondition = (requirement) => requirement.condition || { clauses: [] };

  const updateVisibility = () => {
    const enabled = new Set(
      [...options.querySelectorAll("input[type=checkbox]:checked")].map((input) => input.value),
    );
    declarations.forEach((declaration) => {
      const visible = requirementsFor(declaration).every((requirement) =>
        requirement.branch === "if"
          ? conditionEnabled(requirementCondition(requirement), enabled)
          : !conditionEnabled(requirementCondition(requirement), enabled),
      );
      declaration.classList.toggle("shader-def-inactive", toggle.checked && !visible);
    });
  };

  toggle.addEventListener("change", () => {
    options.hidden = !toggle.checked;
    updateVisibility();
  });
  options.addEventListener("change", updateVisibility);
  options.hidden = !toggle.checked;
  updateVisibility();
})();
