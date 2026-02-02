package gen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerate_ArraysExample_Compiles(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	exampleDir := filepath.Join(repoRoot, "examples", "arrays")
	outDir := filepath.Join(exampleDir, "generated")

	_ = os.RemoveAll(outDir)

	cmd := exec.Command("go", "run", "./cmd/caster-generator", "gen",
		"-pkg", "./examples/arrays",
		"-mapping", filepath.Join(exampleDir, "map.yaml"),
		"-out", outDir,
	)
	cmd.Dir = repoRoot
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gen failed: %v\n%s", err, string(b))
	}

	build := exec.Command("go", "test", "./examples/arrays", "-run", "^$", "-count=1")
	build.Dir = repoRoot
	b, err = build.CombinedOutput()
	if err != nil {
		t.Fatalf("compile failed: %v\n%s", err, string(b))
	}
}
