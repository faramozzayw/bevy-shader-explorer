#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const Handlebars = require("handlebars");

Handlebars.registerHelper("eq", function (a, b) {
  return a === b;
});

Handlebars.registerHelper("neq", function (a, b) {
  return a !== b;
});

const TEMPLATE_SOURCE = fs.readFileSync("template.hbs", "utf-8");
const FUNCTION_PATTERN =
  /(@[^;]*\s+)?(vertex|fragment|compute\s+)?fn\s+([a-zA-Z0-9_]+)\s*\(([^)]*)\)(\s*->\s*([^{]*))?/g;

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
      if (!lineComments[i + 1]) {
        lineComments[i + 1] = line.substring(2).trim();
      } else {
        lineComments[i + 1] += "\n" + line.substring(2).trim();
      }
    } else {
      // If we encounter a non-comment line after collecting comments but before a function,
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
    const visibility = match[2] ? match[2].trim() : "";
    const name = match[3];
    const params = match[4]
      .split(",")
      .map((param) => param.trim())
      .filter((p) => p);
    const returnType = match[6] ? match[6].trim() : "void";

    // Find line number of the function
    const positionInCode = match.index;
    const codeBeforeMatch = fullCode.substring(0, positionInCode);
    const lineNumber = codeBeforeMatch.split("\n").length;

    const comments = getFunctionComments(lineNumber, lineComments);
    const formattedParams = getFunctionParams(params);

    comments.reverse();

    functions.push({
      attributes,
      visibility,
      name,
      params: formattedParams,
      returnType,
      comment: comments.join("\n"),
    });

    lastFunctionLine = lineNumber;
  }

  return functions;
}

function getFunctionParams(params) {
  return params.map((param) => {
    const parts = param.split(":");

    return {
      name: parts[0].trim(),
      type: parts[1].trim(),
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

  fs.writeFileSync(`./outputs/${filename}.html`, output, "utf-8");
}

processWGSLFile("./wgsls/ssr.wgsl");
