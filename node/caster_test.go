package node_test

import (
	"caster-generator/node"
	"fmt"
	"strconv"
)

type moreThanError interface {
	error
	More()
}

func empty()                          { panic("not implemented") }
func wrong(int) (string, error, bool) { panic("not implemented") }

func full(int) (string, bool, error)          { panic("not implemented") }
func customError(int) (string, moreThanError) { panic("not implemented") }

func ExampleCaster() {
	desc, err := node.ParseCaster(full)
	fmt.Println(err, desc.PackageAlias, desc.Name, desc.Src.Kind(), desc.Dst.Kind(), desc.HasBool, desc.HasErr)

	desc, err = node.ParseCaster(strconv.Itoa)
	fmt.Println(err, desc.PackageAlias, desc.Name, desc.Src.Kind(), desc.Dst.Kind(), desc.HasBool, desc.HasErr)

	desc, err = node.ParseCaster(strconv.Atoi)
	fmt.Println(err, desc.PackageAlias, desc.Name, desc.Src.Kind(), desc.Dst.Kind(), desc.HasBool, desc.HasErr)

	desc, err = node.ParseCaster(customError)
	fmt.Println(err, desc.PackageAlias, desc.Name, desc.Src.Kind(), desc.Dst.Kind(), desc.HasBool, desc.HasErr)

	_, err = node.ParseCaster(empty)
	fmt.Println(err)

	_, err = node.ParseCaster(wrong)
	fmt.Println(err)

	// Output:
	// <nil> node_test full int string true true
	// <nil> strconv Itoa int string false false
	// <nil> strconv Atoi string int false true
	// <nil> node_test customError int string false true
	// provided function is not a recognizable caster
	// provided function is not a recognizable caster
}
