# Shader Explorer

Shader Explorer generates docs.rs-style, searchable HTML documentation for WGSL and WESL shaders. It parses shader declarations, links source locations, and can include shader-bearing Bevy dependencies discovered through Cargo. The default file filter includes both `.wgsl` and `.wesl` files; custom filters remain opt-in.

## Quick start

Generate docs for a project:

```bash
go run . generate --project . --output ./dist
```

Serve the generated site locally:

```bash
just serve
```

The optional `wgsl-docs.toml` file can override the project name, description, output path, exclusions, and dependency discovery settings. Without it, name and description are read from `Cargo.toml` when available.

## CLI

```text
wgsl-docs generate [flags]
  --project PATH       project directory (default: .)
  --output PATH        output directory (default: ./shader-docs)
  --exclude PATTERN    exclude a directory or pattern (repeatable)
  --no-deps            disable Cargo dependency shader discovery
  --offline            use Cargo metadata without network access
```

For the bundled Bevy catalogue, `just generate-all` clones the configured source revisions into `sources/` and writes the site to `dist/`. `just deploy-prod` deploys the existing `dist/` output without regenerating it.

The source matrix is defined in `shader-sources.toml`. Run `go run ./cmd/wgsl-docs-build --config path/to/matrix.toml clone` or `generate` to use another release matrix.

## Add a library to the catalogue

Add a `[[sources]]` entry to `shader-sources.toml`. The `root` is where
sources are cloned, and `repo` is used for source links in the generated
pages. Use `versions` with `ref_pattern` when releases follow a predictable
tag convention:

```toml
[[sources]]
name = "bevy_hanabi"
repo = "https://github.com/djeedai/bevy_hanabi.git"
root = "sources/hanabi"
ref_pattern = "v{version}"
versions = ["0.15.0", "0.16.0", "0.17.0", "0.18.0", "0.19.0"]
```

For a project with unusual tags, keep the common pattern and override only
the releases that differ:

```toml
[[sources]]
name = "my-shader-library"
repo = "https://github.com/example/my-shader-library.git"
root = "sources/my-shader-library"
ref_pattern = "v{version}"
versions = ["1.2.0", "1.3.0"]

[[sources.releases]]
version = "1.3.0"
ref = "release-1.3"
```

Then fetch and generate the catalogue:

```bash
just clone-all
just generate-all
```

This is the same pattern used for adding third-party Bevy shader projects;
the generated pages include package versions, shader modules, metadata, and
links back to the original repository.

If you would like a library added but do not want to edit the configuration,
open an issue with its repository and release information. I will most likely
prepare and submit the PR for you.

## License

Shader Explorer is dual-licensed under the [MIT License](LICENSE-MIT) or
the [Apache License 2.0](LICENSE-APACHE), at your option.

Generated documentation may include WGSL files from third-party projects;
those files remain under their original licenses.
