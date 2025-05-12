package wgsl

import "regexp"

var structurePattern = regexp.MustCompile(`struct\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{([^}]*)\}`)
var constPattern = regexp.MustCompile(`const\s+(\w+)\s{0,}(?::\s{0,}(.*))?=\s+(.*);`)
var namedTypeStringPattern = regexp.MustCompile(`^(?:@([^\s]+)\s+)?([a-zA-Z_]\w*):(.+)$`)
var typeStringPattern = regexp.MustCompile(`^(?:@([^\s]+)\s+)?(.+)$`)

var functionPattern = regexp.MustCompile(`(@.*\s+)?(@.*\s+)?\bfn\b\s+([a-zA-Z0-9_]+)[\s\S]*?\{`)
var functionSigWithReturnTypePattern = regexp.MustCompile(`\bfn\b\s+(\w+)\(([\s\S]+)?\)\s+->`)
var functionSigWithoutReturnTypePattern = regexp.MustCompile(`\bfn\b\s+(\w+)\(([\s\S]+)?\).*`)
var workgroupSizePattern = regexp.MustCompile(`@workgroup_size\((.*)\)`)

var bindingPattern = regexp.MustCompile(`@group\((\d+)\)\s{0,}@binding\((\d+)\)\s{0,}var\s{0,}(?:<(.*?)>)?\s{0,}(\w+):\s{0,}(.*);`)
var shaderStagePattern = regexp.MustCompile(`@(vertex|fragment|compute)`)
var vecPattern = regexp.MustCompile(`(vec\d(?:<.*>))`)
var annotationPattern = regexp.MustCompile(`(?:@([^\s]+)\((.*?)\)){1,}`)
