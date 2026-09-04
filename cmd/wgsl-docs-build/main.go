package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/BurntSushi/toml"
)

type versionRef struct {
	Version string `toml:"version"`
	Ref     string `toml:"ref"`
}

type source struct {
	Name     string       `toml:"name"`
	Repo     string       `toml:"repo"`
	Root     string       `toml:"root"`
	Releases []versionRef `toml:"releases"`
}

type release struct {
	Name    string
	Repo    string
	Dir     string
	Version string
	Ref     string
}

type buildConfig struct {
	Sources []source `toml:"sources"`
}

func loadSources(root, path string) ([]source, error) {
	var config buildConfig
	if _, err := toml.DecodeFile(filepath.Join(root, path), &config); err != nil {
		return nil, fmt.Errorf("load build config: %w", err)
	}
	if len(config.Sources) == 0 {
		return nil, fmt.Errorf("build config contains no sources")
	}
	return config.Sources, nil
}

func allReleases(sources []source) []release {
	var result []release
	for _, project := range sources {
		for _, version := range project.Releases {
			dir := project.Root
			if version.Version != "project" {
				dir = filepath.Join(dir, version.Version)
			}
			result = append(result, release{Name: project.Name, Repo: project.Repo, Dir: dir, Version: version.Version, Ref: version.Ref})
		}
	}
	return result
}

func main() {
	flags := flag.NewFlagSet("wgsl-docs-build", flag.ExitOnError)
	configPath := flags.String("config", "wgsl-docs-build.toml", "build matrix configuration")
	if err := flags.Parse(os.Args[1:]); err != nil || flags.NArg() != 1 || (flags.Arg(0) != "clone" && flags.Arg(0) != "generate") {
		fmt.Fprintln(os.Stderr, "usage: wgsl-docs-build [--config path] <clone|generate>")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	sources, err := loadSources(root, *configPath)
	if err != nil {
		fatal(err)
	}
	if flags.Arg(0) == "clone" {
		cloneAll(root, sources)
		return
	}
	generateAll(root, sources)
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

func generateAll(root string, sources []source) {
	releases := allReleases(sources)
	generator := filepath.Join(os.TempDir(), "wgsl-docs-generator")
	build := exec.Command("go", "build", "-o", generator, ".")
	build.Dir = root
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		fatal(err)
	}
	for _, item := range releases {
		fmt.Printf("Generating %s %s\n", item.Name, filepath.Base(item.Dir))
		args := []string{"generate", "--project", filepath.Join(root, item.Dir), "--output", filepath.Join(root, "dist"), "--source-ref", item.Ref}
		if item.Version != "project" {
			args = append(args, "--version", item.Version)
		}
		cmd := exec.Command(generator, args...)
		cmd.Dir = root
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fatal(err)
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
