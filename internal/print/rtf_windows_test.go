//go:build windows

package print

import (
	"syscall"
	"testing"
	"unsafe"
)

func TestFormatRTFToMemoryDC(t *testing.T) {
	displayName := syscall.StringToUTF16Ptr("DISPLAY")
	display, _, _ := procCreateDC.Call(uintptr(unsafe.Pointer(displayName)), 0, 0, 0)
	if display == 0 {
		t.Skip("DISPLAY DC unavailable")
	}
	defer procDeleteDC.Call(display)

	memDC, _, _ := procCreateCompatibleDC.Call(display)
	if memDC == 0 {
		t.Skip("compatible DC unavailable")
	}
	defer procDeleteDC.Call(memDC)

	bmp, _, _ := procCreateCompatibleBitmap.Call(display, 300, 100)
	if bmp == 0 {
		t.Skip("compatible bitmap unavailable")
	}
	defer procDeleteObject.Call(bmp)
	oldBmp, _, _ := procSelectObject.Call(memDC, bmp)
	defer procSelectObject.Call(memDC, oldBmp)

	rtf := []byte(`{\rtf1\ansi{\fonttbl{\f0 Arial;}}\pard\b\f0\fs20 FEITO NO BRASIL\par}`)
	if err := FormatRTFToHDC(memDC, rtf, 0, 0, 300, 100); err != nil {
		t.Fatal(err)
	}
}

func TestRTFPixelRectToTwips(t *testing.T) {
	if got := pixelToTwips(96, 96); got != 1440 {
		t.Fatalf("twips=%d want 1440", got)
	}
	if got := pixelToTwips(48, 96); got != 720 {
		t.Fatalf("twips=%d want 720", got)
	}
}
