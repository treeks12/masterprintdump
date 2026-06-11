//go:build windows

package print

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"masterprint-native/internal/model"
	"masterprint-native/internal/render"
)

const (
	emStreamIn    = 0x0449
	emFormatRange = 0x0439
	sfRTF         = 0x0002
	wsPopup       = 0x80000000
	esMultiline   = 0x0004
	esReadOnly    = 0x0800
	twipsPerInch  = 1440
)

var (
	procCreateWindowEx = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow  = modUser32.NewProc("DestroyWindow")
	procSendMessage    = modUser32.NewProc("SendMessageW")

	richEditLoadOnce sync.Once
	richEditLoadErr  error
	rtfStreamCB      = syscall.NewCallback(rtfStreamCallback)
)

type charRange struct {
	CpMin int32
	CpMax int32
}

type formatRange struct {
	HDC       uintptr
	HDCTarget uintptr
	RC        RECT
	RCPage    RECT
	Chrg      charRange
}

type editStream struct {
	Cookie   uintptr
	Error    uint32
	Callback uintptr
}

type rtfStreamState struct {
	Data []byte
	Off  int
}

func (lp *LabelPrinter) drawRTFElement(baseXMM, baseYMM float64, t model.TextElement) error {
	left, top, right, bottom := render.CadMapaObjectRectPxFromMM(baseXMM+t.XMM, baseYMM+t.YMM, t.WidthMM, t.HeightMM, lp.dpiX, lp.dpiY)
	return FormatRTFToHDC(uintptr(lp.hDC), t.RTFRaw, left, top, right, bottom)
}

func FormatRTFToHDC(hdc uintptr, raw []byte, left, top, right, bottom int) error {
	if hdc == 0 {
		return fmt.Errorf("HDC invalido")
	}
	raw = bytes.TrimRight(raw, "\x00")
	if len(raw) == 0 {
		return nil
	}
	if right <= left || bottom <= top {
		return fmt.Errorf("retangulo RTF invalido")
	}
	hwnd, err := createRichEditWindow()
	if err != nil {
		return err
	}
	defer procDestroyWindow.Call(hwnd)

	state := &rtfStreamState{Data: raw}
	stream := editStream{Cookie: uintptr(unsafe.Pointer(state)), Callback: rtfStreamCB}
	ret, _, callErr := procSendMessage.Call(hwnd, emStreamIn, sfRTF, uintptr(unsafe.Pointer(&stream)))
	runtime.KeepAlive(state)
	runtime.KeepAlive(raw)
	if ret == 0 && stream.Error != 0 {
		return fmt.Errorf("EM_STREAMIN RTF falhou: %w", callErr)
	}

	dpiX, _, _ := procGetDeviceCaps.Call(hdc, LOGPIXELSX)
	dpiY, _, _ := procGetDeviceCaps.Call(hdc, LOGPIXELSY)
	if dpiX == 0 || dpiY == 0 {
		return fmt.Errorf("DPI invalido para RTF")
	}
	r := RECT{
		Left:   int32(pixelToTwips(left, int(dpiX))),
		Top:    int32(pixelToTwips(top, int(dpiY))),
		Right:  int32(pixelToTwips(right, int(dpiX))),
		Bottom: int32(pixelToTwips(bottom, int(dpiY))),
	}
	fr := formatRange{HDC: hdc, HDCTarget: hdc, RC: r, RCPage: r, Chrg: charRange{CpMin: 0, CpMax: -1}}

	stateDC, _, _ := procSaveDC.Call(hdc)
	if stateDC != 0 {
		defer procRestoreDC.Call(hdc, stateDC)
	}
	procSetMapMode.Call(hdc, MM_TEXT)
	procSetBkMode.Call(hdc, 1)
	procSendMessage.Call(hwnd, emFormatRange, 0, 0)
	formatted, _, callErr := procSendMessage.Call(hwnd, emFormatRange, 1, uintptr(unsafe.Pointer(&fr)))
	procSendMessage.Call(hwnd, emFormatRange, 0, 0)
	if formatted == ^uintptr(0) {
		return fmt.Errorf("EM_FORMATRANGE RTF falhou: %w", callErr)
	}
	return nil
}

func pixelToTwips(px, dpi int) int {
	return px * twipsPerInch / dpi
}

func createRichEditWindow() (uintptr, error) {
	if err := ensureRichEditLoaded(); err != nil {
		return 0, err
	}
	classes := []string{"RichEdit20W", "RICHEDIT50W"}
	for _, className := range classes {
		class := syscall.StringToUTF16Ptr(className)
		title := syscall.StringToUTF16Ptr("")
		hwnd, _, _ := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), uintptr(uint32(wsPopup|esMultiline|esReadOnly)), 0, 0, 1, 1, 0, 0, 0, 0)
		if hwnd != 0 {
			return hwnd, nil
		}
	}
	return 0, fmt.Errorf("CreateWindowEx RichEdit falhou")
}

func ensureRichEditLoaded() error {
	richEditLoadOnce.Do(func() {
		if _, err := syscall.LoadLibrary("riched20.dll"); err == nil {
			return
		}
		if _, err := syscall.LoadLibrary("msftedit.dll"); err != nil {
			richEditLoadErr = err
		}
	})
	return richEditLoadErr
}

func rtfStreamCallback(cookie, buff uintptr, cb int32, pcb uintptr) uintptr {
	state := (*rtfStreamState)(unsafe.Pointer(cookie))
	if state == nil || cb <= 0 || pcb == 0 {
		return 1
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(buff)), int(cb))
	n := copy(buf, state.Data[state.Off:])
	state.Off += n
	*(*int32)(unsafe.Pointer(pcb)) = int32(n)
	return 0
}
