package main

import (
	"flag"
	"fmt"
	"log"
)

type Config struct {
	SourcePath      string
	FileFilter      string
	OutputDir       string
	SourceGithubURL string
	Version         string
}

func GetConfig() Config {
	sourcePath := flag.String("source", "", "Source file path")
	fileFilter := flag.String("filter", "*.wgsl", "Source file filter")
	outputDir := flag.String("outputDir", "./dist", "Output directory")
	sourceGithubURL := flag.String("sourceGithubURL", "https://github.com/bevyengine/bevy/tree/release-0.15.0/", "sourceGithubURL")
	version := flag.String("version", "0.15.0", "version")

	flag.Parse()

	if *sourcePath == "" {
		log.Fatal("Error: 'source' is a required argument")
	}

	config := Config{
		SourcePath:      *sourcePath,
		FileFilter:      *fileFilter,
		OutputDir:       *outputDir,
		SourceGithubURL: *sourceGithubURL,
		Version:         *version,
	}

	fmt.Println("🚀 Starting WGSL Documentation Generator")
	fmt.Println("========================================")
	fmt.Printf("📂 Source Directory     : %s\n", config.SourcePath)
	fmt.Printf("🔍 File Filter Pattern  : %s\n", config.FileFilter)
	fmt.Printf("📁 Output Directory     : %s\n", config.OutputDir)
	fmt.Printf("🌐 GitHub Source URL    : %s\n", config.SourceGithubURL)
	fmt.Printf("🏷️ Documentation Version: %s\n", config.Version)
	fmt.Println("========================================")

	return config
}
