package lpfloat

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
)

type SyncBuckets struct {
	m      sync.RWMutex
	layers []f64BucketsLayer
	total  uint64
}

func (b *SyncBuckets) Insert(f float64) {
	b.InsertN(f, 1)
}

func (b *SyncBuckets) InsertN(f float64, count uint64) {
	lpf := FromFloat64(f)
	b.m.RLock()
	for i := range b.layers {
		if b.layers[i].signAndExp == lpf.SignAndExp {
			atomic.AddUint64(&b.total, count)
			atomic.AddUint64(&b.layers[i].buckets[lpf.Fraction], count)
			b.m.RUnlock()
			return
		}
	}

	// cold path
	b.m.RUnlock()
	b.m.Lock()
	atomic.AddUint64(&b.total, count)
	for i := range b.layers {
		if b.layers[i].signAndExp == lpf.SignAndExp {
			atomic.AddUint64(&b.layers[i].buckets[lpf.Fraction], count)
			b.m.Unlock()
			return
		}
	}

	newLayer := f64BucketsLayer{signAndExp: lpf.SignAndExp}
	newLayer.buckets[lpf.Fraction] = count
	b.layers = append(b.layers, newLayer)
	sort.Slice(b.layers, func(i, j int) bool {
		return b.layers[i].unit() < b.layers[j].unit()
	})
	b.m.Unlock()
}

func (b *SyncBuckets) Total() uint64 {
	b.m.RLock()
	n := atomic.LoadUint64(&b.total)
	b.m.RUnlock()
	return n
}

func (b *SyncBuckets) Count(f float64) uint64 {
	lpf := FromFloat64(f)
	b.m.RLock()
	for i := range b.layers {
		layer := &b.layers[i]
		if layer.signAndExp != lpf.SignAndExp {
			continue
		}
		count := atomic.LoadUint64(&layer.buckets[lpf.Fraction])
		b.m.RUnlock()
		return count
	}
	b.m.RUnlock()
	return 0
}

func (b *SyncBuckets) Range(do func(Bucket)) {
	//  locks writing to ensure consistency
	b.m.Lock()
	defer b.m.Unlock()

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

func (b *SyncBuckets) ReverseRange(do func(Bucket)) {
	//  locks writing to ensure consistency
	b.m.Lock()
	defer b.m.Unlock()

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

func (b *SyncBuckets) Buckets() []Bucket {
	var buckets []Bucket
	b.Range(func(bucket Bucket) {
		buckets = append(buckets, bucket)
	})
	return buckets
}

func (b *SyncBuckets) Summary(percentilesCfg []float32) Summary {
	if percentilesCfg == nil {
		percentilesCfg = DefaultPercentilesCfg()
	}
	if err := CheckPercentilesCfg(percentilesCfg); err != nil {
		panic(fmt.Errorf("invalid percentiles cfg %v: %s", percentilesCfg, err))
	}

	summary := makeSummary(percentilesCfg)
	var sum float64
	var percentileIdx int

	//  locks writing to ensure consistency
	b.m.Lock()
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
	b.m.Unlock()

	summary.Sum = FromFloat64(sum)
	summary.Avg = FromFloat64(sum / float64(summary.Total))
	return summary
}
