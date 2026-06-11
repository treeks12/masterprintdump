//go:build windows

package print

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"masterprint-native/internal/model"
	"masterprint-native/internal/printlayout"
	"masterprint-native/internal/render"
)

const (
	mmPerInch           = 25.4
	LOGPIXELSX          = 88
	LOGPIXELSY          = 90
	HORZSIZE            = 4
	VERTSIZE            = 6
	HORZRES             = 8
	VERTRES             = 10
	MM_TEXT             = 1
	MM_ANISOTROPIC      = 8
	TA_LEFT             = 0
	TA_TOP              = 0
	TA_NOUPDATECP       = 0
	FW_NORMAL           = 400
	FW_BOLD             = 700
	DEFAULT_CHARSET     = 1
	OUT_TT_PRECIS       = 4
	CLIP_DEFAULT_PRECIS = 0
	DEFAULT_QUALITY     = 0
	PROOF_QUALITY       = 2
	DEFAULT_PITCH       = 0
	FF_DONTCARE         = 0
	FF_MODERN           = 48
	FF_SWISS            = 32
	PS_SOLID            = 0
	NULL_BRUSH          = 5
	DT_LEFT             = 0x00000000
	DT_CENTER           = 0x00000001
	DT_RIGHT            = 0x00000002
	DT_WORDBREAK        = 0x00000010
	DT_NOPREFIX         = 0x00000800
)

var (
	modGDI32    = syscall.NewLazyDLL("gdi32.dll")
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modWinspool = syscall.NewLazyDLL("winspool.drv")

	procCreateDC               = modGDI32.NewProc("CreateDCW")
	procDeleteDC               = modGDI32.NewProc("DeleteDC")
	procStartDoc               = modGDI32.NewProc("StartDocW")
	procEndDoc                 = modGDI32.NewProc("EndDoc")
	procStartPage              = modGDI32.NewProc("StartPage")
	procEndPage                = modGDI32.NewProc("EndPage")
	procCreateFont             = modGDI32.NewProc("CreateFontA")
	procSelectObject           = modGDI32.NewProc("SelectObject")
	procDeleteObject           = modGDI32.NewProc("DeleteObject")
	procTextOut                = modGDI32.NewProc("TextOutA")
	procDrawText               = modUser32.NewProc("DrawTextA")
	procCreatePen              = modGDI32.NewProc("CreatePen")
	procGetStockObject         = modGDI32.NewProc("GetStockObject")
	procMoveToEx               = modGDI32.NewProc("MoveToEx")
	procLineTo                 = modGDI32.NewProc("LineTo")
	procRectangle              = modGDI32.NewProc("Rectangle")
	procEllipse                = modGDI32.NewProc("Ellipse")
	procSetTextAlign           = modGDI32.NewProc("SetTextAlign")
	procGetDeviceCaps          = modGDI32.NewProc("GetDeviceCaps")
	procSetMapMode             = modGDI32.NewProc("SetMapMode")
	procSetViewportOrgEx       = modGDI32.NewProc("SetViewportOrgEx")
	procSetViewportExtEx       = modGDI32.NewProc("SetViewportExtEx")
	procSetWindowOrgEx         = modGDI32.NewProc("SetWindowOrgEx")
	procSetWindowExtEx         = modGDI32.NewProc("SetWindowExtEx")
	procSetWinMetaFileBits     = modGDI32.NewProc("SetWinMetaFileBits")
	procPlayEnhMetaFile        = modGDI32.NewProc("PlayEnhMetaFile")
	procDeleteEnhMetaFile      = modGDI32.NewProc("DeleteEnhMetaFile")
	procSaveDC                 = modGDI32.NewProc("SaveDC")
	procRestoreDC              = modGDI32.NewProc("RestoreDC")
	procSetBkMode              = modGDI32.NewProc("SetBkMode")
	procCreateCompatibleDC     = modGDI32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modGDI32.NewProc("CreateCompatibleBitmap")
	procGetDIBits              = modGDI32.NewProc("GetDIBits")
	procPatBlt                 = modGDI32.NewProc("PatBlt")
	procEnumPrinters           = modWinspool.NewProc("EnumPrintersW")
)

