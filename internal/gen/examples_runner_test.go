package gen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExamples_RunScripts(t *testing.T) {
	t.Parallel()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	scripts := []string{
		filepath.Join(repoRoot, "examples", "pointers", "run.sh"),
		filepath.Join(repoRoot, "examples", "nested-mixed-structs", "run.sh"),
		filepath.Join(repoRoot, "examples", "arrays", "run.sh"),
		filepath.Join(repoRoot, "examples", "recursive-struct", "run.sh"),
	}

	for _, script := range scripts {
		t.Run(filepath.Base(filepath.Dir(script)), func(t *testing.T) {
			t.Parallel()

			cmd := exec.CommandContext(t.Context(), "bash", script)
			cmd.Dir = repoRoot

			cmd.Env = append(os.Environ(), "CG_NO_PROMPT=1", "CG_DEBUG=1")

			b, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s failed: %v\n%s", script, err, string(b))
			}
		})
	}
}
