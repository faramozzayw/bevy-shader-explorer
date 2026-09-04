package tree_sitter_wesl

// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../src/parser.c"
// #if __has_include("../../src/scanner.c")
// #define kNumXIDContinueRanges wesl_kNumXIDContinueRanges
// #define kNumXIDStartRanges wesl_kNumXIDStartRanges
// #include "../../src/scanner.c"
// #undef kNumXIDContinueRanges
// #undef kNumXIDStartRanges
// #endif
import "C"

import "unsafe"

// Get the tree-sitter Language for this grammar.
func Language() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_wesl())
}
