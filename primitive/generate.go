package primitive

import (
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"text/template"
)

func Generate(src, dst reflect.Type, srcName, dstName, dstStem, funcName string, allowed CategoryEnum) []string {
	srcKind := FromReflectType(src)
	dstKind := FromReflectType(dst)
	pair := ConversionPair{srcKind, dstKind}

	if _, ok := allowedSet(allowed)[pair]; !ok {
		return nil
	}

	if srcKind == KindPrimitiveEnum || dstKind == KindPrimitiveEnum {
		// if src is a primitive enum, cast to interface{ String() string }
		// else src must be KindString
		// if dst is a primitive enum, check it against interface{ IsValid() bool }
		// else trigger compile error

		srcValue := srcName
		if srcKind == KindPrimitiveEnum {
			if src.Implements(reflect.TypeOf((*interface{ String() string })(nil)).Elem()) {
				srcValue += ".String()"
			} else if src.Kind() == reflect.String {
				srcValue = "string(" + srcValue + ")"
			}
		} else if srcKind != KindString {
			panic("src must be KindString or a primitive enum")
		}

		if dstKind == KindPrimitiveEnum {
			if dst.Kind() == reflect.String && dst.Implements(reflect.TypeOf((*interface{ IsValid() bool })(nil)).Elem()) {
				return []string{
					fmt.Sprintf("%s = %s(%s)", dstName, dst.Name(), srcValue),
					"if !" + dstName + ".IsValid() {",
					fmt.Sprintf(`	return fmt.Errorf("%s: %%s is not a valid value for %s", %s)`, funcName, dst.Name(), srcName),
					"}",
				}
			}
		} else if dstKind != KindString {
			panic("dst must be KindString or a primitive enum")
		}

		return []string{dstName + " = " + srcValue}
	}

	res := templates[pair]
	for i, line := range res {
		tmpl, err := template.New("line").Parse(line)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]any{
			"src":      srcName,
			"dst":      dstName,
			"srcType":  src.Name(),
			"dstType":  dst.Name(),
			"dstStem":  dstStem,
			"funcName": funcName,
		})
		if err != nil {
			panic(err)
		}

		res[i] = buf.String()
	}

	return res
}

func allowedSet(allowed CategoryEnum) map[ConversionPair]struct{} {
	res := map[ConversionPair]struct{}{}

	for category := CategoryEnum(1); category&CategoryAll > 0; category <<= 1 {
		if allowed&category == 0 {
			continue
		}

		maps.Copy(res, conversionPairs[category])
	}

	return res
}
