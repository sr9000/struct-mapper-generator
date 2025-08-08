package node

import "strconv"

// NewStem creates a new Stem instance with the provided stem and namespace.
// The nil namespace is treated as a free namespace, meaning all names are available.
func NewStem(stem string, namespace map[string]struct{}) *Stem {
	return &Stem{
		taken: namespace,
		stem:  stem,
		last:  0,
	}
}

type Stem struct {
	taken map[string]struct{}
	stem  string
	last  int
}

func (s *Stem) Next() string {
	if s.taken == nil {
		s.taken = make(map[string]struct{})
	}

	for {
		s.last++
		name := s.stem + strconv.Itoa(s.last)

		if _, ok := s.taken[name]; !ok {
			s.taken[name] = struct{}{}
			return name
		}
	}
}
