//go:build windows

package print

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	dmOutBuffer       = 0x00000002
	dmOrientation     = 0x00000001
	dmCopiesField     = 0x00000100
	dmOrientLandscape = 2
)

var (
	procOpenPrinter        = modWinspool.NewProc("OpenPrinterW")
	procClosePrinter       = modWinspool.NewProc("ClosePrinter")
	procDocumentProperties = modWinspool.NewProc("DocumentPropertiesW")
)

type devmodeHeader struct {
	DmDeviceName    [32]uint16
	DmSpecVersion   uint16
	DmDriverVersion uint16
	DmSize          uint16
	DmDriverExtra   uint16
	DmFields        uint32
	DmOrientation   int16
	DmPaperSize     int16
	DmPaperLength   int16
	DmPaperWidth    int16
	DmScale         int16
	DmCopies        int16
}

func printerDevMode(printerName string, landscape int) ([]byte, error) {
	name := syscall.StringToUTF16Ptr(printerName)
	var hPrinter syscall.Handle
	ret, _, callErr := procOpenPrinter.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(&hPrinter)), 0)
	if ret == 0 {
		return nil, fmt.Errorf("OpenPrinter falhou para %q: %w", printerName, callErr)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	sizeRet, _, callErr := procDocumentProperties.Call(0, uintptr(hPrinter), uintptr(unsafe.Pointer(name)), 0, 0, 0)
	size := int(int32(sizeRet))
	if size <= 0 {
		return nil, fmt.Errorf("DocumentProperties tamanho falhou para %q: %w", printerName, callErr)
	}
	buf := make([]byte, size)
	ret, _, callErr = procDocumentProperties.Call(0, uintptr(hPrinter), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(&buf[0])), 0, dmOutBuffer)
	if int32(ret) < 0 {
		return nil, fmt.Errorf("DocumentProperties falhou para %q: %w", printerName, callErr)
	}
	applyLandscapeOrientation(buf, landscape)
	applyPrinterCopies(buf, 1)
	return buf, nil
}

func applyLandscapeOrientation(devmode []byte, landscape int) {
	if landscape != 1 || len(devmode) < int(unsafe.Sizeof(devmodeHeader{})) {
		return
	}
	dm := (*devmodeHeader)(unsafe.Pointer(&devmode[0]))
	dm.DmFields |= dmOrientation
	dm.DmOrientation = dmOrientLandscape
}

func applyPrinterCopies(devmode []byte, copies int16) {
	if len(devmode) < int(unsafe.Sizeof(devmodeHeader{})) {
		return
	}
	if copies < 1 {
		copies = 1
	}
	dm := (*devmodeHeader)(unsafe.Pointer(&devmode[0]))
	dm.DmFields |= dmCopiesField
	dm.DmCopies = copies
}
