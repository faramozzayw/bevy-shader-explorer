#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { exec } = require("node:child_process");
const Handlebars = require("handlebars");

const CURRENT_VERSION = "release-0.15.3";
const bevyUrl = `https://github.com/bevyengine/bevy/tree/${CURRENT_VERSION}/`;
const wgpuTypes = JSON.parse(fs.readFileSync("./wgpu-types.json", "utf-8"));

Handlebars.registerHelper("eq", (a, b) => a === b);
Handlebars.registerHelper("neq", (a, b) => a !== b);
Handlebars.registerHelper("anyShaderDefs", (input) => {
  return input.some((v) => v?.hasShaderDefs);
});
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
Handlebars.registerPartial(
  "shader-defs-list",
  fs.readFileSync("./templates/partials/shader-defs-list.hbs", "utf-8"),
);
Handlebars.registerPartial(
  "type",
  fs.readFileSync("./templates/partials/type.hbs", "utf-8"),
);
Handlebars.registerPartial(
  "gh-link",
  fs.readFileSync("./templates/partials/gh-link.hbs", "utf-8"),
);

const { source } = require("minimist")(process.argv.slice(2));
const FIND_WGSL = `find ${source} -type f  -name "*.wgsl"`;

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
const CONST_PATTERN = /const\s+(\w+)\s{0,}(?::\s{0,}(.*))?=\s+(.*);/g;
const TYPES_STRING_PATTERN = /^(?:@([^\s]+)\s+)?([a-zA-Z_]\w*):(.+)$/;
const FUNCTION_SIG_WITH_RET_TYPE_PATTERN = /\bfn\b\s+(\w+)\(([\s\S]+)?\)\s+->/;
const FUNCTION_SIG_WITHOUT_RET_TYPE_PATTERN = /\bfn\b\s+(\w+)\(([\s\S]+)?\).*/;
// @see https://regex101.com/r/grsB5b/1
const BINDING_PATTERN =
  /@group\((\d+)\)\s{0,}@binding\((\d+)\)\s{0,}var\s{0,}(?:(<.*?>))?\s{0,}(\w+):\s{0,}(.*);/g;

const OUTPUT_DIR_ROOT = "./dist";
const PUBLIC_FOLDER = path.join(OUTPUT_DIR_ROOT, "public");

/**
 *@param {string} code
 */
function extractShaderDefsBlocks(code) {
  const lines = code.split("\n");
  const blocks = [];
  const stack = [];

  lines.forEach((line, index) => {
    line = line.trim();
    const lineNum = index + 1;

    if (line.includes("#ifdef")) {
      stack.push({
        defName: line.slice(6).trim(),
        ifdefLine: lineNum,
        elseLine: null,
        endifLine: null,
      });
    } else if (line.includes("#else")) {
      const current = stack[stack.length - 1];
      if (current && current.elseLine === null) {
        current.elseLine = lineNum;
      }
    } else if (line.includes("#endif")) {
      const current = stack.pop();
      if (!current) {
        return;
      }

      blocks.push({
        ...current,
        endifLine: lineNum,
      });
    }
  });

  blocks.sort((a, b) => a.ifdefLine - b.ifdefLine);

  return blocks;
}

function getShaderDefsByLine(shaderDefs, lineNumber) {
  const results = [];

  for (const shaderDef of shaderDefs) {
    if (
      lineNumber > shaderDef.ifdefLine &&
      lineNumber < (shaderDef.elseLine || shaderDef.endifLine)
    ) {
      results.push({
        defName: shaderDef.defName,
        branch: "if",
        lineNumber: shaderDef.ifdefLine,
      });
    }

    if (
      shaderDef.elseLine &&
      lineNumber > shaderDef.elseLine &&
      lineNumber < shaderDef.endifLine
    ) {
      results.push({
        defName: shaderDef.defName,
        branch: "else",
        lineNumber: shaderDef.elseLine,
      });
    }
  }

  return results;
}

/**
 * @param {string} wgslCode
 */
