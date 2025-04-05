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

const GREP_WGSL = `grep -rl --include="*.wgsl" .`;

const WGSL_DOC_TEMPLATE_SOURCE = fs.readFileSync(
  "./templates/wgsl-doc.hbs",
  "utf-8",
);
const HOME_DOC_TEMPLATE_SOURCE = fs.readFileSync(
  "./templates/home.hbs",
  "utf-8",
);

const FUNCTION_PATTERN =
  /(@[^;]*\s+)?(vertex|fragment|compute\s+)?fn\s+([a-zA-Z0-9_]+)\s*\(([^)]*)\)(\s*->\s*([^{]*))?/g;
const STRUCTURE_PATTERN = /struct\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{([^}]*)\}/g;
const ANNOTATION_PATTERN =
  /(@[a-zA-Z0-9\(\)\-_]+)?\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:\s*([a-zA-Z0-9\[\]<>,]*)/;

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
    const fieldsString = match[2].trim();
    const fields = fieldsString
      .split(",")
      .map((field) => field.trim())
      .filter(Boolean)
      .map((field) => {
        const annotationMatch = field.match(ANNOTATION_PATTERN);

        if (annotationMatch) {
          const annotation = annotationMatch[1] || "";
          const name = annotationMatch[2];
          const type = annotationMatch[3];
          const typeLink = wgpuTypes?.[type.split("<")[0]] ?? null;

          if (annotation) {
            hasAnnotations = true;
          }

          return { annotation, name, type, typeLink };
        }
        return null;
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
    const attributes = match[1] ? match[1].trim() : "";
    const name = match[3];
    const params = match[4]
      .split(",")
      .map((param) => param.trim())
      .filter((p) => p);
    const returnType = match[6] ? match[6].trim() : "void";

    const positionInCode = match.index;
    const codeBeforeMatch = fullCode.substring(0, positionInCode);
    const lineNumber = codeBeforeMatch.split("\n").length;

    const comments = getFunctionComments(lineNumber, lineComments);
    const formattedParams = getFunctionParams(params);

    const returnTypeLink = wgpuTypes?.[returnType.split("<")[0]] ?? null;

    functions.push({
      attributes,
      name,
      params: formattedParams,
      returnType,
      returnTypeLink,
      comment: comments.join("\n"),
    });

    lastFunctionLine = lineNumber;
  }
  return functions;
}

function getFunctionParams(params) {
  return params.map((param) => {
    const parts = param.split(":");

    const type = parts[1].trim();
    const maybeGenericType = type.split("<")[0];

    return {
      name: parts[0].trim(),
      type,
      typeLink: wgpuTypes?.[maybeGenericType] ?? null,
    };
  });
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
  const fileInfo = path.parse(wgslFilePath);
  const filename = fileInfo.base;

  const innerPath = fileInfo.dir.replace("wgsls", "").replace("wgsls/", "");

  const output = generateFunctionDocsHTML({
    functions,
    structures,
    filename,
  });
  const outputDir = path.join(OUTPUT_DIR_ROOT, innerPath);
  const outputPath = path.join(outputDir, `${fileInfo.name}.html`);

  fs.mkdirSync(outputDir, { recursive: true });
  fs.writeFileSync(outputPath, output, "utf-8");

  return {
    filename,
    functions,
    structures,
    link: path.join(innerPath, `${fileInfo.name}.html`),
  };
}

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
  }

  const homeOutput = Handlebars.compile(HOME_DOC_TEMPLATE_SOURCE)({
    files: filePaths.map((v) => ({
      file: v.split("wgsls/").at(-1).replace(".wgsl", ".html"),
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
    "./templates/search-result.hbs",
  ];

  for (const file of copyToPublic) {
    fs.copyFileSync(
      file,
      path.join(OUTPUT_DIR_ROOT, "public", path.basename(file)),
    );
  }

  fs.copyFileSync("./serve.json", path.join(OUTPUT_DIR_ROOT, "serve.json"));
});
