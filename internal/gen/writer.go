package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// File permission constants.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// WriteFiles writes all generated files to the output directory.
// It creates the directory if it doesn't exist.
func WriteFiles(files []GeneratedFile, outputDir string) error {
	// Create output directory if it doesn't exist
	err := os.MkdirAll(outputDir, dirPerm)
	if err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, file := range files {
		outputPath := filepath.Join(outputDir, file.Filename)

		err := os.WriteFile(outputPath, file.Content, filePerm)
		if err != nil {
			return fmt.Errorf("writing file %s: %w", file.Filename, err)
		}
	}

	return nil
}
