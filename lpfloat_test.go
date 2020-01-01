package lpfloat

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"
	"unsafe"
)

func TestLPFloatSpecialValue(t *testing.T) {
	specialValues := []float64{0, -0, 1, math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, val := range specialValues {
		lpf := FromFloat64(val)
		if math.IsNaN(val) && math.IsNaN(lpf.ToFloat64()) {
			continue
		}
		if val != lpf.ToFloat64() {
			t.Errorf("expected %f, actual %f", val, lpf)
		}
	}
}

func TestFloatFormat(t *testing.T) {
	codes := []string{"b", "f", "F", "g", "G", "e", "E", "x", "X", "v"}
	flags := []string{"", "+", "-", " ", "0", "#"}
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 10; i++ {
		lpf := FromFloat64(rand.NormFloat64())

		expected := fmt.Sprint(lpf.ToFloat64())
		actual := fmt.Sprint(lpf)
		if expected != actual {
			t.Fatalf("print, expected %s, actual %s", expected, actual)
		}
		for _, code := range codes {
			for width := 0; width < 100; width++ {
				for fraction := 0; fraction < 100; fraction++ {
					for _, flag := range flags {
						fmtCode := "%" + flag
						if width > 0 {
							fmtCode += strconv.Itoa(width)
						}
						if fraction > 0 {
							fmtCode += "." + strconv.Itoa(fraction)
						}
						fmtCode += code

						expected := fmt.Sprintf(fmtCode, lpf.ToFloat64())
						actual := fmt.Sprintf(fmtCode, lpf)
						if expected != actual {
							t.Fatalf("%s format, expected %s, actual %s", code, expected, actual)
						}
					}
				}
			}
		}
	}

}

func BenchmarkLPFloatAlmostEqual(b *testing.B) {
	rand.Seed(int64(time.Now().Nanosecond()))
	for i := 0; i < b.N; i++ {
		f := float64(i) * rand.Float64()
		lpf := FromFloat64(f)
		if math.Abs(lpf.ToFloat64()-f)/f >= (1.0 / 256.0) {
			b.Logf("%g, %g, %g%%", f, lpf, 100*(lpf.ToFloat64()-f)/f)
			b.Fatalf("%016x, %06x, %02x", *(*uint64)(unsafe.Pointer(&f)), lpf.SignAndExp, lpf.Fraction)
		}
	}
}

func BenchmarkFromFloat64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = FromFloat64(float64(i))
	}
}

func BenchmarkToFloat64(b *testing.B) {
	n := FromFloat64(rand.NormFloat64())
	for i := 0; i < b.N; i++ {
		_ = n.ToFloat64()
	}
}

func BenchmarkUnSyncBuckets_Insert(b *testing.B) {
	var buckets UnSyncBuckets
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buckets.Insert(float64(i))
	}
	b.StopTimer()
	b.Logf("%.3g", buckets.Summary(DefaultPercentilesCfg()))
}