type DOCINFO struct {
	CbSize       int32
	LpszDocName  *uint16
	LpszOutput   *uint16
	LpszDatatype *uint16
	FwType       uint32
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type LabelPrinter struct {
	PrinterName  string
	hDC          syscall.Handle
	dpiX         int
	dpiY         int
	pageWidthMM  float64
	pageHeightMM float64
	inJob        bool
}

func MmToPixels(mm float64, dpi int) int {
	return int(math.Round(mm * float64(dpi) / mmPerInch))
}

func MmToPixelsF(mm float64, dpi int) float64 {
	return mm * float64(dpi) / mmPerInch
}

func PtToPixels(pt float64, dpi int) int {
	return int(math.Round(pt * float64(dpi) / 72.0))
}

func PixelsToMm(px int, dpi int) float64 {
	return float64(px) * mmPerInch / float64(dpi)
}

func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	n := 0
	for ptr := p; *ptr != 0; ptr = (*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + 2)) {
		n++
	}
	if n == 0 {
		return ""
	}
	s := make([]uint16, n)
	for i := 0; i < n; i++ {
		s[i] = *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(i)*2))
	}
	return syscall.UTF16ToString(s)
}

func EnumPrinters() ([]string, error) {
	var needed, returned uint32
	const flags = 0x00000002
	const level = 1

	procEnumPrinters.Call(
		flags, 0, level, 0, 0,
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)

	if needed == 0 {
		return nil, fmt.Errorf("nenhuma impressora encontrada")
	}

	buf := make([]byte, needed)
	ret, _, err := procEnumPrinters.Call(
		flags, 0, level,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("EnumPrinters falhou: %w", err)
	}

	type printerInfo1 struct {
		Flags       uint32
		Description *uint16
		Name        *uint16
		Comment     *uint16
	}

	printers := make([]string, 0, returned)
	elemSize := unsafe.Sizeof(printerInfo1{})
	for i := uint32(0); i < returned; i++ {
		pi := (*printerInfo1)(unsafe.Pointer(uintptr(unsafe.Pointer(&buf[0])) + uintptr(i)*elemSize))
		if pi.Name != nil {
			printers = append(printers, utf16PtrToString(pi.Name))
		}
	}
	return printers, nil
}

func NewLabelPrinter(printerName string, landscape int) (*LabelPrinter, error) {
	if printerName == "" {
		printers, err := EnumPrinters()
		if err != nil {
			return nil, err
		}
		if len(printers) == 0 {
			return nil, fmt.Errorf("nenhuma impressora disponivel")
		}
		printerName = printers[0]
	}

	driver := syscall.StringToUTF16Ptr("WINSPOOL")
	name := syscall.StringToUTF16Ptr(printerName)
	var devmode []byte
	devmodePtr := uintptr(0)
	if dm, err := printerDevMode(printerName, landscape); err != nil {
		log.Printf("print: DEVMODE padrao usado para %q: %v", printerName, err)
	} else if len(dm) > 0 {
		devmode = dm
		devmodePtr = uintptr(unsafe.Pointer(&devmode[0]))
	}

	hDC, _, err := procCreateDC.Call(
		uintptr(unsafe.Pointer(driver)),
		uintptr(unsafe.Pointer(name)),
		0, devmodePtr,
	)
	runtime.KeepAlive(devmode)
	if hDC == 0 && devmodePtr != 0 {
		log.Printf("print: CreateDC com DEVMODE falhou para %q, tentando sem DEVMODE: %v", printerName, err)
		hDC, _, err = procCreateDC.Call(uintptr(unsafe.Pointer(driver)), uintptr(unsafe.Pointer(name)), 0, 0)
	}
	if hDC == 0 {
		return nil, fmt.Errorf("CreateDC falhou para %q: %w", printerName, err)
	}

	dpiX, _, _ := procGetDeviceCaps.Call(hDC, LOGPIXELSX)
	dpiY, _, _ := procGetDeviceCaps.Call(hDC, LOGPIXELSY)
	horzRes, _, _ := procGetDeviceCaps.Call(hDC, HORZRES)
	vertRes, _, _ := procGetDeviceCaps.Call(hDC, VERTRES)
	horzSize, _, _ := procGetDeviceCaps.Call(hDC, HORZSIZE)
	vertSize, _, _ := procGetDeviceCaps.Call(hDC, VERTSIZE)
	pageW, pageH := render.CadMapaPageSizeMM(render.CadMapaDeviceCaps{HorzRes: int(horzRes), VertRes: int(vertRes), HorzSize: int(horzSize), VertSize: int(vertSize), DPIX: int(dpiX), DPIY: int(dpiY)})

	lp := &LabelPrinter{
		PrinterName:  printerName,
		hDC:          syscall.Handle(hDC),
		dpiX:         int(dpiX),
		dpiY:         int(dpiY),
		pageWidthMM:  pageW,
		pageHeightMM: pageH,
	}

	procSetMapMode.Call(uintptr(lp.hDC), MM_TEXT)
	procSetBkMode.Call(uintptr(lp.hDC), 1)

	return lp, nil
}

