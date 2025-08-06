package primitive

type CategoryEnum int

type ConversionPair struct {
	From, To KindEnum
}

const (
	CategorySafeNumber   CategoryEnum = 1 << iota // int, uint, float without precision loss
	CategoryUnsafeNumber                          // int, uint, float with precision loss
	CategoryTextNumber                            // int, uint, float <-> string: textual number representation
	CategoryNumericBool                           // int <-> bool: 0, 1 representation of boolean values
	CategoryTextualBool                           // string <-> bool: yes, no, on, off, true, false representation of boolean values
	CategoryDatetime                              // string(RFC3339Nano) <-> time.Time: textual date and time representation
	CategoryTimestamp                             // int(Unix seconds) <-> time.Time: Unix timestamp representation
	CategoryDuration                              // string(2h45m) <-> time.Duration: textual duration representation
	CategoryNanoseconds                           // int(nanoseconds) <-> time.Duration: numerical (integer) duration representation
	CategorySeconds                               // float(seconds) <-> time.Duration: numerical (floating-point) duration representation
	CategoryEnumString                            // string <-> enum: textual representation of an enum type (uses parse/isValid/string methods)

	CategoryAll  = (1 << iota) - 1 //all categories combined
	CategoryNone = 0               // no categories selected
)

var conversionPairs map[CategoryEnum]map[ConversionPair]struct{}

func init() {
	conversionPairs = make(map[CategoryEnum]map[ConversionPair]struct{})

	conversionPairs[CategorySafeNumber] = safeNumberConversionPairs()

	// CategoryUnsafeNumber: unsafe number conversions
	conversionPairs[CategoryUnsafeNumber] = map[ConversionPair]struct{}{}
	for fromKind := KindEnum(0); int(fromKind) < KindTotal; fromKind++ {
		if !fromKind.IsNumber() {
			continue
		}

		for toKind := KindEnum(0); int(toKind) < KindTotal; toKind++ {
			if !toKind.IsNumber() {
				continue
			}

			pair := ConversionPair{fromKind, toKind}
			if _, ok := conversionPairs[CategorySafeNumber][pair]; ok {
				continue
			}

			conversionPairs[CategoryUnsafeNumber][pair] = struct{}{}
		}
	}

	// CategoryTextNumber: text <-> number conversions
	conversionPairs[CategoryTextNumber] = map[ConversionPair]struct{}{}
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsNumber() {
			continue
		}

		conversionPairs[CategoryTextNumber][ConversionPair{numberKind, KindString}] = struct{}{}
		conversionPairs[CategoryTextNumber][ConversionPair{KindString, numberKind}] = struct{}{}
	}

	// CategoryNumericBool: int <-> bool conversions
	conversionPairs[CategoryNumericBool] = map[ConversionPair]struct{}{}
	for fromKind := KindEnum(0); int(fromKind) < KindTotal; fromKind++ {
		if !fromKind.IsInteger() {
			continue
		}

		conversionPairs[CategoryNumericBool][ConversionPair{fromKind, KindBool}] = struct{}{}
		conversionPairs[CategoryNumericBool][ConversionPair{KindBool, fromKind}] = struct{}{}
	}

	// string <-> bool: yes, no, on, off, true, false
	conversionPairs[CategoryTextualBool] = map[ConversionPair]struct{}{
		{KindString, KindBool}: {},
		{KindBool, KindString}: {},
	}

	// CategoryDatetime: string(RFC3339Nano) <-> time.Time conversions
	conversionPairs[CategoryDatetime] = map[ConversionPair]struct{}{
		{KindString, KindTime}: {},
		{KindTime, KindString}: {},
	}

	// CategoryTimestamp: int(Unix seconds) <-> time.Time conversions
	conversionPairs[CategoryTimestamp] = map[ConversionPair]struct{}{}
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsInteger() {
			continue
		}

		conversionPairs[CategoryTimestamp][ConversionPair{numberKind, KindTime}] = struct{}{}
		conversionPairs[CategoryTimestamp][ConversionPair{KindTime, numberKind}] = struct{}{}
	}

	// CategoryDuration: string(2h45m) <-> time.Duration conversions
	conversionPairs[CategoryDuration] = map[ConversionPair]struct{}{
		{KindString, KindDuration}: {},
		{KindDuration, KindString}: {},
	}

	// CategoryNanoseconds: int(nanoseconds) <-> time.Duration conversions
	conversionPairs[CategoryNanoseconds] = map[ConversionPair]struct{}{}
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsInteger() || numberKind == KindUint64 {
			continue
		}

		conversionPairs[CategoryNanoseconds][ConversionPair{numberKind, KindDuration}] = struct{}{}
		conversionPairs[CategoryNanoseconds][ConversionPair{KindDuration, numberKind}] = struct{}{}
	}

	// CategorySeconds: float(seconds) <-> time.Duration conversions
	conversionPairs[CategorySeconds] = map[ConversionPair]struct{}{
		{KindFloat32, KindDuration}: {},
		{KindFloat64, KindDuration}: {},
		{KindDuration, KindFloat32}: {},
		{KindDuration, KindFloat64}: {},
	}

	// CategoryEnumString: string <-> enum conversions
	conversionPairs[CategoryEnumString] = map[ConversionPair]struct{}{
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
