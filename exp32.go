package refloat

import "math/bits"

// see exp64.go for details of this algorithm and tools used.

func normalize32(f32 uint32) (uint32, uint32, uint32) {
	lo := mul32(0x346e2bf9, f32)
	hi, md := bits.Mul32(0x5269e12f, f32)
	lo, carry := bits.Add32(md, lo, 0)
	return hi + carry, lo, lo + 1
}

func mul32(f32, mul uint32) uint32 {
	prod, _ := bits.Mul32(f32, mul)
	return prod
}

func exp32pos(f32 uint32) uint32 {
	var approx uint32
	approx = mul32(0x0068109f+approx, f32)
	approx = mul32(0x026ca46f+approx, f32)
	approx = mul32(0x0e3812af+approx, f32)
	approx = mul32(0x3d7f2e74+approx, f32)
	approx = mul32(0xb1721b52+approx, f32)
	return approx - 00000007
}

func exp32neg(f32 uint32) uint32 {
	var approx uint32
	approx = -mul32(0x004995c4+approx, f32)
	approx = -mul32(0x026ed2ac+approx, f32)
	approx = -mul32(0x0e339a87+approx, f32)
	approx = -mul32(0x3d7f433e+approx, f32)
	approx = -mul32(0xb172158e+approx, f32)
	return approx - 00000005
}
