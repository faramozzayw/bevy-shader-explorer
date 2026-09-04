package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"main/config"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "wgsl-docs",
		Short: "Generate documentation for WGSL shaders",
	}
	root.AddCommand(newGenerateCommand())
	return root
}

func newGenerateCommand() *cobra.Command {
	var (
		project   string
		output    string
		format    string
		exclude   []string
		version   string
		sourceURL string
		sourceRef string
		noDeps    bool
		offline   bool
	)

	command := &cobra.Command{
		Use:   "generate",
		Short: "Generate shader documentation",
		RunE: func(command *cobra.Command, _ []string) error {
			if format != "html" {
				return fmt.Errorf("unsupported format %q (only html is currently supported)", format)
			}
			cfg, err := config.Load(project)
			if err != nil {
				return err
			}
			if command.Flags().Changed("project") {
				cfg.SourcePath = project
			}
			if command.Flags().Changed("output") {
				cfg.OutputDir = output
			}
			if command.Flags().Changed("exclude") {
				cfg.Exclude = exclude
			}
			if command.Flags().Changed("no-deps") {
				cfg.NoDeps = noDeps
			}
			if command.Flags().Changed("offline") {
				cfg.Offline = offline
			}
			if command.Flags().Changed("source-url") {
				cfg.SourceGithubURL = sourceURL
			}
			if command.Flags().Changed("source-ref") {
				cfg.SourceGithubRef = sourceRef
			}
			cfg.Version = version
			generate(cfg)
			return nil
		},
	}
	command.Flags().StringVar(&project, "project", ".", "project directory to scan")
	command.Flags().StringVar(&output, "output", "./shader-docs", "documentation output directory")
	command.Flags().StringVar(&format, "format", "html", "documentation format")
	command.Flags().StringArrayVar(&exclude, "exclude", nil, "directory or pattern to exclude (repeatable)")
	command.Flags().StringVar(&version, "version", "project", "documentation version label")
	command.Flags().StringVar(&sourceURL, "source-url", "", "base URL for source links")
	command.Flags().StringVar(&sourceRef, "source-ref", "", "source branch, tag, or commit for source links")
	command.Flags().BoolVar(&noDeps, "no-deps", false, "disable dependency shader discovery")
	command.Flags().BoolVar(&offline, "offline", false, "use Cargo's offline metadata mode")
	return command
}
