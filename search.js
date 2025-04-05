const fuseOptions = {
  includeMatches: true,
  useExtendedSearch: true,
  keys: ["filename", "name", "comment"],
};

async function loadTemplate() {
  const response = await fetch("/public/search-result.hbs");
  const templateSource = await response.text();
  const template = Handlebars.compile(templateSource);
  return template;
}

fetch("/public/search-info.json")
  .then((res) => res.json())
  .then(async (shadersFunctions) => {
    const input = document.getElementById("search-input");
    const resultsContainer = document.getElementById("results");

    const template = await loadTemplate();
    const currentUrl = new URL(window.location);

    const fuse = new Fuse(shadersFunctions, fuseOptions);
    function renderResults(results) {
      const data = results.length > 0 ? results.map((r) => r.item) : [];
      resultsContainer.innerHTML = template(data);
    }

    const search = currentUrl.searchParams.get("search");
    if (search) {
      const query = search.trim();
      input.value = query;
      renderResults(fuse.search(query).slice(0, 10));
    }

    input.addEventListener("input", () => {
      const query = input.value.trim();

      if (query) {
        currentUrl.searchParams.set("search", query);
      } else {
        currentUrl.searchParams.delete("search");
      }
      window.history.pushState({}, "", currentUrl);

      const result = query ? fuse.search(query).slice(0, 10) : [];
      renderResults(result);
    });
  });
