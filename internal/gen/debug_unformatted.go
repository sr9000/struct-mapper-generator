package gen

import (
	"os"
	"path/filepath"
	"strings"
)

// writeDebugUnformatted writes unformatted code to a sidecar file next to the
// intended output. This is best-effort and should never make generation fail
// harder.
func writeDebugUnformatted(outDir, filename string, content []byte) error {
	if outDir == "" || filename == "" {
		return nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	// Keep it a .go file so editors can syntax highlight, but avoid colliding with
	// real output.
	debugName := strings.TrimSuffix(filename, ".go") + ".unformatted.go"
	p := filepath.Join(outDir, debugName)

	return os.WriteFile(p, content, 0o644)
}
