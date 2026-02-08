package gen_test

import (
	"testing"
)

// This is an integration-ish test that ensures the generator can emit compilable code
// for pointer-heavy shapes (nested pointers + slice of pointers).
func TestGenerate_PointersExample_Compiles(t *testing.T) {
	runExampleIntegrationTest(t, "pointers")
}
