package render

import "math"

const mm100PerInch = 2540

func MmFloatTo100(mm float64) int {
	return int(math.Round(mm * 100))
}

func MulDiv(n, numerator, denominator int) int {
	if denominator == 0 {
		return 0
	}
	prod := int64(n) * int64(numerator)
	den := int64(denominator)
	neg := (prod < 0) != (den < 0)
	if prod < 0 {
		prod = -prod
	}
	if den < 0 {
		den = -den
	}
	q := (prod + den/2) / den
	if neg {
		return -int(q)
	}
	return int(q)
}

func Mm100ToPx(mm100, dpi int) int {
	return MulDiv(mm100, dpi, mm100PerInch)
}

func PxToMm100(px, dpi int) int {
	return MulDiv(px, mm100PerInch, dpi)
}
