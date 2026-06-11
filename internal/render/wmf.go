package render

import (
	"encoding/binary"
	"fmt"
)

const PlaceableWMFKey = 0x9ac6cdd7

func ParseWMFBounds(data []byte) (metaX, metaY, metaW, metaH int, metaData []byte, err error) {
	if len(data) < 22 {
		return 0, 0, 0, 0, nil, fmt.Errorf("arquivo WMF muito pequeno")
	}
	if binary.LittleEndian.Uint32(data[0:4]) == PlaceableWMFKey {
		left := int(int16(binary.LittleEndian.Uint16(data[6:8])))
		top := int(int16(binary.LittleEndian.Uint16(data[8:10])))
		right := int(int16(binary.LittleEndian.Uint16(data[10:12])))
		bottom := int(int16(binary.LittleEndian.Uint16(data[12:14])))
		metaW = right - left
		metaH = bottom - top
		if metaW <= 0 || metaH <= 0 {
			return 0, 0, 0, 0, nil, fmt.Errorf("dimensoes invalidas no cabecalho WMF")
		}
		return left, top, metaW, metaH, data[22:], nil
	}
	return 0, 0, 1000, 1000, data, nil
}

func CadMapaWMFPlayRectPx(xMM100, yMM100, widthMM100, heightMM100, dpiX, dpiY int) (left, top, right, bottom int) {
	left = Mm100ToPx(xMM100, dpiX)
	top = Mm100ToPx(yMM100, dpiY)
	right = left + Mm100ToPx(widthMM100, dpiX) - 1
	bottom = top + Mm100ToPx(heightMM100, dpiY) - 1
	return left, top, right, bottom
}

func CadMapaWMFPlayRectFromMM(xMM, yMM, widthMM, heightMM float64, dpiX, dpiY int) (left, top, right, bottom int) {
	return CadMapaWMFPlayRectPx(MmFloatTo100(xMM), MmFloatTo100(yMM), MmFloatTo100(widthMM), MmFloatTo100(heightMM), dpiX, dpiY)
}
