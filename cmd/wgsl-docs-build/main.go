package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/faramozzayw/bevy-shader-explorer/config"
	"github.com/faramozzayw/bevy-shader-explorer/generation"
)

type versionRef struct {
	Version string `toml:"version"`
	Ref     string `toml:"ref"`
}

type source struct {
	Name       string       `toml:"name"`
	Repo       string       `toml:"repo"`
	Root       string       `toml:"root"`
	RefPattern string       `toml:"ref_pattern"`
	Versions   []string     `toml:"versions"`
	Releases   []versionRef `toml:"releases"`
}

type release struct {
	Name    string
	Repo    string
	Dir     string
	Version string
	Ref     string
}

type buildConfig struct {
	SiteURL  string   `toml:"site_url"`
	IssueURL string   `toml:"issue_url"`
	Sources  []source `toml:"sources"`
}

func loadSiteURL(root, path string) (string, error) {
	config, err := loadBuildConfig(root, path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(config.SiteURL, "/"), nil
}

func loadIssueURL(root, path string) (string, error) {
	config, err := loadBuildConfig(root, path)
	if err != nil {
		return "", err
	}
	return config.IssueURL, nil
}

func loadSources(root, path string) ([]source, error) {
	config, err := loadBuildConfig(root, path)
	if err != nil {
		return nil, err
	}
	if len(config.Sources) == 0 {
		return nil, fmt.Errorf("build config contains no sources")
	}
	return config.Sources, nil
}

func loadBuildConfig(root, path string) (buildConfig, error) {
	var config buildConfig
	if _, err := toml.DecodeFile(filepath.Join(root, path), &config); err != nil {
		return buildConfig{}, fmt.Errorf("load build config: %w", err)
	}
	return config, nil
}

func allReleases(sources []source) []release {
	var result []release
	for _, project := range sources {
		versions := make([]versionRef, 0, len(project.Versions)+len(project.Releases))
		indices := make(map[string]int, len(project.Versions))
		for _, version := range project.Versions {
			indices[version] = len(versions)
			versions = append(versions, versionRef{Version: version})
		}
		for _, version := range project.Releases {
			if index, ok := indices[version.Version]; ok {
				versions[index] = version
				continue
			}
			indices[version.Version] = len(versions)
			versions = append(versions, version)
		}
		for _, version := range versions {
			dir := project.Root
			if version.Version != "project" {
				dir = filepath.Join(dir, version.Version)
			}
			ref := version.Ref
			if ref == "" {
				ref = strings.ReplaceAll(project.RefPattern, "{version}", version.Version)
			}
			result = append(result, release{Name: project.Name, Repo: project.Repo, Dir: dir, Version: version.Version, Ref: ref})
		}
	}
	return result
}

func main() {
	flags := flag.NewFlagSet("wgsl-docs-build", flag.ExitOnError)
	configPath := flags.String("config", "shader-sources.toml", "build matrix configuration")
	if err := flags.Parse(os.Args[1:]); err != nil || flags.NArg() != 1 || (flags.Arg(0) != "clone" && flags.Arg(0) != "generate") {
		fmt.Fprintln(os.Stderr, "usage: wgsl-docs-build [--config path] <clone|generate>")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	buildConfig, err := loadBuildConfig(root, *configPath)
	if err != nil {
		fatal(err)
	}
	sources := buildConfig.Sources
	if len(sources) == 0 {
		fatal(fmt.Errorf("build config contains no sources"))
	}
	if flags.Arg(0) == "clone" {
		cloneAll(root, sources)
		return
	}
	generateAll(root, sources, strings.TrimRight(buildConfig.SiteURL, "/"), buildConfig.IssueURL)
}

func cloneAll(root string, sources []source) {
	releases := allReleases(sources)
	jobs := runtime.NumCPU()
	if jobs > 8 {
		jobs = 8
	}
	queue := make(chan release)
	var wg sync.WaitGroup
	for i := 0; i < jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range queue {
				if _, err := os.Stat(filepath.Join(root, item.Dir)); err == nil {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(filepath.Join(root, item.Dir)), 0o755); err != nil {
					fatal(err)
				}
				fmt.Printf("Cloning %s %s (%s)\n", item.Name, filepath.Base(item.Dir), item.Ref)
				cmd := exec.Command("git", "clone", "--branch", item.Ref, "--depth=1", item.Repo, filepath.Join(root, item.Dir))
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				if err := cmd.Run(); err != nil {
					fatal(err)
				}
				if err := os.RemoveAll(filepath.Join(root, item.Dir, ".git")); err != nil {
					fatal(err)
				}
			}
		}()
	}
	for _, item := range releases {
		queue <- item
	}
	close(queue)
	wg.Wait()
}

func generateAll(root string, sources []source, siteURL, issueURL string) {
	releases := allReleases(sources)
	for _, item := range releases {
		projectPath := filepath.Join(root, item.Dir)
		cfg, err := config.Load(projectPath)
		if err != nil {
			fatal(err)
		}
		// Match the generate CLI's explicit --project and --output overrides.
		cfg.SourcePath = projectPath
		cfg.SourceGithubRoot = projectPath
		cfg.OutputDir = filepath.Join(root, "dist")
		cfg.SourceGithubRef = item.Ref
		cfg.Version = item.Version
		cfg.SkipCatalogue = true
		if siteURL != "" {
			cfg.SiteURL = siteURL
		}
		if issueURL != "" {
			cfg.IssueURL = issueURL
		}
		if err := generation.Generate(cfg); err != nil {
			fatal(err)
		}
	}
	cfg, err := config.Load(root)
	if err != nil {
		fatal(err)
	}
	cfg.OutputDir = filepath.Join(root, "dist")
	if siteURL != "" {
		cfg.SiteURL = siteURL
	}
	if issueURL != "" {
		cfg.IssueURL = issueURL
	}
	if err := generation.Finalize(cfg); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
