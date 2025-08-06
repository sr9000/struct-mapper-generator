package primitive

import (
	"fmt"
)

var (
	templates map[ConversionPair][]string
)

func init() {
	templates = map[ConversionPair][]string{}

	// CategorySafeNumber
	// CategoryUnsafeNumber
	for fromKind := KindEnum(0); int(fromKind) < KindTotal; fromKind++ {
		if !fromKind.IsNumber() {
			continue
		}

		for toKind := KindEnum(0); int(toKind) < KindTotal; toKind++ {
			if !toKind.IsNumber() {
				continue
			}

			pair := ConversionPair{fromKind, toKind}
			if _, ok := templates[pair]; ok {
				continue
			}

			templates[pair] = []string{"{{.dst}} = {{.dstType}}({{.src}})"}
		}
	}

	// CategoryTextNumber
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if numberKind.IsSigned() {
			templates[ConversionPair{numberKind, KindString}] = []string{"{{.dst}} = {{.dstType}}(strconv.FormatInt(int64({{.src}}), 10))"}
			templates[ConversionPair{KindString, numberKind}] = []string{
				"var {{.dstStem}} int64",
				fmt.Sprintf("{{.dstStem}}, err = strconv.ParseInt({{.src}}), 10, %d)", numberKind.Bits()),
				"if err != nil {",
				`	return fmt.Errorf("{{.funcName}}: %w", err)`,
				"}",
				"",
				"{{.dst}} = {{.dstType}}({{.dstStem}})",
			}
		}

		if numberKind.IsUnsigned() {
			templates[ConversionPair{numberKind, KindString}] = []string{"{{.dst}} = {{.dstType}}(strconv.FormatUint(uint64({{.src}}), 10))"}
			templates[ConversionPair{KindString, numberKind}] = []string{
				"var {{.dstStem}} uint64",
				fmt.Sprintf("{{.dstStem}}, err = strconv.ParseUint({{.src}}), 10, %d)", numberKind.Bits()),
				"if err != nil {",
				`	return fmt.Errorf("{{.funcName}}: %w", err)`,
				"}",
				"",
				"{{.dst}} = {{.dstType}}({{.dstStem}})",
			}
		}

		if numberKind.IsFloat() {
			templates[ConversionPair{numberKind, KindString}] = []string{"{{.dst}} = {{.dstType}}(strconv.FormatFloat({{.src}}, 'f', -1, 64))"}
			templates[ConversionPair{KindString, numberKind}] = []string{
				"var {{.dstStem}} float64",
				fmt.Sprintf("{{.dstStem}}, err = strconv.ParseFloat({{.src}}, %d)", numberKind.Bits()),
				"if err != nil {",
				`	return fmt.Errorf("{{.funcName}}: %w", err)`,
				"}",
				"",
				"{{.dst}} = {{.dstType}}({{.dstStem}})",
			}
		}
	}

	// CategoryNumericBool
	for fromKind := KindEnum(0); int(fromKind) < KindTotal; fromKind++ {
		if !fromKind.IsInteger() {
			continue
		}

		// 0, 1 - valid, other numbers is error
		templates[ConversionPair{fromKind, KindBool}] = []string{
			"if {{.src}} == 0 {",
			"	{{.dst}} = false",
			"} else if {{.src}} == 1 {",
			"	{{.dst}} = true",
			"} else {",
			`	return fmt.Errorf("{{.funcName}}: only numbers 0 and 1 are allowed for bool, got: %d", {{.src}})`,
			"}",
		}
		templates[ConversionPair{KindBool, fromKind}] = []string{
			"if {{.src}} {",
			"	{{.dst}} = 1",
			"} else {",
			"	{{.dst}} = 0",
			"}",
		}
	}

	// CategoryTextualBool
	templates[ConversionPair{KindString, KindBool}] = []string{
		"switch strings.ToLower({{.src}}) {",
		"	default:",
		`		return fmt.Errorf("{{.funcName}}: only strings true/false, yes/no, on/off are allowed for bool, got: %s", {{.src}})`,
		`	case "true", "yes", "on":`,
		"		{{.dst}} = true",
		`	case "false", "no", "off":`,
		"		{{.dst}} = false",
		"}",
	}
	templates[ConversionPair{KindBool, KindString}] = []string{
		"if {{.src}} {",
		`	{{.dst}} = "true"`,
		"} else {",
		`	{{.dst}} = "false"`,
		"}",
	}

	// CategoryDatetime
	templates[ConversionPair{KindString, KindTime}] = []string{
		"var {{.dstStem}} time.Time",
		"{{.dstStem}}, err = time.Parse(time.RFC3339Nano, {{.src}})",
		"if err != nil {",
		`	return fmt.Errorf("{{.funcName}}: %w", err)`,
		"}",
		"",
		"{{.dst}} = {{.dstStem}}",
	}
	templates[ConversionPair{KindTime, KindString}] = []string{"{{.dst}} = {{.src}}.Format(time.RFC3339Nano)"}

	// CategoryTimestamp
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsInteger() || numberKind == KindUint64 {
			continue
		}

		templates[ConversionPair{numberKind, KindTime}] = []string{"{{.dst}} = time.Unix(int64({{.src}}), 0)"}

		if numberKind.IsSigned() {
			templates[ConversionPair{KindTime, numberKind}] = []string{
				"{{.dst}} = {{.src}}.Unix()", // trigger compile error (and manual control) if dst is not int64
			}
		}
	}

	// CategoryDuration
	templates[ConversionPair{KindString, KindDuration}] = []string{
		"var {{.dstStem}} time.Duration",
		"{{.dstStem}}, err = time.ParseDuration({{.src}})",
		"if err != nil {",
		`	return fmt.Errorf("{{.funcName}}: %w", err)`,
		"}",
		"",
		"{{.dst}} = {{.dstStem}}",
	}
	templates[ConversionPair{KindDuration, KindString}] = []string{"{{.dst}} = {{.src}}.String()"}

	// CategoryNanoseconds
	for numberKind := KindEnum(0); int(numberKind) < KindTotal; numberKind++ {
		if !numberKind.IsInteger() || numberKind == KindUint64 {
			continue
		}

		templates[ConversionPair{numberKind, KindDuration}] = []string{"{{.dst}} = time.Duration({{.src}})"}

		if numberKind.IsSigned() {
			templates[ConversionPair{KindDuration, numberKind}] = []string{
				"{{.dst}} = {{.src}}.Nanoseconds()", // trigger compile error (and manual control) if dst is not int64
			}
		}
	}

	// CategorySeconds
	templates[ConversionPair{KindFloat32, KindDuration}] = []string{"{{.dst}} = time.Duration({{.src}} * float32(time.Second))"}
	templates[ConversionPair{KindFloat64, KindDuration}] = []string{"{{.dst}} = time.Duration({{.src}} * float64(time.Second))"}
	templates[ConversionPair{KindDuration, KindFloat32}] = []string{"{{.dst}} = float32({{.src}}.Seconds())"}
	templates[ConversionPair{KindDuration, KindFloat64}] = []string{"{{.dst}} = {{.src}}.Seconds()"}
}
