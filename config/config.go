package config

type Config struct {
	SourcePath           string
	FileFilter           string
	OutputDir            string
	SourceGithubURL      string
	Version              string
	Exclude              []string
	NoDeps               bool
	Offline              bool
	DependencyInclude    []string
	DependencyTransitive bool
}
