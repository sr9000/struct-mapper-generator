package node

import (
	"caster-generator/options"
	"caster-generator/primitive"
	"fmt"
	"reflect"
	"strings"
)

func genTypeOrPointer(
	src, dst reflect.Type,
	srcExpr, dstVar, dstStem, funcName string,
	nameOf FuncNamer,
	allowed options.CategoryEnum,
) Result {
	var out Result
	out.Imports = map[string]struct{}{}

	sBase := base(src)
	dBase := base(dst)

	// Case 1: identical types and pointer shapes
	if src == dst {
		out.Statements = append(out.Statements, fmt.Sprintf("%s = %s", dstVar, srcExpr))
		return out
	}

	// Case 2: primitive sub-case (value to value) with pointer wrappers
	srcPrim := primitive.FromReflectType(sBase)
	dstPrim := primitive.FromReflectType(dBase)
	if srcPrim != 0 && dstPrim != 0 {
		out = genPrimitiveValueWithPointers(src, dst, srcExpr, dstVar, dstStem, funcName, allowed)
		return out
	}

	// Case 3: struct-to-struct (respect pointer wrappers). Use nested caster function.
	if sBase.Kind() == reflect.Struct && dBase.Kind() == reflect.Struct {
		callArg := srcExpr
		// source pointer -> require nil check + dereference
		if src.Kind() == reflect.Ptr {
			out.Statements = append(out.Statements,
				fmt.Sprintf("if %s == nil {", srcExpr),
				`  panic("source pointer is nil")`,
				"}",
			)
			callArg = "*" + srcExpr
		}

		fn := nameOf(sBase, dBase)
		out.Needs = append(out.Needs, StructPair{Src: sBase, Dst: dBase})
		out.Imports["fmt"] = struct{}{}

		switch dst.Kind() {
		default:
			// dst is value: call nested caster directly
			out.Statements = append(out.Statements,
				fmt.Sprintf("%s, ok, err = %s(%s)", dstVar, fn, callArg),
				"if err != nil {",
				fmt.Sprintf(`  return fmt.Errorf("%s failed to call %s: %%w", err)`, funcName, fn),
				"}",
			)
		case reflect.Ptr:
			// dst is pointer: call nested caster into temp, then take address
			tmp := "" //tmpName(dstVar, "val")
			out.Statements = append(out.Statements,
				fmt.Sprintf("var %s %s", tmp, typeStr(dBase)),
				fmt.Sprintf("%s, ok, err = %s(%s)", tmp, fn, callArg),
				"if err != nil {",
				fmt.Sprintf(`  return fmt.Errorf("%s failed to call %s: %%w", err)`, funcName, fn),
				"}",
				fmt.Sprintf("%s = &%s", dstVar, tmp),
			)
		}
		return out
	}

	// Case 4: fallback stub
	out.Statements = append(out.Statements,
		//fmt.Sprintf("var %s %s // dst.%s", dstVar, typeStr(dst), cleanLeaf(dstVar)),
		fmt.Sprintf("_ = %s // suggestion", srcExpr),
	)
	return out
}

func genPrimitiveValueWithPointers(
	src, dst reflect.Type,
	srcExpr, dstVar, dstStem, funcName string,
	allowed options.CategoryEnum,
) Result {
	var out Result
	out.Imports = map[string]struct{}{}

	valDstVar := dstVar
	valSrcExpr := srcExpr
	assignNeedsAddr := false

	// source pointer -> nil check and deref
	if src.Kind() == reflect.Ptr {
		out.Statements = append(out.Statements,
			fmt.Sprintf("if %s == nil {", srcExpr),
			`  panic("source pointer is nil")`,
			"}",
		)
		valSrcExpr = "*" + srcExpr
	}

	// destination pointer -> produce into temp value then take address
	if dst.Kind() == reflect.Ptr {
		assignNeedsAddr = true
		valDstVar = "" //tmpName(dstVar, "val")
		out.Statements = append(out.Statements,
			fmt.Sprintf("var %s %s", valDstVar, typeStr(dst.Elem())),
		)
	}

	// primitive conversion (value-to-value)
	lines := primitive.Generate(base(src), base(dst), valSrcExpr, valDstVar, dstStem, funcName, allowed)
	out.Statements = append(out.Statements, lines...)
	// Collect common imports heuristically from emitted lines
	for _, ln := range lines {
		if strings.Contains(ln, "fmt.") {
			out.Imports["fmt"] = struct{}{}
		}
		if strings.Contains(ln, "time.") {
			out.Imports["time"] = struct{}{}
		}
		if strings.Contains(ln, "strconv.") {
			out.Imports["strconv"] = struct{}{}
		}
		if strings.Contains(ln, "strings.") {
			out.Imports["strings"] = struct{}{}
		}
	}

	if assignNeedsAddr {
		out.Statements = append(out.Statements, fmt.Sprintf("%s = &%s", dstVar, valDstVar))
	}

	return out
}
