#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { exec } = require("node:child_process");
const Handlebars = require("handlebars");

const wgpuTypes = JSON.parse(fs.readFileSync("./wgpu-types.json", "utf-8"));

Handlebars.registerHelper("eq", (a, b) => a === b);
Handlebars.registerHelper("neq", (a, b) => a !== b);
Handlebars.registerHelper("linkify", function (text) {
  const urlPattern = /(?:https?|ftp):\/\/[\n\S]+/g;
  return text.replace(urlPattern, function (url) {
    return `<a href="${url}" target="_blank">${url}</a>`;
  });
});
Handlebars.registerHelper("code-highlight", function (text) {
  return text.replace(/`(.*)`/g, function (_, v) {
    return `<code>${v}</code>`;
  });
});
Handlebars.registerHelper("contains", function (needle, haystack, options) {
  return haystack.indexOf(needle) > -1
    ? options.fn(this)
    : options.inverse(this);
});

const { source } = require("minimist")(process.argv.slice(2));
const GREP_WGSL = `find ${source} -type f  -name "*.wgsl"`;

const WGSL_DOC_TEMPLATE_SOURCE = fs.readFileSync(
  "./templates/wgsl-doc.hbs",
  "utf-8",
);
const HOME_DOC_TEMPLATE_SOURCE = fs.readFileSync(
  "./templates/home.hbs",
  "utf-8",
);

const FUNCTION_PATTERN =
  /(@[^;]*\s+)?(vertex|fragment|compute\s+)?\bfn\b\s+([a-zA-Z0-9_]+)[\s\S]*?\{/g;
const STRUCTURE_PATTERN = /struct\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{([^}]*)\}/g;
const TYPE_PATTERN_GLOBAL = /(@\w+\([^)]+\))?(\w+):\s+([^,]+)?/g;

const OUTPUT_DIR_ROOT = "./dist";

function extractWGSLItems(wgslCode) {
  const normalizedCode = wgslCode.replace(/\r\n/g, "\n");
  const lines = normalizedCode.split("\n");

  const lineComments = {};

  // First pass: collect comments
  let commentBuffer = [];
  let isCollectingComment = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();

    // Handle multi-line comments
    if (line.includes("/*")) {
      isCollectingComment = true;
      commentBuffer.push(line.replace(/\/\*/, "").trim());
      // Check if multi-line comment ends on the same line
      if (line.includes("*/")) {
        isCollectingComment = false;
        commentBuffer[commentBuffer.length - 1] = commentBuffer[
          commentBuffer.length - 1
        ]
          .replace(/\*\/.*$/, "")
          .trim();
        lineComments[i + 1] = commentBuffer.join("\n");
        commentBuffer = [];
      }
    } else if (isCollectingComment) {
      if (line.includes("*/")) {
        commentBuffer.push(line.replace(/\*\/.*$/, "").trim());
        isCollectingComment = false;
        lineComments[i + 1] = commentBuffer.join("\n");
        commentBuffer = [];
      } else {
        commentBuffer.push(line.replace(/^\*/, "").trim());
      }
    }

    // Handle single-line comments
    else if (line.startsWith("//")) {
      lineComments[i + 1] = line
        .substring(line.startsWith("///") ? 3 : 2)
        .trim();
    } else {
      // NOTE: If we encounter a non-comment line after collecting comments but before a function,
      // store these comments for the next line
      if (commentBuffer.length > 0) {
        lineComments[i + 1] = commentBuffer.join("\n");
        commentBuffer = [];
      }
    }
  }

  // Second pass: match functions and associate comments
  const functions = extractFunctions(normalizedCode, lineComments);
  const structures = extractStructures(normalizedCode);

  return {
    functions,
    structures,
  };
}

function extractStructures(normalizedCode) {
  let match;
  const structures = [];
  let hasAnnotations = false;

  while ((match = STRUCTURE_PATTERN.exec(normalizedCode)) !== null) {
    const name = match[1];
    const fieldsString = match[2].trim().replaceAll(/\/{1,3}.*/g, "");
    const fields = [...fieldsString.matchAll(TYPE_PATTERN_GLOBAL)]
      .map((match) => {
        const annotation = match?.[1]?.trim() ?? null;

        if (annotation) {
          hasAnnotations = true;
        }

        const type = match[3].trim();

        return {
          annotation,
          name: match[2].trim(),
          type,
          typeLink: wgpuTypes?.[type.split("<")[0]] ?? null,
        };
      })
      .filter(Boolean);

    structures.push({
      hasAnnotations,
      name,
      fields,
    });
  }
  return structures;
}