func (lp *LabelPrinter) Close() {
	if lp.hDC != 0 {
		procDeleteDC.Call(uintptr(lp.hDC))
		lp.hDC = 0
	}
}

func (lp *LabelPrinter) DPI() (int, int) {
	return lp.dpiX, lp.dpiY
}

func (lp *LabelPrinter) PageSizeMM() (float64, float64) {
	return lp.pageWidthMM, lp.pageHeightMM
}

func (lp *LabelPrinter) BeginDocument(docName string) error {
	if lp.inJob {
		return fmt.Errorf("documento ja iniciado")
	}
	di := DOCINFO{
		CbSize:      int32(unsafe.Sizeof(DOCINFO{})),
		LpszDocName: syscall.StringToUTF16Ptr(docName),
	}
	ret, _, err := procStartDoc.Call(uintptr(lp.hDC), uintptr(unsafe.Pointer(&di)))
	if int32(ret) <= 0 {
		return fmt.Errorf("StartDoc falhou: %w", err)
	}
	lp.inJob = true
	return nil
}

func (lp *LabelPrinter) EndDocument() error {
	if !lp.inJob {
		return fmt.Errorf("nenhum documento ativo")
	}
	ret, _, err := procEndDoc.Call(uintptr(lp.hDC))
	lp.inJob = false
	if int32(ret) <= 0 {
		return fmt.Errorf("EndDoc falhou: %w", err)
	}
	return nil
}

func (lp *LabelPrinter) BeginPage() error {
	ret, _, err := procStartPage.Call(uintptr(lp.hDC))
	if int32(ret) <= 0 {
		return fmt.Errorf("StartPage falhou: %w", err)
	}
	return nil
}

func (lp *LabelPrinter) EndPage() error {
	ret, _, err := procEndPage.Call(uintptr(lp.hDC))
	if int32(ret) <= 0 {
		return fmt.Errorf("EndPage falhou: %w", err)
	}
	return nil
}

func (lp *LabelPrinter) createFont(fontName string, heightPx, widthPx int, escapement int16, bold, italic, underline, strikeout bool) (syscall.Handle, error) {
	weight := uintptr(FW_NORMAL)
	if bold {
		weight = FW_BOLD
	}
	italicFlag := uintptr(0)
	if italic {
		italicFlag = 1
	}
	underlineFlag := uintptr(0)
	if underline {
		underlineFlag = 1
	}
	strikeoutFlag := uintptr(0)
	if strikeout {
		strikeoutFlag = 1
	}
	face, faceErr := syscall.BytePtrFromString(fontName)
	if faceErr != nil {
		return 0, faceErr
	}
	hFont, _, err := procCreateFont.Call(
		uintptr(heightPx), uintptr(widthPx), uintptr(int32(escapement)), 0,
		weight, italicFlag, underlineFlag, strikeoutFlag,
		DEFAULT_CHARSET, OUT_TT_PRECIS,
		CLIP_DEFAULT_PRECIS, PROOF_QUALITY,
		DEFAULT_PITCH, uintptr(unsafe.Pointer(face)),
	)
	if hFont == 0 {
		return 0, fmt.Errorf("CreateFont falhou para %q: %w", fontName, err)
	}
	return syscall.Handle(hFont), nil
}

