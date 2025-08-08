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
func Generate(
	src, dst reflect.Type,
	srcExpr, dstVar, dstStem, funcName string,
) []string {
	out := Result{Imports: map[string]struct{}{}}

	//srcDepth, srcBase := ptrDepthAndBase(src)

	switch base(dst).Kind() {
	case reflect.Slice, reflect.Array:
		//out = genSliceOrArray(src, dst, srcExpr, dstVar, dstStem, funcName, nameOf, allowed)
	case reflect.Map:
		//out = genMap(src, dst, srcExpr, dstVar, dstStem, funcName, nameOf, allowed)
	default:
		//out = genTypeOrPointer(src, dst, srcExpr, dstVar, dstStem, funcName, nameOf, allowed)
	}

	fmt.Println(out)
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
	depth = 0
	for t != nil && t.Kind() == reflect.Ptr {
		depth++
		t = t.Elem()
	}
	return depth, t
}
