//go:build windows

package render

import (
	"math"
	"syscall"
	"unsafe"
)

const (
	cadMapaCharWidthSeedFactor = 0.8
	cadMapaFWNormal            = 400
	cadMapaFWBold              = 700
	cadMapaDefaultCharset      = 1
	cadMapaOutTTPrecision      = 4
	cadMapaClipDefaultPrecis   = 0
	cadMapaProofQuality        = 2
	cadMapaDefaultPitch        = 0
)

var (
	procCreateCompatibleDCR    = modGDI32Render.NewProc("CreateCompatibleDC")
	procDeleteDCR              = modGDI32Render.NewProc("DeleteDC")
	procCreateFontAR           = modGDI32Render.NewProc("CreateFontA")
	procSelectObjectR          = modGDI32Render.NewProc("SelectObject")
	procDeleteObjectR          = modGDI32Render.NewProc("DeleteObject")
	procGetTextExtentPoint32AR = modGDI32Render.NewProc("GetTextExtentPoint32A")
)

type gdiSize struct {
	CX int32
	CY int32
}

func CadMapaGDIStyle(style byte) (bold, italic, underline, strikeout bool) {
	return style&0x01 != 0, style&0x02 != 0, style&0x04 != 0, style&0x08 != 0
}

func CadMapaFitCharWidthPx(hdc uintptr, text string, rectW, rectH int, face string, style byte, escapement int16) int {
	textBytes := CadMapaANSIBytes(text)
	if len(textBytes) == 0 || rectW <= 0 || rectH <= 0 {
		return 0
	}
	seed := int(math.Round((float64(rectW) / float64(len(textBytes))) * cadMapaCharWidthSeedFactor))
	if seed <= 0 {
		return seed
	}

	memDC, _, _ := procCreateCompatibleDCR.Call(hdc)
	if memDC == 0 {
		return 0
	}
	defer procDeleteDCR.Call(memDC)

	candidate := seed
	direction := 0
	lastGood := seed
	for candidate > 0 {
		textW, ok := cadMapaMeasureTextWidth(memDC, textBytes, rectH, candidate, face, style, escapement)
		if !ok {
			return 0
		}
		if direction == 1 && rectW < textW {
			return lastGood
		}
		lastGood = candidate
		if rectW < textW {
			direction = -1
		} else {
			direction = 1
		}
		candidate += direction
	}
	return seed
}

func cadMapaMeasureTextWidth(hdc uintptr, textBytes []byte, height, width int, face string, style byte, escapement int16) (int, bool) {
	bold, italic, underline, strikeout := CadMapaGDIStyle(style)
	weight := cadMapaFWNormal
	if bold {
		weight = cadMapaFWBold
	}
	faceBytes, err := syscall.BytePtrFromString(face)
	if err != nil {
		return 0, false
	}
	hFont, _, _ := procCreateFontAR.Call(
		uintptr(height), uintptr(width), uintptr(int32(escapement)), 0,
		uintptr(weight), boolUintptr(italic), boolUintptr(underline), boolUintptr(strikeout),
		cadMapaDefaultCharset, cadMapaOutTTPrecision, cadMapaClipDefaultPrecis, cadMapaProofQuality,
		cadMapaDefaultPitch, uintptr(unsafe.Pointer(faceBytes)),
	)
	if hFont == 0 {
		return 0, false
	}
	defer procDeleteObjectR.Call(hFont)

	oldFont, _, _ := procSelectObjectR.Call(hdc, hFont)
	defer procSelectObjectR.Call(hdc, oldFont)

	var size gdiSize
	ret, _, _ := procGetTextExtentPoint32AR.Call(hdc, uintptr(unsafe.Pointer(&textBytes[0])), uintptr(len(textBytes)), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return 0, false
	}
	return int(size.CX), true
}

func CadMapaANSIBytes(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r <= 0xff {
			out = append(out, byte(r))
		} else {
			out = append(out, '?')
		}
	}
	return out
}

func boolUintptr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}
