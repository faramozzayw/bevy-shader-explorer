package generation

import "path/filepath"

// searchIndexRenderer owns search artifacts independently from page rendering.
// This keeps HTML/JSON document renderers from depending on index layout.
type searchIndexRenderer interface {
	WritePublic(outputDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error
	WritePending(outputDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error
	WriteGlobal(outputDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error
}

type filesystemSearchIndexRenderer struct{}

func (filesystemSearchIndexRenderer) WritePublic(outputDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error {
	return copyItemsToPublic(outputDir, searchInfo, packages)
}

func (filesystemSearchIndexRenderer) WritePending(outputDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error {
	return writePendingSearchIndexes(outputDir, searchInfo, packages)
}

func (filesystemSearchIndexRenderer) WriteGlobal(outputDir string, searchInfo []ShaderSearchableInfo, packages []documentationPackagePage) error {
	return writeGlobalSearchIndex(filepath.Join(outputDir, "public"), searchInfo, packages)
}
