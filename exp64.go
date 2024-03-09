package refloat

import "math/bits"

// the polynomial is generated using "sollya" (https://www.sollya.org/)
// sollya is licensed under CeCILL-C.
//
// command used (replace two x with -x for exp64neg):
// prec = 256;
// display = hexadecimal;
// poly = remez(2^x, 10, [0;0.5]);
// print(poly);
// norm = supnorm(poly, 2^x, [0;0.5], absolute, 2^(-64));
// print(norm);
// quit;

func exp64pos(f64 uint64) uint64 {
	var approx uint64
	approx = mul64(0x000000240f7385cf+approx, f64)
	approx = mul64(0x000001ae189374a9+approx, f64)
	approx = mul64(0x00001630cfccd8b0+approx, f64)
	approx = mul64(0x0000ffe3f8fcc2be+approx, f64)
	approx = mul64(0x000a1849231599e0+approx, f64)
	approx = mul64(0x005761ff8619c241+approx, f64)
	approx = mul64(0x0276556df9e057dc+approx, f64)
	approx = mul64(0x0e35846b8226d4b6+approx, f64)
	approx = mul64(0x3d7f7bff058c732a+approx, f64)
	approx = mul64(0xb17217f7d1cf7562+approx, f64)
	return approx + 0x0000000000000002
}

func exp64neg(f64 uint64) uint64 {
	var approx uint64
	approx = -mul64(0x000000197f9f2cf6+approx, f64)
	approx = -mul64(0x000001af9dcd1e57+approx, f64)
	approx = -mul64(0x000016285b20389a+approx, f64)
	approx = -mul64(0x0000ffe47c526796+approx, f64)
	approx = -mul64(0x000a1848312c5dfa+approx, f64)
	approx = -mul64(0x005761ff8c9de4a9+approx, f64)
	approx = -mul64(0x0276556df56a7024+approx, f64)
	approx = -mul64(0x0e35846b82328319+approx, f64)
	approx = -mul64(0x3d7f7bff058a2901+approx, f64)
	approx = -mul64(0xb17217f7d1cf76a1+approx, f64)
	return approx - 0x0000000000000002
}

func mul64(f64, mul uint64) uint64 {
	prod, _ := bits.Mul64(f64, mul)
	return prod
}

func normalize64(f64 uint64) (uint64, uint64, uint64) {
	// top 128bits of the fraction in log_2(5).
	// the integer part 2 is hardcoded into the parser itself.
	lo := mul64(0x24afdbfd36bf6d33, f64)
	hi, md := bits.Mul64(0x5269e12f346e2bf9, f64)
	lo, carry := bits.Add64(md, lo, 0)
	return hi + carry, lo, lo + 1
}
