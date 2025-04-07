const fuseOptions = {
  includeMatches: true,
  useExtendedSearch: true,
  keys: ["filename", "name", "comment", "stageAttribute", "type"],
};

const currentUrl = new URL(window.location);

async function loadTemplate() {
  const response = await fetch("/public/search-result.hbs");
  const templateSource = await response.text();
  const template = Handlebars.compile(templateSource);
  return template;
}

const parseQuery = (rawQuery) => {
  const stageAttributeRegex = /@(\w+)/g;
  const flags = [];
  let cleanedQuery = rawQuery
    .replace(stageAttributeRegex, (match, flag) => {
      flags.push(flag.toLowerCase());
      return "";
    })
    .trim();
  return { cleanedQuery, flags };
};

fetch("/public/search-info.json")
  .then((res) => res.json())
  .then(async (shadersFunctions) => {
    const input = document.getElementById("search-input");
    const resultsContainer = document.getElementById("results");

    const template = await loadTemplate();

    function renderResults(results) {
      const data = results.length > 0 ? results.map((r) => r.item) : [];
      resultsContainer.innerHTML = template(data);
    }

    function doSearch(query) {
      query = query.trim();
      if (!query) return [];

      const { cleanedQuery, flags } = parseQuery(query);

      let filteredData = shadersFunctions;
      if (flags.length) {
        filteredData = filteredData.filter((item) =>
          flags.includes(item.stageAttribute?.toLowerCase()),
        );
      }

      const fuse = new Fuse(filteredData, fuseOptions);
      return fuse.search(cleanedQuery).slice(0, 10);
    }

    const search = currentUrl.searchParams.get("search") ?? "";

    // init render
    input.value = search;
    renderResults(doSearch(search));

    input.addEventListener("input", () => {
      const query = input.value.trim();

      if (query) {
        currentUrl.searchParams.set("search", query);
      } else {
        currentUrl.searchParams.delete("search");
      }
      window.history.pushState({}, "", currentUrl);
      renderResults(doSearch(query));
    });
  });
