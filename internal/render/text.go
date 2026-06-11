package render

const (
	DrawTextLeft      = 0x0000
	DrawTextCenter    = 0x0001
	DrawTextRight     = 0x0002
	DrawTextWordBreak = 0x0010
	DrawTextRTL       = 0x0100
	DrawTextNoPrefix  = 0x0800
)

func CadMapaDrawTextFlags(align int16, rtl bool) uint32 {
	flags := uint32(DrawTextNoPrefix | DrawTextWordBreak | DrawTextLeft)
	switch align {
	case 1:
		flags |= DrawTextCenter
	case 2:
		flags |= DrawTextRight
	}
	if rtl {
		flags |= DrawTextRTL
	}
	return flags
}

func CadMapaFontHeightPx(rectHeightMM100, dpiY int) int {
	return Mm100ToPx(rectHeightMM100, dpiY)
}

func CadMapaTextRectPx(xMM100, yMM100, widthMM100, heightMM100, dpiX, dpiY int) (left, top, right, bottom int) {
	left = Mm100ToPx(xMM100, dpiX)
	top = Mm100ToPx(yMM100, dpiY)
	right = left + Mm100ToPx(widthMM100, dpiX)
	bottom = top + Mm100ToPx(heightMM100, dpiY)
	return left, top, right, bottom
}

func CadMapaObjectRectPxFromMM(xMM, yMM, widthMM, heightMM float64, dpiX, dpiY int) (left, top, right, bottom int) {
	return CadMapaTextRectPx(MmFloatTo100(xMM), MmFloatTo100(yMM), MmFloatTo100(widthMM), MmFloatTo100(heightMM), dpiX, dpiY)
}
