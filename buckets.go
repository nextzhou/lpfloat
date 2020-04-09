package lpfloat

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Buckets interface {
	Insert(float64)
	InsertN(float64, uint64)
	Total() uint64
	Sum() float64
	Count(float64) uint64
	Range(func(Bucket))
	ReverseRange(func(Bucket))
	Buckets() []Bucket
	Summary([]float32) Summary
	Reset()
}

var (
	_ Buckets = &UnSyncBuckets{}
	_ Buckets = &SyncBuckets{}

	emptyBuckets = [256]uint64{}
)

type Bucket struct {
	Value LPFloat
	Count uint64
}

type Summary struct {
	Min         LPFloat
	Max         LPFloat
	Avg         LPFloat
	Sum         LPFloat
	Total       uint64
	Percentiles []PercentilePair
}

func makeSummary(p []float32) Summary {
	s := Summary{
		Min:         _NaN,
		Max:         _NaN,
		Avg:         _NaN,
		Sum:         _Zero,
		Total:       0,
		Percentiles: make([]PercentilePair, len(p)),
	}
	for i := range p {
		s.Percentiles[i].Percentile = p[i]
		s.Percentiles[i].LessThan = _NaN
	}
	return s
}

func (s Summary) String() string {
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, "Summary{Total: %d, Sum: %v, Avg: %v, Max: %v, Min: %v, Percentiles: %v}",
		s.Total, s.Sum, s.Avg, s.Max, s.Min, s.Percentiles)
	return buf.String()
}

func (s Summary) Format(f fmt.State, c rune) {
	fmtCode := toFormatCode(f, c)
	fmtStr := "Summary{Total: %d, Sum: _CODE_, Avg: _CODE_, Max: _CODE_, Min: _CODE_, Percentiles: _CODE_}"
	fmtStr = strings.Replace(fmtStr, "_CODE_", fmtCode, -1)
	_, _ = f.Write([]byte(fmt.Sprintf(fmtStr, s.Total, s.Sum, s.Avg, s.Max, s.Min, s.Percentiles)))
}

type PercentilePair struct {
	Percentile float32 // [0, 100]
	LessThan   LPFloat
}

func (p PercentilePair) String() string {
	return fmt.Sprintf("P%g: %v", p.Percentile, p.LessThan)
}

func (p PercentilePair) Format(f fmt.State, c rune) {
	fmtStr := "P%g: " + toFormatCode(f, c)
	_, _ = f.Write([]byte(fmt.Sprintf(fmtStr, p.Percentile, p.LessThan)))
}

func DefaultPercentilesCfg() []float32 {
	defaultCfg := []float32{50, 80, 90, 95, 99, 99.9}
	ret := make([]float32, len(defaultCfg))
	copy(ret, defaultCfg)
	return ret
}

func CheckPercentilesCfg(cfg []float32) error {
	if len(cfg) == 0 {
		return nil
	}
	sort.Slice(cfg, func(i, j int) bool {
		return cfg[i] < cfg[j]
	})
	if cfg[0] <= 0 || cfg[len(cfg)-1] >= 100 {
		return errors.New("the percentile should be between 0 and 100")
	}
	return nil
}
