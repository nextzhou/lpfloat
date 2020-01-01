package lpfloat

import "math"

var (
	_Zero   = FromFloat64(0.0)
	_One    = FromFloat64(1.0)
	_NaN    = FromFloat64(math.NaN())
	_PosInf = FromFloat64(math.Inf(1))
	_NegInf = FromFloat64(math.Inf(-1))
)

func Zero() LPFloat {
	return _Zero
}

func One() LPFloat {
	return _One
}

func NaN() LPFloat {
	return _NaN
}

func Inf(sign int) LPFloat {
	if sign >= 0 {
		return _PosInf
	}
	return _NegInf
}

func PosInf() LPFloat {
	return _PosInf
}

func NegInf() LPFloat {
	return _NegInf
}
