package refloat

import (
	"math/big"
)

var (
	fiv = big.NewInt(5)
)

func bigParseFloat(num string, extend int) (uint64, int, error) {
	const fnc = "ParseFloat"
	width2 := 32 << extend
	width10 := 10 << extend
	var (
		sign  int
		mant  big.Int
		exp10 int
	)

	var offset int
	if offset >= len(num) {
		panic("bigParseFloat was called with empty string")
	} else if num[offset] == '+' {
		offset++
	} else if num[offset] == '-' {
		offset++
		sign = 1
	}

	// Infs and NaNs are checked in fast-path.
	var temp big.Int
	var point bool
	for ; offset < len(num); offset++ {
		char := num[offset]
		if char == '.' && !point {
			point = true
			continue
		}
		if char == '_' {
			// also checked by fast-path.
			continue
		}
		char -= '0'
		if char > '9'-'0' {
			break
		}
		if point {
			exp10--
		}
		// * 10 might compile down to multiplication unlike
		// word size math, so we manually make shifts.
		// (x*10 = x*8 + x*2 = x<<3 + x<<1)
		temp.Lsh(&mant, 3)
		mant.Lsh(&mant, 1)
		mant.Add(&mant, &temp)
		temp.SetUint64(uint64(char))
		mant.Add(&mant, &temp)
	}

	if offset < len(num) && num[offset]|0x20 == 'e' {
		// log10(2)*1024 ~= 308.
		// max exponent + "mant" length + log10(2)*width.
		limit := 308 + int(mant.BitLen()) + width10
		var shift int
		var esign bool
		offset++
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
				continue
			}
			char -= '0'
			if char > '9'-'0' {
				break
			}
			// definitely an overflow/
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
	}

	exp := exp10
	abs := big.NewInt(0)
	abs.SetUint64(uint64(max(exp10, -exp10)))
	temp.Exp(fiv, abs, nil)
	var trunc bool
	if exp10 >= 0 {
		mant.Mul(&mant, &temp)
		log := mant.BitLen() - width2
		trunc = int(mant.TrailingZeroBits()) < log
		if log > 0 {
			mant.Rsh(&mant, uint(log))
			exp += log
		}
	} else {
		var rem big.Int
		log := temp.BitLen() - mant.BitLen() + width2
		if log > 0 {
			mant.Lsh(&mant, uint(log))
			exp -= log
		}
		mant.DivMod(&mant, &temp, &rem)
		trunc = rem.Sign() != 0
	}

	var prec, bias int
	switch width2 {
	case 64:
		prec = 52
		bias = 1023
	case 32:
		prec = 23
		bias = 127
	}

	log := mant.BitLen() - prec - 1 - 1
	exp += log + bias + prec + 1
	if log > 0 {
		// TODO(?): find a better way to find truncated bits.
		// is TrailingZeros enough?
		if int(mant.TrailingZeroBits()) < log {
			trunc = true
		}
		mant.Rsh(&mant, uint(log))
	} else {
		mant.Lsh(&mant, uint(-log))
	}
	if exp <= 0 {
		if int(mant.TrailingZeroBits()) < 1-exp {
			trunc = true
		}
		mant.Rsh(&mant, uint(1-exp))
		exp = 0
	}
	bit := mant.Uint64()
	// see hex.go for explanation of rounding.
	round := bit & 1
	if round != 0 && !trunc {
		round &= bit >> 1
	}
	bit += round
	bit >>= 1
	if bit>>prec != 0 && exp == 0 {
		exp++
	}
	carry := bit >> (prec + 1)
	bit >>= carry
	exp += int(carry)
	bit &= 1<<prec - 1
	if extend == 1 && exp >= 0x7ff {
		return uint64(0x7ff)<<prec | uint64(sign)<<(width2-1), offset, errorRange(fnc, num)
	}
	if extend == 0 && exp >= 0x0ff {
		return uint64(0x0ff)<<prec | uint64(sign)<<(width2-1), offset, errorRange(fnc, num)
	}
	bit |= uint64(exp) << prec
	bit |= uint64(sign) << (width2 - 1)
	return bit, offset, nil
}
