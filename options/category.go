package options

type CategoryEnum int

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
	CategorySafeArray                             // slice <-> array: slice perfectly fits into an array
	CategoryUnsafeArray                           // slice <-> array: slice does not fit into an array, slices are cut, arrays leaved with zero values

	CategoryAll  = (1 << iota) - 1 //all categories combined
	CategoryNone = 0               // no categories selected
)
