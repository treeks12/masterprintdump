//go:build windows

package render

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modGDI32Render          = syscall.NewLazyDLL("gdi32.dll")
	procSetWinMetaFileBitsR = modGDI32Render.NewProc("SetWinMetaFileBits")
	procGetEnhMetaFileHdrR  = modGDI32Render.NewProc("GetEnhMetaFileHeader")
	procPlayEnhMetaFileR    = modGDI32Render.NewProc("PlayEnhMetaFile")
	procDeleteEnhMetaFileR  = modGDI32Render.NewProc("DeleteEnhMetaFile")
	procSaveDCR             = modGDI32Render.NewProc("SaveDC")
	procRestoreDCR          = modGDI32Render.NewProc("RestoreDC")
	procSetMapModeR         = modGDI32Render.NewProc("SetMapMode")
)

const (
	wmfMapModeText        = 1
	wmfMapModeAnisotropic = 8
)

type wmfPlayRECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type metafilePict struct {
	MM   int32
	XExt int32
	YExt int32
	HMF  uintptr
}

type enhMetaHeader100 struct {
	IType           uint32
	NSize           uint32
	RclBounds       wmfPlayRECT
	RclFrame        wmfPlayRECT
	DSignature      uint32
	NVersion        uint32
	NBytes          uint32
	NRecords        uint32
	NHandles        uint16
	SReserved       uint16
	NDescription    uint32
	OffDescription  uint32
	NPalEntries     uint32
	SzlDeviceX      int32
	SzlDeviceY      int32
	SzlMillimetersX int32
	SzlMillimetersY int32
	CbPixelFormat   uint32
	OffPixelFormat  uint32
	BOpenGL         uint32
}

func PlayWMFBytes(hdc uintptr, data []byte, left, top, right, bottom int) error {
	if hdc == 0 {
		return fmt.Errorf("HDC invalido")
	}
	if right < left || bottom < top {
		return fmt.Errorf("retangulo WMF invalido")
	}
	_, _, _, _, metaData, err := ParseWMFBounds(data)
	if err != nil {
		return err
	}
	if len(metaData) == 0 {
		return fmt.Errorf("WMF sem dados de metafile")
	}
	hEMF, err := cadMapaEnhMetaFileFromWMF(metaData)
	if err != nil {
		return err
	}
	defer procDeleteEnhMetaFileR.Call(hEMF)

	state, _, _ := procSaveDCR.Call(hdc)
	if state != 0 {
		defer procRestoreDCR.Call(hdc, state)
	}
	procSetMapModeR.Call(hdc, wmfMapModeText)
	r := wmfPlayRECT{Left: int32(left), Top: int32(top), Right: int32(right), Bottom: int32(bottom)}
	ret, _, callErr := procPlayEnhMetaFileR.Call(hdc, hEMF, uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return fmt.Errorf("PlayEnhMetaFile falhou: %w", callErr)
	}
	return nil
}

func cadMapaEnhMetaFileFromWMF(metaData []byte) (uintptr, error) {
	if len(metaData) == 0 {
		return 0, fmt.Errorf("WMF sem dados de metafile")
	}
	mp := metafilePict{MM: wmfMapModeAnisotropic}
	hEMF, _, callErr := procSetWinMetaFileBitsR.Call(uintptr(len(metaData)), uintptr(unsafe.Pointer(&metaData[0])), 0, uintptr(unsafe.Pointer(&mp)))
	if hEMF == 0 {
		return 0, fmt.Errorf("SetWinMetaFileBits falhou: %w", callErr)
	}

	var hdr enhMetaHeader100
	ret, _, callErr := procGetEnhMetaFileHdrR.Call(hEMF, unsafe.Sizeof(hdr), uintptr(unsafe.Pointer(&hdr)))
	procDeleteEnhMetaFileR.Call(hEMF)
	if ret == 0 {
		return 0, fmt.Errorf("GetEnhMetaFileHeader falhou: %w", callErr)
	}

	mp = metafilePict{MM: wmfMapModeAnisotropic, XExt: hdr.RclFrame.Right, YExt: hdr.RclFrame.Bottom}
	hEMF, _, callErr = procSetWinMetaFileBitsR.Call(uintptr(len(metaData)), uintptr(unsafe.Pointer(&metaData[0])), 0, uintptr(unsafe.Pointer(&mp)))
	if hEMF == 0 {
		return 0, fmt.Errorf("SetWinMetaFileBits final falhou: %w", callErr)
	}
	return hEMF, nil
}