function extractWGSLItems(wgslCode) {
  const normalizedCode = wgslCode.replace(/\r\n/g, "\n");
  const lines = normalizedCode.split("\n");

  const lineComments = extractComments(lines);

  const importPath =
    normalizedCode.match(/#define_import_path\s+(.*)/)?.[1] ?? null;
  const shaderDefs = extractShaderDefsBlocks(normalizedCode);

  return {
    consts: extractConsts(normalizedCode, lineComments, shaderDefs),
    bindings: extractBindings(normalizedCode, lineComments, shaderDefs),
    functions: extractFunctions(normalizedCode, lineComments, shaderDefs),
    structures: extractStructures(normalizedCode, lineComments, shaderDefs),
    importPath,
  };
}

function extractComments(lines) {
  const lineComments = {};
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
  return lineComments;
}

/**
  @param str {?string}
 */
function removePath(str) {
  return str?.trim()?.split("::")?.at(-1) ?? "";
}

/**
 * @param str {string}
 */
function splitParams(str) {
  if (!str) return [];

  const parts = [];
  let current = "";
  let depth = 0;

  for (let i = 0; i < str.length; i++) {
    const char = str[i];

    if (char === "<") depth++;
    else if (char === ">") depth--;
    else if (char === "," && depth === 0) {
      parts.push(current.trim());
      current = "";
      continue;
    }

    current += char;
  }

  if (current.trim()) parts.push(current.trim());
  return parts;
}

function getLineNumber(fullCode, matchIndex) {
  const codeBeforeMatch = fullCode.substring(0, matchIndex);
  return codeBeforeMatch.split("\n").length;
}

function getTypeLink(type) {
  return wgpuTypes?.[type.trim().split("<")[0]] ?? null;
}

function normalizeLink(link) {
  return link.startsWith("/") ? link : "/" + link;
}

/**
 * @param str {string}
 * @param fullCode {string}
 */
function parseTypesString(str, fullCode, shaderDefs) {
  if (!str) return [];

  str = str
    ?.replaceAll(/#ifdef\s+\w+/g, "")
    ?.replaceAll("#endif", "")
    ?.replaceAll("#else", "")
    ?.replaceAll(/\n/g, "")
    ?.trim();

  const entries = splitParams(str);
  const result = [];

  for (const entry of entries) {
    const match = entry.match(TYPES_STRING_PATTERN);
    if (!match) continue;
    let [, annotation, name, type] = match;
    type = removePath(type);

    const regex = new RegExp(`\\b${name}\\s{0,}:`);
    const lineNumber = getLineNumber(fullCode, fullCode.match(regex)?.index);
    const thisShaderDefs = getShaderDefsByLine(shaderDefs, lineNumber);

    result.push({
      annotation: annotation ?? null,
      name,
      type: type,
      hasShaderDefs: !!thisShaderDefs.length,
      shaderDefs: thisShaderDefs,
      typeLink: getTypeLink(type),
    });
  }

  return result;
}

/**
 * @param normalizedCode {string}
 */
function extractBindings(normalizedCode, lineComments, shaderDefs) {
  let fullCode = normalizedCode;

  return [...normalizedCode.matchAll(BINDING_PATTERN)].map((match) => {
    let [_, groupIndex, bindingIndex, bindingType, name, type] = match;
    const lineNumber = getLineNumber(fullCode, match.index);
    type = removePath(type);

    const thisShaderDefs = getShaderDefsByLine(shaderDefs, lineNumber);

    return {
      lineNumber,
      name,
      groupIndex,
      bindingIndex,
      bindingType,
      type,
      hasShaderDefs: !!thisShaderDefs.length,
      shaderDefs: thisShaderDefs,
      typeLink: getTypeLink(type),
    };
  });
}

/**
 * @param normalizedCode {string}
 */

function extractConsts(normalizedCode, lineComments, shaderDefs) {
  let fullCode = normalizedCode;

  return [...normalizedCode.matchAll(CONST_PATTERN)].map((match) => {
    const lineNumber = getLineNumber(fullCode, match.index);
    let [, name, type, value] = match;

    const thisShaderDefs = getShaderDefsByLine(shaderDefs, lineNumber);

    if (!type) {
      const vecRegex = /(vec\d(?:<.*>))/;
      if (/^\d+\.\d+/.test(value)) {
        type = "AbstractFloat";
      } else if (vecRegex.test(value)) {
        type = value.match(vecRegex)[1];
      } else if (/\d+u$/.test(value)) {
        type = "u32";
      } else if (/\d+$/.test(value)) {
        type = "AbstractInt";
      }
    }
    type = removePath(type);

    return {
      lineNumber,
      name,
      type,
      value,
      hasShaderDefs: !!thisShaderDefs.length,
      shaderDefs: thisShaderDefs,
      typeLink: getTypeLink(type),
    };
  });
}

/**
 * @param normalizedCode {string}
 * @param lineComments {}
 */
function extractStructures(normalizedCode, lineComments, shaderDefs) {
  let match;
  let fullCode = normalizedCode;
  const structures = [];

  while ((match = STRUCTURE_PATTERN.exec(normalizedCode)) !== null) {
    const name = match[1];
    const fieldsString = match[2].trim().replaceAll(/\/{1,3}.*/g, "");
    const fields = parseTypesString(fieldsString, fullCode, shaderDefs);
    const lineNumber = getLineNumber(fullCode, match.index);
    const comments = getItemComments(lineNumber, lineComments);

    const thisShaderDefs = getShaderDefsByLine(shaderDefs, lineNumber);

    structures.push({
      hasAnnotations: fields.some((v) => Boolean(v.annotation)),
      name,
      fields,
      lineNumber,
      hasShaderDefs: !!thisShaderDefs.length,
      shaderDefs: thisShaderDefs,
      comment: comments.join("\n"),
    });
  }

  return structures;
}

/**
 * @param normalizedCode {string}
 */
function extractFunctions(normalizedCode, lineComments, shaderDefs) {
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
        ? FUNCTION_SIG_WITH_RET_TYPE_PATTERN
        : FUNCTION_SIG_WITHOUT_RET_TYPE_PATTERN,
    );
    const name = matchNameAndParams[1];
    let rawParams = matchNameAndParams?.[2];

    const params = parseTypesString(rawParams, fullCode, shaderDefs);
    const returnType = signature.match(/->(.*)/)?.[1]?.trim() ?? "void";
    const lineNumber = getLineNumber(fullCode, match.index);
    const comments = getItemComments(lineNumber, lineComments);
    const thisShaderDefs = getShaderDefsByLine(shaderDefs, lineNumber);

    const returnTypeLink = returnType
      ? (wgpuTypes?.[returnType.split("<")[0]] ?? null)
      : null;

    functions.push({
      stageAttribute,
      name,
      lineNumber,
      params,
      returnType,
      returnTypeLink,
      hasShaderDefs: !!thisShaderDefs.length,
      shaderDefs: thisShaderDefs,
      comment: comments.join("\n"),
    });

    lastFunctionLine = lineNumber;
  }
  return functions;
}

