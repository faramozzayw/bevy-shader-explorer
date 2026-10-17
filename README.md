# Shader Explorer

Shader Explorer generates docs.rs-style, searchable HTML documentation for
WGSL and WESL shaders. It parses declarations, links source locations, and can
include shader-bearing Bevy dependencies discovered through Cargo.

## Quick start

Generate docs for a project:

```bash
go run . generate --project . --output ./dist
```

Serve the generated site locally:

```bash
just serve
```

The optional `wgsl-docs.toml` file configures project metadata, output paths,
exclusions, and dependency discovery. Without it, name and description are
read from `Cargo.toml` when available.

## Documentation

See the [user and generation guide](docs/guide.md) for:

- CLI flags and catalogue workflows
- search and SEO behavior
- used-by relationships
- the generation pipeline and documentation model
- adding libraries to the catalogue

## License

Bevy Shader Explorer is dual-licensed under the [MIT License](LICENSE-MIT) or
the [Apache License 2.0](LICENSE-APACHE), at your option.

Generated documentation may include WGSL files from third-party projects;
those files remain under their original licenses.
