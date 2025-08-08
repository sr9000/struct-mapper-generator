package node_test

import (
	"caster-generator/node"
	"fmt"
)

func ExampleStem() {
	st := node.NewStem("id", nil)
	fmt.Println(st.Next(), st.Next(), st.Next())

	st = node.NewStem("val", map[string]struct{}{"val2": {}})
	fmt.Println(st.Next(), st.Next(), st.Next())

	// Output:
	// id1 id2 id3
	// val1 val3 val4
}