func (lp *LabelPrinter) DrawText(text string, xMM, yMM float64, fontName string, sizePt float64, bold bool) error {
	return lp.DrawTextElement(xMM, yMM, model.TextElement{Text: text, FontName: fontName, FontSize: sizePt, Bold: bold})
}

func (lp *LabelPrinter) DrawTextElement(baseXMM, baseYMM float64, t model.TextElement) error {
	if t.Text == "" && len(t.RTFRaw) == 0 {
		return nil
	}
	xMM := baseXMM + t.XMM
	yMM := baseYMM + t.YMM
	useBox := t.WidthMM > 0 && t.HeightMM > 0
	rtfFallback := false
	if useBox && len(t.RTFRaw) > 0 {
		if err := lp.drawRTFElement(baseXMM, baseYMM, t); err == nil {
			return nil
		} else {
			log.Printf("print: RTF fallback to plain text off=%#x: %v", t.FileOffset, err)
			rtfFallback = true
		}
	}
	heightPx := PtToPixels(t.FontSize, lp.dpiY)
	if useBox {
		heightPx = render.CadMapaFontHeightPx(render.MmFloatTo100(t.HeightMM), lp.dpiY)
	}
	if heightPx <= 0 {
		heightPx = PtToPixels(8, lp.dpiY)
	}

	widthPx := 0
	styleByte := printTextStyleByte(t)
	bold, italic, underline, strikeout := render.CadMapaGDIStyle(styleByte)
	if useBox && (len(t.RTFRaw) == 0 || rtfFallback) {
		left, _, right, _ := render.CadMapaObjectRectPxFromMM(xMM, yMM, t.WidthMM, t.HeightMM, lp.dpiX, lp.dpiY)
		widthPx = render.CadMapaFitCharWidthPx(uintptr(lp.hDC), t.Text, right-left, heightPx, t.FontName, styleByte, 0)
	}

	hFont, err := lp.createFont(t.FontName, heightPx, widthPx, 0, bold, italic, underline, strikeout)
	if err != nil {
		return err
	}
	defer procDeleteObject.Call(uintptr(hFont))

	oldFont, _, _ := procSelectObject.Call(uintptr(lp.hDC), uintptr(hFont))
	defer procSelectObject.Call(uintptr(lp.hDC), oldFont)

	if useBox {
		left, top, right, bottom := render.CadMapaObjectRectPxFromMM(xMM, yMM, t.WidthMM, t.HeightMM, lp.dpiX, lp.dpiY)
		r := RECT{
			Left:   int32(left),
			Top:    int32(top),
			Right:  int32(right),
			Bottom: int32(bottom),
		}
		textBytes := render.CadMapaANSIBytes(t.Text)
		if len(textBytes) == 0 {
			return nil
		}
		procDrawText.Call(
			uintptr(lp.hDC),
			uintptr(unsafe.Pointer(&textBytes[0])),
			uintptr(len(textBytes)),
			uintptr(unsafe.Pointer(&r)),
			uintptr(printTextFlags(t.Align)),
		)
		return nil
	}
	return lp.drawTextOut(t.Text, MmToPixels(xMM, lp.dpiX), MmToPixels(yMM, lp.dpiY))
}

func printTextStyleByte(t model.TextElement) byte {
	if t.StyleByte != 0 && len(t.RTFRaw) == 0 {
		return t.StyleByte
	}
	var style byte
	if t.Bold {
		style |= 0x01
	}
	if t.Italic {
		style |= 0x02
	}
	if t.Underline {
		style |= 0x04
	}
	return style
}

