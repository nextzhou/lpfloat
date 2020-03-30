package lpfloat

import (
	"fmt"
	"sort"
)

type UnSyncBuckets struct {
	layers []f64BucketsLayer
}

type f64BucketsLayer struct {
	count      uint64
	sum        float64
	signAndExp int16
	buckets    [256]uint64
}

func (l *f64BucketsLayer) unit() float64 {
	return compose(l.signAndExp, _One.Fraction).ToFloat64()
}

func (b *UnSyncBuckets) Insert(f float64) {
	lpf := FromFloat64(f)
	for i := range b.layers {
		layer := &b.layers[i]
		if layer.signAndExp == lpf.SignAndExp {
			layer.count++
			layer.sum += f
			layer.buckets[lpf.Fraction]++
			return
		}
	}

	// cold path
	newLayer := f64BucketsLayer{signAndExp: lpf.SignAndExp}
	newLayer.buckets[lpf.Fraction]++
	newLayer.count++
	newLayer.sum += f
	b.layers = append(b.layers, newLayer)
	sort.Slice(b.layers, func(i, j int) bool {
		return b.layers[i].unit() < b.layers[j].unit()
	})
}

func (b *UnSyncBuckets) InsertN(f float64, count uint64) {
	lpf := FromFloat64(f)
	for i := range b.layers {
		layer := &b.layers[i]
		if layer.signAndExp == lpf.SignAndExp {
			layer.buckets[lpf.Fraction] += count
			layer.count += count
			layer.sum += float64(count) * f
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

func (b *UnSyncBuckets) Total() uint64 {
	total := uint64(0)
	for i := range b.layers {
		total += b.layers[i].count
	}
	return total
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

	total := b.Total()
	for i := range b.layers {
		layer := &b.layers[i]
		sum += layer.sum
		for fraction, count := range layer.buckets {
			if count == 0 {
				continue
			}

			lpf := compose(layer.signAndExp, uint8(fraction))
			if summary.Total == 0 {
				summary.Min = lpf
			}
			summary.Total += count
			if summary.Total == total {
				summary.Max = lpf
			}
			for percentileIdx < len(percentilesCfg) &&
				float64(summary.Total)*100 >= float64(total)*float64(percentilesCfg[percentileIdx]) {
				summary.Percentiles[percentileIdx].LessThan = lpf
				percentileIdx++
			}
		}
	}
	summary.Sum = FromFloat64(sum)
	summary.Avg = FromFloat64(sum / float64(summary.Total))
	return summary
}

func (b *UnSyncBuckets) Reset() {
	for i := range b.layers {
		layer := &b.layers[i]
		layer.buckets = emptyBuckets
		layer.count = 0
		layer.sum = 0
	}
}
