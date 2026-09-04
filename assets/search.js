const fuseOptions = {
  includeMatches: true,
  useExtendedSearch: true,
  keys: ["filename", "name", "comment", "stageAttribute", "type", "packageName", "version", "description"],
};

const currentUrl = new URL(window.location);
const searchHeader = document.querySelector("header[data-search-scope]");
const pathParts = currentUrl.pathname.split("/").filter(Boolean);
const legacyPackagePage = !searchHeader && pathParts.length >= 2;
const searchScope = searchHeader?.dataset.searchScope || (legacyPackagePage ? "package" : "packages");
const searchPackage = searchHeader?.dataset.searchPackage || (legacyPackagePage ? pathParts[0] : "");
const searchVersion = searchHeader?.dataset.searchVersion || (legacyPackagePage ? pathParts[1] : "");
const searchIndex = searchHeader?.dataset.searchIndex || (legacyPackagePage ? pathParts[1] : "project");

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
    .replace(stageAttributeRegex, (_match, flag) => {
      flags.push(flag.toLowerCase());
      return "";
    })
    .trim();
  return { cleanedQuery, flags };
};

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    document.activeElement.blur();
    return;
  }

  if (event.key === "s" || event.key === "S" || event.key === "/") {
    const searchInput = document.querySelector("input#search-input");

    if (document.activeElement !== searchInput) {
      event.preventDefault();
    }

    if (searchInput) searchInput.focus();
  }
});

const loadSearchData = async () => {
  if (searchScope === "packages") {
    const response = await fetch("/public/packages.json");
    const packages = await response.json();
    return packages.map((pkg) => ({
      kind: "package",
      name: pkg.packageName,
      packageName: pkg.packageName,
      version: pkg.version,
      description: pkg.description,
      comment: pkg.description,
      link: `/${pkg.detailPath}`,
      filename: "",
    }));
  }
  const response = await fetch(`/public/search-info-${searchIndex}.json`);
  const items = await response.json();
  return items.filter((item) => item.packageName === searchPackage && item.packageVersion === searchVersion);
};

loadSearchData()
  .then(async (shadersFunctions) => {
    const input = document.getElementById("search-input");
    const resultsContainer = document.getElementById("results");

    const template = await loadTemplate();

    function renderResults(results) {
      if (results.length === 0) {
        resultsContainer.innerHTML = null;
      } else {
        resultsContainer.innerHTML = template(results.map((r) => r.item));
      }
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
