package node_test

import (
	"caster-generator/node"
	"fmt"
	"reflect"
)

func ExampleDealer() {
	var d node.Dealer

	d.Needs(reflect.TypeFor[int](), reflect.TypeFor[string]())
	src, dst, ok := d.NextNeeds()
	fmt.Println("int & string:", src, dst, ok)

	_, _, ok = d.NextNeeds()
	fmt.Println("empty:", ok)

	d.Needs(reflect.TypeFor[int](), reflect.TypeFor[string]())
	_, _, ok = d.NextNeeds()
	fmt.Println("no duplicates:", ok)

	d.Needs(reflect.TypeFor[int](), reflect.TypeFor[int]())
	d.Needs(reflect.TypeFor[string](), reflect.TypeFor[string]())
	_, _, ok = d.NextNeeds()
	fmt.Println("some pair:", ok)

	_, _, ok = d.NextNeeds()
	fmt.Println("another pair:", ok)

	_, _, ok = d.NextNeeds()
	fmt.Println("no more pairs:", ok)

	// Output:
	// int & string: int string true
	// empty: false
	// no duplicates: false
	// some pair: true
	// another pair: true
	// no more pairs: false
}
