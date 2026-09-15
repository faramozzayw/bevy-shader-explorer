package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorProducesCompleteFixtureSite(t *testing.T) {
	project := t.TempDir()
	shaderDir := filepath.Join(project, "shaders")
	if err := os.MkdirAll(shaderDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "Cargo.toml"), []byte(`[package]
name = "fixture"
version = "0.1.0"
description = "Generator fixture"
repository = "https://github.com/example/fixture"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(shaderDir, "simple.wgsl"), `struct Vertex {
    position: vec4<f32>,
}

@vertex
fn vertex_main(@location(0) position: vec3<f32>) -> @builtin(position) vec4<f32> {
    return vec4<f32>(position, 1.0);
}
	`)

	output := filepath.Join(t.TempDir(), "dist")
	root := repoRoot(t)
	runGenerator(t, root, project, output, "--no-deps", "--source-ref", "release-test")

	packagePage := filepath.Join(output, "fixture", "0.1.0", "index.html")
	shaderPage := filepath.Join(output, "fixture", "0.1.0", "shaders", "simple.html")
	for _, path := range []string{packagePage, shaderPage, filepath.Join(output, "public", "search-info-0.1.0.json"), filepath.Join(output, "public", "package-versions.json")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %s: %v", path, err)
		}
	}
	page, err := os.ReadFile(packagePage)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "simple") || !strings.Contains(string(page), "Generator fixture") {
		t.Fatalf("package page is missing fixture content")
	}
	shader, err := os.ReadFile(shaderPage)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(shader), "github.com/example/fixture/blob/release-test/shaders/simple.wgsl") {
		t.Fatalf("shader page is missing versioned source link")
	}
	search, err := os.ReadFile(filepath.Join(output, "public", "search-info-0.1.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(search), `"name": "vertex_main"`) || !strings.Contains(string(search), `"packageName": "fixture"`) {
		t.Fatalf("search index is missing fixture declarations")
	}
}

func TestGeneratorDisambiguatesWGSLAndWESLNames(t *testing.T) {
	project := t.TempDir()
	shaderDir := filepath.Join(project, "shaders")
	if err := os.MkdirAll(shaderDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(project, "Cargo.toml"), `[package]
name = "collision-fixture"
version = "0.1.0"
`)
	writeFixtureFile(t, filepath.Join(shaderDir, "shared.wgsl"), "const WGSL_VALUE: f32 = 1.0;\n")
	writeFixtureFile(t, filepath.Join(shaderDir, "shared.wesl"), "const WESL_VALUE: f32 = 2.0;\n")

	output := filepath.Join(t.TempDir(), "dist")
	runGenerator(t, repoRoot(t), project, output, "--no-deps")
	for _, name := range []string{"shared.wgsl.html", "shared.wesl.html"} {
		if _, err := os.Stat(filepath.Join(output, "collision-fixture", "0.1.0", "shaders", name)); err != nil {
			t.Fatalf("expected collision-disambiguated page %s: %v", name, err)
		}
	}
}

func TestGeneratorJSONMirrorsPageModel(t *testing.T) {
	project := t.TempDir()
	writeFixtureFile(t, filepath.Join(project, "Cargo.toml"), `[package]
name = "json-fixture"
version = "0.1.0"
description = "JSON fixture"
repository = "https://github.com/example/json-fixture"
`)
	writeFixtureFile(t, filepath.Join(project, "src", "lib.rs"), "pub fn fixture() {}\n")
	writeFixtureFile(t, filepath.Join(project, "shaders", "page.wgsl"), `const JSON_VALUE: f32 = 1.0;
@group(0) @binding(0) var page_texture: texture_2d<f32>;
fn page_main() {}
`)
	output := filepath.Join(t.TempDir(), "json")
	runGenerator(t, repoRoot(t), project, output, "--format", "json")

	data, err := os.ReadFile(filepath.Join(output, "documentation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Home struct {
			ShaderCount int `json:"shaderCount"`
			Sections    []struct {
				Groups []struct {
					PackageName string `json:"packageName"`
				} `json:"groups"`
			} `json:"sections"`
		} `json:"home"`
		Packages []struct {
			Name     string `json:"name"`
			Metadata struct {
				Repository string `json:"repository"`
			} `json:"metadata"`
		} `json:"packages"`
		ShaderPages []struct {
			Items struct {
				Constants []struct {
					Name string `json:"name"`
				} `json:"constants"`
				Bindings []struct {
					Name string `json:"name"`
				} `json:"bindings"`
				Functions []struct {
					Name string `json:"name"`
				} `json:"functions"`
			} `json:"items"`
		} `json:"shaderPages"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("invalid documentation JSON: %v", err)
	}
	if document.Home.ShaderCount != 1 || len(document.Home.Sections) != 1 || len(document.Home.Sections[0].Groups) != 1 || document.Home.Sections[0].Groups[0].PackageName != "json-fixture" {
		t.Fatalf("JSON omitted home page model: %+v", document.Home)
	}
	if len(document.Packages) != 1 || document.Packages[0].Metadata.Repository != "https://github.com/example/json-fixture" {
		t.Fatalf("JSON omitted package page model: %+v", document.Packages)
	}
	if len(document.ShaderPages) != 1 || len(document.ShaderPages[0].Items.Constants) != 1 || len(document.ShaderPages[0].Items.Bindings) != 1 || len(document.ShaderPages[0].Items.Functions) != 1 {
		t.Fatalf("JSON omitted shader page items: %+v", document.ShaderPages)
	}
	if _, err := os.Stat(filepath.Join(output, "json-fixture", "0.1.0", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("JSON output should not render HTML pages")
	}
	htmlOutput := filepath.Join(t.TempDir(), "html")
	runGenerator(t, repoRoot(t), project, htmlOutput, "--no-deps")
	for _, page := range []string{
		filepath.Join(htmlOutput, "json-fixture", "0.1.0", "index.html"),
		filepath.Join(htmlOutput, "json-fixture", "0.1.0", "shaders", "page.html"),
	} {
		if _, err := os.Stat(page); err != nil {
			t.Fatalf("HTML renderer omitted page represented in JSON: %s: %v", page, err)
		}
	}
}

func TestGeneratorExcludesWorkspaceRootPackage(t *testing.T) {
	project := t.TempDir()
	writeFixtureFile(t, filepath.Join(project, "Cargo.toml"), `[package]
name = "workspace-root"
version = "0.1.0"

[workspace]
members = ["crates/member"]
`)
	writeFixtureFile(t, filepath.Join(project, "src", "lib.rs"), "pub fn root() {}\n")
	writeFixtureFile(t, filepath.Join(project, "root.wgsl"), "const ROOT_VALUE: f32 = 1.0;\n")
	memberDir := filepath.Join(project, "crates", "member")
	writeFixtureFile(t, filepath.Join(memberDir, "Cargo.toml"), `[package]
name = "member"
version = "0.1.0"
`)
	writeFixtureFile(t, filepath.Join(memberDir, "src", "lib.rs"), "pub fn member() {}\n")
	writeFixtureFile(t, filepath.Join(memberDir, "member.wgsl"), "const MEMBER_VALUE: f32 = 2.0;\n")

	output := filepath.Join(t.TempDir(), "dist")
	runGenerator(t, repoRoot(t), project, output)
	if _, err := os.Stat(filepath.Join(output, "workspace-root", "0.1.0")); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(output)
		t.Fatalf("workspace root package should not be generated (output=%s entries=%v err=%v)", output, entries, err)
	}
	if _, err := os.Stat(filepath.Join(output, "member", "0.1.0", "index.html")); err != nil {
		t.Fatalf("workspace member package was not generated: %v", err)
	}
}

func runGenerator(t *testing.T, root, project, output string, extra ...string) {
	t.Helper()
	args := []string{"run", ".", "generate", "--project", project, "--output", output, "--version", "0.1.0"}
	args = append(args, extra...)
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generator failed: %v\n%s", err, result)
	}
}

func writeFixtureFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(root)
}
