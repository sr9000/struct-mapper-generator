package primitive_test

import (
	"caster-generator/primitive"
	"fmt"
	"reflect"
	"time"
)

func Example() {
	type IntEnum int
	type StringEnum string
	type Empty struct{}

	fmt.Println(primitive.FromReflectType(reflect.TypeOf(int(0))))
	fmt.Println(primitive.FromReflectType(reflect.TypeOf("")))
	fmt.Println(primitive.FromReflectType(reflect.TypeOf(IntEnum(0))))
	fmt.Println(primitive.FromReflectType(reflect.TypeOf(StringEnum(""))))
	fmt.Println(primitive.FromReflectType(reflect.TypeOf(time.Duration(0))))
	fmt.Println(primitive.FromReflectType(reflect.TypeOf(time.Time{})))
	fmt.Println(primitive.FromReflectType(reflect.TypeOf(Empty{})))
	// Output:
	// KindInt
	// KindString
	// KindPrimitiveEnum
	// KindPrimitiveEnum
	// KindDuration
	// KindTime
	// KindEnum(0)
}
