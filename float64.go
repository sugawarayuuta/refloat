package refloat

import (
	"math"
	"math/bits"
)

var (
	// powers of 10 can be represented exactly by binary64, ignoring
	// all the trailing zeros.
	pow10float64 = [...]float64{
		1e00, 1e01, 1e02, 1e03, 1e04, 1e05, 1e06, 1e07, 1e08, 1e09,
		1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
		1e20, 1e21, 1e22,
	}
)

func parseFloat64(num string) (float64, int, error) {
	const fnc = "ParseFloat"
	var (
		sign  int
		mant  uint64
		exp10 int
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

	if offset >= len(num) {
		return 0, 0, errorSyntax(fnc, num)
	}
	// ORing 0x20 gives lowercased characters.
	if num[offset]|0x20 == 'i' {
		comm := common(num[offset+1:], "nfinity")
		if comm == 7 {
			return math.Inf(-sign), offset + 8, nil
		}
		// don't consume something like "infin" more than "inf".
		if comm == 2 {
			return math.Inf(-sign), offset + 3, nil
		}
		return 0, 0, errorSyntax(fnc, num)
	}

	if num[offset]|0x20 == 'n' {
		comm := common(num[offset+1:], "an")
		// NaN cannot be signed.
		if comm == 2 && offset == 0 {
			return math.NaN(), offset + 3, nil
		}
		return 0, 0, errorSyntax(fnc, num)
	}

	if offset+1 < len(num) && num[offset] == '0' && num[offset+1]|0x20 == 'x' {
		u64, offset, err := hexParseFloat(num, 1)
		return math.Float64frombits(u64), offset, err
	}

	// the limit of being able to do
	// mant = mant*10 + 9.
	const limit = 0x1999999999999999
	var point, edigit, line bool
	for ; offset < len(num); offset++ {
		char := num[offset]
		if char == '.' && !point {
			point = true
			continue
		}
		if char == '_' {
			line = true
			continue
		}
		char -= '0'
		if char > '9'-'0' {
			break
		}
		edigit = true
		if point {
			exp10--
		}
		if mant >= limit {
			exp10++
			continue
		}
		mant = mant*10 + uint64(char)
	}

	if !edigit {
		return 0, 0, errorSyntax(fnc, num)
	}

	if offset < len(num) && num[offset]|0x20 == 'e' {
		// max exponent + "mant" variable size + subnormal range.
		// all parts are taken as upper bounds.
		const limit = 308 + 20 + 20
		var shift int
		var esign, edigit bool
		offset++
		if offset >= len(num) {
			return 0, 0, errorSyntax(fnc, num)
		} else if num[offset] == '+' {
			offset++
		} else if offset < len(num) && num[offset] == '-' {
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
			// definitely an overflow
			if esign && exp10-shift < -limit || !esign && exp10+shift > limit {
				continue
			}
			shift = shift*10 + int(char)
		}
		if esign {
			exp10 -= shift
		} else {
			exp10 += shift
		}
		if !edigit {
			return 0, 0, errorSyntax(fnc, num)
		}
	}

	if line {
		for idx := 0; idx < len(num); idx++ {
			if num[idx] != '_' {
				continue
			}
			if idx == 0 || idx == len(num)-1 {
				return 0, 0, errorSyntax(fnc, num)
			}
			lo, hi := num[idx-1], num[idx+1]
			if lo-'0' > '9'-'0' || hi-'0' > '9'-'0' {
				return 0, 0, errorSyntax(fnc, num)
			}
		}
	}

	abs := max(exp10, -exp10)
	if abs <= 22 && mant < 1<<53 {
		// even if it can't represent the number exactly,
		// as long as it's under these conditions,
		// it returns the correctly rounded values.
		f64 := float64(mant)
		if exp10 > 0 {
			f64 *= pow10float64[abs]
		} else {
			f64 /= pow10float64[abs]
		}
		if sign > 0 {
			f64 = -f64
		}
		return f64, offset, nil
	}

	if mant == 0 {
		f64 := math.Float64frombits(uint64(sign) << 63)
		return f64, offset, nil
	}

	exp := exp10
	nat, lo, hi := normalize64(uint64(abs))

	flag := lo >> 63
	if flag != 0 {
		nat++
		lo = -lo
		hi = -hi
	}
	exp += exp10 * 2
	if exp10 > 0 {
		exp += int(nat)
	} else {
		exp -= int(nat)
	}

	zero := bits.LeadingZeros64(mant)
	lom := mant << zero
	him := lom
	if mant >= limit {
		// mant >= limit when it couldn't represent
		// the mantissa exactly. create an upper bound.
		him += 1 << zero
	}
	exp -= int(zero)

	mx := max(hi, lo)
	mn := min(hi, lo)
	// diff (1):
	// difference between the answer and the output with floored constants.
	// norm (2):
	// max error, also known as Uniform norm but rounded up away from zero.
	flag ^= uint64(exp10) >> 63
	if flag == 0 {
		const diff = 3 // (1) ceil(~2.74) == 3.
		const norm = 3 // (2) ceil(~2.27) == 3.
		e64 := exp64pos(mn)
		hi, lo = e64+norm+diff, e64-norm
	} else {
		const diff = 2 // (1) ceil(~1.36) == 2.
		const norm = 2 // (2) ceil(~1.60) == 2.
		e64 := exp64neg(mx)
		hi, lo = e64+norm+diff, e64-norm
	}

	lop := mul64(lom, lo)
	hip := mul64(him, hi) + 1
	if flag == 0 {
		los, loc := bits.Add64(lop, lom, 0)
		his, hic := bits.Add64(hip, him, 0)
		// loc and hic are ORed to not lose bits.
		// always prioritize the one with a carry.
		lop = los>>(loc|hic) | loc<<63
		hip = his>>(loc|hic) | hic<<63
		exp += int(loc | hic)
	}

	const bias = 1023
	const prec = 52

	// lop and hip are NOT-ANDed to not lose bits,
	// just like the OR in the above block.
	// always prioritize the one with non-zero MSB.
	flip := ^lop & ^hip >> 63
	lop <<= flip
	hip <<= flip
	exp += bias + 63 - int(flip)
	if exp <= 0 {
		lop >>= 1 - exp
		hip >>= 1 - exp
		exp = 0
	}

	hir := hip << (prec + 1)
	lor := lop << (prec + 1)
	hip = hip>>(63-prec) + hir>>63
	lop = lop>>(63-prec) + lor>>63

	if lop>>prec != 0 && exp == 0 {
		exp++
	}

	carry := lop >> (prec + 1)
	hip >>= carry
	lop >>= carry
	exp += int(carry)
	if lor <= 1<<63 && hir > 1<<63 || hip != lop {
		// the former condition ensures there is no possibilities
		// of ending up in the "ties".
		u64, offset, err := bigParseFloat(num, 1)
		return math.Float64frombits(u64), offset, err
	}

	if exp >= 0x7ff {
		return math.Inf(-sign), offset, errorRange(fnc, num)
	}

	bit := lop & (1<<prec - 1)
	bit |= uint64(exp) << prec
	bit |= uint64(sign) << 63
	return math.Float64frombits(bit), offset, nil
}

func common(str, cmp string) int {
	for idx := 0; idx < len(str) && idx < len(cmp); idx++ {
		if str[idx]|0x20 != cmp[idx] {
			return idx
		}
	}
	return min(len(str), len(cmp))
}
