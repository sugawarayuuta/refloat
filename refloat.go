package refloat

// ParseFloat converts the string num to a floating-point number
// with the precision specified by size: 32 for float32, or 64 for float64.
// When size=32, the result still has type float64, but it will be
// convertible to float32 without changing its value.
//
// ParseFloat accepts decimal and hexadecimal floating-point numbers
// as defined by the Go syntax for [floating-point literals].
// If num is well-formed and near a valid floating-point number,
// ParseFloat returns the nearest floating-point number rounded
// using IEEE754 unbiased rounding.
// (Parsing a hexadecimal floating-point value only rounds when
// there are more bits in the hexadecimal representation than
// will fit in the mantissa.)
//
// The errors that ParseFloat returns have concrete type *NumError
// and include err.Num = num.
//
// If num is not syntactically well-formed, ParseFloat returns err.Err = ErrSyntax.
//
// If num is syntactically well-formed but is more than 1/2 ULP
// away from the largest floating point number of the given size,
// ParseFloat returns f = ±Inf, err.Err = ErrRange.
//
// ParseFloat recognizes the string "NaN", and the (possibly signed) strings "Inf" and "Infinity"
// as their respective special floating point values. It ignores case when matching.
//
// [floating-point literals]: https://go.dev/ref/spec#Floating-point_literals
func ParseFloat(num string, size int) (float64, error) {
	const fnc = "ParseFloat"
	f64, read, err := parseFloat(num, size)
	if read != len(num) && (err == nil || err.(*NumError).Err != ErrSyntax) {
		return 0, errorSyntax(fnc, num)
	}
	return f64, err
}

func parseFloat(num string, size int) (float64, int, error) {
	if size == 32 {
		f32, read, err := parseFloat32(num)
		return float64(f32), read, err
	}
	return parseFloat64(num)
}
