package node

import (
	"caster-generator/options"
	"caster-generator/primitive"
	"fmt"
	"reflect"
)

type FuncNamer func(src, dst reflect.Type) string

type Generator struct {
	funcNamer   FuncNamer
	needsDealer Dealer
	allowed     options.CategoryEnum
}

type Result struct {
	Statements []string
	Needs      []StructPair
	Imports    map[string]struct{} // e.g. "fmt", "time", "strconv", "strings"
}

func WithCustomName(fn FuncNamer, src, dst reflect.Type, name string) FuncNamer {
	if fn == nil {
		panic("customized namer function cannot be nil")
	}

	return func(srcIn, dstIn reflect.Type) string {
		if srcIn == src && dstIn == dst {
			return name
		}

		return fn(srcIn, dstIn)
	}
}

func Dispatch(src, dst reflect.Type) DispatcherEnum {
	if src.Kind() == reflect.Ptr || dst.Kind() == reflect.Ptr {
		panic("dispatcher is not allowing pointer reflect types")
	}

	if dst.Kind() == reflect.Interface {
		return DispatcherInterface
	}

	if dst.Kind() == reflect.Slice || dst.Kind() == reflect.Array {
		if src.Kind() == reflect.Slice || src.Kind() == reflect.Array {
			return DispatcherSlice
		}

		return DispatcherUnknown
	}

	if dst.Kind() == reflect.Map {
		if src.Kind() == reflect.Map {
			return DispatcherMap
		}

		return DispatcherUnknown
	}

	dstKind := primitive.FromReflectType(dst)
	if dstKind != 0 {
		srcKind := primitive.FromReflectType(src)
		if srcKind != 0 {
			return DispatcherPrimitive
		}

		return DispatcherUnknown
	}

	if dst.Kind() == reflect.Struct {
		if src.Kind() == reflect.Struct {
			return DispatcherStruct
		}

		return DispatcherUnknown
	}

	return DispatcherUnknown
}

// Generate dispatches conversion generation for three cases by dst shape:
// 1) type/pointer-to-type
// 2) slice/array
// 3) map
// It returns statements to compute dstVar from srcExpr, a set of needed nested struct casters, and import hints.
func (g *Generator) Generate(
	src, dst reflect.Type, casters ...any,
) []string {
	//out := Result{Imports: map[string]struct{}{}}

	srcDepth, srcBase := ptrDepthAndBase(src)
	dstDepth, dstBase := ptrDepthAndBase(dst)

	fmt.Println(srcDepth, dstDepth)

	switch Dispatch(srcBase, dstBase) {
	case DispatcherInterface:
	case DispatcherPrimitive:
	case DispatcherSlice:
	case DispatcherMap:
	case DispatcherStruct:
	case DispatcherUnknown:

	}

	return nil
}

func base(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// ptrDepthAndBase returns the pointer depth and the final base type.
func ptrDepthAndBase(t reflect.Type) (depth int, base reflect.Type) {
	depth, base = 0, t
	for t != nil && base.Kind() == reflect.Ptr {
		depth++
		base = base.Elem()
	}

	return
}
