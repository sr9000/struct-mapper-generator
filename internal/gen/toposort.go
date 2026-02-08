package gen

import (
	"errors"
	"fmt"
	"sort"
)

// topoSortAssignments returns indices in execution order.
//
// Nodes are by index in the input slice.
// depsFn(i) yields indices that must be executed before i.
//
// The result is deterministic: when multiple nodes are available, we pick the
// smallest index. If a cycle exists, an error is returned.
func topoSortAssignments(n int, depsFn func(i int) []int) ([]int, error) {
	if n <= 0 {
		return nil, nil
	}

	indeg := make([]int, n)
	out := make([][]int, n)

	for i := range n {
		deps := depsFn(i)
		for _, d := range deps {
			if d < 0 || d >= n {
				return nil, fmt.Errorf("dependency index out of range: %d depends on %d", i, d)
			}

			indeg[i]++
			out[d] = append(out[d], i)
		}
	}

	// Deterministic traversal.
	for i := range out {
		sort.Ints(out[i])
	}

	var ready []int

	for i := range n {
		if indeg[i] == 0 {
			ready = append(ready, i)
		}
	}

	sort.Ints(ready)

	order := make([]int, 0, n)

	for len(ready) > 0 {
		i := ready[0]
		ready = ready[1:]

		order = append(order, i)
		for _, j := range out[i] {
			indeg[j]--
			if indeg[j] == 0 {
				// Insert while keeping ready sorted.
				k := sort.SearchInts(ready, j)
				ready = append(ready, 0)
				copy(ready[k+1:], ready[k:])
				ready[k] = j
			}
		}
	}

	if len(order) != n {
		return nil, errors.New("cycle detected")
	}

	return order, nil
}
