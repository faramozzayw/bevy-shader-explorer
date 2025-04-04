#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { exec } = require("node:child_process");
const Handlebars = require("handlebars");

const wgpuTypes = JSON.parse(fs.readFileSync("./wgpu-types.json", "utf-8"));

Handlebars.registerHelper("eq", (a, b) => a === b);
Handlebars.registerHelper("neq", (a, b) => a !== b);

const TEMPLATE_SOURCE = fs.readFileSync("template.hbs", "utf-8");
const FUNCTION_PATTERN =
  /(@[^;]*\s+)?(vertex|fragment|compute\s+)?fn\s+([a-zA-Z0-9_]+)\s*\(([^)]*)\)(\s*->\s*([^{]*))?/g;
const OUTPUT_DIR = "./outputs";

function extractWGSLFunctions(wgslCode) {
  const normalizedCode = wgslCode.replace(/\r\n/g, "\n");
  const lines = normalizedCode.split("\n");

  const functions = [];
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

function generateFunctionDocsHTML(functions) {
  return Handlebars.compile(TEMPLATE_SOURCE)({ functions });
}

function processWGSLFile(wgslFilePath) {
  const wgslCode = fs.readFileSync(wgslFilePath, "utf-8");
  const functions = extractWGSLFunctions(wgslCode);
  const output = generateFunctionDocsHTML(functions);
  const filename = path.parse(wgslFilePath).name;

  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  fs.writeFileSync(path.join(OUTPUT_DIR, `${filename}.html`), output, {
    encoding: "utf-8",
  });
}

const GREP_WGSL = `grep -rl --include="*.wgsl" .`;

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

  for (const filePath of filePaths) {
    processWGSLFile(filePath);
  }
});
