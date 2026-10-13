(() => {
  const toc = document.querySelector(".shader-toc");
  if (toc && window.matchMedia("(max-width: 700px)").matches) {
    toc.open = false;
  }

  const toggle = document.getElementById("interactive-shader-defs");
  const options = document.getElementById("shader-def-options");
  if (!toggle || !options) return;

  const declarations = [...document.querySelectorAll("[data-shader-defs]")];
  const requirementsFor = (declaration) => {
    try {
      const requirements = JSON.parse(declaration.dataset.shaderDefs || "[]");
      return Array.isArray(requirements) ? requirements : [];
    } catch (_error) {
      return [];
    }
  };

  const conditionEnabled = (expression, enabled) => {
    const alternatives = expression.split("||");
    return alternatives.some((alternative) =>
      alternative.split("&&").every((term) => {
        term = term.trim().replace(/[()]/g, "");
        const negated = term.startsWith("!");
        const name = negated ? term.slice(1).trim() : term;
        return negated ? !enabled.has(name) : enabled.has(name);
      }),
    );
  };

  const updateVisibility = () => {
    const enabled = new Set(
      [...options.querySelectorAll("input[type=checkbox]:checked")].map((input) => input.value),
    );
    declarations.forEach((declaration) => {
      const visible = requirementsFor(declaration).every((requirement) =>
        requirement.branch === "if"
          ? conditionEnabled(requirement.defName, enabled)
          : !conditionEnabled(requirement.defName, enabled),
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
