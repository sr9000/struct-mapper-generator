package node

import (
	"reflect"
	"strconv"
	"strings"
)

func typeStr(t reflect.Type) string {
	// fully qualified named types, or builtin string for basics
	switch t.Kind() {
	case reflect.Ptr:
		return "*" + typeStr(t.Elem())
	case reflect.Slice:
		return "[]" + typeStr(t.Elem())
	case reflect.Array:
		return "[" + strconv.Itoa(t.Len()) + "]" + typeStr(t.Elem())
	case reflect.Map:
		return "map[" + typeStr(t.Key()) + "]" + typeStr(t.Elem())
	default:
		if t.PkgPath() == "" {
			return t.String()
		}
		return t.PkgPath() + "." + t.Name()
	}
}

func indentAll(lines []string, tabs int) []string {
	if len(lines) == 0 {
		return lines
	}
	prefix := strings.Repeat("\t", tabs)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			out = append(out, l)
		} else {
			out = append(out, prefix+l)
		}
	}
	return out
}

func mergeResult(dst, src *Result) {
	dst.Needs = append(dst.Needs, src.Needs...)
	for k := range src.Imports {
		dst.Imports[k] = struct{}{}
	}
}

func isError(t reflect.Type) bool {
	if t == nil {
		return false
	}

	terr := reflect.TypeOf((*error)(nil)).Elem()

	return t.Implements(terr)
}
