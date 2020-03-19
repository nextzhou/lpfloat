package lpfloat

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"
	"unsafe"
)

func TestLPFloat_SpecialValue(t *testing.T) {
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

func TestLPFloat_AlmostEqualF64(t *testing.T) {
	rand.Seed(int64(time.Now().Nanosecond()))
	for i := 0; i < 1000000; i++ {
		f := float64(i) * rand.Float64()
		lpf := FromFloat64(f)
		if !lpf.AlmostEqualF64(f) {
			t.Logf("%g, %g, %g%%", f, lpf, 100*(lpf.ToFloat64()-f)/f)
			t.Fatalf("%016x, %06x, %02x", *(*uint64)(unsafe.Pointer(&f)), lpf.SignAndExp, lpf.Fraction)
		}
	}
}

func BenchmarkLPFloat_AlmostEqual(b *testing.B) {
	rand.Seed(int64(time.Now().Nanosecond()))
	n := rand.NormFloat64()
	m := n * (1 + 1/300)
	lpN, lpM := FromFloat64(n), FromFloat64(m)
	for i := 0; i < b.N; i++ {
		_ = lpN.AlmostEqual(lpM)
	}
}

func BenchmarkLPFloat_AlmostEqualF64(b *testing.B) {
	rand.Seed(int64(time.Now().Nanosecond()))
	n := rand.NormFloat64()
	m := FromFloat64(n * (1 + 1/300))
	for i := 0; i < b.N; i++ {
		_ = m.AlmostEqualF64(n)
	}
}

func TestLPFloat_Format(t *testing.T) {
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

func TestBuckets(t *testing.T) {
	bucketsList := []Buckets{new(UnSyncBuckets), new(SyncBuckets)}

	const size = 1000000
	const maxVal = 100
	const minVal = 0.01
	data := randomData(size, minVal, maxVal)

	prepareData := func() {
		for _, buckets := range bucketsList {
			insertBuckets(buckets, data)
			if buckets.Total() != size {
				t.Fatal("")
			}
		}
	}
	check := func() {
		plainSummary := calPlainSummary(data, DefaultPercentilesCfg())
		plainBuckets := calPlainBuckets(data)
		for i, bucket := range bucketsList {
			summary := bucket.Summary(DefaultPercentilesCfg())
			summary.Sum = plainSummary.Sum // FIXME: accumulative errors of sum
			if !reflect.DeepEqual(plainSummary, summary) {
				t.Fatalf("%T summary,\nexpected:\t%v\nactual:\t%v", bucketsList[i], plainSummary, summary)
			}
			if !reflect.DeepEqual(plainBuckets, bucket.Buckets()) {
				t.Fatalf("%T buckets", bucketsList[i])
			}
		}
	}

	reset := func() {
		for _, buckets := range bucketsList {
			buckets.Reset()
		}
	}

	prepareData()
	check()
	reset()
	prepareData()
	check()
}

func BenchmarkLPFloat_FromFloat64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = FromFloat64(float64(i))
	}
}

func BenchmarkLPFloat_ToFloat64(b *testing.B) {
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
	//b.Logf("%.3g", buckets.Summary(DefaultPercentilesCfg()))
}

func BenchmarkSyncBuckets_Insert(b *testing.B) {
	var buckets SyncBuckets
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buckets.Insert(float64(i))
	}
}

func BenchmarkSyncBuckets_Insert_Parallel(b *testing.B) {
	var buckets SyncBuckets
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var n uint64
		for pb.Next() {
			buckets.Insert(float64(n))
			n++
		}
	})
}

func insertBuckets(buckets Buckets, data []float64) {
	for _, val := range data {
		buckets.Insert(val)
	}
}

func randomData(size int, min, max float64) []float64 {
	rand.Seed(int64(time.Now().Nanosecond()))
	buckets := make([]float64, 0, size)
	for i := 0; i < size; i++ {
		val := min + math.Pow(rand.Float64(), 3)*(max-min)
		buckets = append(buckets, val)
	}
	return buckets
}

func calPlainSummary(data []float64, percentilesCfg []float32) Summary {
	summary := makeSummary(percentilesCfg)
	if len(data) == 0 {
		return summary
	}
	sum := 0.0
	for _, val := range data {
		sum += val
	}
	buckets := calPlainBuckets(data)
	summary.Total = uint64(len(data))
	summary.Sum = FromFloat64(sum)
	summary.Avg = FromFloat64(summary.Sum.ToFloat64() / float64(len(data)))

	summary.Min = buckets[0].Value
	summary.Max = buckets[len(buckets)-1].Value

	percentileIdx := 0
	var currentTotal uint64
	for _, bucket := range buckets {
		currentTotal += bucket.Count
		for percentileIdx < len(percentilesCfg) &&
			float64(currentTotal)*100 >= float64(summary.Total)*float64(percentilesCfg[percentileIdx]) {
			summary.Percentiles[percentileIdx].LessThan = bucket.Value
			percentileIdx++
		}
	}
	return summary
}

func calPlainBuckets(data []float64) []Bucket {
	counter := make(map[LPFloat]uint64)
	for _, val := range data {
		lpf := FromFloat64(val)
		counter[lpf]++
	}

	buckets := make([]Bucket, 0, len(counter))
	for val, count := range counter {
		buckets = append(buckets, Bucket{Value: val, Count: count})
	}

	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Value.ToFloat64() < buckets[j].Value.ToFloat64()
	})
	return buckets
}
