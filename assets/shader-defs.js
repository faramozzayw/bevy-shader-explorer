(() => {
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

  const updateVisibility = () => {
    const enabled = new Set(
      [...options.querySelectorAll("input[type=checkbox]:checked")].map((input) => input.value),
    );
    declarations.forEach((declaration) => {
      const visible = requirementsFor(declaration).every((requirement) =>
        requirement.branch === "if"
          ? enabled.has(requirement.defName)
          : !enabled.has(requirement.defName),
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
