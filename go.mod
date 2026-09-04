module main

go 1.23.1

require (
	github.com/aymerick/raymond v2.0.2+incompatible
	github.com/gomarkdown/markdown v0.0.0-20250311123330-531bef5e742b
	github.com/gpuweb/tree-sitter-wgsl v0.0.0
	github.com/webgpu-tools/tree-sitter-wesl v0.0.0
	github.com/samber/lo v1.49.1
	github.com/schollz/progressbar/v3 v3.18.0
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
	github.com/tree-sitter/go-tree-sitter v0.25.0
)

replace github.com/gpuweb/tree-sitter-wgsl => ./third_party/tree-sitter-wgsl

replace github.com/webgpu-tools/tree-sitter-wesl => ./third_party/tree-sitter-wesl

require (
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
