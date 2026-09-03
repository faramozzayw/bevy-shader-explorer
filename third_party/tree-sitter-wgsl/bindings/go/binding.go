// Package tree_sitter_wgsl exposes the parser generated from GPUWeb's
// official WGSL grammar.
package tree_sitter_wgsl

// #cgo CFLAGS: -std=c11 -fPIC -I${SRCDIR}/../../src
// #include "../../src/parser.c"
// #include "../../src/scanner.c"
import "C"

import "unsafe"

// Language returns the Tree-sitter language exported by the generated parser.
func Language() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_wgsl())
}
