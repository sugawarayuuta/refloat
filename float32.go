package refloat

import (
	"math"
	"math/bits"
)

var (
	pow10float32 = [...]float32{
		1e00, 1e01, 1e02, 1e03, 1e04, 1e05, 1e06, 1e07, 1e08, 1e09,
		1e10,
	}
)

const (
	inf32 = 0x7f800000
	nan32 = 0x7f800001
)

func parseFloat32(num string) (float32, int, error) {
	const fnc = "ParseFloat"
	var (
		sign  int
		mant  uint32
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

	if num[offset]|0x20 == 'i' {
		comm := common(num[offset+1:], "nfinity")
		if comm == 7 {
			return math.Float32frombits(inf32 | uint32(sign)<<31), offset + 8, nil
		}
		if comm == 2 {
			return math.Float32frombits(inf32 | uint32(sign)<<31), offset + 3, nil
		}
		return 0, 0, errorSyntax(fnc, num)
	}

	if num[offset]|0x20 == 'n' {
		comm := common(num[offset+1:], "an")
		if comm == 2 && offset == 0 {
			return math.Float32frombits(nan32), offset + 3, nil
		}
		return 0, 0, errorSyntax(fnc, num)
	}

	if offset+1 < len(num) && num[offset] == '0' && num[offset+1]|0x20 == 'x' {
		u64, offset, err := hexParseFloat(num, 0)
		return math.Float32frombits(uint32(u64)), offset, err
	}

	const limit = 0x19999999
	var point, digit, line bool
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
		digit = true
		if point {
			exp10--
		}
		if mant >= limit {
			exp10++
			continue
		}
		mant = mant*10 + uint32(char)
	}

	if !digit {
		return 0, 0, errorSyntax(fnc, num)
	}

	if offset < len(num) && num[offset]|0x20 == 'e' {
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
			if mant == 0 {
				continue
			}
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
	if abs <= 10 && mant < 1<<24 {
		f32 := float32(mant)
		if exp10 > 0 {
			f32 *= pow10float32[abs]
		} else {
			f32 /= pow10float32[abs]
		}
		if sign > 0 {
			f32 = -f32
		}
		return f32, offset, nil
	}

	if mant == 0 {
		f32 := math.Float32frombits(uint32(sign) << 31)
		return f32, offset, nil
	}

	exp := exp10
	nat, lo, hi := normalize32(uint32(abs))

	flag := lo >> 31
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

	zero := bits.LeadingZeros32(mant)
	lom := mant << zero
	him := lom
	if mant >= limit {
		him += 1 << zero
	}
	exp -= zero

	mn := min(hi, lo)
	flag ^= uint32(exp10) >> 31
	if flag == 0 {
		const diff = 2 // (1) ceil(~1.94) == 2.
		const norm = 7 // (2) ceil(~6.00) == 7.
		e64 := exp32pos(mn)
		hi, lo = e64+norm+diff, e64-norm
	} else {
		const diff = 1 // (1) ceil(~0.72) == 1.
		const norm = 5 // (2) ceil(~4.24) == 5.
		e64 := exp32neg(mn)
		hi, lo = e64+norm+diff, e64-norm
	}

	lop := mul32(lom, lo)
	hip := mul32(him, hi) + 1
	if flag == 0 {
		los, loc := bits.Add32(lop, lom, 0)
		his, hic := bits.Add32(hip, him, 0)
		lop = los>>(loc|hic) | loc<<31
		hip = his>>(loc|hic) | hic<<31
		exp += int(loc | hic)
	}

	const bias = 127
	const prec = 23

	flip := ^lop & ^hip >> 31
	lop <<= flip
	hip <<= flip
	exp += bias + 31 - int(flip)
	if exp <= 0 {
		lop >>= 1 - exp
		hip >>= 1 - exp
		exp = 0
	}

	hir := hip << (prec + 1)
	lor := lop << (prec + 1)
	hip = hip>>(31-prec) + hir>>31
	lop = lop>>(31-prec) + lor>>31

	if lop>>prec != 0 && exp == 0 {
		exp++
	}

	carry := lop >> (prec + 1)
	hip >>= carry
	lop >>= carry
	exp += int(carry)
	if lor <= 1<<31 && hir > 1<<31 || hip != lop {
		u64, offset, err := bigParseFloat(num, 0)
		return math.Float32frombits(uint32(u64)), offset, err
	}

	if exp >= 0x0ff {
		return math.Float32frombits(inf32 | uint32(sign)<<31), offset, errorRange(fnc, num)
	}

	bit := lop & (1<<prec - 1)
	bit |= uint32(exp) << prec
	bit |= uint32(sign) << 31
	return math.Float32frombits(bit), offset, nil
}
