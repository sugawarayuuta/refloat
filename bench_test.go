package refloat_test

import (
	"math"
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/sugawarayuuta/refloat"
)

type (
	float64String struct {
		out float64
		inp string
	}
	float32String struct {
		out float32
		inp string
	}
)

var (
	once       sync.Once
	randbits64 []float64String
	randnorm64 []float64String
	randbits32 []float32String
	randnorm32 []float32String
)

func initOnce() {
	if testing.Short() {
		randbits64 = make([]float64String, 1e2)
		randnorm64 = make([]float64String, 1e2)
		randbits32 = make([]float32String, 1e2)
		randnorm32 = make([]float32String, 1e2)
	} else {
		randbits64 = make([]float64String, 1e4)
		randnorm64 = make([]float64String, 1e4)
		randbits32 = make([]float32String, 1e4)
		randnorm32 = make([]float32String, 1e4)
	}
	for idx := range randbits64 {
		f64 := math.Float64frombits(rand.Uint64())
		randbits64[idx] = float64String{
			inp: strconv.FormatFloat(f64, 'g', -1, 64),
			out: f64,
		}
	}
	for idx := range randbits32 {
		f32 := math.Float32frombits(rand.Uint32())
		randbits32[idx] = float32String{
			inp: strconv.FormatFloat(float64(f32), 'g', -1, 32),
			out: f32,
		}
	}
	for idx := range randnorm64 {
		f64 := rand.NormFloat64()
		randnorm64[idx] = float64String{
			inp: strconv.FormatFloat(f64, 'g', -1, 64),
			out: f64,
		}
	}
	for idx := range randnorm32 {
		f32 := float32(rand.NormFloat64())
		randnorm32[idx] = float32String{
			inp: strconv.FormatFloat(float64(f32), 'g', -1, 32),
			out: f32,
		}
	}
}

func benchmarkParseFloat64(b *testing.B, fnc func(string, int) (float64, error), tab []float64String) {
	for try := 0; try < b.N; try++ {
		ent := tab[try%len(tab)]
		f64, err := fnc(ent.inp, 64)
		if err != nil {
			b.Fatal(ent.inp, err)
		}
		if f64 == f64 && ent.out == ent.out && f64 != ent.out {
			b.Fatal(ent.inp, f64)
		}
	}
}

func benchmarkParseFloat32(b *testing.B, fnc func(string, int) (float64, error), tab []float32String) {
	for try := 0; try < b.N; try++ {
		ent := tab[try%len(tab)]
		f64, err := fnc(ent.inp, 32)
		if err != nil {
			b.Fatal(ent.inp, err)
		}
		f32 := float32(f64)
		if f32 == f32 && ent.out == ent.out && f32 != ent.out {
			b.Fatal(ent.inp, f32)
		}
	}
}

func BenchmarkParseFloat64(b *testing.B) {
	once.Do(initOnce)
	b.ResetTimer()
	b.Run("strconv/bits", func(b *testing.B) {
		benchmarkParseFloat64(b, strconv.ParseFloat, randbits64)
	})
	b.Run("strconv/norm", func(b *testing.B) {
		benchmarkParseFloat64(b, strconv.ParseFloat, randnorm64)
	})
	b.Run("refloat/bits", func(b *testing.B) {
		benchmarkParseFloat64(b, refloat.ParseFloat, randbits64)
	})
	b.Run("refloat/norm", func(b *testing.B) {
		benchmarkParseFloat64(b, refloat.ParseFloat, randnorm64)
	})
}

func BenchmarkParseFloat32(b *testing.B) {
	once.Do(initOnce)
	b.ResetTimer()
	b.Run("strconv/bits", func(b *testing.B) {
		benchmarkParseFloat32(b, strconv.ParseFloat, randbits32)
	})
	b.Run("strconv/norm", func(b *testing.B) {
		benchmarkParseFloat32(b, strconv.ParseFloat, randnorm32)
	})
	b.Run("refloat/bits", func(b *testing.B) {
		benchmarkParseFloat32(b, refloat.ParseFloat, randbits32)
	})
	b.Run("refloat/norm", func(b *testing.B) {
		benchmarkParseFloat32(b, refloat.ParseFloat, randnorm32)
	})
}

func BenchmarkParseFloat64Special(b *testing.B) {
	b.Run("strconv", func(b *testing.B) {
		benchmarkParseFloat64(b, strconv.ParseFloat, []float64String{
			{inp: "NaN", out: math.NaN()},
			{inp: "nan", out: math.NaN()},
			{inp: "Inf", out: math.Inf(1)},
			{inp: "inf", out: math.Inf(1)},
			{inp: "+Infinity", out: math.Inf(1)},
			{inp: "-inf", out: math.Inf(-1)},
		})
	})
	b.Run("refloat", func(b *testing.B) {
		benchmarkParseFloat64(b, refloat.ParseFloat, []float64String{
			{inp: "NaN", out: math.NaN()},
			{inp: "nan", out: math.NaN()},
			{inp: "Inf", out: math.Inf(1)},
			{inp: "inf", out: math.Inf(1)},
			{inp: "+Infinity", out: math.Inf(1)},
			{inp: "-inf", out: math.Inf(-1)},
		})
	})
}

func BenchmarkParseFloat32Special(b *testing.B) {
	b.Run("strconv", func(b *testing.B) {
		benchmarkParseFloat32(b, strconv.ParseFloat, []float32String{
			{inp: "NaN", out: float32(math.NaN())},
			{inp: "nan", out: float32(math.NaN())},
			{inp: "Inf", out: float32(math.Inf(1))},
			{inp: "inf", out: float32(math.Inf(1))},
			{inp: "+Infinity", out: float32(math.Inf(1))},
			{inp: "-inf", out: float32(math.Inf(-1))},
		})
	})
	b.Run("refloat", func(b *testing.B) {
		benchmarkParseFloat32(b, refloat.ParseFloat, []float32String{
			{inp: "NaN", out: float32(math.NaN())},
			{inp: "nan", out: float32(math.NaN())},
			{inp: "Inf", out: float32(math.Inf(1))},
			{inp: "inf", out: float32(math.Inf(1))},
			{inp: "+Infinity", out: float32(math.Inf(1))},
			{inp: "-inf", out: float32(math.Inf(-1))},
		})
	})
}