func printTextFlags(align string) uint32 {
	return render.CadMapaDrawTextFlags(alignToCadMapa(align), false)
}

func alignToCadMapa(align string) int16 {
	switch align {
	case "center":
		return 1
	case "right":
		return 2
	default:
		return 0
	}
}

func (lp *LabelPrinter) drawTextOut(text string, x, y int) error {
	procSetTextAlign.Call(uintptr(lp.hDC), TA_LEFT|TA_TOP|TA_NOUPDATECP)
	textBytes := render.CadMapaANSIBytes(text)
	if len(textBytes) == 0 {
		return nil
	}
	ret, _, callErr := procTextOut.Call(
		uintptr(lp.hDC),
		uintptr(x), uintptr(y),
		uintptr(unsafe.Pointer(&textBytes[0])),
		uintptr(len(textBytes)),
	)
	if ret == 0 {
		return fmt.Errorf("TextOut falhou: %w", callErr)
	}
	return nil
}

func (lp *LabelPrinter) DrawWMF(filePath string, xMM, yMM, widthMM, heightMM float64) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("leitura WMF: %w", err)
	}
	return lp.drawWMFBytes(data, xMM, yMM, widthMM, heightMM)
}

func (lp *LabelPrinter) DrawWMFSymbol(sym model.WMFSymbol, cellXMM, cellYMM float64) error {
	data, err := WMFBytes(sym)
	if err != nil {
		return err
	}
	return lp.drawWMFBytes(data, cellXMM+sym.XMM, cellYMM+sym.YMM, sym.WidthMM, sym.HeightMM)
}

func (lp *LabelPrinter) drawWMFBytes(data []byte, xMM, yMM, widthMM, heightMM float64) error {
	left, top, right, bottom := render.CadMapaWMFPlayRectFromMM(xMM, yMM, widthMM, heightMM, lp.dpiX, lp.dpiY)
	return render.PlayWMFBytes(uintptr(lp.hDC), data, left, top, right, bottom)
}

func (lp *LabelPrinter) PrintLabel(docName string, layout model.LayoutDefinition, sheet printlayout.Sheet, copies int, label model.Label) error {
	if err := lp.BeginDocument(docName); err != nil {
		return err
	}
	defer lp.EndDocument()

	if sheet.PageWidthMM <= 0 {
		sheet.PageWidthMM = lp.pageWidthMM
	}
	if sheet.PageHeightMM <= 0 {
		sheet.PageHeightMM = lp.pageHeightMM
	}
	cells := printlayout.Cells(layout, sheet)
	if len(cells) == 0 {
		return fmt.Errorf("nenhuma celula de impressao calculada")
	}
	pages := printlayout.PageCount(copies, len(cells))
	for pageIndex := 0; pageIndex < pages; pageIndex++ {
		if err := lp.BeginPage(); err != nil {
			return err
		}
		pageEnded := false
		for _, cell := range printlayout.CellsForPage(cells, copies, pageIndex) {
			if len(label.Objects) > 0 {
				if err := lp.drawPrintObjects(layout, cell.XMM, cell.YMM, label.Objects); err != nil {
					if !pageEnded {
						_ = lp.EndPage()
					}
					return err
				}
			} else {
				if err := lp.drawLegacyLabelObjects(cell.XMM, cell.YMM, label); err != nil {
					if !pageEnded {
						_ = lp.EndPage()
					}
					return err
				}
			}
		}
		if err := lp.EndPage(); err != nil {
			return err
		}
		pageEnded = true
	}
	return nil
}

func (lp *LabelPrinter) drawPrintObjects(layout model.LayoutDefinition, cellXMM, cellYMM float64, objects []model.PrintObject) error {
	if tr, ok := NewLandscapeCellTransform(layout); ok {
		objects = landscapePrintObjects(tr, objects)
	}
	for _, obj := range objects {
		switch obj.Type {
		case "text":
			if err := lp.DrawTextElement(cellXMM, cellYMM, obj.Text); err != nil {
				return err
			}
		case "image":
			if err := lp.DrawWMFSymbol(obj.WMF, cellXMM, cellYMM); err != nil {
				return err
			}
		case "line", "rect", "ellipse":
			if err := lp.DrawShapeElement(cellXMM, cellYMM, obj.Type, obj.Shape); err != nil {
				return err
			}
		}
	}
	return nil
}

