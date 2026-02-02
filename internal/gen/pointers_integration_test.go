package gen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// This is an integration-ish test that ensures the generator can emit compilable code
// for pointer-heavy shapes (nested pointers + slice of pointers).
func TestGenerate_PointersExample_Compiles(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	exampleDir := filepath.Join(repoRoot, "examples", "pointers")
	outDir := filepath.Join(exampleDir, "generated")

	// Ensure a clean output dir so the test is repeatable.
	_ = os.RemoveAll(outDir)

	cmd := exec.Command("go", "run", "./cmd/caster-generator", "gen",
		"-pkg", "./examples/pointers",
		"-mapping", filepath.Join(exampleDir, "map.yaml"),
		"-out", outDir,
	)
	cmd.Dir = repoRoot
	b, err := cmd.CombinedOutput()
	if err != nil {
		// Best-effort: if any file got written, dump it for easier debugging.
		if entries, readErr := os.ReadDir(outDir); readErr == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				p := filepath.Join(outDir, e.Name())
				if fb, rerr := os.ReadFile(p); rerr == nil {
					t.Logf("generated file %s:\n%s", p, string(fb))
				}
			}
		}
		t.Fatalf("gen failed: %v\n%s", err, string(b))
	}

	// Compile the example package (including generated code).
	build := exec.Command("go", "test", "./examples/pointers", "-run", "^$", "-count=1")
	build.Dir = repoRoot
	b, err = build.CombinedOutput()
	if err != nil {
		t.Fatalf("compile failed: %v\n%s", err, string(b))
	}
}
