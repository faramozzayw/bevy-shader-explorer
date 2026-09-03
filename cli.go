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
	)

	command := &cobra.Command{
		Use:   "generate",
		Short: "Generate shader documentation",
		RunE: func(_ *cobra.Command, _ []string) error {
			if format != "html" {
				return fmt.Errorf("unsupported format %q (only html is currently supported)", format)
			}
			generate(config.Config{
				SourcePath:      project,
				FileFilter:      "*.wgsl",
				OutputDir:       output,
				SourceGithubURL: sourceURL,
				Version:         version,
				Exclude:         exclude,
			})
			return nil
		},
	}
	command.Flags().StringVar(&project, "project", ".", "project directory to scan")
	command.Flags().StringVar(&output, "output", "./shader-docs", "documentation output directory")
	command.Flags().StringVar(&format, "format", "html", "documentation format")
	command.Flags().StringArrayVar(&exclude, "exclude", nil, "directory or pattern to exclude (repeatable)")
	command.Flags().StringVar(&version, "version", "project", "documentation version label")
	command.Flags().StringVar(&sourceURL, "source-url", "", "base URL for source links")
	return command
}
