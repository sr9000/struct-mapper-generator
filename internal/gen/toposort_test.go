package gen

import "testing"

func TestTopoSortAssignments_Order(t *testing.T) {
	order, err := topoSortAssignments(3, func(i int) []int {
		switch i {
		case 0:
			return nil
		case 1:
			return []int{0}
		case 2:
			return []int{1}
		default:
			return nil
		}
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	exp := []int{0, 1, 2}
	if len(order) != len(exp) {
		t.Fatalf("expected %v, got %v", exp, order)
	}

	for i := range exp {
		if order[i] != exp[i] {
			t.Fatalf("expected %v, got %v", exp, order)
		}
	}
}

func TestTopoSortAssignments_Cycle(t *testing.T) {
	_, err := topoSortAssignments(2, func(i int) []int {
		if i == 0 {
			return []int{1}
		}

		return []int{0}
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
