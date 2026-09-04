# Shader Explorer

Shader Explorer generates docs.rs-style, searchable HTML documentation for WGSL shaders. It parses shader declarations, links source locations, and can include shader-bearing Bevy dependencies discovered through Cargo.

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