func landscapePrintObjects(tr LandscapeCellTransform, objects []model.PrintObject) []model.PrintObject {
	out := make([]model.PrintObject, len(objects))
	for i, obj := range objects {
		out[i] = landscapePrintObject(tr, obj)
	}
	return out
}

func landscapePrintObject(tr LandscapeCellTransform, obj model.PrintObject) model.PrintObject {
	switch obj.Type {
	case "text":
		obj.Text = landscapeTextElement(tr, obj.Text)
	case "image":
		obj.WMF = landscapeWMFSymbol(tr, obj.WMF)
	case "line", "rect", "ellipse":
		obj.Shape = landscapeShapeElement(tr, obj.Shape)
	}
	return obj
}

func landscapeTextElement(tr LandscapeCellTransform, t model.TextElement) model.TextElement {
	r := tr.MapDesignRectToCellMM(render.RectMM{X: t.XMM, Y: t.YMM, Width: t.WidthMM, Height: t.HeightMM})
	t.XMM, t.YMM, t.WidthMM, t.HeightMM = r.X, r.Y, r.Width, r.Height
	return t
}

func landscapeWMFSymbol(tr LandscapeCellTransform, s model.WMFSymbol) model.WMFSymbol {
	r := tr.MapDesignRectToCellMM(render.RectMM{X: s.XMM, Y: s.YMM, Width: s.WidthMM, Height: s.HeightMM})
	s.XMM, s.YMM, s.WidthMM, s.HeightMM = r.X, r.Y, r.Width, r.Height
	return s
}

func landscapeShapeElement(tr LandscapeCellTransform, s model.ShapeElement) model.ShapeElement {
	r := tr.MapDesignRectToCellMM(render.RectMM{X: s.XMM, Y: s.YMM, Width: s.WidthMM, Height: s.HeightMM})
	s.XMM, s.YMM, s.WidthMM, s.HeightMM = r.X, r.Y, r.Width, r.Height
	return s
}

