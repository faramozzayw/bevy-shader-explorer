const currentUrl = new URL(window.location);
const maxSearchResults = 50;
const maxTermResults = 5000;
const searchCacheDatabase = "bevy-shader-explorer";
const searchCacheStore = "search-index";
const searchFields = ["filename", "name", "comment", "stageAttribute", "type", "packageName", "packageVersion", "version", "description"];

async function loadTemplate() {
  const response = await fetch("/public/search-result.hbs");
  const templateSource = await response.text();
  Handlebars.registerHelper("eq", (left, right) => left === right);
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

const openSearchCache = () => new Promise((resolve, reject) => {
  if (!window.indexedDB) {
    resolve(null);
    return;
  }
  const request = indexedDB.open(searchCacheDatabase, 1);
  request.onupgradeneeded = () => request.result.createObjectStore(searchCacheStore);
  request.onsuccess = () => resolve(request.result);
  request.onerror = () => reject(request.error);
});

const readCachedSearchData = async () => {
  let database;
  try {
    database = await openSearchCache();
    if (!database) return null;
    return await new Promise((resolve, reject) => {
      const request = database.transaction(searchCacheStore, "readonly").objectStore(searchCacheStore).get("global");
      request.onsuccess = () => resolve(request.result || null);
      request.onerror = () => reject(request.error);
    });
  } catch {
    return null;
  } finally {
    database?.close();
  }
};

const writeCachedSearchData = async (value) => {
  let database;
  try {
    database = await openSearchCache();
    if (!database) return;
    await new Promise((resolve, reject) => {
      const transaction = database.transaction(searchCacheStore, "readwrite");
      transaction.objectStore(searchCacheStore).put(value, "global");
      transaction.oncomplete = resolve;
      transaction.onerror = () => reject(transaction.error);
    });
  } catch {
    // Search remains functional without persistent browser storage.
  } finally {
    database?.close();
  }
};

const loadSearchData = async () => {
  const cached = await readCachedSearchData();
  let manifest;
  try {
    const response = await fetch("/public/search-index-manifest.json", { cache: "no-store" });
    if (response.ok) manifest = await response.json();
  } catch {
    if (cached?.items) return cached.items;
    try {
      const response = await fetch("/public/search-info-all.json");
      return response.ok ? await response.json() : [];
    } catch {
      return [];
    }
  }

  if (manifest && cached?.hash === manifest.hash) return cached.items;

  try {
    const url = manifest?.url || "/public/search-info-all.json";
    const response = await fetch(url);
    if (!response.ok) throw new Error("global search index unavailable");
    const items = await response.json();
    await writeCachedSearchData({ hash: manifest?.hash || "", items });
    return items;
  } catch {
    return cached?.items || [];
  }
};

loadSearchData()
  .then(async (shadersFunctions) => {
    const input = document.getElementById("search-input");
    const resultsContainer = document.getElementById("results");

    const template = await loadTemplate();
    shadersFunctions.forEach((item) => {
      if (item.versions) {
        item.versions.sort((left, right) =>
          right.label.localeCompare(left.label, undefined, { numeric: true }),
        );
        if (item.kind === "declaration" && item.versions[0]) {
          item.link = item.versions[0].url.split("#", 1)[0];
        }
      }
    });
    const searchIndex = window.FlexSearch
      ? new FlexSearch.Document({
        tokenize: "forward",
        document: { id: "id", index: searchFields, store: true },
      })
      : null;
    if (searchIndex) {
      shadersFunctions.forEach((item, id) => searchIndex.add({ ...item, id }));
    }

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
      const terms = cleanedQuery.toLowerCase().match(/[^\s]+/g) || [];
      if (terms.length === 0) return [];

      const rankResults = (items) => items
        .map((item) => ({ item, score: scoreResult(item, terms) }))
        .sort((left, right) => right.score - left.score || left.item.name.localeCompare(right.item.name))
        .slice(0, maxSearchResults)
        .map((result) => ({ item: result.item }));

      if (!searchIndex) {
        const matches = shadersFunctions
          .filter((item) => !flags.length || flags.includes(item.stageAttribute?.toLowerCase()))
          .filter((item) => terms.every((term) => searchFields.some((field) => String(item[field] || "").toLowerCase().includes(term))));
        return rankResults(matches);
      }

      let matches;
      for (const term of terms) {
        const termMatches = new Map(
          searchIndex
            .search(term, { enrich: true, merge: true, limit: maxTermResults })
            .map((result) => [result.id, result.doc]),
        );
        if (!matches) {
          matches = termMatches;
        } else {
          matches = new Map([...matches].filter(([id]) => termMatches.has(id)));
        }
        if (matches.size === 0) break;
      }

      return rankResults([...matches?.values() || []].filter((item) =>
        !flags.length || flags.includes(item.stageAttribute?.toLowerCase()),
      ));
    }

    function scoreResult(item, terms) {
      const fields = [
        ["name", 1000],
        ["filename", 700],
        ["packageName", 500],
        ["packageVersion", 450],
        ["version", 450],
        ["type", 250],
        ["comment", 100],
        ["description", 100],
      ];
      return terms.reduce((total, term) => {
        let best = 0;
        for (const [field, weight] of fields) {
          const value = String(item[field] || "").toLowerCase();
          if (value === term) best = Math.max(best, weight);
          else if (value.startsWith(term)) best = Math.max(best, weight * 0.8);
          else if (value.includes(term)) best = Math.max(best, weight * 0.4);
        }
        return total + best;
      }, 0);
    }

    const search = currentUrl.searchParams.get("search") ?? "";

    // init render
    input.value = search;
    renderResults(doSearch(search));

    let searchTimer;
    input.addEventListener("input", () => {
      const query = input.value.trim();

      if (query) {
        currentUrl.searchParams.set("search", query);
      } else {
        currentUrl.searchParams.delete("search");
      }
      window.history.replaceState({}, "", currentUrl);
      clearTimeout(searchTimer);
      searchTimer = setTimeout(() => renderResults(doSearch(query)), 80);
    });
  });