function extractFunctions(normalizedCode, lineComments) {
  const functions = [];
  let lastFunctionLine = -1;
  let fullCode = normalizedCode;
  let match;

  while ((match = FUNCTION_PATTERN.exec(fullCode)) !== null) {
    const signature = match[0].replace("{", "").trim();

    const stageAttribute =
      signature.match(/@(vertex|fragment|compute)/)?.[1] ?? null;
    const matchNameAndParams = signature.match(
      signature.includes("->")
        ? /fn\s+(\w+)\(([\s\S]+)?\)\s+->/
        : /fn\s+(\w+)\(([\s\S]+)?\).*/,
    );
    const name = matchNameAndParams[1];
    let rawParams = matchNameAndParams?.[2];

    const defsMatches = rawParams
      ? [...rawParams.matchAll(/#ifdef\s+(\w+)/g)]
      : [];
    const defs = defsMatches.map((match) => match[1]);
    rawParams = rawParams
      ?.replaceAll(/#ifdef\s+\w+/g, "")
      ?.replaceAll("#endif", "")
      ?.replaceAll("#else", "")
      ?.trim();

    const params =
      [...(rawParams ?? "").matchAll(TYPE_PATTERN_GLOBAL)]?.map((match) => {
        return {
          attr: match?.[1] ?? null,
          name: match[2],
          type: match[3],
          typeLink: wgpuTypes?.[match[3].split("<")[0]] ?? null,
        };
      }) ?? [];
    const returnType = signature.match(/->(.*)/)?.[1]?.trim() ?? "void";

    const positionInCode = match.index;
    const codeBeforeMatch = fullCode.substring(0, positionInCode);
    const lineNumber = codeBeforeMatch.split("\n").length;

    const comments = getFunctionComments(lineNumber, lineComments);

    const returnTypeLink = returnType
      ? (wgpuTypes?.[returnType.split("<")[0]] ?? null)
      : null;

    functions.push({
      stageAttribute,
      name,
      params,
      returnType,
      returnTypeLink,
      comment: comments.join("\n"),
    });

    lastFunctionLine = lineNumber;
  }
  return functions;
}

function getFunctionComments(lineNumber, lineComments) {
  let comments = [];
  let currentLine = lineNumber;

  while (currentLine > 0 && lineComments[currentLine - 1] !== undefined) {
    if (lineComments[currentLine] !== undefined) {
      comments.push(lineComments[currentLine]);
    }
    currentLine--;
  }

  if (lineComments[currentLine] !== undefined) {
    comments.push(lineComments[currentLine]);
  }

  comments.reverse();

  return comments;
}

function generateFunctionDocsHTML(params) {
  return Handlebars.compile(WGSL_DOC_TEMPLATE_SOURCE)(params);
}

function processWGSLFile(wgslFilePath) {
  const wgslCode = fs.readFileSync(wgslFilePath, "utf-8");
  const { functions, structures } = extractWGSLItems(wgslCode);
  const { base: basename, name: filename, dir } = path.parse(wgslFilePath);
  const innerPath = path.relative(source, dir);

  const output = generateFunctionDocsHTML({
    functions,
    structures,
    filename: basename,
  });
  const outputDir = path.join(OUTPUT_DIR_ROOT, innerPath);
  const outputPath = path.join(outputDir, `${filename}.html`);

  fs.mkdirSync(outputDir, { recursive: true });
  fs.writeFileSync(outputPath, output, "utf-8");

  return {
    filename: basename,
    functions,
    structures,
    link: path.join(innerPath, `${filename}.html`),
  };
}

// entrypoint
exec(GREP_WGSL, (error, stdout, stderr) => {
  if (error) {
    console.error(`❌ Error: ${error.message}`);
    return;
  }
  if (stderr) {
    console.error(`⚠️ stderr: ${stderr}`);
    return;
  }

  const filePaths = stdout.trim().split("\n");
  let searchInfo = [];

  for (const filePath of filePaths) {
    try {
      const shaderFunctions = processWGSLFile(filePath);
      const functions = shaderFunctions.functions.map((func) =>
        Object.assign(
          {
            link: shaderFunctions.link.startsWith("/")
              ? shaderFunctions.link
              : "/" + shaderFunctions.link,
            filename: shaderFunctions.filename,
          },
          func,
        ),
      );
      const structures = shaderFunctions.structures.map((struct) =>
        Object.assign(
          {
            link: shaderFunctions.link.startsWith("/")
              ? shaderFunctions.link
              : "/" + shaderFunctions.link,
            filename: shaderFunctions.filename,
          },
          struct,
        ),
      );
      searchInfo = searchInfo.concat(functions);
      searchInfo = searchInfo.concat(structures);
    } catch (error) {
      console.log(`Cannot build for ${filePath}, error: `, error);
    }
  }

  const homeOutput = Handlebars.compile(HOME_DOC_TEMPLATE_SOURCE)({
    files: filePaths.map((v) => ({
      file: path.relative(source, v).replace(".wgsl", ".html"),
    })),
  });

  fs.writeFileSync(path.join(OUTPUT_DIR_ROOT, `index.html`), homeOutput, {
    encoding: "utf-8",
  });

  fs.mkdirSync(path.join(OUTPUT_DIR_ROOT, "public"), { recursive: true });
  fs.writeFileSync(
    path.join(OUTPUT_DIR_ROOT, "public", "search-info.json"),
    JSON.stringify(searchInfo, null, 2),
    "utf-8",
  );

  const copyToPublic = [
    "styles.css",
    "favicon.ico",
    "search.js",
    "wgsl.png",
    "github.png",
    "templates/search-result.hbs",
  ];

  for (const file of copyToPublic) {
    fs.copyFileSync(
      file,
      path.join(OUTPUT_DIR_ROOT, "public", path.basename(file)),
    );
  }

  fs.copyFileSync("./serve.json", path.join(OUTPUT_DIR_ROOT, "serve.json"));
});