func (lp *LabelPrinter) DrawShapeElement(baseXMM, baseYMM float64, shapeType string, s model.ShapeElement) error {
	left, top, right, bottom := render.CadMapaObjectRectPxFromMM(baseXMM+s.XMM, baseYMM+s.YMM, s.WidthMM, s.HeightMM, lp.dpiX, lp.dpiY)
	pen, _, err := procCreatePen.Call(PS_SOLID, 1, 0)
	if pen == 0 {
		return fmt.Errorf("CreatePen falhou: %w", err)
	}
	defer procDeleteObject.Call(pen)
	oldPen, _, _ := procSelectObject.Call(uintptr(lp.hDC), pen)
	defer procSelectObject.Call(uintptr(lp.hDC), oldPen)
	nullBrush, _, _ := procGetStockObject.Call(NULL_BRUSH)
	oldBrush := uintptr(0)
	if nullBrush != 0 {
		oldBrush, _, _ = procSelectObject.Call(uintptr(lp.hDC), nullBrush)
		defer procSelectObject.Call(uintptr(lp.hDC), oldBrush)
	}
	switch shapeType {
	case "line":
		procMoveToEx.Call(uintptr(lp.hDC), uintptr(left), uintptr(top), 0)
		ret, _, callErr := procLineTo.Call(uintptr(lp.hDC), uintptr(right), uintptr(bottom))
		if ret == 0 {
			return fmt.Errorf("LineTo falhou: %w", callErr)
		}
	case "rect":
		ret, _, callErr := procRectangle.Call(uintptr(lp.hDC), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
		if ret == 0 {
			return fmt.Errorf("Rectangle falhou: %w", callErr)
		}
	case "ellipse":
		ret, _, callErr := procEllipse.Call(uintptr(lp.hDC), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
		if ret == 0 {
			return fmt.Errorf("Ellipse falhou: %w", callErr)
		}
	}
	return nil
}

func (lp *LabelPrinter) drawLegacyLabelObjects(cellXMM, cellYMM float64, label model.Label) error {
	for _, t := range label.Texts {
		if err := lp.DrawTextElement(cellXMM, cellYMM, t); err != nil {
			return err
		}
	}
	for _, img := range label.WMFSymbols {
		if err := lp.DrawWMFSymbol(img, cellXMM, cellYMM); err != nil {
			return err
		}
	}
	return nil
}

func WMFToPNG(wmfPath string, widthMM, heightMM float64) (string, error) {
	const renderDPI = 300
	w := MmToPixels(widthMM, renderDPI)
	h := MmToPixels(heightMM, renderDPI)
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("dimensoes invalidas")
	}

	data, err := os.ReadFile(wmfPath)
	if err != nil {
		return "", err
	}

	metaX, metaY, metaW, metaH, metaData, err := render.ParseWMFBounds(data)
	if err != nil {
		return "", err
	}

	screenDC, _, err := procCreateDC.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("DISPLAY"))),
		0, 0, 0,
	)
	if screenDC == 0 {
		return "", fmt.Errorf("CreateDC DISPLAY falhou: %w", err)
	}
	defer procDeleteDC.Call(screenDC)

	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return "", fmt.Errorf("CreateCompatibleDC falhou")
	}
	defer procDeleteDC.Call(memDC)

	bmp, _, _ := procCreateCompatibleBitmap.Call(screenDC, uintptr(w), uintptr(h))
	if bmp == 0 {
		return "", fmt.Errorf("CreateCompatibleBitmap falhou")
	}
	defer procDeleteObject.Call(bmp)

	oldBmp, _, _ := procSelectObject.Call(memDC, bmp)
	defer procSelectObject.Call(memDC, oldBmp)

	const whiteness = 0x00FF0062
	procPatBlt.Call(memDC, 0, 0, uintptr(w), uintptr(h), whiteness)

	hEMF, _, err := procSetWinMetaFileBits.Call(
		uintptr(len(metaData)),
		uintptr(unsafe.Pointer(&metaData[0])),
		memDC, 0,
	)
	if hEMF == 0 {
		return "", fmt.Errorf("SetWinMetaFileBits falhou: %w", err)
	}
	defer procDeleteEnhMetaFile.Call(hEMF)

	state, _, _ := procSaveDC.Call(memDC)
	defer procRestoreDC.Call(memDC, state)

	procSetMapMode.Call(memDC, MM_ANISOTROPIC)
	procSetWindowOrgEx.Call(memDC, uintptr(metaX), uintptr(metaY), 0)
	procSetWindowExtEx.Call(memDC, uintptr(metaW), uintptr(metaH), 0)
	procSetViewportOrgEx.Call(memDC, 0, 0, 0)
	procSetViewportExtEx.Call(memDC, uintptr(w), uintptr(h), 0)
	r := RECT{Left: 0, Top: 0, Right: int32(w - 1), Bottom: int32(h - 1)}
	procPlayEnhMetaFile.Call(memDC, hEMF, uintptr(unsafe.Pointer(&r)))

	bi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:     uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:    int32(w),
			BiHeight:   -int32(h),
			BiPlanes:   1,
			BiBitCount: 32,
		},
	}

	pixels := make([]byte, w*h*4)
	ret, _, _ := procGetDIBits.Call(
		memDC, bmp, 0, uintptr(h),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bi)), 0,
	)
	if ret == 0 {
		return "", fmt.Errorf("GetDIBits falhou")
	}

	tmpFile, err := os.CreateTemp("", "wmf-*.png")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	if err := saveBGRAasPNG(tmpPath, w, h, pixels); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func saveBGRAasPNG(path string, width, height int, bgra []byte) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			src := (y*width + x) * 4
			img.SetRGBA(x, y, color.RGBA{
				R: bgra[src+2],
				G: bgra[src+1],
				B: bgra[src+0],
				A: 255,
			})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
