package node

import "reflect"

type StructPair struct{ Src, Dst reflect.Type }

type Dealer struct {
	needs map[StructPair]struct{}
	done  map[StructPair]struct{}
}

func (d *Dealer) NextNeeds() (src, dst reflect.Type, ok bool) {
	if len(d.needs) == 0 {
		return
	}

	for pair := range d.needs {
		delete(d.needs, pair)

		if _, exists := d.done[pair]; !exists {
			d.Done(pair.Src, pair.Dst)

			return pair.Src, pair.Dst, true
		}
	}

	return
}

func (d *Dealer) Needs(src, dst reflect.Type) {
	if d.needs == nil {
		d.needs = make(map[StructPair]struct{})
	}

	pair := StructPair{Src: src, Dst: dst}
	if _, exists := d.done[pair]; !exists {
		d.needs[pair] = struct{}{}
	}
}

func (d *Dealer) Done(src, dst reflect.Type) {
	if d.done == nil {
		d.done = make(map[StructPair]struct{})
	}

	pair := StructPair{Src: src, Dst: dst}
	delete(d.needs, pair)
	d.done[pair] = struct{}{}
}
