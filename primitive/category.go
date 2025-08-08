package primitive

import "caster-generator/options"

type ConversionPair struct {
	From, To KindEnum
}

var conversionPairs map[options.CategoryEnum]map[ConversionPair]struct{}

func init() {
	conversionPairs = make(map[options.CategoryEnum]map[ConversionPair]struct{})

	conversionPairs[options.CategorySafeNumber] = safeNumberConversionPairs()

	// CategoryUnsafeNumber: unsafe number conversions
	conversionPairs[options.CategoryUnsafeNumber] = map[ConversionPair]struct{}{}
	for fromKind := KindEnum(0); int(fromKind) < KindTotal; fromKind++ {
		if !fromKind.IsNumber() {
			continue
		}

		for toKind := KindEnum(0); int(toKind) < KindTotal; toKind++ {
			if !toKind.IsNumber() {
				continue
			}

			pair := ConversionPair{fromKind, toKind}
			if _, ok := conversionPairs[options.CategorySafeNumber][pair]; ok {
				continue
			}

			conversionPairs[options.CategoryUnsafeNumber][pair] = struct{}{}
		}
	}

	// CategoryTextNumber: text <-> number conversions
	conversionPairs[options.CategoryTextNumber] = map[ConversionPair]struct{}{}
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsNumber() {
			continue
		}

		conversionPairs[options.CategoryTextNumber][ConversionPair{numberKind, KindString}] = struct{}{}
		conversionPairs[options.CategoryTextNumber][ConversionPair{KindString, numberKind}] = struct{}{}
	}

	// CategoryNumericBool: int <-> bool conversions
	conversionPairs[options.CategoryNumericBool] = map[ConversionPair]struct{}{}
	for fromKind := KindEnum(0); int(fromKind) < KindTotal; fromKind++ {
		if !fromKind.IsInteger() {
			continue
		}

		conversionPairs[options.CategoryNumericBool][ConversionPair{fromKind, KindBool}] = struct{}{}
		conversionPairs[options.CategoryNumericBool][ConversionPair{KindBool, fromKind}] = struct{}{}
	}

	// string <-> bool: yes, no, on, off, true, false
	conversionPairs[options.CategoryTextualBool] = map[ConversionPair]struct{}{
		{KindString, KindBool}: {},
		{KindBool, KindString}: {},
	}

	// CategoryDatetime: string(RFC3339Nano) <-> time.Time conversions
	conversionPairs[options.CategoryDatetime] = map[ConversionPair]struct{}{
		{KindString, KindTime}: {},
		{KindTime, KindString}: {},
	}

	// CategoryTimestamp: int(Unix seconds) <-> time.Time conversions
	conversionPairs[options.CategoryTimestamp] = map[ConversionPair]struct{}{}
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsInteger() {
			continue
		}

		conversionPairs[options.CategoryTimestamp][ConversionPair{numberKind, KindTime}] = struct{}{}
		conversionPairs[options.CategoryTimestamp][ConversionPair{KindTime, numberKind}] = struct{}{}
	}

	// CategoryDuration: string(2h45m) <-> time.Duration conversions
	conversionPairs[options.CategoryDuration] = map[ConversionPair]struct{}{
		{KindString, KindDuration}: {},
		{KindDuration, KindString}: {},
	}

	// CategoryNanoseconds: int(nanoseconds) <-> time.Duration conversions
	conversionPairs[options.CategoryNanoseconds] = map[ConversionPair]struct{}{}
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsInteger() || numberKind == KindUint64 {
			continue
		}

		conversionPairs[options.CategoryNanoseconds][ConversionPair{numberKind, KindDuration}] = struct{}{}
		conversionPairs[options.CategoryNanoseconds][ConversionPair{KindDuration, numberKind}] = struct{}{}
	}

	// CategorySeconds: float(seconds) <-> time.Duration conversions
	conversionPairs[options.CategorySeconds] = map[ConversionPair]struct{}{
		{KindFloat32, KindDuration}: {},
		{KindFloat64, KindDuration}: {},
		{KindDuration, KindFloat32}: {},
		{KindDuration, KindFloat64}: {},
	}

	// CategoryEnumString: string <-> enum conversions
	conversionPairs[options.CategoryEnumString] = map[ConversionPair]struct{}{
		{KindString, KindPrimitiveEnum}:        {},
		{KindPrimitiveEnum, KindString}:        {},
		{KindPrimitiveEnum, KindPrimitiveEnum}: {},
	}
}

func safeNumberConversionPairs() map[ConversionPair]struct{} {
	return map[ConversionPair]struct{}{
		{KindInt, KindInt}:   {}, // int can be any wide from 32 upto 64
		{KindInt, KindInt64}: {},

		{KindInt8, KindInt}:     {}, // int8 can be safely converted to any signed int
		{KindInt8, KindInt8}:    {},
		{KindInt8, KindInt16}:   {},
		{KindInt8, KindInt32}:   {},
		{KindInt8, KindInt64}:   {},
		{KindInt8, KindFloat32}: {},
		{KindInt8, KindFloat64}: {},

		{KindInt16, KindInt}:     {},
		{KindInt16, KindInt16}:   {}, // int16 omitting narrowing to int8
		{KindInt16, KindInt32}:   {},
		{KindInt16, KindInt64}:   {},
		{KindInt16, KindFloat32}: {},
		{KindInt16, KindFloat64}: {},

		{KindInt32, KindInt}:     {},
		{KindInt32, KindInt32}:   {}, // int32 omitting narrowing to int8/16
		{KindInt32, KindInt64}:   {},
		{KindInt32, KindFloat64}: {}, // int32 is wider than float32 mantissa

		{KindInt64, KindInt64}: {}, // int64 is the widest signed integer type

		{KindUint, KindUint}:   {}, // uint can be any wide from 32 upto 64
		{KindUint, KindUint64}: {},

		{KindUint8, KindUint}:    {}, // uint8 can be safely converted to any unsigned int
		{KindUint8, KindUint8}:   {},
		{KindUint8, KindUint16}:  {},
		{KindUint8, KindUint32}:  {},
		{KindUint8, KindUint64}:  {},
		{KindUint8, KindInt}:     {}, // also uint8 can be converted to any wider signed int
		{KindUint8, KindInt16}:   {},
		{KindUint8, KindInt32}:   {},
		{KindUint8, KindInt64}:   {},
		{KindUint8, KindFloat32}: {},
		{KindUint8, KindFloat64}: {},

		{KindUint16, KindUint}:    {},
		{KindUint16, KindUint16}:  {}, // uint16 omitting narrowing to uint8
		{KindUint16, KindUint32}:  {},
		{KindUint16, KindUint64}:  {},
		{KindUint16, KindInt}:     {}, // also uint16 can be converted to any wider signed int
		{KindUint16, KindInt32}:   {},
		{KindUint16, KindInt64}:   {},
		{KindUint16, KindFloat32}: {},
		{KindUint16, KindFloat64}: {},

		{KindUint32, KindUint32}:  {},
		{KindUint32, KindUint64}:  {}, // uint32 omitting narrowing to uint8/16
		{KindUint32, KindInt64}:   {}, // also only int64 is wide enough to hold uint32
		{KindUint32, KindFloat64}: {}, // uint32 is wider than float32 mantissa

		{KindUint64, KindUint64}: {}, // uint64 is the widest unsigned integer type

		{KindFloat32, KindFloat32}: {},
		{KindFloat32, KindFloat64}: {},

		{KindFloat64, KindFloat64}: {},
	}
}
