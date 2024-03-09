package refloat_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/sugawarayuuta/refloat"
)

func Fuzz(f *testing.F) {
	for _, test := range atoftests {
		f.Add(test.in, 64)
	}
	for _, test := range atof32tests {
		f.Add(test.in, 32)
	}
	f.Fuzz(func(t *testing.T, num string, size int) {
		fstd, estd := strconv.ParseFloat(num, size)
		fref, eref := refloat.ParseFloat(num, size)
		if !isFloat64(fstd, fref) || !isError(estd, eref) {
			bstd := math.Float64bits(fstd)
			bref := math.Float64bits(fref)
			t.Errorf("\nstd: %064b, %v\nref: %064b, %v\nParseFloat(%s, %d)\n", bstd, estd, bref, eref, num, size)
		}
	})
}

func isFloat64(l64, r64 float64) bool {
	return math.IsNaN(l64) || math.IsNaN(r64) || math.Float64bits(l64) == math.Float64bits(r64)
}

func isError(ler, rer error) bool {
	return (ler != nil) == (rer != nil)
}
