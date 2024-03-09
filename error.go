package refloat

import "errors"

type (
	// A NumError records a failed conversion.
	NumError struct {
		Func string // the failing function (ParseBool, ParseInt, ParseUint, ParseFloat, ParseComplex)
		Num  string // the input
		Err  error  // the reason the conversion failed (e.g. ErrRange, ErrSyntax, etc.)
	}
)

var (
	// ErrRange indicates that a value is out of range for the target type.
	ErrRange = errors.New("value out of range")
	// ErrSyntax indicates that a value does not have the right syntax for the target type.
	ErrSyntax = errors.New("invalid syntax")
)

func (err *NumError) Error() string {
	return "refloat." + err.Func + ": " + "parsing " + quoteASCII(err.Num) + ": " + err.Err.Error()
}

func (err *NumError) Unwrap() error {
	return err.Err
}

func errorSyntax(fnc, num string) *NumError {
	return &NumError{Func: fnc, Num: string([]byte(num)), Err: ErrSyntax}
}

func errorRange(fnc, num string) *NumError {
	return &NumError{Func: fnc, Num: string([]byte(num)), Err: ErrRange}
}

func quoteASCII(str string) string {
	// TODO: implement proper Quote().
	const hex = "0123456789abcdef"
	data := make([]byte, 0, len(str))
	data = append(data, '"')
	for idx := 0; idx < len(str); idx++ {
		char := str[idx]
		if char >= ' ' && char <= '~' && char != '\\' && char != '"' {
			data = append(data, char)
			continue
		}
		switch char {
		case '\\', '"':
			data = append(data, '\\', char)
		case '\n':
			data = append(data, '\\', 'n')
		case '\r':
			data = append(data, '\\', 'r')
		case '\t':
			data = append(data, '\\', 't')
		default:
			data = append(data, '\\', 'x', hex[char>>4], hex[char&0xf])
		}
	}
	data = append(data, '"')
	return string(data)
}
