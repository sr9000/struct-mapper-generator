package node

import (
	"caster-generator/options"
	"caster-generator/primitive"
	"fmt"
	"reflect"
	"strings"
)

func genMap(
	src, dst reflect.Type,
	srcExpr, dstVar, dstStem, funcName string,
	nameOf FuncNamer,
	allowed options.CategoryEnum,
) Result {
	var out Result
	out.Imports = map[string]struct{}{}

	sBase, dBase := base(src), base(dst)
	if sBase.Kind() != reflect.Map {
		// Fallback stub
		out.Statements = append(out.Statements,
			//fmt.Sprintf("var %s %s // dst.%s", dstVar, typeStr(dst), cleanLeaf(dstVar)),
			fmt.Sprintf("_ = %s // suggestion", srcExpr),
		)
		return out
	}

	srcKey := base(sBase.Key())
	//srcVal := base(sBase.Elem())
	dstKey := base(dBase.Key())
	dstVal := base(dBase.Elem())

	// Prepare destination (map or *map)
	if dst.Kind() == reflect.Ptr {
		out.Statements = append(out.Statements,
			fmt.Sprintf("%s = new(%s)", dstVar, typeStr(dBase)),
			fmt.Sprintf("*%s = make(%s, len(%s))", dstVar, typeStr(dBase), srcExpr),
		)
	} else {
		out.Statements = append(out.Statements, fmt.Sprintf("%s = make(%s, len(%s))", dstVar, typeStr(dBase), srcExpr))
	}

	builder := dstVar
	if dst.Kind() == reflect.Ptr {
		builder = "*" + dstVar
	}

	out.Imports["fmt"] = struct{}{}
	out.Statements = append(out.Statements,
		fmt.Sprintf("for k, v := range %s {", srcExpr),
	)

	// Key conversion (primitive-only)
	//keyVar := "kk"
	if srcKey == dstKey {
		out.Statements = append(out.Statements, "  kk := k")
	} else {
		// Try primitive conversion path
		kPrimFrom := primitive.FromReflectType(srcKey)
		kPrimTo := primitive.FromReflectType(dstKey)
		if kPrimFrom != 0 && kPrimTo != 0 {
			out.Statements = append(out.Statements, fmt.Sprintf("  var kk %s", typeStr(dstKey)))
			lines := primitive.Generate(srcKey, dstKey, "k", "kk", dstStem, funcName, allowed)
			out.Statements = append(out.Statements, indentAll(lines, 2)...)
			for _, ln := range lines {
				if strings.Contains(ln, "time.") {
					out.Imports["time"] = struct{}{}
				}
				if strings.Contains(ln, "strconv.") {
					out.Imports["strconv"] = struct{}{}
				}
				if strings.Contains(ln, "strings.") {
					out.Imports["strings"] = struct{}{}
				}
				if strings.Contains(ln, "fmt.") {
					out.Imports["fmt"] = struct{}{}
				}
			}
		} else {
			out.Statements = append(out.Statements,
				fmt.Sprintf("  var kk %s // key stub", typeStr(dstKey)),
				"  _ = k // suggestion",
			)
		}
	}

	// Value conversion (recursive)
	valVar := "vv"
	out.Statements = append(out.Statements, fmt.Sprintf("  var %s %s", valVar, typeStr(dstVal)))

	// Pointer source value -> value destination requires nil check
	srcValExpr := "v"
	if sBase.Elem().Kind() == reflect.Ptr && dBase.Elem().Kind() != reflect.Ptr {
		out.Statements = append(out.Statements,
			"  if v == nil {",
			`    panic("source map value is nil")`,
			"  }",
		)
		srcValExpr = "*v"
	}

	// Recursively convert value into vv
	valRes := genValueInto(dstVal, srcValExpr, valVar, dstStem, funcName, nameOf, allowed)
	out.Statements = append(out.Statements, indentAll(valRes.Statements, 2)...)
	mergeResult(&out, &valRes)

	// Error/ok handling for nested casters in value
	out.Statements = append(out.Statements,
		"  if err != nil {",
		fmt.Sprintf(`    return fmt.Errorf("%s: map value conversion failed for key %%v: %%w", k, err)`, funcName),
		"  }",
		"  if !ok {",
		"    continue",
		"  }",
	)

	// Assign
	out.Statements = append(out.Statements, fmt.Sprintf("  %s[kk] = %s", builder, valVar), "}")

	return out
}

func genValueInto(
	dstVal reflect.Type,
	srcExpr string,
	dstVar string,
	dstStem string,
	funcName string,
	nameOf FuncNamer,
	allowed options.CategoryEnum,
) Result {
	var out Result
	out.Imports = map[string]struct{}{}

	// struct value -> nested caster
	if dstVal.Kind() == reflect.Struct {
		fn := nameOf(dstVal, dstVal) // placeholder: caller should pass actual src type mapping
		out.Needs = append(out.Needs, StructPair{Src: dstVal, Dst: dstVal})
		out.Statements = append(out.Statements,
			fmt.Sprintf("%s, ok, err = %s(%s)", dstVar, fn, srcExpr),
		)
		out.Imports["fmt"] = struct{}{}
		return out
	}

	// primitive value
	lines := primitive.Generate(nil, dstVal, srcExpr, dstVar, dstStem, funcName, allowed)
	out.Statements = append(out.Statements, lines...)
	for _, ln := range lines {
		if strings.Contains(ln, "time.") {
			out.Imports["time"] = struct{}{}
		}
		if strings.Contains(ln, "strconv.") {
			out.Imports["strconv"] = struct{}{}
		}
		if strings.Contains(ln, "strings.") {
			out.Imports["strings"] = struct{}{}
		}
		if strings.Contains(ln, "fmt.") {
			out.Imports["fmt"] = struct{}{}
		}
	}
	return out
}