function getItemComments(lineNumber, lineComments) {
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
  const items = extractWGSLItems(wgslCode);
  const { base: basename, name: filename, dir } = path.parse(wgslFilePath);
  const innerPath = path.relative(source, dir);

  const output = generateFunctionDocsHTML({
    ...items,
    githubLink: new URL(path.join(innerPath, basename), bevyUrl).toString(),
    filename: basename,
  });
  const outputDir = path.join(OUTPUT_DIR_ROOT, innerPath);
  const outputPath = path.join(outputDir, `${filename}.html`);

  fs.mkdirSync(outputDir, { recursive: true });
  fs.writeFileSync(outputPath, output, "utf-8");

  return {
    ...items,
    filename: basename,
    link: path.join(innerPath, `${filename}.html`),
  };
}

// entrypoint
exec(FIND_WGSL, (error, stdout, stderr) => {
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
      const {
        importPath,
        filename,
        link: baseLink,
        ...shaderFunctions
      } = processWGSLFile(filePath);
      const common = {
        link: normalizeLink(baseLink),
        filename,
        exportable: !!importPath,
      };

      const functions = shaderFunctions.functions.map(
        ({ name, stageAttribute, comment }) => ({
          ...common,
          name,
          stageAttribute,
          comment,
          type: "function",
        }),
      );
      const structures = shaderFunctions.structures.map(({ name }) => ({
        ...common,
        name,
        type: "struct",
      }));
      const consts = shaderFunctions.consts.map(({ name }) => ({
        ...common,
        name,
        type: "const",
      }));
      const bindings = shaderFunctions.bindings.map(({ name }) => ({
        ...common,
        type: "binding",
        name,
      }));
      searchInfo = searchInfo
        .concat(functions)
        .concat(structures)
        .concat(consts)
        .concat(bindings);
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

  fs.mkdirSync(PUBLIC_FOLDER, { recursive: true });
  fs.writeFileSync(
    path.join(PUBLIC_FOLDER, "search-info.json"),
    JSON.stringify(searchInfo),
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
    fs.copyFileSync(file, path.join(PUBLIC_FOLDER, path.basename(file)));
  }

  fs.copyFileSync("./serve.json", path.join(OUTPUT_DIR_ROOT, "serve.json"));
});
