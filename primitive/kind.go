package primitive

import (
	"math"
	"reflect"
	"time"
)

//go:generate go tool stringer -type=KindEnum -output=kind_string.go

type KindEnum int

const (
	_ KindEnum = iota // skip zero value, use it as a default (invalid) value for KindEnum

	KindInt
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindUint
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindFloat32
	KindFloat64
	KindBool
	KindString
	KindTime
	KindDuration
	KindPrimitiveEnum // alias to any integer number, boolean or string

	// KindTotal is a constant that represents the total number of kinds defined
	KindTotal = int(iota)
)

func (k KindEnum) IsNumber() bool {
	switch k {
	default:
		return false
	case KindInt, KindInt8, KindInt16, KindInt32, KindInt64,
		KindUint, KindUint8, KindUint16, KindUint32, KindUint64,
		KindFloat32, KindFloat64:
		return true
	}
}

func (k KindEnum) IsInteger() bool {
	switch k {
	default:
		return false
	case KindInt, KindInt8, KindInt16, KindInt32, KindInt64,
		KindUint, KindUint8, KindUint16, KindUint32, KindUint64:
		return true
	}
}

func (k KindEnum) IsFloat() bool {
	switch k {
	default:
		return false
	case KindFloat32, KindFloat64:
		return true
	}
}

func (k KindEnum) IsSigned() bool {
	switch k {
	default:
		return false
	case KindInt, KindInt8, KindInt16, KindInt32, KindInt64:
		return true
	}
}

func (k KindEnum) IsUnsigned() bool {
	switch k {
	default:
		return false
	case KindUint, KindUint8, KindUint16, KindUint32, KindUint64:
		return true
	}
}

func (k KindEnum) Bits() int {
	switch k {
	default:
		panic("only integer kinds has meaningful bits amount, but requested for: " + k.String())
	case KindInt, KindUint:
		power := 0
		for n := uint(math.MaxUint); n > 0; n >>= 1 {
			power++
		}
		return power
	case KindInt8, KindUint8:
		return 8
	case KindInt16, KindUint16:
		return 16
	case KindInt32, KindUint32:
		return 32
	case KindInt64, KindUint64:
		return 64
	case KindFloat32:
		return 32
	case KindFloat64:
		return 64
	}
}

func FromReflectType(rtype reflect.Type) KindEnum {
	if rtype == nil {
		return 0
	}

	// check if true primitive type
	switch rtype {
	case reflect.TypeOf(int(0)):
		return KindInt
	case reflect.TypeOf(int8(0)):
		return KindInt8
	case reflect.TypeOf(int16(0)):
		return KindInt16
	case reflect.TypeOf(int32(0)):
		return KindInt32
	case reflect.TypeOf(int64(0)):
		return KindInt64
	case reflect.TypeOf(uint(0)):
		return KindUint
	case reflect.TypeOf(uint8(0)):
		return KindUint8
	case reflect.TypeOf(uint16(0)):
		return KindUint16
	case reflect.TypeOf(uint32(0)):
		return KindUint32
	case reflect.TypeOf(uint64(0)):
		return KindUint64
	case reflect.TypeOf(float32(0)):
		return KindFloat32
	case reflect.TypeOf(float64(0)):
		return KindFloat64
	case reflect.TypeOf(false):
		return KindBool
	case reflect.TypeOf(""):
		return KindString
	case reflect.TypeOf(time.Time{}):
		return KindTime
	case reflect.TypeOf(time.Duration(0)):
		return KindDuration
	}

	// check if it's a primitive enum type
	switch rtype.Kind() {
	default:
		return 0
	case reflect.Int, reflect.String:
		return KindPrimitiveEnum
	}
}
