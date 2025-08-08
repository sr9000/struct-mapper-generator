package node

import (
	"caster-generator/options"
	"fmt"
	"reflect"
	"strings"
)

type FieldMatch struct {
	SrcName string
	Found   bool
}

// genStructLiteral creates a dst struct value into dstVar by mapping fields with recursion.
// It emits:
// - "var <dstVar> <DstType>"
// - per-field local variables or direct assigns
// - final "<dstVar> = <DstType>{ ... }"
func (g *Generator) genStructLiteral(
	src, dst reflect.Type,
	srcExpr, dstVar, funcName string,
	nameOf FuncNamer,
	allowed options.CategoryEnum,
) Result {
	var out Result
	out.Imports = map[string]struct{}{}
	out.Statements = append(out.Statements, fmt.Sprintf("var %s %s", dstVar, typeStr(dst)))

	// gather field build lines and return entries
	var fieldTemps []string
	var retEntries []string

	for i := 0; i < dst.NumField(); i++ {
		df := dst.Field(i)
		match := matchField(src, df)
		fieldVar := strings.ToLower(df.Name)

		if !match.Found {
			// stub
			fieldTemps = append(fieldTemps,
				fmt.Sprintf("var %s %s // dst.%s", fieldVar, typeStr(df.Type), df.Name),
				fmt.Sprintf("_ = %s // suggestion", srcExpr),
			)
			retEntries = append(retEntries, fmt.Sprintf("%s: %s,", df.Name, fieldVar))
			continue
		}

		sf, _ := src.FieldByName(match.SrcName)

		// Direct assign if equal types and shapes
		if sf.Type == df.Type {
			retEntries = append(retEntries, fmt.Sprintf("%s: %s.%s,", df.Name, srcExpr, match.SrcName))
			continue
		}

		// Otherwise, build fieldVar via recursion then place into literal
		fieldTemps = append(fieldTemps, fmt.Sprintf("// build %s", df.Name))

		// Determine per-case generation into fieldVar
		//fieldRes := Generate(sf.Type, df.Type, fmt.Sprintf("%s.%s", srcExpr, match.SrcName), fieldVar, fieldVar+"Tmp", funcName, nameOf, allowed)
		//fieldTemps = append(fieldTemps, fieldRes.Statements...)
		//mergeResult(&out, &fieldRes)

		retEntries = append(retEntries, fmt.Sprintf("%s: %s,", df.Name, fieldVar))
	}

	// Emit field temps
	out.Statements = append(out.Statements, fieldTemps...)

	// Create struct literal
	out.Statements = append(out.Statements, fmt.Sprintf("%s = %s{", dstVar, typeStr(dst)))
	for _, e := range retEntries {
		out.Statements = append(out.Statements, "  "+e)
	}
	out.Statements = append(out.Statements, "}")

	// If any nested caster errors inside fields, we already emitted error-handling lines there.

	// Collect import hints
	for _, ln := range out.Statements {
		if strings.Contains(ln, "fmt.") {
			out.Imports["fmt"] = struct{}{}
		}
	}

	return out
}

// matchField tries: `cast:"SrcName"`, json tag match, exact name, case-insensitive name
func matchField(src reflect.Type, dstField reflect.StructField) FieldMatch {
	// 1) cast tag
	if tag := dstField.Tag.Get("cast"); tag != "" {
		if _, ok := src.FieldByName(tag); ok {
			return FieldMatch{SrcName: tag, Found: true}
		}
	}

	// 2) json tag (match by json name)
	dstJSON := jsonTagName(dstField)
	if dstJSON != "" {
		for i := 0; i < src.NumField(); i++ {
			sf := src.Field(i)
			if jsonTagName(sf) == dstJSON {
				return FieldMatch{SrcName: sf.Name, Found: true}
			}
		}
	}

	// 3) exact name
	if _, ok := src.FieldByName(dstField.Name); ok {
		return FieldMatch{SrcName: dstField.Name, Found: true}
	}

	// 4) case-insensitive
	lw := strings.ToLower(dstField.Name)
	for i := 0; i < src.NumField(); i++ {
		if strings.ToLower(src.Field(i).Name) == lw {
			return FieldMatch{SrcName: src.Field(i).Name, Found: true}
		}
	}

	return FieldMatch{}
}

func jsonTagName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	// trim options
	if idx := strings.IndexByte(tag, ','); idx >= 0 {
		tag = tag[:idx]
	}
	return tag
}
