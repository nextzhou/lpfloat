package lpfloat

import (
	"fmt"
	"sort"
)

type UnSyncBuckets struct {
	layers []f64BucketsLayer
	total  uint64
}

type f64BucketsLayer struct {
	signAndExp int16
	buckets    [256]uint64
}

func (l *f64BucketsLayer) unit() float64 {
	return compose(l.signAndExp, _One.Fraction).ToFloat64()
}

func (b *UnSyncBuckets) Insert(f float64) {
	b.total++
	lpf := FromFloat64(f)
	for i := range b.layers {
		if b.layers[i].signAndExp == lpf.SignAndExp {
			b.layers[i].buckets[lpf.Fraction]++
			return
		}
	}

	// cold path
	newLayer := f64BucketsLayer{signAndExp: lpf.SignAndExp}
	newLayer.buckets[lpf.Fraction]++
	b.layers = append(b.layers, newLayer)
	sort.Slice(b.layers, func(i, j int) bool {
		return b.layers[i].unit() < b.layers[j].unit()
	})
}

func (b *UnSyncBuckets) InsertN(f float64, count uint64) {
	b.total += count
	lpf := FromFloat64(f)
	for i := range b.layers {
		if b.layers[i].signAndExp == lpf.SignAndExp {
			b.layers[i].buckets[lpf.Fraction] += count
			return
		}
	}

	// cold path
	newLayer := f64BucketsLayer{signAndExp: lpf.SignAndExp}
	newLayer.buckets[lpf.Fraction] += count
	b.layers = append(b.layers, newLayer)
	sort.Slice(b.layers, func(i, j int) bool {
		return b.layers[i].unit() < b.layers[j].unit()
	})
}

func (b *UnSyncBuckets) Remove(f float64) {
	lpf := FromFloat64(f)
	for i := range b.layers {
		layer := &b.layers[i]
		if layer.signAndExp != lpf.SignAndExp {
			continue
		}
		count := layer.buckets[lpf.Fraction]
		if count > 0 {
			b.total--
			layer.buckets[lpf.Fraction]--
		}
	}
}

func (b *UnSyncBuckets) RemoveN(f float64, count uint64) {
	lpf := FromFloat64(f)
	for i := range b.layers {
		layer := &b.layers[i]
		if layer.signAndExp != lpf.SignAndExp {
			continue
		}
		c := layer.buckets[lpf.Fraction]
		if c >= count {
			b.total -= count
			layer.buckets[lpf.Fraction] -= count
		}
	}
}

func (b *UnSyncBuckets) Total() uint64 {
	return b.total
}

func (b *UnSyncBuckets) Count(f float64) uint64 {
	lpf := FromFloat64(f)
	for i := range b.layers {
		layer := &b.layers[i]
		if layer.signAndExp != lpf.SignAndExp {
			continue
		}
		return layer.buckets[lpf.Fraction]
	}
	return 0
}

func (b *UnSyncBuckets) Range(do func(Bucket)) {
	for i := range b.layers {
		layer := &b.layers[i]
		for fraction, count := range layer.buckets {
			if count == 0 {
				continue
			}
			do(Bucket{Value: compose(layer.signAndExp, uint8(fraction)), Count: count})
		}
	}
}

func (b *UnSyncBuckets) ReverseRange(do func(Bucket)) {
	layersLen := len(b.layers)
	for i := range b.layers {
		layer := &b.layers[layersLen-i-1]
		for i := range layer.buckets {
			fraction := 0xff - i
			count := layer.buckets[fraction]
			if count == 0 {
				continue
			}
			do(Bucket{Value: compose(layer.signAndExp, uint8(fraction)), Count: count})
		}
	}
}

func (b *UnSyncBuckets) Buckets() []Bucket {
	var buckets []Bucket
	b.Range(func(bucket Bucket) {
		buckets = append(buckets, bucket)
	})
	return buckets
}

func (b *UnSyncBuckets) Summary(percentilesCfg []float32) Summary {
	if percentilesCfg == nil {
		percentilesCfg = DefaultPercentilesCfg()
	}
	if err := CheckPercentilesCfg(percentilesCfg); err != nil {
		panic(fmt.Errorf("invalid percentiles cfg %v: %s", percentilesCfg, err))
	}

	summary := makeSummary(percentilesCfg)
	var sum float64
	var percentileIdx int

	for i := range b.layers {
		layer := &b.layers[i]
		for fraction, count := range layer.buckets {
			if count == 0 {
				continue
			}

			lpf := compose(layer.signAndExp, uint8(fraction))
			if summary.Total == 0 {
				summary.Min = lpf
			}
			summary.Total += count
			sum += lpf.ToFloat64() * float64(count)
			if summary.Total == b.total {
				summary.Max = lpf
			}
			for percentileIdx < len(percentilesCfg) &&
				float64(summary.Total)*100 >= float64(b.total)*float64(percentilesCfg[percentileIdx]) {
				summary.Percentiles[percentileIdx].LessThan = lpf
				percentileIdx++
			}
		}
	}
	summary.Sum = FromFloat64(sum)
	summary.Avg = FromFloat64(sum / float64(summary.Total))
	return summary
}
