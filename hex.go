package refloat

import (
	"math/bits"
)

func hexParseFloat(num string, extend int) (uint64, int, error) {
	const fnc = "ParseFloat"
	width := 32 << extend
	var (
		sign int
		mant uint64
		exp  int
	)

	var offset int
	if offset >= len(num) {
		return 0, 0, errorSyntax(fnc, num)
	} else if num[offset] == '+' {
		offset++
	} else if num[offset] == '-' {
		offset++
		sign = 1
	}

	offset += len("0x") // length of hexadecimal prefix.

	// one hexadecimal digit is 4 bits (2^4 == 16 patterns).
	const hexLen = 4
	// point == true: we already saw a decimal point.
	// trunc == true: non-zero digits are truncated and not in "mant".
	// digit == true: we at least saw one digit.
	// line  == true: we at least saw one underline.
	var point, trunc, digit, line bool
	for ; offset < len(num); offset++ {
		char := num[offset]
		if char == '_' {
			line = true
			continue
		}
		if char == '.' && !point {
			point = true
			continue
		}
		isaf := char|0x20-'a' <= 'f'-'a'
		is09 := char-'0' <= '9'-'0'
		if !isaf && !is09 {
			break
		}
		digit = true
		if point {
			exp -= hexLen
		}
		if mant>>(width-hexLen) != 0 {
			// truncation of 0 digits doesn't matter.
			trunc = trunc || char != '0'
			exp += hexLen
			continue
		}
		if isaf {
			mant = mant<<hexLen | uint64(char|0x20-'a'+10)
		}
		if is09 {
			mant = mant<<hexLen | uint64(char-'0')
		}
	}

	if !digit {
		return 0, 0, errorSyntax(fnc, num)
	}

	if offset >= len(num) || num[offset]|0x20 != 'p' {
		// according to strconv.readFloat, exponent
		// is required in hexadecimal.
		return 0, 0, errorSyntax(fnc, num)
	}
	offset += len("p") // already checked above.

	// max exponent (of positive or negative) + "mant" width + subnormal range.
	limit := 1024 + width + width
	var shift int
	// esign  == true: additional exponent part is negative
	// edigit == true: we at least saw one digit in exponent.
	var esign, edigit bool
	if offset >= len(num) {
		return 0, 0, errorSyntax(fnc, num)
	} else if num[offset] == '+' {
		offset++
	} else if num[offset] == '-' {
		offset++
		esign = true
	}

	for ; offset < len(num); offset++ {
		char := num[offset]
		if char == '_' {
			line = true
			continue
		}
		char -= '0'
		if char > '9'-'0' {
			break
		}
		edigit = true
		// exponent does not matter for 0; just consume.
		if mant == 0 {
			continue
		}
		// definitely an overflow.
		if esign && exp-shift < -limit || !esign && exp+shift > limit {
			continue
		}
		shift = shift*10 + int(char)
	}

	if esign {
		exp -= shift
	} else {
		exp += shift
	}

	if !edigit {
		return 0, 0, errorSyntax(fnc, num)
	}

	if line {
		// checking syntax related to underlines here
		// allows us to simply skip those while reading.
		// underlines often don't exist at all in number literals.
		for idx := 0; idx < len(num); idx++ {
			if num[idx] != '_' {
				continue
			}
			// '_' must separate successive digits
			// note, it does allow you to put them after "0x" prefix.
			if idx == 0 || idx == len(num)-1 {
				return 0, 0, errorSyntax(fnc, num)
			}
			lo, hi := num[idx-1], num[idx+1]
			if lo|0x20 != 'x' && lo-'0' > '9'-'0' && lo|0x20-'a' > 'f'-'a' {
				return 0, 0, errorSyntax(fnc, num)
			}
			if hi-'0' > '9'-'0' && hi|0x20-'a' > 'f'-'a' {
				return 0, 0, errorSyntax(fnc, num)
			}
		}
	}

	if mant == 0 {
		// the MSB is the sign bit. width -1 brings us just that.
		return uint64(sign) << (width - 1), offset, nil
	}

	// IEEE-754 mantissa length and exponent bias
	// for single/doube precision.
	var prec, bias int
	switch width {
	case 64:
		prec = 52
		bias = 1023
	case 32:
		prec = 23
		bias = 127
	}

	log := bits.Len64(mant) - prec - 1 - 1
	exp += log + bias + prec + 1
	// when we right shift, always check for truncated bits.
	// note: mant<<(width-log) != 0 is a wrong implementation,
	// especially for width == 32.
	if log > 0 {
		if mant&(1<<log-1) != 0 {
			trunc = true
		}
		mant >>= log
	} else {
		mant <<= -log
	}
	if exp <= 0 {
		if mant&(2<<-exp-1) != 0 {
			trunc = true
		}
		mant >>= 1 - exp
		exp = 0
	}
	// first part: round to nearest.
	round := mant & 1
	if round != 0 && !trunc {
		// second part: ties to even.
		// if all truncated bits are zero, we're at exactly the middle
		// of 2 floating points. in that case, go to the closest even
		// number (mant >> 1).
		round &= mant >> 1
	}
	mant += round
	// now we don't need the bit we used for rounding, cut off.
	mant >>= 1
	// this is for a case where we first thought it was subnormal,
	// but rounding made it slightly higher and made it to normal range.
	// if that happened, we simply increment exp (mantissa should be kept 0).
	if mant>>prec != 0 && exp == 0 {
		exp++
	}
	// handle carries from rounding.
	// mant >> (prec+1) becomes 1 when it overflows.
	carry := mant >> (prec + 1)
	mant >>= carry
	exp += int(carry)
	// implicit bit; hide the highest 1.
	mant &= 1<<prec - 1
	if extend == 1 && exp >= 0x7ff {
		return uint64(0x7ff)<<prec | uint64(sign)<<(width-1), offset, errorRange(fnc, num)
	}
	if extend == 0 && exp >= 0x0ff {
		return uint64(0x0ff)<<prec | uint64(sign)<<(width-1), offset, errorRange(fnc, num)
	}
	mant |= uint64(exp) << prec
	mant |= uint64(sign) << (width - 1)
	return mant, offset, nil
}
