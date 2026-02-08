package gen_test

import (
	"testing"
)

func TestGenerate_NestedMixedExample_Compiles(t *testing.T) {
	runExampleIntegrationTest(t, "nested-mixed-structs")
}
