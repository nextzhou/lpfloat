package lpfloat

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

type (
	t64bits = uint64
)

const (
	signExpMask  = 0xfff0000000000000
	fractionMask = 0x000ff00000000000

	signExpShift  = 48
	fractionShift = 44
)

// LPFloat means low precision float
type LPFloat struct {
	SignAndExp int16 // 12bits
	Fraction   uint8
}

var (
	_ fmt.Stringer     = LPFloat{}
	_ fmt.GoStringer   = LPFloat{}
	_ fmt.Formatter    = LPFloat{}
	_ json.Marshaler   = LPFloat{}
	_ json.Unmarshaler = &LPFloat{}
)

func FromFloat64(f float64) LPFloat {
	var lp LPFloat
	bits := math.Float64bits(f)
	lp.SignAndExp = int16((bits & signExpMask) >> signExpShift)
	lp.Fraction = uint8((bits & fractionMask) >> fractionShift)
	return lp
}

func compose(signAndExp int16, fraction uint8) LPFloat {
	return LPFloat{
		SignAndExp: signAndExp,
		Fraction:   fraction,
	}
}

func (f LPFloat) ToFloat64() float64 {
	var bits t64bits
	bits |= t64bits(f.SignAndExp) << signExpShift
	bits |= t64bits(f.Fraction) << fractionShift
	return math.Float64frombits(bits)
}

func (f LPFloat) String() string {
	return strconv.FormatFloat(f.ToFloat64(), 'g', -1, 64)
}

func (f LPFloat) GoString() string {
	return f.String()
}

func (f LPFloat) Format(s fmt.State, c rune) {
	_, _ = s.Write([]byte(fmt.Sprintf(toFormatCode(s, c), f.ToFloat64())))
}

func toFormatCode(s fmt.State, c rune) string {
	fmtCode := "%"
	if s.Flag('+') {
		fmtCode += "+"
	}
	if s.Flag('-') {
		fmtCode += "-"
	}
	if s.Flag('0') {
		fmtCode += "0"
	}
	if s.Flag('#') {
		fmtCode += "#"
	}
	if s.Flag('+') {
		fmtCode += "+"
	}
	if s.Flag(' ') {
		fmtCode += " "
	}
	if width, ok := s.Width(); ok {
		fmtCode += strconv.Itoa(width)
	}
	if precision, ok := s.Precision(); ok {
		fmtCode += "." + strconv.Itoa(precision)
	}
	fmtCode += string(c)
	return fmtCode
}

func (f LPFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.ToFloat64())
}

func (f *LPFloat) UnmarshalJSON(data []byte) error {
	var n float64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*f = FromFloat64(n)
	return nil
}
