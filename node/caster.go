package node

import (
	"caster-generator/utils"
	"errors"
	"path"
	"reflect"
	"runtime"
	"strings"
)

var (
	ErrIsNotACaster         = errors.New("provided function is not a recognizable caster")
	ErrCasterIsNotAFunction = errors.New("provided caster is not a function")
	ErrDoublePointer        = errors.New("caster function does not support double pointers")
)

type Caster struct {
	Src, Dst     reflect.Type
	PackageAlias string
	Name         string
	HasBool      bool
	HasErr       bool
}

// ParseCaster inspects the provided function and returns a Caster struct if it is a valid caster function.
//
// Supports interfaces:
//   - func(src Type) (dst Type)
//   - func(src Type) (dst Type, bool)
//   - func(src Type) (dst Type, error)
//   - func(src Type) (dst Type, bool, error)
func ParseCaster(fn any) (Caster, error) {
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()
	if fnType.Kind() != reflect.Func {
		return Caster{}, ErrCasterIsNotAFunction
	}

	if fnType.NumIn() != 1 || fnType.NumOut() == 0 {
		return Caster{}, ErrIsNotACaster
	}

	src := fnType.In(0)
	if src.Kind() == reflect.Ptr && src.Elem().Kind() == reflect.Ptr {
		return Caster{}, ErrDoublePointer
	}

	dst := fnType.Out(0)
	if dst.Kind() == reflect.Ptr && dst.Elem().Kind() == reflect.Ptr {
		return Caster{}, ErrDoublePointer
	}

	// Get the pointer to the function
	// Get the function object from the pointer
	fnPC := runtime.FuncForPC(fnVal.Pointer())
	alias, name := utils.Unpack2(strings.SplitN(fnPC.Name(), ".", 2))

	caster := Caster{
		Src:          src,
		Dst:          dst,
		Name:         name,
		PackageAlias: utils.Second(path.Split(alias)),
	}

	switch fnType.NumOut() {
	default:
		return Caster{}, ErrIsNotACaster

	case 1:
		return caster, nil

	case 2:
		last := fnType.Out(1)

		switch {
		default:
			return Caster{}, ErrIsNotACaster
		case last.Kind() == reflect.Bool:
			caster.HasBool = true
		case isError(last):
			caster.HasErr = true
		}
		return caster, nil

	case 3:
		tbool, terr := fnType.Out(1), fnType.Out(2)
		if tbool.Kind() != reflect.Bool || !isError(terr) {
			return Caster{}, ErrIsNotACaster
		}

		caster.HasBool = true
		caster.HasErr = true
		return caster, nil
	}
}
