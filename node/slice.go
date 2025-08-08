package node

import (
	"caster-generator/options"
	"fmt"
	"reflect"
	"strings"
)

func genSliceOrArray(
	src, dst reflect.Type,
	srcExpr, dstVar, dstStem, funcName string,
	nameOf FuncNamer,
	allowed options.CategoryEnum,
) Result {
	var out Result
	out.Imports = map[string]struct{}{}

	sBase, dBase := base(src), base(dst)
	if sBase.Kind() != reflect.Slice && sBase.Kind() != reflect.Array {
		// Fallback stub
		out.Statements = append(out.Statements,
			//fmt.Sprintf("var %s %s // dst.%s", dstVar, typeStr(dst), cleanLeaf(dstVar)),
			fmt.Sprintf("_ = %s // suggestion", srcExpr),
		)
		return out
	}

	//srcElem := baseElem(sBase)
	dstElem := baseElem(dBase)

	// For arrays, we create a temp slice builder, then copy into array (or produce best-effort stub)
	isDstArray := dBase.Kind() == reflect.Array
	dstElemStr := typeStr(dstElem)
	dstSliceStr := "[]" + dstElemStr

	builder := dstVar
	if isDstArray {
		builder = "" //tmpName(dstVar, "slice")
		out.Statements = append(out.Statements, fmt.Sprintf("%s := make(%s, 0, len(%s))", builder, dstSliceStr, srcExpr))
	} else {
		// dst slice or pointer to slice
		if dst.Kind() == reflect.Ptr {
			out.Statements = append(out.Statements,
				fmt.Sprintf("%s = new(%s)", dstVar, typeStr(dBase)),
				fmt.Sprintf("*%s = make(%s, 0, len(%s))", dstVar, typeStr(dBase), srcExpr),
			)
			builder = "*" + dstVar
		} else {
			out.Statements = append(out.Statements, fmt.Sprintf("%s = make(%s, 0, len(%s))", dstVar, typeStr(dBase), srcExpr))
			builder = dstVar
		}
	}

	// Loop with recursive element conversion
	out.Imports["fmt"] = struct{}{} // for error wrapping when nested casters are used
	out.Statements = append(out.Statements,
		fmt.Sprintf("for i := 0; i < len(%s); i++ {", srcExpr),
		fmt.Sprintf("  var tmp %s", dstElemStr),
	)

	elemSrcExpr := fmt.Sprintf("%s[i]", srcExpr)
	// element recursion:
	elemRes := genElementConversion(elemSrcExpr, "tmp", dstElem, funcName, nameOf, allowed)
	out.Statements = append(out.Statements, indentAll(elemRes.Statements, 2)...)
	mergeResult(&out, &elemRes)

	// skip optional elements if nested caster returns ok=false (we follow the convention that nested casters return (v, ok, err))
	out.Statements = append(out.Statements,
		"  if err != nil {",
		fmt.Sprintf(`    return fmt.Errorf("%s: element %%d conversion failed: %%w", i, err)`, funcName),
		"  }",
		"  if !ok {",
		"    continue",
		"  }",
		fmt.Sprintf("  %s = append(%s, tmp)", builder, builder),
		"}",
	)

	// if destination is array, copy and truncate if needed
	if isDstArray {
		out.Statements = append(out.Statements,
			fmt.Sprintf("for i := 0; i < len(%s) && i < %d; i++ {", builder, dBase.Len()),
			fmt.Sprintf("  %s[i] = %s[i]", dstVar, builder),
			"}",
		)
	}

	// Collect import hints from nested primitives
	for _, ln := range out.Statements {
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

	return out
}

func baseElem(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return base(t.Elem())
	}
	return t
}

// genElementConversion converts a single element (value-level). Handles both primitive and struct via nested caster.
func genElementConversion(
	srcExpr string,
	dstVar string,
	dstElem reflect.Type,
	funcName string,
	nameOf FuncNamer,
	allowed options.CategoryEnum,
) Result {
	var out Result
	out.Imports = map[string]struct{}{}

	// We don't know src type statically here, rely on embedded expression's type in real generator;
	// for snippet generation we assume src expression has the right base type shape at runtime,
	// so detect via dstElem shape and use nested caster for structs,
	// otherwise assume primitive via value cast where possible (caller supplies src reflect.Type if needed).
	if dstElem.Kind() == reflect.Struct {
		// In a full implementation you would carry the actual src element type through recursion;
		// here we treat src element = dst element's source twin, and ask namer for function name.
		// Caller should pass proper namer that maps source->destination types.
		out.Needs = append(out.Needs, StructPair{Src: dstElem, Dst: dstElem})
		fn := nameOf(dstElem, dstElem) // placeholder; in real usage, pass actual srcElem type
		out.Statements = append(out.Statements,
			fmt.Sprintf("%s, ok, err = %s(%s)", dstVar, fn, srcExpr),
		)
		out.Imports["fmt"] = struct{}{}
		return out
	}

	// Primitive fallback (assume src expr type equals dstElem or convertible; for exactness,
	// prefer calling primitive.Generate from callers that know src reflect.Type)
	// Minimal: direct cast stub if unknown
	out.Statements = append(out.Statements, fmt.Sprintf("%s = %s(%s)", dstVar, typeStr(dstElem), srcExpr))
	return out
}
