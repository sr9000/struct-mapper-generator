//go:build wubba_lubba_dub_dub

package example_test

import (
	"fmt"
	"reflect"
	"runtime"
)

type CasterPattern int

const (
	Struct2Struct CasterPattern = iota
	Pointer2Pointer
	Struct2Optional
	Struct2Error
	Struct2Full
	Pointer2Error
	UnknownPattern
)

type CasterFuncInfo struct {
	Name    string
	Pattern CasterPattern
	SrcType reflect.Type
	DstType reflect.Type
}

func patternName(p CasterPattern) string {
	switch p {
	case Struct2Struct:
		return "Struct2Struct"
	case Pointer2Pointer:
		return "Pointer2Pointer"
	case Struct2Optional:
		return "Struct2Optional"
	case Struct2Error:
		return "Struct2Error"
	case Struct2Full:
		return "Struct2Full"
	case Pointer2Error:
		return "Pointer2Error"
	default:
		return "Unknown"
	}
}

func analyzeCasterFunc(fn any) (CasterFuncInfo, error) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return CasterFuncInfo{}, fmt.Errorf("not a function")
	}
	info := CasterFuncInfo{Name: runtimeFuncName(fn)}

	// Analyze input
	if t.NumIn() != 1 {
		return info, fmt.Errorf("expected 1 input, got %d", t.NumIn())
	}
	in := t.In(0)
	info.SrcType = in

	// Analyze output
	switch t.NumOut() {
	case 1:
		out := t.Out(0)
		info.DstType = out
		if in.Kind() == reflect.Ptr && out.Kind() == reflect.Ptr {
			info.Pattern = Pointer2Pointer
		} else {
			info.Pattern = Struct2Struct
		}
	case 2:
		out0, out1 := t.Out(0), t.Out(1)
		info.DstType = out0
		if out1.Kind() == reflect.Bool {
			info.Pattern = Struct2Optional
		} else if out1.Name() == "error" && out1.PkgPath() == "" {
			if in.Kind() == reflect.Ptr && out0.Kind() == reflect.Ptr {
				info.Pattern = Pointer2Error
			} else {
				info.Pattern = Struct2Error
			}
		} else {
			info.Pattern = UnknownPattern
		}
	case 3:
		out0, out1, out2 := t.Out(0), t.Out(1), t.Out(2)
		info.DstType = out0
		if out1.Kind() == reflect.Bool && out2.Name() == "error" && out2.PkgPath() == "" {
			info.Pattern = Struct2Full
		} else {
			info.Pattern = UnknownPattern
		}
	default:
		info.Pattern = UnknownPattern
	}
	return info, nil
}

func runtimeFuncName(fn any) string {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return ""
	}
	pc := v.Pointer()
	f := runtime.FuncForPC(pc)
	if f == nil {
		return ""
	}
	return f.Name()
}

// Example functions
func cast1(a int) (b string)                     { return "" }
func cast2(a *int) (b *string)                   { return nil }
func cast3(a int) (b string, ok bool)            { return "", true }
func cast4(a int) (b string, err error)          { return "", nil }
func cast5(a int) (b string, ok bool, err error) { return "", true, nil }
func cast6(a *int) (b *string, err error)        { return nil, nil }

func Example() {
	funcs := []any{cast1, cast2, cast3, cast4, cast5, cast6}

	for i, fn := range funcs {
		info, err := analyzeCasterFunc(fn)
		if err != nil {
			fmt.Printf("Func %d: error: %v\n", i, err)
			continue
		}
		fmt.Printf("Func %d (%s): %s -> %s, pattern: %s\n",
			i,
			info.Name,
			info.SrcType,
			info.DstType,
			patternName(info.Pattern),
		)
	}

	// Output:
	// Func 0 (caster-generator/test1_test.cast1): int -> string, pattern: Struct2Struct
	// Func 1 (caster-generator/test1_test.cast2): *int -> *string, pattern: Pointer2Pointer
	// Func 2 (caster-generator/test1_test.cast3): int -> string, pattern: Struct2Optional
	// Func 3 (caster-generator/test1_test.cast4): int -> string, pattern: Struct2Error
	// Func 4 (caster-generator/test1_test.cast5): int -> string, pattern: Struct2Full
	// Func 5 (caster-generator/test1_test.cast6): *int -> *string, pattern: Pointer2Error
}
