# GPUWeb WGSL grammar

This directory contains the generated C parser and scanner for the official
[GPUWeb WGSL Tree-sitter grammar](https://github.com/gpuweb/tree-sitter-wgsl).

- Upstream revision: `52e3c620a9c316cc8c1d504dd1908eb8cebe255b`
- Generator: `tree-sitter-cli@0.25.9`

`grammar.js` and `src/scanner.c` come from that upstream revision. `src/parser.c`
is generated from them. The Go binding is local because the upstream project
does not publish one.

To update the parser, replace `grammar.js` and `src/scanner.c` from a chosen
upstream revision, then run `./generate.sh` from this directory. Do not edit
`src/parser.c` manually.
