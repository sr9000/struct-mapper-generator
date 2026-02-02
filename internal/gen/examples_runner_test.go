package gen_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExamples_RunScripts(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	scripts := []string{
		filepath.Join(repoRoot, "examples", "pointers", "run.sh"),
		filepath.Join(repoRoot, "examples", "nested-mixed", "run.sh"),
		filepath.Join(repoRoot, "examples", "arrays", "run.sh"),
	}

	for _, script := range scripts {
		script := script
		t.Run(filepath.Base(filepath.Dir(script)), func(t *testing.T) {
			t.Parallel()

			cmd := exec.Command("bash", script)
			cmd.Dir = repoRoot
			b, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s failed: %v\n%s", script, err, string(b))
			}
		})
	}
}
