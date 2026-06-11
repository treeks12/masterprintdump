//go:build windows

package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"masterprint-native/internal/etq"
	"masterprint-native/internal/inf"
	"masterprint-native/internal/model"
	"masterprint-native/internal/print"
	"masterprint-native/internal/printlayout"
	"masterprint-native/internal/render"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/tailscale/win"
)

var (
	grayBrush   *walk.SolidColorBrush
	whiteBrush  *walk.SolidColorBrush
	darkBrush   *walk.SolidColorBrush
	rulerBrush  *walk.SolidColorBrush
	shadowBrush *walk.SolidColorBrush
	blackBrush  *walk.SolidColorBrush
)

const (
	topbarHeight = 102
	menuTopY     = 0
	menuHeight   = 22
	clipsHeight  = 65
	rulerSize    = 23
)

func initBrushes() {
	grayBrush, _ = walk.NewSolidColorBrush(walk.RGB(127, 127, 127))
	whiteBrush, _ = walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	darkBrush, _ = walk.NewSolidColorBrush(walk.RGB(224, 224, 224))
	rulerBrush, _ = walk.NewSolidColorBrush(walk.RGB(232, 232, 232))
	shadowBrush, _ = walk.NewSolidColorBrush(walk.RGB(60, 60, 60))
	blackBrush, _ = walk.NewSolidColorBrush(walk.RGB(0, 0, 0))
}

type App struct {
	dataDir       string
	clipartDir    string
	layouts       map[string][]model.LayoutDefinition
	pageOverrides map[string]model.PrintPage
	symbolNames   []string

	mainWindow    *walk.MainWindow
	toolbar       *walk.CustomWidget
	canvas        *walk.CustomWidget
	symbolList    *walk.ListBox
	symbolStrip   *walk.CustomWidget
	clipFilter    *walk.LineEdit
	clipsPanel    *walk.Composite
	toolbarGlyphs map[string]walk.Image

	elements          []LabelElement
	selectedIdx       int
	clipboardElem     *LabelElement
	tool              string
	zoom              float64
	defaultFontName   string
	defaultFontSize   float64
	defaultTextColor  color.Color
	currentLayout     *model.LayoutDefinition
	currentDocPath    string
	currentPrinter    string
	currentLayoutType string
	currentTemplate   string
	unknownObjects    []etq.ETQUnknownObject
	etqSourcePath     string
	etqBaseline       []etqElementSnapshot

	undoStack [][]LabelElement
	redoStack [][]LabelElement

	dragging        bool
	dragStart       image.Point
	dragHandle      string
	dragOrigX       float64
	dragOrigY       float64
	dragOrigW       float64
	dragOrigH       float64
	dragUndoPending bool

	panning  bool
	panStart image.Point
	scrollX  int
	scrollY  int

	lastClickAt    time.Time
	lastClickPoint image.Point

	clipsCollapsed    bool
	selectedSymbol    int
	symbolOffset      int
	dpi               int
	showGrid          bool
	mergeField        string
	persistenceStatus string
}

type ToolbarButton struct {
	ID       string
	Label    string
	Tooltip  string
	Rect     walk.Rectangle
	Row      int
	Enabled  bool
	GapAfter bool
}

type MenuHotspot struct {
	ID   string
	Text string
	Rect walk.Rectangle
}

type menuItemAction struct {
	label string
	sep   bool
	stub  bool
	fn    func()
}

type LabelElement struct {
	ID         int
	Type       string
	FileOffset int
	FEFlags    uint32
	FETag      uint32
	PayloadRaw string
	RTFRaw     string
	WMFRaw     string
	WMFPreRaw  string
	StyleByte  byte
	NextX      uint32
	NextY      uint32
	XMM        float64
	YMM        float64
	WidthMM    float64
	HeightMM   float64
	Text       string
	FontName   string
	FontSize   float64
	Bold       bool
	Italic     bool
	Underline  bool
	Locked     bool
	Color      color.Color
	ImagePath  string
	SymbolName string
	Align      string
}

type savedDocument struct {
	SchemaVersion  int                  `json:"schemaVersion"`
	DocumentKind   string               `json:"documentKind,omitempty"`
	LayoutName     string               `json:"layoutName"`
	PrinterName    string               `json:"printerName,omitempty"`
	LayoutType     string               `json:"layoutType,omitempty"`
	TemplateName   string               `json:"templateName,omitempty"`
	UnknownObjects []savedUnknownObject `json:"unknownObjects,omitempty"`
	Elements       []savedElement       `json:"elements"`
}

type savedUnknownObject struct {
	Offset int    `json:"offset"`
	Flags  uint32 `json:"flags"`
	Tag    uint32 `json:"tag"`
	Kind   string `json:"kind"`
}

type savedElement struct {
	Type       string  `json:"type"`
	FileOffset int     `json:"fileOffset,omitempty"`
	FEFlags    uint32  `json:"feFlags,omitempty"`
	FETag      uint32  `json:"feTag,omitempty"`
	PayloadRaw string  `json:"payloadRaw,omitempty"`
	RTFRaw     string  `json:"rtfRaw,omitempty"`
	WMFRaw     string  `json:"wmfRaw,omitempty"`
	WMFPreRaw  string  `json:"wmfPreRaw,omitempty"`
	StyleByte  byte    `json:"styleByte,omitempty"`
	NextX      uint32  `json:"nextX,omitempty"`
	NextY      uint32  `json:"nextY,omitempty"`
	XMM        float64 `json:"xMM"`
	YMM        float64 `json:"yMM"`
	WidthMM    float64 `json:"widthMM"`
	HeightMM   float64 `json:"heightMM"`
	Text       string  `json:"text,omitempty"`
	FontName   string  `json:"fontName,omitempty"`
	FontSize   float64 `json:"fontSize,omitempty"`
	Bold       bool    `json:"bold,omitempty"`
	Italic     bool    `json:"italic,omitempty"`
	Underline  bool    `json:"underline,omitempty"`
	ImagePath  string  `json:"imagePath,omitempty"`
	SymbolName string  `json:"symbolName,omitempty"`
	Align      string  `json:"align,omitempty"`
}

var nextElemID = 1

func NewApp() *App {
	dataDir := `C:\Program Files (x86)\paulimaq`
	if customDir := os.Getenv("MASTERPRINT_DATA"); customDir != "" {
		dataDir = customDir
	}
	return &App{
		dataDir:          dataDir,
		clipartDir:       filepath.Join(dataDir, "CLIPART", "S\u00edmbolos"),
		layouts:          make(map[string][]model.LayoutDefinition),
		pageOverrides:    make(map[string]model.PrintPage),
		selectedIdx:      -1,
		selectedSymbol:   -1,
		symbolOffset:     0,
		tool:             "select",
		zoom:             1.0,
		defaultFontName:  "Arial",
		defaultFontSize:  8,
		defaultTextColor: color.Black,
		clipsCollapsed:   false,
		toolbarGlyphs:    make(map[string]walk.Image),
	}
}

func (a *App) loadAllLayouts() {
	if pages, err := inf.ParsePageOverride(filepath.Join(a.dataDir, "pageovrr.ini")); err == nil {
		a.pageOverrides = pages
	}
	if catalogs, err := inf.LoadCatalogs(a.dataDir); err == nil {
		for key, layouts := range catalogs {
			a.layouts[key] = layouts
		}
	}
}

func (a *App) renderDPI() int {
	dpi := a.dpi
	if dpi <= 0 {
		dpi = 96
	}
	return int(float64(dpi) * a.zoom)
}

func (a *App) mmToPxX(mm float64) int {
	return a.mmToPxLenX(mm) - a.scrollX
}

func (a *App) mmToPxY(mm float64) int {
	return a.mmToPxLenY(mm) - a.scrollY

}

func (a *App) mmToPxLenX(mm float64) int {
	return render.Mm100ToPx(render.MmFloatTo100(mm), a.renderDPI())
}

func (a *App) mmToPxLenY(mm float64) int {
	return render.Mm100ToPx(render.MmFloatTo100(mm), a.renderDPI())
}

func (a *App) pxToMmX(px int) float64 {
	dpi := a.renderDPI()
	if dpi <= 0 {
		return 0
	}
	return float64(render.PxToMm100(px+a.scrollX, dpi)) / 100.0
}

func (a *App) pxToMmY(py int) float64 {
	dpi := a.renderDPI()
	if dpi <= 0 {
		return 0
	}
	return float64(render.PxToMm100(py+a.scrollY, dpi)) / 100.0
}

func (a *App) run() error {
	app, err := walk.InitApp()
	if err != nil {
		return err
	}
	initBrushes()
	if err := a.buildUI(); err != nil {
		return err
	}
	a.dpi = a.canvas.DPI()
	a.loadToolbarGlyphs()
	app.Run()
	a.disposeToolbarGlyphs()
	return nil
}

func (a *App) toolbarGlyphDir() string {
	if dir := os.Getenv("MASTERPRINT_GLYPHS"); dir != "" {
		return dir
	}
	exeDir := filepath.Dir(os.Args[0])
	candidates := []string{
		filepath.Join("assets", "cadmapa_glyphs"),
		filepath.Join(exeDir, "assets", "cadmapa_glyphs"),
		filepath.Join(exeDir, "..", "assets", "cadmapa_glyphs"),
	}
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir
		}
	}
	return ""
}

func (a *App) loadToolbarGlyphs() {
	dir := a.toolbarGlyphDir()
	if dir == "" {
		return
	}
	for _, id := range []string{
		"new", "open", "save", "print", "cut", "copy", "paste", "alignDialog", "group", "ungroup", "bring", "send", "zoomPage", "zoomWidth", "zoom100", "help", "exit",
		"printSetup", "bold", "italic", "underline", "strike", "alignL", "alignC", "alignR", "bullets",
		"select", "zoom", "line", "roundRect", "rect", "ellipse", "simpleText", "text", "artText", "barcode", "image", "mapaRisc", "ole", "fileMan", "mergeToggle",
		"navFirst", "navPrev", "navNext", "navLast",
	} {
		img, err := walk.NewImageFromFile(filepath.Join(dir, id+".png"))
		if err == nil {
			a.toolbarGlyphs[id] = img
		}
	}
}

func (a *App) disposeToolbarGlyphs() {
	for id, img := range a.toolbarGlyphs {
		img.Dispose()
		delete(a.toolbarGlyphs, id)
	}
}

func (a *App) buildUI() error {
	err := MainWindow{
		AssignTo: &a.mainWindow,
		Title:    "Paulimaq MasterPrint 3.0",
		MinSize:  Size{Width: 900, Height: 650},
		Size:     Size{Width: 1100, Height: 750},
		Layout:   VBox{MarginsZero: true, SpacingZero: true},
		OnKeyDown: func(key walk.Key) {
			a.handleKeyDown(key)
		},
		Children: []Widget{
			CustomWidget{
				AssignTo:            &a.toolbar,
				ClearsBackground:    true,
				InvalidatesOnResize: true,
				MinSize:             Size{Height: topbarHeight},
				MaxSize:             Size{Height: topbarHeight},
				StretchFactor:       0,
				Paint:               a.paintToolbar,
				OnMouseDown:         a.toolbarMouseDown,
			},
			CustomWidget{
				AssignTo:            &a.canvas,
				ClearsBackground:    true,
				InvalidatesOnResize: true,
				StretchFactor:       1,
				Paint:               a.paintCanvas,
				OnMouseDown:         a.canvasMouseDown,
				OnMouseMove:         a.canvasMouseMove,
				OnMouseUp:           a.canvasMouseUp,
			},
			Composite{
				AssignTo:      &a.clipsPanel,
				Layout:        VBox{MarginsZero: true, SpacingZero: true},
				MinSize:       Size{Height: clipsHeight},
				MaxSize:       Size{Height: clipsHeight},
				StretchFactor: 0,
				Children: []Widget{
					CustomWidget{AssignTo: &a.symbolStrip, MinSize: Size{Height: clipsHeight}, Paint: a.paintSymbolStrip, OnMouseDown: a.symbolStripMouseDown},
				},
			},
		},
		StatusBarItems: []StatusBarItem{{Text: "MasterPrint 3.0 - Arquivo \u2192 Abrir", Width: 500}},
	}.Create()

	if err != nil {
		return fmt.Errorf("criando janela: %w", err)
	}
	a.mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if !a.confirmDiscardChanges() {
			*canceled = true
		}
	})

	a.loadSymbolList()
	return nil
}

func (a *App) setTool(tool string) { a.tool = tool }

func (a *App) defaultTextWalkColor() walk.Color {
	if a.defaultTextColor == nil {
		return walk.RGB(0, 0, 0)
	}
	r, g, b, _ := a.defaultTextColor.RGBA()
	return walk.RGB(byte(r>>8), byte(g>>8), byte(b>>8))
}

func (a *App) snapshot() []LabelElement {
	snap := make([]LabelElement, len(a.elements))
	copy(snap, a.elements)
	return snap
}

func (a *App) pushUndo() {
	a.undoStack = append(a.undoStack, a.snapshot())
	if len(a.undoStack) > 50 {
		a.undoStack = a.undoStack[1:]
	}
	a.redoStack = nil
	a.persistenceStatus = ""
	a.updateWindowTitle()
}

func (a *App) resetHistory() {
	a.undoStack = nil
	a.redoStack = nil
}

func (a *App) onUndo() {
	if len(a.undoStack) == 0 {
		return
	}
	a.redoStack = append(a.redoStack, a.snapshot())
	snap := a.undoStack[len(a.undoStack)-1]
	a.undoStack = a.undoStack[:len(a.undoStack)-1]
	a.elements = make([]LabelElement, len(snap))
	copy(a.elements, snap)
	if a.selectedIdx >= len(a.elements) {
		a.selectedIdx = -1
	}
	a.invalidateCanvas()
	a.updateWindowTitle()
	a.updateStatus()
}

func (a *App) onRedo() {
	if len(a.redoStack) == 0 {
		return
	}
	a.undoStack = append(a.undoStack, a.snapshot())
	snap := a.redoStack[len(a.redoStack)-1]
	a.redoStack = a.redoStack[:len(a.redoStack)-1]
	a.elements = make([]LabelElement, len(snap))
	copy(a.elements, snap)
	if a.selectedIdx >= len(a.elements) {
		a.selectedIdx = -1
	}
	a.invalidateCanvas()
	a.updateWindowTitle()
	a.updateStatus()
}

func (a *App) renderW() float64 {
	if a.currentLayout == nil {
		return 25
	}
	w, _ := render.LandscapeDesignSize(*a.currentLayout)
	for _, el := range a.elements {
		if right := el.XMM + el.WidthMM; right > w {
			w = right
		}
	}
	return w
}

func (a *App) renderH() float64 {
	if a.currentLayout == nil {
		return 55.5
	}
	_, h := render.LandscapeDesignSize(*a.currentLayout)
	for _, el := range a.elements {
		if bottom := el.YMM + el.HeightMM; bottom > h {
			h = bottom
		}
	}
	return h
}

func (a *App) setZoomToPage(widthOnly bool) {
	if a.currentLayout == nil || a.canvas == nil {
		return
	}
	b := a.canvas.ClientBoundsPixels()
	dpi := float64(a.dpi)
	if dpi <= 0 {
		dpi = 96
	}
	baseW := a.renderW() * dpi / 25.4
	baseH := a.renderH() * dpi / 25.4
	availW := float64(b.Width - rulerSize - 24)
	availH := float64(b.Height - rulerSize - 24)
	if availW <= 0 || availH <= 0 || baseW <= 0 || baseH <= 0 {
		return
	}
	if widthOnly {
		a.zoom = math.Max(0.25, math.Min(4.0, availW/baseW))
	} else {
		a.zoom = math.Max(0.25, math.Min(4.0, math.Min(availW/baseW, availH/baseH)))
	}
	a.scrollX = 0
	a.scrollY = 0
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) handleKeyDown(key walk.Key) {
	mods := walk.ModifiersDown()
	if mods&walk.ModControl != 0 {
		switch key {
		case walk.KeyC:
			a.onCopy()
		case walk.KeyD:
			a.onCopy()
			a.onPaste()
		case walk.KeyV:
			a.onPaste()
		case walk.KeyX:
			a.onCopy()
			a.onDelete()
		case walk.KeyS:
			a.onSave()
		case walk.KeyO:
			a.onOpen()
		case walk.KeyP:
			a.onPrint()
		case walk.KeyZ:
			a.onUndo()
		case walk.KeyY:
			a.onRedo()
		case walk.KeyAdd, walk.KeyOEMPlus:
			a.zoom = math.Min(4.0, a.zoom*1.25)
			a.invalidateCanvas()
			a.updateStatus()
		case walk.KeySubtract, walk.KeyOEMMinus:
			a.zoom = math.Max(0.25, a.zoom/1.25)
			a.invalidateCanvas()
			a.updateStatus()
		case walk.Key0:
			a.zoom = 1.0
			a.scrollX = 0
			a.scrollY = 0
			a.invalidateCanvas()
			a.updateStatus()
		}
		return
	}
	if key == walk.KeyDelete {
		a.onDelete()
		return
	}
	if key == walk.KeyEscape {
		a.setTool("select")
		if a.toolbar != nil {
			a.toolbar.Invalidate()
		}
		return
	}
	if key == walk.KeyHome {
		a.scrollX = 0
		a.scrollY = 0
		if mods&walk.ModControl != 0 {
			a.setZoomToPage(false)
			return
		}
		a.invalidateCanvas()
		return
	}
	if key == walk.KeyReturn {
		a.onProperties()
		return
	}
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.elements) {
		return
	}
	step := 0.5
	if mods&walk.ModShift != 0 {
		step = 0.1
	}
	el := &a.elements[a.selectedIdx]
	switch key {
	case walk.KeyLeft:
		a.pushUndo()
		el.XMM -= step
	case walk.KeyRight:
		a.pushUndo()
		el.XMM += step
	case walk.KeyUp:
		a.pushUndo()
		el.YMM -= step
	case walk.KeyDown:
		a.pushUndo()
		el.YMM += step
	default:
		return
	}
	a.clampElementToLabel(el)
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) toolbarButtons(bounds walk.Rectangle) []ToolbarButton {
	// Order follows CadMapa's TLAYOUTDESKTOP resource: Toolbar974 (standard),
	// Toolbar971 (font), Toolbar973 (objects), and Toolbar972 (database merge).
	standard := []ToolbarButton{
		{ID: "new", Tooltip: "Nova Etiqueta"}, {ID: "open", Tooltip: "Abrir Etiqueta"}, {ID: "save", Tooltip: "Salvar Etiqueta", GapAfter: true},
		{ID: "print", Tooltip: "Imprimir", GapAfter: true},
		{ID: "cut", Tooltip: "Cortar"}, {ID: "copy", Tooltip: "Copiar"}, {ID: "paste", Tooltip: "Colar", GapAfter: true},
		{ID: "alignDialog", Tooltip: "Alinhamento", GapAfter: true},
		{ID: "group", Tooltip: "Agrupar"}, {ID: "ungroup", Tooltip: "Desagrupar", GapAfter: true},
		{ID: "bring", Tooltip: "Trazer para Frente"}, {ID: "send", Tooltip: "Enviar para Trás", GapAfter: true},
		{ID: "zoomPage", Tooltip: "Zoom de Página Inteira"}, {ID: "zoomWidth", Tooltip: "Zoom de Largura da Página"}, {ID: "zoom100", Tooltip: "Zoom de 100%", GapAfter: true},
		{ID: "zoomCombo", Label: "100%", Tooltip: "Zoom", GapAfter: true},
		{ID: "help", Tooltip: "Ajuda"}, {ID: "exit", Tooltip: "Sair"},
	}
	fontTools := []ToolbarButton{
		{ID: "fontCombo", Tooltip: "Fontes"}, {ID: "printSetup", Tooltip: "Impressora"}, {ID: "fontSize", Tooltip: "Tamanho da Fonte", GapAfter: true},
		{ID: "bold", Tooltip: "Negrito"}, {ID: "italic", Tooltip: "Itálico"}, {ID: "underline", Tooltip: "Sublinhado"}, {ID: "strike", Tooltip: "Cortado", GapAfter: true},
		{ID: "alignL", Tooltip: "Alinhamento a Esquerda"}, {ID: "alignC", Tooltip: "Alinhamento ao Centro"}, {ID: "alignR", Tooltip: "Alinhamento a Direita"}, {ID: "bullets", Tooltip: "Marcadores"}, {ID: "color", Tooltip: "Cor do Texto", GapAfter: true},
	}
	objectsAndDB := []ToolbarButton{
		{ID: "select", Tooltip: "Apontador", GapAfter: true},
		{ID: "zoom", Tooltip: "Zoom", GapAfter: true},
		{ID: "line", Tooltip: "Linha"}, {ID: "roundRect", Tooltip: "Retângulo Ovalado"}, {ID: "rect", Tooltip: "Quadrado"}, {ID: "ellipse", Tooltip: "Oval", GapAfter: true},
		{ID: "simpleText", Tooltip: "Texto Simples"}, {ID: "text", Tooltip: "Caixa de Texto"}, {ID: "artText", Tooltip: "Texto Artístico", GapAfter: true},
		{ID: "barcode", Tooltip: "Código de Barras"}, {ID: "image", Tooltip: "Figura"}, {ID: "ole", Tooltip: "Ole"}, {ID: "mapaRisc", Tooltip: "Figura"}, {ID: "fileMan", Tooltip: "Figura", GapAfter: true},
		{ID: "mergeToggle", Tooltip: "Ativar/Desativar Mescla"}, {ID: "merge", Label: "Mesclar:", Tooltip: "Mesclar"}, {ID: "navFirst", Tooltip: "Primeiro Registro"}, {ID: "navPrev", Tooltip: "Registro Anterior"}, {ID: "navNext", Tooltip: "Próximo Registro"}, {ID: "navLast", Tooltip: "Último Registro"},
	}
	buttons := make([]ToolbarButton, 0, len(standard)+len(fontTools)+len(objectsAndDB))
	decodedLeft := map[string]int{
		"new": 0, "open": 22, "save": 43, "print": 69, "cut": 95, "copy": 116, "paste": 137,
		"alignDialog": 163, "group": 189, "ungroup": 210, "bring": 236, "send": 257,
		"zoomPage": 283, "zoomWidth": 304, "zoom100": 325, "zoomCombo": 387, "help": 470, "exit": 491,
		"fontCombo": 0, "printSetup": 219, "fontSize": 256, "bold": 334, "italic": 356, "underline": 378, "strike": 400,
		"alignL": 405, "alignC": 426, "alignR": 447, "bullets": 473, "color": 518,
		"select": 0, "zoom": 27, "line": 54, "roundRect": 76, "rect": 98, "ellipse": 120,
		"simpleText": 147, "text": 169, "artText": 191, "barcode": 218, "image": 245, "ole": 267, "mapaRisc": 321, "fileMan": 343,
		"mergeToggle": 357, "merge": 401, "navFirst": 532, "navPrev": 545, "navNext": 558, "navLast": 571,
	}
	place := func(row []ToolbarButton, y int) {
		x := bounds.X + 4
		for i := range row {
			w := 24
			if row[i].ID == "fontCombo" {
				w = 219
			}
			if row[i].ID == "printSetup" {
				w = 28
			}
			if row[i].ID == "fontSize" {
				w = 54
			}
			if row[i].ID == "color" {
				w = 41
			}
			if row[i].ID == "merge" {
				w = 118
			}
			if row[i].ID == "zoomCombo" {
				w = 114
			}
			left := x
			if decodedX, ok := decodedLeft[row[i].ID]; ok {
				left = bounds.X + 4 + decodedX
			}
			row[i].Rect = walk.Rectangle{X: left, Y: bounds.Y + y, Width: w, Height: 22}
			row[i].Enabled = !isStubToolbarCommand(row[i].ID)
			buttons = append(buttons, row[i])
			x += w + 3
			if row[i].GapAfter {
				x += 8
			}
		}
	}
	place(standard, 23)
	place(fontTools, 49)
	place(objectsAndDB, 75)
	return buttons
}

func (a *App) paintToolbar(canvas *walk.Canvas, bounds walk.Rectangle) error {
	if a.toolbar != nil {
		bounds = a.toolbar.ClientBoundsPixels()
	}
	canvas.FillRectanglePixels(darkBrush, walk.Rectangle{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: topbarHeight})
	font, _ := walk.NewFont("MS Sans Serif", 8, 0)
	defer font.Dispose()
	sepPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(160, 160, 160))
	defer sepPen.Dispose()
	for _, btn := range a.toolbarButtons(bounds) {
		if btn.ID == "fontCombo" || btn.ID == "fontSize" || btn.ID == "zoomCombo" || btn.ID == "merge" || btn.ID == "color" {
			a.drawToolbarCombo(canvas, btn, font)
			continue
		}
		face := walk.RGB(224, 224, 224)
		if !btn.Enabled {
			face = walk.RGB(192, 192, 192)
			brush, _ := walk.NewSolidColorBrush(face)
			canvas.FillRectanglePixels(brush, btn.Rect)
			brush.Dispose()
		}
		activeTool := btn.Enabled && (btn.ID == a.tool || (a.tool == "text" && (btn.ID == "simpleText" || btn.ID == "artText")) || (a.tool == "rect" && btn.ID == "roundRect"))
		if activeTool {
			face = walk.RGB(210, 225, 245)
			brush, _ := walk.NewSolidColorBrush(face)
			canvas.FillRectanglePixels(brush, btn.Rect)
			brush.Dispose()
			pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(128, 128, 128))
			canvas.DrawRectanglePixels(pen, btn.Rect)
			pen.Dispose()
		}
		textColor := walk.RGB(0, 0, 0)
		if !btn.Enabled {
			textColor = walk.RGB(128, 128, 128)
		} else if btn.ID == "help" {
			textColor = walk.RGB(220, 0, 0)
		}
		a.drawToolbarIcon(canvas, btn, font, textColor)
		if btn.GapAfter {
			x := btn.Rect.X + btn.Rect.Width + 4
			canvas.DrawLinePixels(sepPen, walk.Point{X: x, Y: btn.Rect.Y + 2}, walk.Point{X: x, Y: btn.Rect.Y + btn.Rect.Height - 2})
		}
	}
	_ = a.paintMenuBar(canvas, walk.Rectangle{X: bounds.X, Y: bounds.Y + menuTopY, Width: bounds.Width, Height: menuHeight})
	return nil
}

func (a *App) drawToolbarCombo(canvas *walk.Canvas, btn ToolbarButton, font *walk.Font) {
	brush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 255))
	canvas.FillRectanglePixels(brush, btn.Rect)
	brush.Dispose()
	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(160, 160, 160))
	canvas.DrawRectanglePixels(pen, btn.Rect)
	pen.Dispose()
	text := btn.Label
	if btn.ID == "fontCombo" {
		text = a.defaultFontName
		if text == "" {
			text = "Arial"
		}
		if a.selectedIdx >= 0 && a.selectedIdx < len(a.elements) {
			text = a.elements[a.selectedIdx].FontName
		}
		if len(text) > 20 {
			text = text[:20]
		}
	}
	if btn.ID == "fontSize" {
		text = fmt.Sprintf("%.0f", a.defaultFontSize)
		if a.defaultFontSize <= 0 {
			text = "8"
		}
		if a.selectedIdx >= 0 && a.selectedIdx < len(a.elements) {
			text = fmt.Sprintf("%.0f", a.elements[a.selectedIdx].FontSize)
		}
	}
	if btn.ID == "zoomCombo" {
		text = fmt.Sprintf("%.0f%%", a.zoom*100)
	}
	if btn.ID == "merge" {
		text = "Mesclar:  (nenhum)"
		if a.mergeField != "" {
			text = "Mesclar:  " + a.mergeField
		}
	}
	if btn.ID == "color" {
		text = ""
		brushColor := a.defaultTextWalkColor()
		if a.selectedIdx >= 0 && a.selectedIdx < len(a.elements) && a.elements[a.selectedIdx].Color != nil {
			cr, cg, cb, _ := a.elements[a.selectedIdx].Color.RGBA()
			brushColor = walk.RGB(byte(cr>>8), byte(cg>>8), byte(cb>>8))
		}
		brush, _ := walk.NewSolidColorBrush(brushColor)
		canvas.FillRectanglePixels(brush, walk.Rectangle{X: btn.Rect.X + 5, Y: btn.Rect.Y + 5, Width: 14, Height: 10})
		brush.Dispose()
	}
	canvas.DrawTextPixels(text, font, walk.RGB(0, 0, 0), walk.Rectangle{X: btn.Rect.X + 4, Y: btn.Rect.Y + 3, Width: btn.Rect.Width - 20, Height: btn.Rect.Height - 5}, 0)
	canvas.DrawTextPixels("▼", font, walk.RGB(0, 0, 0), walk.Rectangle{X: btn.Rect.X + btn.Rect.Width - 18, Y: btn.Rect.Y + 3, Width: 16, Height: btn.Rect.Height - 5}, walk.TextCenter)
}

func (a *App) drawToolbarIcon(canvas *walk.Canvas, btn ToolbarButton, font *walk.Font, color walk.Color) {
	r := btn.Rect
	cx, cy := r.X+r.Width/2, r.Y+r.Height/2
	if img, ok := a.toolbarGlyphs[btn.ID]; ok {
		size := 16
		_ = canvas.DrawImageStretchedPixels(img, walk.Rectangle{X: cx - size/2, Y: cy - size/2, Width: size, Height: size})
		return
	}
	black, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
	blue, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 170))
	red, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(200, 0, 0))
	defer black.Dispose()
	defer blue.Dispose()
	defer red.Dispose()
	icon := walk.Rectangle{X: r.X + 4, Y: r.Y + 3, Width: r.Width - 8, Height: r.Height - 6}
	switch btn.ID {
	case "select":
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 2, Y: icon.Y + 1}, walk.Point{X: icon.X + 11, Y: icon.Y + 9})
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 2, Y: icon.Y + 1}, walk.Point{X: icon.X + 5, Y: icon.Y + 12})
	case "zoom":
		canvas.DrawEllipsePixels(blue, walk.Rectangle{X: icon.X + 1, Y: icon.Y + 1, Width: 9, Height: 9})
		canvas.DrawLinePixels(blue, walk.Point{X: icon.X + 9, Y: icon.Y + 9}, walk.Point{X: icon.X + 14, Y: icon.Y + 14})
	case "zoom100":
		canvas.DrawTextPixels("100", font, walk.RGB(0, 0, 0), r, walk.TextCenter|walk.TextVCenter)
	case "line":
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 1, Y: icon.Y + 13}, walk.Point{X: icon.X + 14, Y: icon.Y + 2})
	case "rect":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 3, Width: 12, Height: 9})
	case "roundRect":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 3, Width: 12, Height: 9})
		canvas.DrawEllipsePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 3, Width: 4, Height: 4})
	case "ellipse":
		canvas.DrawEllipsePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 3, Width: 12, Height: 9})
	case "text", "font", "field", "simpleText", "artText":
		canvas.DrawTextPixels("A", font, walk.RGB(0, 0, 170), r, walk.TextCenter|walk.TextVCenter)
		if btn.ID == "field" || btn.ID == "text" {
			canvas.DrawRectanglePixels(black, walk.Rectangle{X: r.X + 13, Y: r.Y + 12, Width: 6, Height: 5})
		}
		if btn.ID == "artText" {
			canvas.DrawLinePixels(red, walk.Point{X: icon.X + 2, Y: icon.Y + 12}, walk.Point{X: icon.X + 14, Y: icon.Y + 12})
		}
	case "barcode":
		for i := 0; i < 6; i++ {
			x := icon.X + 2 + i*2
			canvas.DrawLinePixels(black, walk.Point{X: x, Y: icon.Y + 2}, walk.Point{X: x, Y: icon.Y + 13})
		}
	case "image":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 2, Width: 12, Height: 11})
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 3, Y: icon.Y + 12}, walk.Point{X: icon.X + 8, Y: icon.Y + 7})
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 8, Y: icon.Y + 7}, walk.Point{X: icon.X + 14, Y: icon.Y + 12})
	case "db", "dbopen", "dbtable", "mergeToggle":
		canvas.DrawTextPixels("DB", font, walk.RGB(0, 0, 0), r, walk.TextCenter|walk.TextVCenter)
	case "new":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 3, Y: icon.Y + 2, Width: 10, Height: 12})
	case "open":
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 1, Y: icon.Y + 6}, walk.Point{X: icon.X + 5, Y: icon.Y + 3})
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 6, Width: 13, Height: 8})
	case "save":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 2, Width: 12, Height: 12})
		canvas.DrawRectanglePixels(blue, walk.Rectangle{X: icon.X + 5, Y: icon.Y + 3, Width: 6, Height: 4})
	case "print":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 2, Y: icon.Y + 6, Width: 13, Height: 7})
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 4, Y: icon.Y + 2, Width: 9, Height: 5})
	case "printSetup":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 1, Y: icon.Y + 6, Width: 12, Height: 7})
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 3, Y: icon.Y + 2, Width: 8, Height: 5})
		canvas.DrawTextPixels("▼", font, walk.RGB(0, 0, 0), walk.Rectangle{X: r.X + r.Width - 9, Y: r.Y + 5, Width: 8, Height: 10}, walk.TextCenter)
	case "cut":
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 2, Y: icon.Y + 3}, walk.Point{X: icon.X + 13, Y: icon.Y + 13})
		canvas.DrawLinePixels(black, walk.Point{X: icon.X + 13, Y: icon.Y + 3}, walk.Point{X: icon.X + 2, Y: icon.Y + 13})
	case "copy", "paste", "props", "grid", "bring", "send", "lock", "preview", "alignDialog", "group", "ungroup", "zoomPage", "zoomWidth", "exit":
		canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 3, Y: icon.Y + 3, Width: 10, Height: 10})
		if btn.ID == "copy" || btn.ID == "paste" {
			canvas.DrawRectanglePixels(black, walk.Rectangle{X: icon.X + 1, Y: icon.Y + 1, Width: 10, Height: 10})
		}
		if btn.ID == "zoomPage" || btn.ID == "zoomWidth" {
			canvas.DrawEllipsePixels(blue, walk.Rectangle{X: icon.X + 5, Y: icon.Y + 5, Width: 8, Height: 8})
		}
		if btn.ID == "exit" {
			canvas.DrawLinePixels(red, walk.Point{X: icon.X + 3, Y: icon.Y + 3}, walk.Point{X: icon.X + 13, Y: icon.Y + 13})
			canvas.DrawLinePixels(red, walk.Point{X: icon.X + 13, Y: icon.Y + 3}, walk.Point{X: icon.X + 3, Y: icon.Y + 13})
		}
	case "delete", "stop":
		canvas.DrawRectanglePixels(red, walk.Rectangle{X: icon.X + 4, Y: icon.Y + 4, Width: 8, Height: 8})
	case "bold":
		canvas.DrawTextPixels("N", font, walk.RGB(90, 90, 90), r, walk.TextCenter|walk.TextVCenter)
	case "italic":
		canvas.DrawTextPixels("I", font, walk.RGB(90, 90, 90), r, walk.TextCenter|walk.TextVCenter)
	case "underline":
		canvas.DrawTextPixels("S", font, walk.RGB(90, 90, 90), r, walk.TextCenter|walk.TextVCenter)
	case "strike":
		canvas.DrawTextPixels("ABC", font, walk.RGB(90, 90, 90), r, walk.TextCenter|walk.TextVCenter)
	case "alignL", "alignC", "alignR", "bullets":
		y := icon.Y + 3
		for i := 0; i < 3; i++ {
			canvas.DrawLinePixels(black, walk.Point{X: icon.X + 2, Y: y + i*4}, walk.Point{X: icon.X + 14, Y: y + i*4})
		}
	case "color":
		brush, _ := walk.NewSolidColorBrush(walk.RGB(0, 0, 0))
		canvas.FillRectanglePixels(brush, walk.Rectangle{X: icon.X + 3, Y: icon.Y + 4, Width: 11, Height: 8})
		brush.Dispose()
	case "help":
		canvas.DrawTextPixels("?", font, walk.RGB(220, 0, 0), r, walk.TextCenter|walk.TextVCenter)
	case "navFirst", "navPrev", "navNext", "navLast":
		label := "◀"
		if btn.ID == "navNext" || btn.ID == "navLast" {
			label = "▶"
		}
		canvas.DrawTextPixels(label, font, walk.RGB(0, 0, 0), r, walk.TextCenter|walk.TextVCenter)
	default:
		canvas.DrawTextPixels(btn.Label, font, color, r, walk.TextCenter|walk.TextVCenter)
	}
	_ = cx
	_ = cy
}

func (a *App) toolbarMouseDown(x, y int, button walk.MouseButton) {
	log.Printf("toolbarMouseDown: x=%d y=%d button=%v", x, y, button)
	if button != walk.LeftButton || a.toolbar == nil {
		return
	}
	b := a.toolbar.ClientBoundsPixels()
	if y >= menuTopY && y < menuTopY+menuHeight {
		menuBounds := walk.Rectangle{X: b.X, Y: b.Y + menuTopY, Width: b.Width, Height: menuHeight}
		for _, item := range a.menuHotspots(menuBounds) {
			if x >= item.Rect.X && x <= item.Rect.X+item.Rect.Width && y >= item.Rect.Y && y <= item.Rect.Y+item.Rect.Height {
				log.Printf("menu command: %s", item.ID)
				a.runMenuCommand(item.ID, item.Rect)
				return
			}
		}
		return
	}
	for _, btn := range a.toolbarButtons(b) {
		if !btn.Enabled {
			continue
		}
		if x >= btn.Rect.X && x <= btn.Rect.X+btn.Rect.Width && y >= btn.Rect.Y && y <= btn.Rect.Y+btn.Rect.Height {
			log.Printf("toolbar command: %s", btn.ID)
			switch btn.ID {
			case "fontCombo":
				a.showToolbarPopup(btn, a.buildFontMenu)
			case "fontSize":
				a.showToolbarPopup(btn, a.buildFontSizeMenu)
			case "zoomCombo":
				a.showToolbarPopup(btn, a.buildZoomMenu)
			case "merge":
				a.showToolbarPopup(btn, a.buildMergeMenu)
			default:
				a.runToolbarCommand(btn.ID)
			}
			if a.toolbar != nil {
				a.toolbar.Invalidate()
			}
			return
		}
	}
}

func (a *App) menuHotspots(bounds walk.Rectangle) []MenuHotspot {
	items := []MenuHotspot{{"arquivo", "Arquivo", walk.Rectangle{}}, {"editar", "Editar", walk.Rectangle{}}, {"objeto", "Objeto", walk.Rectangle{}}, {"banco", "Banco de Dados", walk.Rectangle{}}, {"opcoes", "Opções", walk.Rectangle{}}, {"ajuda", "Ajuda ?", walk.Rectangle{}}}
	x := bounds.X + 6
	for i := range items {
		w := 52
		if items[i].ID == "banco" {
			w = 104
		}
		if items[i].ID == "opcoes" || items[i].ID == "arquivo" {
			w = 58
		}
		items[i].Rect = walk.Rectangle{X: x, Y: bounds.Y + 2, Width: w, Height: bounds.Height - 4}
		x += w + 2
	}
	return items
}

func (a *App) paintMenuBar(canvas *walk.Canvas, bounds walk.Rectangle) error {
	canvas.FillRectanglePixels(whiteBrush, bounds)
	font, _ := walk.NewFont("MS Sans Serif", 8, 0)
	defer font.Dispose()
	for _, item := range a.menuHotspots(bounds) {
		canvas.DrawTextPixels(item.Text, font, walk.RGB(0, 0, 0), item.Rect, walk.TextVCenter)
	}
	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(220, 220, 220))
	canvas.DrawLinePixels(pen, walk.Point{X: bounds.X, Y: bounds.Y + bounds.Height - 1}, walk.Point{X: bounds.X + bounds.Width, Y: bounds.Y + bounds.Height - 1})
	pen.Dispose()
	return nil
}
func (a *App) runMenuCommand(id string, rect walk.Rectangle) {
	switch id {
	case "arquivo":
		a.showMenuPopup(rect, a.buildArquivoMenu)
	case "editar":
		a.showMenuPopup(rect, a.buildEditarMenu)
	case "objeto":
		a.showMenuPopup(rect, a.buildObjetoMenu)
	case "banco":
		walk.MsgBox(a.mainWindow, "Banco de Dados", stubFeatureMessage("Cadastro de pessoas/produtos"), walk.MsgBoxIconInformation)
	case "opcoes":
		walk.MsgBox(a.mainWindow, "Opções", stubFeatureMessage("Configuracoes de pagina"), walk.MsgBoxIconInformation)
	case "ajuda":
		walk.MsgBox(a.mainWindow, "MasterPrint 3.0", "Paulimaq MasterPrint 3.0", walk.MsgBoxIconInformation)
	}
}

func (a *App) showMenuPopup(rect walk.Rectangle, build func(*walk.Menu) error) {
	a.showToolbarPopup(ToolbarButton{Rect: rect}, build)
}

func (a *App) buildArquivoMenu(menu *walk.Menu) error {
	return addMenuItems(menu, a.arquivoMenuItems())
}

func (a *App) arquivoMenuItems() []menuItemAction {
	return []menuItemAction{
		{label: "Novo", fn: a.onNew},
		{label: "Abrir", fn: a.onOpen},
		{label: "Salvar", fn: a.onSave},
		{label: "Salvar Como...", fn: a.onSaveAs},
		{label: "Salvar Como Modelo...", stub: true},
		{label: "Reabrir", stub: true},
		{label: "Exportar...", stub: true},
		{label: "Configurar Documento", stub: true},
		{label: "Imprimir", fn: a.onPrint},
		{label: "Sair", fn: a.onExit},
	}
}

func (a *App) buildEditarMenu(menu *walk.Menu) error {
	return addMenuItems(menu, a.editarMenuItems())
}

func (a *App) editarMenuItems() []menuItemAction {
	return []menuItemAction{
		{label: "Recortar", fn: func() { a.onCopy(); a.onDelete() }},
		{label: "Copiar", fn: a.onCopy},
		{label: "Colar", fn: a.onPaste},
		{label: "Apagar", fn: a.onDelete},
	}
}

func (a *App) buildObjetoMenu(menu *walk.Menu) error {
	return addMenuItems(menu, a.objetoMenuItems())
}

func (a *App) objetoMenuItems() []menuItemAction {
	return []menuItemAction{
		{label: "Borda", stub: true},
		{label: "Preenchimento", stub: true},
		{label: "Fonte", stub: true},
		{sep: true},
		{label: "Enviar Para Trás", fn: a.onSendToBack},
		{label: "Trazer Para Frente", fn: a.onBringToFront},
		{sep: true},
		{label: "A&grupar", stub: true},
		{label: "Desagr&upar", stub: true},
		{sep: true},
		{label: "&Alinhar", stub: true},
		{label: "Escalonar", stub: true},
		{sep: true},
		{label: "Propriedades", fn: a.onProperties},
	}
}

func (a *App) buildPopupMenu1(menu *walk.Menu) error {
	return addMenuItems(menu, a.popupMenu1Items())
}

func (a *App) popupMenu1Items() []menuItemAction {
	return a.objetoMenuItems()
}

func addMenuAction(menu *walk.Menu, label string, fn func()) error {
	act := walk.NewAction()
	if err := act.SetText(label); err != nil {
		return err
	}
	act.Triggered().Attach(fn)
	return menu.Actions().Add(act)
}

func addMenuItems(menu *walk.Menu, items []menuItemAction) error {
	for _, item := range items {
		if item.sep {
			if err := menu.Actions().Add(walk.NewSeparatorAction()); err != nil {
				return err
			}
			continue
		}
		stub := item.stub || isStubMenuLabel(item.label)
		act := walk.NewAction()
		if err := act.SetText(item.label); err != nil {
			return err
		}
		if stub {
			act.SetEnabled(false)
		} else {
			act.Triggered().Attach(item.fn)
		}
		if err := menu.Actions().Add(act); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) showCanvasPopup(x, y int) {
	if a.canvas == nil || a.currentLayout == nil {
		return
	}
	menu, err := walk.NewMenu()
	if err != nil {
		return
	}
	defer menu.Dispose()
	if err := a.buildPopupMenu1(menu); err != nil {
		log.Printf("canvas popup build failed: %v", err)
		return
	}
	a.canvas.SetContextMenu(menu)
	defer a.canvas.SetContextMenu(nil)
	pt := win.POINT{X: int32(x), Y: int32(y)}
	win.ClientToScreen(a.canvas.Handle(), &pt)
	win.SendMessage(a.canvas.Handle(), win.WM_CONTEXTMENU, uintptr(a.canvas.Handle()), uintptr(win.MAKELONG(uint16(pt.X), uint16(pt.Y))))
}

func (a *App) runToolbarCommand(id string) {
	switch id {
	case "new":
		a.onNew()
	case "open":
		a.onOpen()
	case "save":
		a.onSave()
	case "print":
		a.onPrint()
	case "select", "text", "line", "rect", "ellipse":
		a.setTool(id)
	case "roundRect":
		a.setTool("rect")
	case "simpleText", "artText":
		a.setTool("text")
	case "zoom":
		a.setTool("zoom")
	case "zoom100":
		a.zoom = 1.0
		a.scrollX = 0
		a.scrollY = 0
		a.invalidateCanvas()
		a.updateStatus()
	case "zoomCombo":
		a.onZoomPopup()
	case "zoomPage":
		a.setZoomToPage(false)
	case "zoomWidth":
		a.setZoomToPage(true)
	case "delete", "cut":
		if id == "cut" {
			a.onCopy()
		}
		a.onDelete()
	case "copy":
		a.onCopy()
	case "paste":
		a.onPaste()
	case "props", "font":
		a.onProperties()
	case "bold":
		a.toggleSelectedTextStyle("bold")
	case "italic":
		a.toggleSelectedTextStyle("italic")
	case "underline":
		a.toggleSelectedTextStyle("underline")
	case "alignL":
		a.setSelectedTextAlign("left")
	case "alignC":
		a.setSelectedTextAlign("center")
	case "alignR":
		a.setSelectedTextAlign("right")
	case "fontCombo":
		a.onFontPopup()
	case "fontSize":
		a.onFontSizePopup()
	case "color":
		a.onColorPopup()
	case "merge":
		a.onMergePopup()
	case "navFirst", "navPrev", "navNext", "navLast":
		return
	case "exit":
		a.onExit()
	case "bring":
		a.onBringToFront()
	case "send":
		a.onSendToBack()
	case "grid":
		a.showGrid = !a.showGrid
		a.invalidateCanvas()
	case "lock":
		if a.selectedIdx >= 0 && a.selectedIdx < len(a.elements) {
			a.pushUndo()
			a.elements[a.selectedIdx].Locked = !a.elements[a.selectedIdx].Locked
			a.invalidateCanvas()
			a.updateStatus()
		}
	case "preview":
		a.onPrintPreview()
	case "barcode":
		a.setTool("text")
	case "strike":
		if a.selectedIdx >= 0 && a.selectedIdx < len(a.elements) {
			a.pushUndo()
			a.elements[a.selectedIdx].Italic = !a.elements[a.selectedIdx].Italic
			syncEditableTextPayload(&a.elements[a.selectedIdx])
			a.invalidateCanvas()
			a.updateStatus()
		}
	case "printSetup":
		a.onPrintSetup()
	case "image":
		a.onInsertImage()
	}
}

func (a *App) toggleSelectedTextStyle(style string) {
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.elements) || a.elements[a.selectedIdx].Type != "text" {
		return
	}
	a.pushUndo()
	el := &a.elements[a.selectedIdx]
	switch style {
	case "bold":
		el.Bold = !el.Bold
	case "italic":
		el.Italic = !el.Italic
	case "underline":
		el.Underline = !el.Underline
	}
	syncEditableTextPayload(el)
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) setSelectedTextAlign(align string) {
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.elements) || a.elements[a.selectedIdx].Type != "text" {
		return
	}
	a.pushUndo()
	el := &a.elements[a.selectedIdx]
	el.Align = align
	syncEditableTextPayload(el)
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) cycleZoom() {
	levels := []float64{0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 2.0, 3.0, 4.0}
	best := 0
	for i, z := range levels {
		if a.zoom <= z+0.01 {
			best = i
			break
		}
	}
	best = (best + 1) % len(levels)
	a.zoom = levels[best]
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) onFontPopup() { a.onFontPicker() }

func (a *App) onFontSizePopup() { a.onFontSizeCycle() }

func (a *App) onColorPopup() { a.onColorPicker() }

func (a *App) onZoomPopup() { a.onZoomPicker() }

func (a *App) onMergePopup() {
	var dlg *walk.Dialog
	var list *walk.ListBox
	items := []string{"(nenhum)"}
	Dialog{
		AssignTo: &dlg, Title: "Mesclar", MinSize: Size{Width: 220, Height: 160}, Layout: VBox{},
		Children: []Widget{
			ListBox{AssignTo: &list, MinSize: Size{Height: 80}, OnItemActivated: func() { dlg.Accept() }},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	list.SetModel(items)
	list.SetCurrentIndex(0)
	dlg.Run()
	dlg.Dispose()
}

func (a *App) showToolbarPopup(btn ToolbarButton, build func(*walk.Menu) error) {
	if a.toolbar == nil {
		return
	}
	menu, err := walk.NewMenu()
	if err != nil {
		return
	}
	defer menu.Dispose()
	if err := build(menu); err != nil {
		log.Printf("toolbar popup build failed: %v", err)
		return
	}
	a.toolbar.SetContextMenu(menu)
	defer a.toolbar.SetContextMenu(nil)
	pt := win.POINT{X: int32(btn.Rect.X), Y: int32(btn.Rect.Y + btn.Rect.Height)}
	win.ClientToScreen(a.toolbar.Handle(), &pt)
	win.SendMessage(a.toolbar.Handle(), win.WM_CONTEXTMENU, uintptr(a.toolbar.Handle()), uintptr(win.MAKELONG(uint16(pt.X), uint16(pt.Y))))
}

func (a *App) buildFontMenu(menu *walk.Menu) error {
	current := a.defaultFontName
	if el := a.selectedTextElement(); el != nil && el.FontName != "" {
		current = el.FontName
	}
	for _, fontName := range toolbarFontNames() {
		name := fontName
		act := walk.NewAction()
		if err := act.SetText(name); err != nil {
			return err
		}
		if strings.EqualFold(name, current) {
			_ = act.SetCheckable(true)
			_ = act.SetChecked(true)
		}
		act.Triggered().Attach(func() { a.applyFontName(name) })
		if err := menu.Actions().Add(act); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) buildFontSizeMenu(menu *walk.Menu) error {
	current := a.defaultFontSize
	if el := a.selectedTextElement(); el != nil && el.FontSize > 0 {
		current = el.FontSize
	}
	for _, size := range toolbarFontSizes() {
		s := size
		act := walk.NewAction()
		if err := act.SetText(fmt.Sprintf("%.0f pt", s)); err != nil {
			return err
		}
		if math.Abs(current-s) < 0.5 {
			_ = act.SetCheckable(true)
			_ = act.SetChecked(true)
		}
		act.Triggered().Attach(func() { a.applyFontSize(s) })
		if err := menu.Actions().Add(act); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) buildZoomMenu(menu *walk.Menu) error {
	for _, level := range toolbarZoomLevels() {
		z := level
		act := walk.NewAction()
		if err := act.SetText(fmt.Sprintf("%.0f%%", z*100)); err != nil {
			return err
		}
		if math.Abs(a.zoom-z) < 0.01 {
			_ = act.SetCheckable(true)
			_ = act.SetChecked(true)
		}
		act.Triggered().Attach(func() { a.applyZoom(z) })
		if err := menu.Actions().Add(act); err != nil {
			return err
		}
	}
	for _, item := range []struct {
		label string
		fn    func()
	}{
		{label: "Página Inteira", fn: func() { a.setZoomToPage(false) }},
		{label: "Largura da Página", fn: func() { a.setZoomToPage(true) }},
	} {
		act := walk.NewAction()
		if err := act.SetText(item.label); err != nil {
			return err
		}
		fn := item.fn
		act.Triggered().Attach(fn)
		if err := menu.Actions().Add(act); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) buildMergeMenu(menu *walk.Menu) error {
	for _, item := range []string{"(nenhum)"} {
		value := item
		act := walk.NewAction()
		if err := act.SetText(value); err != nil {
			return err
		}
		if a.mergeField == "" && value == "(nenhum)" {
			_ = act.SetCheckable(true)
			_ = act.SetChecked(true)
		}
		act.Triggered().Attach(func() { a.applyMergeField(value) })
		if err := menu.Actions().Add(act); err != nil {
			return err
		}
	}
	return nil
}

func toolbarFontNames() []string {
	return []string{"Arial", "Consolas", "Courier New", "MS Sans Serif", "Serif", "Tahoma", "Times New Roman", "Verdana"}
}

func toolbarFontSizes() []float64 {
	return []float64{4, 5, 6, 7, 8, 9, 10, 11, 12, 14, 16, 18, 20, 24, 28, 32, 36, 48, 72}
}

func toolbarZoomLevels() []float64 {
	return []float64{0.25, 0.5, 0.75, 1.0, 1.5, 2.0, 3.0, 4.0}
}

func (a *App) applyFontName(name string) {
	if el := a.selectedTextElement(); el != nil {
		a.pushUndo()
		el.FontName = name
	} else {
		a.defaultFontName = name
	}
	a.invalidateCanvas()
	a.updateStatus()
	if a.toolbar != nil {
		a.toolbar.Invalidate()
	}
}

func (a *App) applyFontSize(size float64) {
	if el := a.selectedTextElement(); el != nil {
		a.pushUndo()
		el.FontSize = size
	} else {
		a.defaultFontSize = size
	}
	a.invalidateCanvas()
	a.updateStatus()
	if a.toolbar != nil {
		a.toolbar.Invalidate()
	}
}

func (a *App) applyZoom(level float64) {
	a.zoom = level
	a.invalidateCanvas()
	a.updateStatus()
	if a.toolbar != nil {
		a.toolbar.Invalidate()
	}
}

func (a *App) applyMergeField(field string) {
	if field == "(nenhum)" {
		a.mergeField = ""
	} else {
		a.mergeField = field
	}
	a.updateStatus()
	if a.toolbar != nil {
		a.toolbar.Invalidate()
	}
}

func (a *App) onClipCategoryPopup() {
	var dlg *walk.Dialog
	var list *walk.ListBox
	items := []string{"Símbolos"}
	Dialog{
		AssignTo: &dlg, Title: "Cliparts", MinSize: Size{Width: 220, Height: 150}, Layout: VBox{},
		Children: []Widget{
			ListBox{AssignTo: &list, MinSize: Size{Height: 70}, OnItemActivated: func() { dlg.Accept() }},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	list.SetModel(items)
	list.SetCurrentIndex(0)
	dlg.Run()
	dlg.Dispose()
}

func (a *App) selectedEditableElement() *LabelElement {
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.elements) {
		return nil
	}
	if a.elements[a.selectedIdx].Type == "preview" {
		return nil
	}
	return &a.elements[a.selectedIdx]
}

func (a *App) selectedTextElement() *LabelElement {
	el := a.selectedEditableElement()
	if el == nil || el.Type != "text" {
		return nil
	}
	return el
}

func (a *App) onZoomPicker() {
	levels := []float64{0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 2.0, 3.0, 4.0}
	items := make([]string, len(levels))
	for i, z := range levels {
		items[i] = fmt.Sprintf("%.0f%%", z*100)
	}
	var dlg *walk.Dialog
	var list *walk.ListBox
	Dialog{
		AssignTo: &dlg, Title: "Zoom", MinSize: Size{Width: 180, Height: 260}, Layout: VBox{},
		Children: []Widget{
			ListBox{AssignTo: &list, MinSize: Size{Height: 180}, OnItemActivated: func() { dlg.Accept() }},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	list.SetModel(items)
	for i, z := range levels {
		if math.Abs(a.zoom-z) < 0.01 {
			list.SetCurrentIndex(i)
			break
		}
	}
	if dlg.Run() == walk.DlgCmdOK {
		if idx := list.CurrentIndex(); idx >= 0 && idx < len(levels) {
			a.zoom = levels[idx]
			a.invalidateCanvas()
			a.updateStatus()
		}
	}
	dlg.Dispose()
}

func (a *App) onFontPicker() {
	fonts := []string{"Arial", "Consolas", "Courier New", "MS Sans Serif", "Serif", "Tahoma", "Times New Roman", "Verdana"}
	var dlg *walk.Dialog
	var list *walk.ListBox
	Dialog{
		AssignTo: &dlg, Title: "Selecionar Fonte", MinSize: Size{Width: 260, Height: 320}, Layout: VBox{},
		Children: []Widget{
			ListBox{AssignTo: &list, MinSize: Size{Height: 240}, OnItemActivated: func() { dlg.Accept() }},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	list.SetModel(fonts)
	current := a.defaultFontName
	if el := a.selectedTextElement(); el != nil && el.FontName != "" {
		current = el.FontName
	}
	for i, f := range fonts {
		if strings.EqualFold(f, current) {
			list.SetCurrentIndex(i)
			break
		}
	}
	if dlg.Run() == walk.DlgCmdOK {
		if idx := list.CurrentIndex(); idx >= 0 {
			if el := a.selectedTextElement(); el != nil {
				a.pushUndo()
				el.FontName = fonts[idx]
			} else {
				a.defaultFontName = fonts[idx]
			}
			a.invalidateCanvas()
			a.updateStatus()
		}
	}
	dlg.Dispose()
}

func (a *App) onFontSizeCycle() {
	sizes := []float64{4, 5, 6, 7, 8, 9, 10, 11, 12, 14, 16, 18, 20, 24, 28, 32, 36, 48, 72}
	var dlg *walk.Dialog
	var list *walk.ListBox
	Dialog{
		AssignTo: &dlg, Title: "Tamanho da Fonte", MinSize: Size{Width: 200, Height: 340}, Layout: VBox{},
		Children: []Widget{
			ListBox{AssignTo: &list, MinSize: Size{Height: 260}, OnItemActivated: func() { dlg.Accept() }},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	strSizes := make([]string, len(sizes))
	for i, s := range sizes {
		strSizes[i] = fmt.Sprintf("%.0f pt", s)
	}
	list.SetModel(strSizes)
	current := a.defaultFontSize
	if el := a.selectedTextElement(); el != nil && el.FontSize > 0 {
		current = el.FontSize
	}
	for i, s := range sizes {
		if math.Abs(current-s) < 0.5 {
			list.SetCurrentIndex(i)
			break
		}
	}
	if dlg.Run() == walk.DlgCmdOK {
		if idx := list.CurrentIndex(); idx >= 0 && idx < len(sizes) {
			if el := a.selectedTextElement(); el != nil {
				a.pushUndo()
				el.FontSize = sizes[idx]
			} else {
				a.defaultFontSize = sizes[idx]
			}
			a.invalidateCanvas()
			a.updateStatus()
		}
	}
	dlg.Dispose()
}

func (a *App) onColorPicker() {
	colors := []walk.Color{
		walk.RGB(0, 0, 0),
		walk.RGB(255, 255, 255),
		walk.RGB(255, 0, 0),
		walk.RGB(0, 128, 0),
		walk.RGB(0, 0, 255),
		walk.RGB(255, 255, 0),
		walk.RGB(255, 0, 255),
		walk.RGB(0, 255, 255),
		walk.RGB(128, 128, 128),
		walk.RGB(128, 0, 0),
		walk.RGB(0, 0, 128),
		walk.RGB(0, 128, 128),
	}
	current := a.defaultTextWalkColor()
	if el := a.selectedEditableElement(); el != nil && el.Color != nil {
		cr, cg, cb, _ := el.Color.RGBA()
		current = walk.RGB(byte(cr>>8), byte(cg>>8), byte(cb>>8))
	}
	idx := 0
	for i, c := range colors {
		if c == current {
			idx = i
			break
		}
	}
	idx = (idx + 1) % len(colors)
	newColor := color.RGBA{uint8(colors[idx]), uint8(colors[idx] >> 8), uint8(colors[idx] >> 16), 255}
	if el := a.selectedEditableElement(); el != nil {
		a.pushUndo()
		el.Color = newColor
	} else {
		a.defaultTextColor = newColor
	}
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) toggleClips() {
	a.clipsCollapsed = !a.clipsCollapsed
	a.clipsPanel.SetVisible(!a.clipsCollapsed)
}

func (a *App) loadSymbolList() {
	entries, err := os.ReadDir(a.clipartDir)
	if err != nil {
		return
	}
	filter := ""
	if a.clipFilter != nil {
		filter = strings.ToLower(a.clipFilter.Text())
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".wmf") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
			continue
		}
		names = append(names, name)
	}
	names = orderSymbolNames(names)
	a.symbolNames = names
	if a.symbolOffset >= len(a.symbolNames) {
		a.symbolOffset = 0
	}
	if a.selectedSymbol >= len(a.symbolNames) {
		a.selectedSymbol = -1
	}
	if a.symbolList != nil {
		a.symbolList.SetModel(names)
	}
	if a.symbolStrip != nil {
		a.symbolStrip.Invalidate()
	}
}

func orderSymbolNames(names []string) []string {
	preferred := []string{
		"cloro", "clorom", "clorox",
		"ferro---", "ferro--", "ferro-", "ferrox",
		"lav--30", "lav--40", "lav-30", "lav-40", "lav-50", "lav-60", "lav-95",
		"lav30", "lav40", "lav50", "lav60", "lav70", "lav95", "lavmao",
		"lavp--30", "lavp--40", "lavp-30", "lavp-40", "lavp-50", "lavp-60",
		"lavp30", "lavp40", "lavp50", "lavp60", "lavp70", "lavp95", "lavx",
		"secag", "secah", "secas", "secav", "seco-f", "seco-p", "seco-w", "secof", "secop", "secow", "secox", "seco--w", "tambor--", "tambor-", "tamborx",
	}
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	var out []string
	used := make(map[string]bool)
	for _, name := range preferred {
		if set[name] {
			out = append(out, name)
			used[name] = true
		}
	}
	var rest []string
	for _, name := range names {
		if !used[name] {
			rest = append(rest, name)
		}
	}
	sort.Slice(rest, func(i, j int) bool { return strings.ToLower(rest[i]) < strings.ToLower(rest[j]) })
	return append(out, rest...)
}

func (a *App) paintSymbolStrip(canvas *walk.Canvas, bounds walk.Rectangle) error {
	if a.symbolStrip != nil {
		bounds = a.symbolStrip.ClientBoundsPixels()
	}
	canvas.FillRectanglePixels(whiteBrush, bounds)
	borderPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(130, 130, 130))
	canvas.DrawLinePixels(borderPen, walk.Point{X: bounds.X, Y: bounds.Y}, walk.Point{X: bounds.X + bounds.Width, Y: bounds.Y})
	canvas.DrawLinePixels(borderPen, walk.Point{X: bounds.X, Y: bounds.Y + bounds.Height - 1}, walk.Point{X: bounds.X + bounds.Width, Y: bounds.Y + bounds.Height - 1})
	borderPen.Dispose()

	font, _ := walk.NewFont("MS Sans Serif", 8, 0)
	leftW := 130
	pageTop := bounds.Y + 3
	pageHeight := 53
	comboPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(120, 120, 120))
	canvas.DrawRectanglePixels(comboPen, walk.Rectangle{X: bounds.X + 1, Y: bounds.Y + 1, Width: 124, Height: 18})
	comboPen.Dispose()
	canvas.DrawTextPixels("Símbolos", font, walk.RGB(0, 0, 0), walk.Rectangle{X: bounds.X + 4, Y: bounds.Y + 3, Width: 94, Height: 14}, 0)
	canvas.DrawTextPixels("⌄", font, walk.RGB(0, 0, 0), walk.Rectangle{X: bounds.X + 110, Y: bounds.Y + 2, Width: 12, Height: 14}, walk.TextCenter)
	filterPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(160, 160, 160))
	canvas.DrawRectanglePixels(filterPen, walk.Rectangle{X: bounds.X + 1, Y: bounds.Y + 24, Width: 124, Height: 34})
	canvas.DrawLinePixels(filterPen, walk.Point{X: bounds.X + 126, Y: bounds.Y}, walk.Point{X: bounds.X + 126, Y: bounds.Y + 63})
	filterPen.Dispose()
	selectedName := ""
	if a.selectedSymbol >= 0 && a.selectedSymbol < len(a.symbolNames) {
		selectedName = a.symbolNames[a.selectedSymbol]
	}
	canvas.DrawTextPixels(selectedName, font, walk.RGB(0, 0, 0), walk.Rectangle{X: bounds.X + 3, Y: bounds.Y + 28, Width: 120, Height: 26}, walk.TextCenter|walk.TextVCenter)
	font.Dispose()

	tileW := 52
	tileH := pageHeight
	arrowW := 15
	startX := bounds.X + leftW
	if a.symbolOffset > 0 {
		leftArrowRect := walk.Rectangle{X: bounds.X + leftW, Y: pageTop, Width: arrowW, Height: tileH}
		arrowBrush, _ := walk.NewSolidColorBrush(walk.RGB(232, 242, 255))
		canvas.FillRectanglePixels(arrowBrush, leftArrowRect)
		arrowBrush.Dispose()
		arrowPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(120, 160, 210))
		canvas.DrawRectanglePixels(arrowPen, leftArrowRect)
		arrowPen.Dispose()
		arrowFont, _ := walk.NewFont("MS Sans Serif", 10, walk.FontBold)
		canvas.DrawTextPixels("◀", arrowFont, walk.RGB(0, 70, 160), leftArrowRect, walk.TextCenter|walk.TextVCenter)
		arrowFont.Dispose()
		startX += arrowW
	}
	visibleRight := bounds.X + bounds.Width - arrowW
	for i := a.symbolOffset; i < len(a.symbolNames); i++ {
		name := a.symbolNames[i]
		x := startX + (i-a.symbolOffset)*tileW
		if x+tileW > visibleRight {
			break
		}
		r := walk.Rectangle{X: x + 2, Y: pageTop, Width: tileW - 4, Height: tileH}
		penColor := walk.RGB(180, 180, 180)
		if i == a.selectedSymbol {
			penColor = walk.RGB(0, 80, 200)
		}
		p, _ := walk.NewCosmeticPen(walk.PenSolid, penColor)
		canvas.DrawRectanglePixels(p, r)
		p.Dispose()
		img, err := walk.NewImageFromFile(filepath.Join(a.clipartDir, name+".wmf"))
		if err == nil {
			canvas.DrawImageStretchedPixels(img, walk.Rectangle{X: r.X + 6, Y: r.Y + 5, Width: r.Width - 12, Height: r.Height - 10})
			img.Dispose()
		}
	}
	if len(a.symbolNames) > 0 && visibleSymbolCount(bounds.Width, leftW, tileW, arrowW) < len(a.symbolNames) {
		arrowRect := walk.Rectangle{X: bounds.X + bounds.Width - arrowW, Y: pageTop, Width: arrowW, Height: tileH}
		arrowBrush, _ := walk.NewSolidColorBrush(walk.RGB(232, 242, 255))
		canvas.FillRectanglePixels(arrowBrush, arrowRect)
		arrowBrush.Dispose()
		arrowPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(120, 160, 210))
		canvas.DrawRectanglePixels(arrowPen, arrowRect)
		arrowPen.Dispose()
		arrowFont, _ := walk.NewFont("MS Sans Serif", 10, walk.FontBold)
		label := "▶"
		if a.symbolOffset > 0 && a.symbolOffset+visibleSymbolCount(bounds.Width, leftW, tileW, arrowW) >= len(a.symbolNames) {
			label = "◀"
		}
		canvas.DrawTextPixels(label, arrowFont, walk.RGB(0, 70, 160), arrowRect, walk.TextCenter|walk.TextVCenter)
		arrowFont.Dispose()
	}
	return nil
}

func visibleSymbolCount(width, leftW, tileW, arrowW int) int {
	usable := width - leftW - arrowW
	if usable <= 0 {
		return 0
	}
	return usable / tileW
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (a *App) symbolStripMouseDown(x, y int, button walk.MouseButton) {
	if button != walk.LeftButton {
		return
	}
	if a.symbolStrip == nil {
		return
	}
	b := a.symbolStrip.ClientBoundsPixels()
	leftW, tileW, arrowW := 130, 52, 15
	if x >= 1 && x <= 125 && y >= 1 && y <= 19 {
		a.onClipCategoryPopup()
		return
	}
	count := visibleSymbolCount(b.Width, leftW, tileW, arrowW)
	if a.symbolOffset > 0 && x >= leftW && x < leftW+arrowW {
		a.symbolOffset -= count
		if a.symbolOffset < 0 {
			a.symbolOffset = 0
		}
		a.symbolStrip.Invalidate()
		return
	}
	if x >= b.Width-arrowW {
		if count <= 0 || len(a.symbolNames) <= count {
			return
		}
		if a.symbolOffset+count >= len(a.symbolNames) {
			a.symbolOffset = 0
		} else {
			a.symbolOffset += count
			if a.symbolOffset >= len(a.symbolNames) {
				a.symbolOffset = len(a.symbolNames) - count
			}
		}
		a.symbolStrip.Invalidate()
		return
	}
	startX := leftW
	if a.symbolOffset > 0 {
		startX += arrowW
	}
	if x < startX {
		return
	}
	idx := a.symbolOffset + (x-startX)/tileW
	if idx < 0 || idx >= len(a.symbolNames) {
		return
	}
	a.selectedSymbol = idx
	if a.symbolStrip != nil {
		a.symbolStrip.Invalidate()
	}
	a.addSymbolByName(a.symbolNames[idx])
}

func (a *App) addSymbolToCanvas() {
	if a.symbolList == nil || a.currentLayout == nil {
		return
	}
	idx := a.symbolList.CurrentIndex()
	if idx < 0 || idx >= len(a.symbolNames) {
		return
	}
	name := a.symbolNames[idx]
	a.pushUndo()
	el := a.newSymbolElement(name)
	nextElemID++
	a.elements = append(a.elements, el)
	a.selectedIdx = len(a.elements) - 1
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) addSymbolByName(name string) {
	if a.currentLayout == nil || name == "" {
		return
	}
	a.pushUndo()
	el := a.newSymbolElement(name)
	nextElemID++
	a.elements = append(a.elements, el)
	a.selectedIdx = len(a.elements) - 1
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) onNew() {
	if !a.confirmDiscardChanges() {
		return
	}
	a.showOpenDialog()
}

func (a *App) onOpen() {
	if !a.confirmDiscardChanges() {
		return
	}
	a.showOpenDocumentDialog()
}

func (a *App) onExit() {
	a.mainWindow.Close()
}

func (a *App) onPrint() {
	if a.currentLayout == nil {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Abra uma etiqueta antes de imprimir.", walk.MsgBoxIconExclamation)
		return
	}
	printers, err := print.EnumPrinters()
	if err != nil || len(printers) == 0 {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Nenhuma impressora encontrada.", walk.MsgBoxIconExclamation)
		return
	}
	defaultCopies := defaultPrintCopies()
	layout := *a.currentLayout
	var dlg *walk.Dialog
	var list *walk.ListBox
	var copiesEdit *walk.NumberEdit
	var factsLabel *walk.TextLabel
	refreshPrintDialogFacts := func() {
		if factsLabel == nil {
			return
		}
		copies := int(copiesEdit.Value())
		printerName := a.currentPrinter
		if idx := list.CurrentIndex(); idx >= 0 && idx < len(printers) {
			printerName = printers[idx]
		}
		facts := buildPrintDialogFacts(PrintDialogInput{
			Layout:       layout,
			TemplateName: a.currentTemplate,
			LayoutType:   a.currentLayoutType,
			PrinterName:  printerName,
			UnknownCount: len(a.unknownObjects),
			Copies:       copies,
		})
		factsLabel.SetText(strings.Join(facts.SummaryLines(), "\n"))
	}
	Dialog{
		AssignTo: &dlg, Title: "Imprimir", MinSize: Size{Width: 420, Height: 380}, Layout: VBox{},
		Children: []Widget{
			TextLabel{AssignTo: &factsLabel},
			Label{Text: "Selecione a impressora:"},
			ListBox{AssignTo: &list, OnCurrentIndexChanged: func() { refreshPrintDialogFacts() }},
			Composite{Layout: Grid{Columns: 2}, Children: []Widget{
				Label{Text: "Cópias:"},
				NumberEdit{AssignTo: &copiesEdit, Value: float64(defaultCopies), Decimals: 0, OnValueChanged: func() { refreshPrintDialogFacts() }},
			}},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "Imprimir", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	list.SetModel(printers)
	if a.currentPrinter != "" {
		for i, p := range printers {
			if strings.EqualFold(p, a.currentPrinter) {
				list.SetCurrentIndex(i)
				break
			}
		}
	}
	if list.CurrentIndex() < 0 && len(printers) > 0 {
		list.SetCurrentIndex(0)
	}
	refreshPrintDialogFacts()
	if dlg.Run() == walk.DlgCmdOK {
		if idx := list.CurrentIndex(); idx >= 0 && idx < len(printers) {
			copies := int(copiesEdit.Value())
			if copies < 1 {
				copies = 1
			}
			a.currentPrinter = printers[idx]
			a.doPrint(a.currentPrinter, copies)
		}
	}
	dlg.Dispose()
}

func (a *App) onPrintSetup() {
	printers, err := print.EnumPrinters()
	if err != nil || len(printers) == 0 {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Nenhuma impressora encontrada.", walk.MsgBoxIconExclamation)
		return
	}
	var dlg *walk.Dialog
	var list *walk.ListBox
	Dialog{
		AssignTo: &dlg, Title: "Impressora", MinSize: Size{Width: 380, Height: 260}, Layout: VBox{},
		Children: []Widget{
			Label{Text: "Selecione a impressora:"},
			ListBox{AssignTo: &list},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	list.SetModel(printers)
	if a.currentPrinter != "" {
		for i, p := range printers {
			if strings.EqualFold(p, a.currentPrinter) {
				list.SetCurrentIndex(i)
				break
			}
		}
	}
	if list.CurrentIndex() < 0 && len(printers) > 0 {
		list.SetCurrentIndex(0)
	}
	if dlg.Run() == walk.DlgCmdOK {
		if idx := list.CurrentIndex(); idx >= 0 && idx < len(printers) {
			a.currentPrinter = printers[idx]
			a.updateStatus()
		}
	}
	dlg.Dispose()
}

func (a *App) printDocument(printerName string, copies int) error {
	if a.currentLayout == nil {
		return fmt.Errorf("nenhum layout carregado")
	}
	layout := *a.currentLayout
	lp, err := print.NewLabelPrinter(printerName, layout.Landscape)
	if err != nil {
		return err
	}
	defer lp.Close()
	texts, wmfs, printObjects := printModelFromElements(a.elements)
	pageW, pageH := lp.PageSizeMM()
	page := a.resolvePrintPage()
	sheet := printlayout.SheetForPage(layout, pageW, pageH, page)
	label := model.Label{Name: filepath.Base(a.currentDocPath), PrinterName: a.currentPrinter, LayoutType: a.currentLayoutType, TemplateName: a.currentTemplate, Texts: texts, WMFSymbols: wmfs, Objects: printObjects}
	return lp.PrintLabel("MasterPrint", layout, sheet, totalPrintLabels(layout, copies), label)
}

func (a *App) doPrint(printerName string, copies int) {
	if err := a.printDocument(printerName, copies); err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
	} else {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Impress\u00e3o enviada!", walk.MsgBoxIconInformation)
	}
}

func defaultPrintCopies() int {
	return 1
}

func totalPrintLabels(layout model.LayoutDefinition, copies int) int {
	if copies < 1 {
		copies = 1
	}
	if layout.CopiesPerColumn > 0 {
		return copies * layout.CopiesPerColumn
	}
	return copies
}

func printModelFromElements(elements []LabelElement) ([]model.TextElement, []model.WMFSymbol, []model.PrintObject) {
	texts := make([]model.TextElement, 0)
	wmfs := make([]model.WMFSymbol, 0)
	objects := make([]model.PrintObject, 0)
	for _, el := range elements {
		switch el.Type {
		case "text":
			text := textElementFromLabel(el)
			texts = append(texts, text)
			objects = append(objects, model.PrintObject{Type: "text", Text: text})
		case "image":
			wmf := wmfSymbolFromElement(el)
			wmfs = append(wmfs, wmf)
			objects = append(objects, model.PrintObject{Type: "image", WMF: wmf})
		case "line", "rect", "ellipse":
			shape := model.ShapeElement{XMM: el.XMM, YMM: el.YMM, WidthMM: el.WidthMM, HeightMM: el.HeightMM}
			objects = append(objects, model.PrintObject{Type: el.Type, Shape: shape})
		}
	}
	return texts, wmfs, objects
}

func wmfBytesFromElement(el LabelElement) ([]byte, error) {
	return print.WMFBytes(wmfSymbolFromElement(el))
}

func (a *App) drawWMFElement(canvas *walk.Canvas, r walk.Rectangle, el LabelElement) error {
	data, err := wmfBytesFromElement(el)
	if err != nil {
		return err
	}
	return render.PlayWMFBytes(uintptr(canvas.HDC()), data, r.X, r.Y, r.X+r.Width-1, r.Y+r.Height-1)
}

func textElementFromLabel(el LabelElement) model.TextElement {
	var payloadRaw []byte
	if el.PayloadRaw != "" {
		if raw, err := base64.StdEncoding.DecodeString(el.PayloadRaw); err == nil {
			payloadRaw = raw
		}
	}
	var rtfRaw []byte
	if el.RTFRaw != "" {
		if raw, err := base64.StdEncoding.DecodeString(el.RTFRaw); err == nil {
			rtfRaw = raw
		}
	}
	return model.TextElement{FileOffset: el.FileOffset, FEFlags: el.FEFlags, FETag: el.FETag, PayloadRaw: payloadRaw, RTFRaw: rtfRaw, StyleByte: el.StyleByte, NextX: el.NextX, NextY: el.NextY, Text: el.Text, XMM: el.XMM, YMM: el.YMM, WidthMM: el.WidthMM, HeightMM: el.HeightMM, FontName: el.FontName, FontSize: el.FontSize, Bold: el.Bold, Italic: el.Italic, Underline: el.Underline, Align: el.Align}
}

func wmfSymbolFromElement(el LabelElement) model.WMFSymbol {
	sym := model.WMFSymbol{FileOffset: el.FileOffset, FilePath: el.ImagePath, StyleByte: el.StyleByte, NextX: el.NextX, NextY: el.NextY, XMM: el.XMM, YMM: el.YMM, WidthMM: el.WidthMM, HeightMM: el.HeightMM}
	if el.WMFRaw != "" {
		if raw, err := base64.StdEncoding.DecodeString(el.WMFRaw); err == nil && len(raw) > 0 {
			sym.Embedded = raw
		}
	}
	return sym
}

func (a *App) resolvePrintPage() model.PrintPage {
	if len(a.pageOverrides) == 0 || a.currentLayoutType == "" {
		return model.PrintPage{}
	}
	want := normalizePrintSection(a.currentLayoutType)
	for section, page := range a.pageOverrides {
		got := normalizePrintSection(section)
		if got == want || strings.HasPrefix(want, got) || strings.HasPrefix(got, want) {
			return page
		}
	}
	return model.PrintPage{}
}

func normalizePrintSection(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"Á", "a", "À", "a", "Ã", "a", "Â", "a", "Ä", "a",
		"é", "e", "ê", "e", "É", "e", "Ê", "e",
		"í", "i", "Í", "i",
		"ó", "o", "õ", "o", "ô", "o", "Ó", "o", "Õ", "o", "Ô", "o",
		"ú", "u", "Ú", "u",
		"ç", "c", "Ç", "c",
	)
	return strings.ToLower(strings.TrimSpace(replacer.Replace(s)))
}

func (a *App) onPrintPreview() {
	if a.currentLayout == nil {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Abra uma etiqueta primeiro.", walk.MsgBoxIconExclamation)
		return
	}
	a.setZoomToPage(false)
}

func (a *App) onInsertImage() {
	if a.currentLayout == nil {
		return
	}
	if a.symbolList != nil && a.selectedSymbol >= 0 && a.selectedSymbol < len(a.symbolNames) {
		name := a.symbolNames[a.selectedSymbol]
		a.pushUndo()
		a.elements = append(a.elements, a.newSymbolElement(strings.TrimSuffix(name, filepath.Ext(name))))
		nextElemID++
		a.selectedIdx = len(a.elements) - 1
		a.invalidateCanvas()
		a.updateStatus()
		return
	}
	walk.MsgBox(a.mainWindow, "MasterPrint", "Selecione um simbolo no painel de clips primeiro.", walk.MsgBoxIconInformation)
}

func (a *App) newSymbolElement(name string) LabelElement {
	img := filepath.Join(a.clipartDir, name+".wmf")
	w, h := wmfIntrinsicSizeMM(img)
	return LabelElement{ID: nextElemID, Type: "image", XMM: 0, YMM: 0, WidthMM: w, HeightMM: h, ImagePath: img, SymbolName: name}
}

func wmfIntrinsicSizeMM(path string) (float64, float64) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 22 || binary.LittleEndian.Uint32(data[:4]) != 0x9ac6cdd7 {
		return 0, 0
	}
	left := int(int16(binary.LittleEndian.Uint16(data[6:8])))
	top := int(int16(binary.LittleEndian.Uint16(data[8:10])))
	right := int(int16(binary.LittleEndian.Uint16(data[10:12])))
	bottom := int(int16(binary.LittleEndian.Uint16(data[12:14])))
	inch := int(binary.LittleEndian.Uint16(data[14:16]))
	if inch == 0 {
		inch = 96
	}
	w := render.MulDiv(right-left, 2540, inch)
	h := render.MulDiv(bottom-top, 2540, inch)
	return float64(w) / 100.0, float64(h) / 100.0
}

func (a *App) onDelete() {
	if a.selectedIdx >= 0 && a.selectedIdx < len(a.elements) {
		a.pushUndo()
		a.elements = append(a.elements[:a.selectedIdx], a.elements[a.selectedIdx+1:]...)
		a.selectedIdx = -1
		a.invalidateCanvas()
		a.updateStatus()
	}
}

func (a *App) onSave() {
	if a.currentLayout == nil {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Selecione um layout antes de salvar.", walk.MsgBoxIconInformation)
		return
	}
	if a.currentDocPath == "" {
		a.onSaveAs()
		return
	}
	path := a.currentDocPath
	nativeDoc := isMPNDocument(path)
	if nativeDoc {
		if err := a.saveMPNDocument(path); err != nil {
			walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
			return
		}
	} else if err := a.saveSidecar(path); err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
		return
	}
	etqApplied := false
	if !nativeDoc {
		var err error
		etqApplied, err = a.maybeSaveETQ(path)
		if err != nil {
			walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	a.markDocumentSaved()
	a.setPersistenceStatus(saveStatusText(nativeDoc, etqApplied))
}

func (a *App) onSaveAs() {
	if a.currentLayout == nil {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Selecione um layout antes de salvar.", walk.MsgBoxIconInformation)
		return
	}
	if !a.hasETQSource() {
		a.onSaveAsMPN()
		return
	}
	dlg := new(walk.FileDialog)
	dlg.Title = "Salvar Como"
	dlg.InitialDirPath = filepath.Join(a.dataDir, "ARQUIVOS")
	dlg.Filter = "Etiquetas Paulimaq (*.ETQ)|*.ETQ"
	if a.currentTemplate != "" {
		dlg.FilePath = a.currentTemplate + ".ETQ"
	} else if a.currentLayout.Name != "" {
		dlg.FilePath = a.currentLayout.Name + ".ETQ"
	}
	ok, err := dlg.ShowSave(a.mainWindow)
	if err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
		return
	}
	if !ok {
		return
	}
	path := ensureETQExtension(dlg.FilePath)
	etqApplied, err := a.saveAsETQToPath(path)
	if err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
		return
	}
	a.updateWindowTitle()
	a.setPersistenceStatus(saveStatusText(false, etqApplied))
}

func (a *App) saveAsETQToPath(path string) (bool, error) {
	if a.etqSourcePath == "" {
		return false, fmt.Errorf("salvar ETQ: arquivo de origem ausente")
	}
	path = ensureETQExtension(path)
	if !strings.EqualFold(path, a.etqSourcePath) {
		if err := etq.CopyETQ(a.etqSourcePath, path); err != nil {
			return false, err
		}
	}
	if err := a.saveSidecar(path); err != nil {
		return false, err
	}
	etqApplied, err := a.maybeSaveETQ(path)
	if err != nil {
		return false, err
	}
	a.currentDocPath = path
	a.etqSourcePath = path
	if a.currentTemplate == "" {
		a.currentTemplate = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	a.markDocumentSaved()
	return etqApplied, nil
}

func (a *App) onSaveAsMPN() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Salvar Documento MasterPrint"
	dlg.InitialDirPath = filepath.Join(a.dataDir, "ARQUIVOS")
	dlg.Filter = "Documento MasterPrint (*.mpn)|*.mpn"
	if a.currentTemplate != "" {
		dlg.FilePath = a.currentTemplate + mpnDocumentExt
	} else if a.currentLayout != nil && a.currentLayout.Name != "" {
		dlg.FilePath = a.currentLayout.Name + mpnDocumentExt
	}
	ok, err := dlg.ShowSave(a.mainWindow)
	if err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
		return
	}
	if !ok {
		return
	}
	path := ensureMPNExtension(dlg.FilePath)
	a.currentDocPath = path
	if a.currentTemplate == "" {
		a.currentTemplate = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if err := a.saveMPNDocument(path); err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
		return
	}
	a.markDocumentSaved()
	a.setPersistenceStatus(saveStatusText(true, false))
}

func saveStatusText(nativeDoc bool, etqApplied bool) string {
	if nativeDoc {
		return "Documento salvo (.mpn)"
	}
	if etqApplied {
		return "Alteracoes salvas no .ETQ (experimental)"
	}
	return "Alteracoes salvas no documento auxiliar; o .ETQ original nao foi alterado"
}

func (a *App) saveSidecar(path string) error {
	doc := a.buildSavedDocument(1, "")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("salvando alteracoes: %w", err)
	}
	var lastErr error
	wrote := false
	for _, candidate := range sidecarPaths(path) {
		if err := os.MkdirAll(filepath.Dir(candidate), 0755); err != nil {
			lastErr = err
			continue
		}
		if err := os.WriteFile(candidate, data, 0644); err != nil {
			lastErr = err
			continue
		}
		wrote = true
	}
	if wrote {
		return nil
	}
	return fmt.Errorf("salvando alteracoes: %w", lastErr)
}

func (a *App) loadSidecar(path, expectedLayout string) bool {
	return a.loadSidecarWithBaseline(path, expectedLayout, nil).Applied
}

func (a *App) loadSidecarWithBaseline(path, expectedLayout string, baseline map[int]LabelElement) sidecarLoadOutcome {
	candidates := sidecarCandidates(path)
	evalCandidates := make([]sidecarCandidateData, len(candidates))
	for i, candidate := range candidates {
		evalCandidates[i] = sidecarCandidateData{Path: candidate.path, JSON: candidate.data}
	}
	outcome := evaluateSidecarLoad(evalCandidates, expectedLayout, envFlag("MASTERPRINT_ALLOW_LEGACY_SIDECAR"), envFlag("MASTERPRINT_ALLOW_SIDECAR_LAYOUT_MISMATCH"))
	if !outcome.Applied {
		if outcome.IgnoredReason != "" {
			log.Printf("sidecar ignored: %s", outcome.IgnoredReason)
		}
		return outcome
	}
	var doc savedDocument
	for _, candidate := range candidates {
		if candidate.path != outcome.AppliedPath {
			continue
		}
		if err := json.Unmarshal(candidate.data, &doc); err != nil {
			outcome.Applied = false
			outcome.AppliedPath = ""
			outcome.IgnoredReason = fmt.Sprintf("JSON invalido (%s)", filepath.Base(candidate.path))
			log.Printf("sidecar ignored: invalid JSON in %s: %v", candidate.path, err)
			return outcome
		}
		break
	}
	if len(baseline) > 0 {
		fillMissingRawPayloads(&doc, baseline)
	}
	a.applySavedDocument(doc)
	log.Printf("sidecar applied: %s", outcome.AppliedPath)
	return outcome
}

func elementsByFileOffset(elements []LabelElement) map[int]LabelElement {
	out := make(map[int]LabelElement)
	for _, el := range elements {
		if el.FileOffset != 0 {
			out[el.FileOffset] = el
		}
	}
	return out
}

func fillMissingRawPayloads(doc *savedDocument, baseline map[int]LabelElement) {
	if doc == nil {
		return
	}
	for i := range doc.Elements {
		se := &doc.Elements[i]
		base, ok := baseline[se.FileOffset]
		if !ok {
			continue
		}
		textMatchesBaseline := savedTextCompatibleWithRaw(*se, base)
		imageMatchesBaseline := se.Type == "image" && (se.ImagePath == "" || base.ImagePath == "" || strings.EqualFold(filepath.Clean(se.ImagePath), filepath.Clean(base.ImagePath)))
		if textMatchesBaseline && se.PayloadRaw == "" {
			se.PayloadRaw = base.PayloadRaw
		}
		if textMatchesBaseline && se.RTFRaw == "" {
			se.RTFRaw = base.RTFRaw
		}
		if imageMatchesBaseline && se.WMFRaw == "" {
			se.WMFRaw = base.WMFRaw
		}
		if imageMatchesBaseline && se.WMFPreRaw == "" {
			se.WMFPreRaw = base.WMFPreRaw
		}
	}
}

func savedTextCompatibleWithRaw(se savedElement, base LabelElement) bool {
	if se.Type != "text" || se.Text != base.Text {
		return false
	}
	return se.FontName == base.FontName && nearFloat64(se.FontSize, base.FontSize) && se.Bold == base.Bold && se.Italic == base.Italic && se.Underline == base.Underline && se.Align == base.Align && se.StyleByte == base.StyleByte
}

func nearFloat64(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

type sidecarCandidate struct {
	path    string
	data    []byte
	modTime time.Time
}

func sidecarCandidates(path string) []sidecarCandidate {
	var out []sidecarCandidate
	for _, candidate := range sidecarPaths(path) {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		out = append(out, sidecarCandidate{path: candidate, data: data, modTime: info.ModTime()})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].modTime.After(out[j].modTime) })
	return out
}

func sidecarExists(path string) (string, bool) {
	for _, candidate := range sidecarPaths(path) {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func envFlag(name string) bool {
	v := strings.TrimSpace(os.Getenv(name))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func sidecarPaths(path string) []string {
	h := sha1.Sum([]byte(strings.ToLower(path)))
	userDir := filepath.Join(os.Getenv("APPDATA"), "MasterPrintNative", "overlays")
	if userDir == filepath.Join("", "MasterPrintNative", "overlays") {
		if cfg, err := os.UserConfigDir(); err == nil {
			userDir = filepath.Join(cfg, "MasterPrintNative", "overlays")
		}
	}
	return []string{path + ".masterprint-native.json", filepath.Join(userDir, hex.EncodeToString(h[:])+".json")}
}

func (a *App) onCopy() {
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.elements) {
		return
	}
	el := a.elements[a.selectedIdx]
	if el.Type == "preview" {
		return
	}
	a.clipboardElem = &el
}

func (a *App) onPaste() {
	if a.clipboardElem == nil || a.currentLayout == nil {
		return
	}
	a.pushUndo()
	el := *a.clipboardElem
	el.ID = nextElemID
	nextElemID++
	el.XMM += 1
	el.YMM += 1
	a.clampElementToLabel(&el)
	a.elements = append(a.elements, el)
	a.selectedIdx = len(a.elements) - 1
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) onBringToFront() {
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.elements)-1 {
		return
	}
	a.pushUndo()
	el := a.elements[a.selectedIdx]
	a.elements = append(a.elements[:a.selectedIdx], a.elements[a.selectedIdx+1:]...)
	a.elements = append(a.elements, el)
	a.selectedIdx = len(a.elements) - 1
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) onSendToBack() {
	if a.selectedIdx <= 0 || a.selectedIdx >= len(a.elements) {
		return
	}
	a.pushUndo()
	el := a.elements[a.selectedIdx]
	a.elements = append(a.elements[:a.selectedIdx], a.elements[a.selectedIdx+1:]...)
	a.elements = append([]LabelElement{el}, a.elements...)
	a.selectedIdx = 0
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) onProperties() {
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.elements) {
		return
	}
	a.showPropsDialog()
}

func (a *App) showOpenDocumentDialog() {
	dlg := new(walk.FileDialog)
	dlg.Title = "Abrir Documento"
	dlg.InitialDirPath = filepath.Join(a.dataDir, "ARQUIVOS")
	dlg.Filter = "Documentos MasterPrint (*.ETQ;*.mpn)|*.ETQ;*.mpn|Etiquetas Paulimaq (*.ETQ)|*.ETQ|Documento MasterPrint (*.mpn)|*.mpn"
	ok, err := dlg.ShowOpen(a.mainWindow)
	if err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
		return
	}
	if !ok {
		return
	}
	if err := a.loadDocumentPath(dlg.FilePath); err != nil {
		walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
	}
}

func (a *App) loadDocumentPath(path string) error {
	if isMPNDocument(path) {
		return a.loadMPNDocument(path)
	}
	return a.loadETQDocument(path)
}

func (a *App) loadETQDocument(path string) error {
	doc, err := etq.ParseETQ(path)
	if err != nil {
		return err
	}
	templateName := doc.TemplateName
	layoutCategory := layoutCategoryFromHeader(doc.LayoutType)
	if layoutCategory == "" {
		return fmt.Errorf("categoria de layout nao reconhecida: %q", doc.LayoutType)
	}
	var layout *model.LayoutDefinition
	if templateName != "" {
		layout = a.findLayout(layoutCategory, templateName)
	}
	if layout == nil {
		if a.mainWindow == nil {
			if templateName == "" {
				return fmt.Errorf("modelo da etiqueta nao encontrado no arquivo")
			}
			return fmt.Errorf("layout %s/%s nao encontrado", layoutCategory, templateName)
		}
		selectedLayout, ok := a.chooseETQLayout(layoutCategory, templateName, filepath.Base(path))
		if !ok {
			return nil
		}
		layout = selectedLayout
		templateName = selectedLayout.Name
	}
	a.currentLayout = layout
	a.currentDocPath = path
	a.currentPrinter = doc.PrinterName
	a.currentLayoutType = doc.LayoutType
	a.currentTemplate = templateName
	a.unknownObjects = make([]etq.ETQUnknownObject, 0, len(doc.UnknownObjects))
	for _, obj := range doc.UnknownObjects {
		a.unknownObjects = append(a.unknownObjects, etq.ETQUnknownObject{Offset: obj.Offset, Flags: obj.Flags, Tag: obj.Tag, Kind: obj.Kind})
	}
	a.elements = nil
	a.selectedIdx = -1
	a.resetHistory()
	type loadedETQElement struct {
		order int
		el    LabelElement
	}
	loadedElements := make([]loadedETQElement, 0, len(doc.TextElements)+len(doc.WMFElements))
	for _, t := range doc.TextElements {
		// CadMapa's text renderer builds the font from RECT height
		// (FUN_004c036c). Keep ETQ RECT-like values instead of inventing a box.
		h := t.HeightMM
		if h <= 0 {
			h = t.FontSize * 25.4 / 72.0
		}
		w := t.WidthMM
		align := t.Align
		if align == "" {
			align = "left"
		}
		payloadRaw := ""
		if len(t.PayloadRaw) > 0 {
			payloadRaw = base64.StdEncoding.EncodeToString(t.PayloadRaw)
		}
		rtfRaw := ""
		if len(t.RTFRaw) > 0 {
			rtfRaw = base64.StdEncoding.EncodeToString(t.RTFRaw)
		}
		loadedElements = append(loadedElements, loadedETQElement{order: t.FileOffset, el: LabelElement{ID: nextElemID, Type: "text", FileOffset: t.FileOffset, FEFlags: t.FEFlags, FETag: t.FETag, PayloadRaw: payloadRaw, RTFRaw: rtfRaw, StyleByte: t.StyleByte, NextX: t.NextX, NextY: t.NextY, XMM: t.XMM, YMM: t.YMM, WidthMM: w, HeightMM: h, Text: t.Text, FontName: t.FontName, FontSize: t.FontSize, Bold: t.Bold, Italic: t.Italic, Underline: t.Underline, Color: color.Black, Align: align}})
		nextElemID++
	}
	for _, s := range doc.WMFElements {
		img := s.FilePath
		if !filepath.IsAbs(img) {
			img = filepath.Join(a.clipartDir, img)
		}
		wmfRaw := ""
		if len(s.Embedded) > 0 {
			wmfRaw = base64.StdEncoding.EncodeToString(s.Embedded)
		}
		wmfPreRaw := ""
		if len(s.PreBlock) > 0 {
			wmfPreRaw = base64.StdEncoding.EncodeToString(s.PreBlock)
		}
		loadedElements = append(loadedElements, loadedETQElement{order: s.FileOffset, el: LabelElement{ID: nextElemID, Type: "image", FileOffset: s.FileOffset, WMFRaw: wmfRaw, WMFPreRaw: wmfPreRaw, StyleByte: s.StyleByte, NextX: s.NextX, NextY: s.NextY, XMM: s.XMM, YMM: s.YMM, WidthMM: s.WidthMM, HeightMM: s.HeightMM, ImagePath: img, SymbolName: strings.TrimSuffix(filepath.Base(img), filepath.Ext(img))}})
		nextElemID++
	}
	sort.SliceStable(loadedElements, func(i, j int) bool { return loadedElements[i].order < loadedElements[j].order })
	for _, item := range loadedElements {
		a.elements = append(a.elements, item.el)
	}
	baseline := elementsByFileOffset(a.elements)
	a.etqSourcePath = path
	sidecarOutcome := a.loadSidecarWithBaseline(path, templateName, baseline)
	a.markDocumentSaved()
	a.persistenceStatus = openStatusText(path, templateName, sidecarOutcome, len(a.unknownObjects))
	a.updateWindowTitle()
	a.restoreDocumentView()
	a.invalidateCanvas()
	a.updateStatus()
	a.notifyUnknownObjectsIfAny()
	return nil
}

func (a *App) chooseETQLayout(category, requestedTemplate, fileName string) (*model.LayoutDefinition, bool) {
	layouts := a.layouts[category]
	if len(layouts) == 0 {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Nenhum layout encontrado para este tipo de etiqueta.", walk.MsgBoxIconExclamation)
		return nil, false
	}
	names := make([]string, len(layouts))
	selected := 0
	for i, l := range layouts {
		names[i] = fmt.Sprintf("%s (%.0fx%.0fmm)", l.Name, l.WidthMM, l.HeightMM)
		if requestedTemplate != "" && strings.EqualFold(l.Name, requestedTemplate) {
			selected = i
		}
	}
	message := fmt.Sprintf("Selecione o layout para %s.", fileName)
	if requestedTemplate != "" {
		message = fmt.Sprintf("Layout %q nao encontrado. Selecione o layout correto para %s.", requestedTemplate, fileName)
	}
	var dlg *walk.Dialog
	var list *walk.ListBox
	Dialog{
		AssignTo: &dlg, Title: "Selecionar Layout", MinSize: Size{Width: 520, Height: 360}, Layout: VBox{},
		Children: []Widget{
			Label{Text: message},
			ListBox{AssignTo: &list, MinSize: Size{Width: 460, Height: 220}, OnItemActivated: func() { dlg.Accept() }},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() { dlg.Accept() }},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	list.SetModel(names)
	list.SetCurrentIndex(selected)
	defer dlg.Dispose()
	if dlg.Run() != walk.DlgCmdOK {
		return nil, false
	}
	idx := list.CurrentIndex()
	if idx < 0 || idx >= len(layouts) {
		return nil, false
	}
	return &layouts[idx], true
}

func layoutCategoryPairs() []struct{ header, category string } {
	return []struct{ header, category string }{
		{"Etiq. para Composições em Folhas", "etiqueta"},
		{"Etiq. para Composições em Formulários", "etiqueta_m"},
		{"Etiq. para Composições em Rolo", "etiqueta_r"},
		{"TAG'S em Folhas e Formulários", "tag"},
		{"Etiq. Ades. Fast Label (Padrão)", "tag2"},
		{"Pauli - Tab", "tag3"},
		{"Etiquetas para Jóias", "joia"},
		{"Etiq. para Caixas de Calçados", "sapato"},
		{"Cartões de Visita - PRINT CARD", "cartao"},
		{"Convites - PRINT INVITE", "invite"},
		{"Etiq. Box para CD - PRINT CD FACE", "box"},
		{"Caixa de Cartões - PRINT BOX", "caixa"},
		{"Etiq. para Plantas", "plantas"},
		{"Pulseiras Bands", "fixbands"},
		{"Etiquetas para CD - CD Center", "cdcenter"},
		{"Etiquetas para CD - CD FAST LABEL 2", "cdfastlab"},
		{"Etiquetas para CD - PRINT CD LABEL 2", "cdlab"},
		{"Etiquetas para CD - Mini CD", "minicd"},
		{"Etiquetas para CD - PRINT CD LABEL 3", "ncd"},
		{"Print CD Cards", "pcd"},
		{"Photo Quality Álbum", "photoa4"},
	}
}

func layoutCategoryFromHeader(header string) string {
	normalized := normalizePrintSection(header)
	for _, pair := range layoutCategoryPairs() {
		if normalizePrintSection(pair.header) == normalized {
			return pair.category
		}
	}
	switch {
	case strings.Contains(normalized, "composi") || strings.Contains(normalized, "etiq"):
		return "etiqueta"
	case strings.Contains(normalized, "tag"):
		return "tag"
	case strings.Contains(normalized, "cart"):
		return "cartao"
	case strings.Contains(normalized, "joia"):
		return "joia"
	case strings.Contains(normalized, "sapato"):
		return "sapato"
	}
	return ""
}

func layoutHeaderFromCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	for _, pair := range layoutCategoryPairs() {
		if pair.category == category {
			return pair.header
		}
	}
	return ""
}

func statusTipoModelo(layoutType, template, layoutName string) string {
	tipo := layoutType
	if header := layoutHeaderFromCategory(layoutType); header != "" {
		tipo = header
	}
	modelo := template
	if modelo == "" {
		modelo = layoutName
	}
	return fmt.Sprintf("Tipo: %s    Modelo: %s", tipo, modelo)
}

func ensureETQExtension(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".etq") {
		return path
	}
	return path + ".ETQ"
}

func (a *App) findLayout(category, name string) *model.LayoutDefinition {
	layouts := a.layouts[category]
	for i := range layouts {
		if strings.EqualFold(layouts[i].Name, name) {
			return &layouts[i]
		}
	}
	if len(layouts) > 0 && envFlag("MASTERPRINT_ALLOW_LAYOUT_FALLBACK") {
		log.Printf("layout fallback opt-in: category=%q requested=%q using=%q", category, name, layouts[0].Name)
		return &layouts[0]
	}
	return nil
}

func (a *App) paintDocumentPreview(canvas *walk.Canvas, bounds walk.Rectangle, doc *etq.ETQFile) error {
	canvas.FillRectanglePixels(whiteBrush, bounds)
	pen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(160, 160, 160))
	canvas.DrawRectanglePixels(pen, bounds)
	pen.Dispose()
	if doc == nil {
		return nil
	}
	if doc.PreviewPath != "" {
		img, err := walk.NewImageFromFile(doc.PreviewPath)
		if err == nil {
			canvas.DrawImageStretchedPixels(img, walk.Rectangle{X: bounds.X + 8, Y: bounds.Y + 6, Width: bounds.Width - 16, Height: bounds.Height - 12})
			img.Dispose()
			return nil
		}
	}
	rx, ry, rw, rh := bounds.X+18, bounds.Y+12, 76, 180
	canvas.FillRectanglePixels(whiteBrush, walk.Rectangle{X: rx, Y: ry, Width: rw, Height: rh})
	bp, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
	canvas.DrawRectanglePixels(bp, walk.Rectangle{X: rx, Y: ry, Width: rw, Height: rh})
	bp.Dispose()
	for _, t := range doc.TextElements {
		fontSize := int(t.FontSize)
		if fontSize < 5 {
			fontSize = 5
		}
		style := walk.FontStyle(0)
		if t.Bold {
			style |= walk.FontBold
		}
		if t.Italic {
			style |= walk.FontItalic
		}
		if t.Underline {
			style |= walk.FontUnderline
		}
		font, _ := walk.NewFont(t.FontName, fontSize, style)
		x := rx + int(t.XMM/25.0*float64(rw))
		y := ry + int(t.YMM/55.5*float64(rh))
		flags := drawTextFlags(t.Align)
		canvas.DrawTextPixels(t.Text, font, walk.RGB(0, 0, 0), walk.Rectangle{X: x, Y: y, Width: rw - 6, Height: 18}, flags)
		font.Dispose()
	}
	return nil
}

func drawTextFlags(align string) walk.DrawTextFormat {
	flags := walk.DrawTextFormat(walk.TextNoPrefix | walk.TextWordbreak)
	switch align {
	case "center":
		flags |= walk.TextCenter
	case "right":
		flags |= walk.TextRight
	default:
		flags |= walk.TextLeft
	}
	return flags
}

func (a *App) showOpenDialog() {
	cats := make([]string, 0, len(a.layouts))
	for k := range a.layouts {
		cats = append(cats, k)
	}
	sort.Strings(cats)
	if len(cats) == 0 {
		walk.MsgBox(a.mainWindow, "MasterPrint", "Nenhum layout encontrado. Instale o Paulimaq ou defina MASTERPRINT_DATA para a pasta correta.", walk.MsgBoxIconExclamation)
		return
	}

	var dlg *walk.Dialog
	var catList, layoutList *walk.ListBox
	selectedCat := cats[0]
	var confirmedCat string
	var confirmedLayoutIdx int = -1
	acceptSelection := func() {
		idx := layoutList.CurrentIndex()
		if idx < 0 {
			walk.MsgBox(dlg, "Erro", "Selecione um dos items!", walk.MsgBoxIconExclamation)
			return
		}
		confirmedCat = selectedCat
		confirmedLayoutIdx = idx
		dlg.Accept()
	}
	Dialog{
		AssignTo: &dlg, Title: "Novo Documento", MinSize: Size{Width: 550, Height: 380}, Layout: VBox{},
		Children: []Widget{
			Label{Text: "Selecione categoria e layout:"},
			HSplitter{Children: []Widget{
				Composite{Layout: VBox{}, Children: []Widget{
					Label{Text: "Categoria:"},
					ListBox{AssignTo: &catList, MinSize: Size{Width: 140, Height: 200}, OnCurrentIndexChanged: func() {
						if idx := catList.CurrentIndex(); idx >= 0 && idx < len(cats) {
							selectedCat = cats[idx]
							layouts := a.layouts[selectedCat]
							names := make([]string, len(layouts))
							for i, l := range layouts {
								names[i] = fmt.Sprintf("%s (%.0fx%.0fmm)", l.Name, l.WidthMM, l.HeightMM)
							}
							layoutList.SetModel(names)
							if len(names) > 0 {
								layoutList.SetCurrentIndex(0)
							}
						}
					}},
				}},
				Composite{Layout: VBox{}, Children: []Widget{
					Label{Text: "Layout:"},
					ListBox{AssignTo: &layoutList, MinSize: Size{Width: 200, Height: 200}, OnItemActivated: acceptSelection},
				}},
			}},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: acceptSelection},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	catList.SetModel(cats)
	if len(cats) > 0 {
		catList.SetCurrentIndex(0)
	}

	result := dlg.Run()

	if result == walk.DlgCmdOK && confirmedLayoutIdx >= 0 {
		log.Printf("loading: cat=%s layoutIdx=%d", confirmedCat, confirmedLayoutIdx)
		if err := a.applySelectedLayout(confirmedCat, confirmedLayoutIdx); err != nil {
			walk.MsgBox(a.mainWindow, "Erro", err.Error(), walk.MsgBoxIconError)
		}
	}
	dlg.Dispose()
}

func (a *App) applySelectedLayout(category string, idx int) error {
	layouts := a.layouts[category]
	if idx < 0 || idx >= len(layouts) {
		return fmt.Errorf("layout invalido")
	}
	a.currentLayout = &layouts[idx]
	a.currentDocPath = ""
	a.currentPrinter = ""
	a.currentTemplate = a.currentLayout.Name
	if header := layoutHeaderFromCategory(category); header != "" {
		a.currentLayoutType = header
	} else {
		a.currentLayoutType = category
	}
	a.unknownObjects = nil
	a.etqSourcePath = ""
	a.etqBaseline = nil
	a.elements = nil
	a.selectedIdx = -1
	a.persistenceStatus = ""
	a.markDocumentSaved()
	a.restoreDocumentView()
	log.Printf("layout loaded: %s w=%.1f h=%.1f zoom=%.1f", a.currentLayout.Name, a.currentLayout.WidthMM, a.currentLayout.HeightMM, a.zoom)
	a.invalidateCanvas()
	a.updateStatus()
	return nil
}

func (a *App) restoreDocumentView() {
	a.scrollX = 0
	a.scrollY = 0
	if a.currentLayout != nil && a.canvas != nil {
		a.setZoomToPage(false)
	}
}

func (a *App) showPropsDialog() {
	el := &a.elements[a.selectedIdx]
	var dlg *walk.Dialog
	var fontCombo *walk.ComboBox
	var sizeEdit *walk.NumberEdit
	var xEdit, yEdit, wEdit, hEdit *walk.NumberEdit
	var boldCB, italicCB, underlineCB *walk.CheckBox
	var textEdit *walk.TextEdit
	fonts := []string{"Arial", "Consolas", "Courier New", "MS Sans Serif", "Tahoma", "Times New Roman", "Verdana"}
	Dialog{
		AssignTo: &dlg, Title: "Propriedades", MinSize: Size{Width: 340, Height: 350}, Layout: VBox{},
		Children: []Widget{
			TabWidget{Pages: []TabPage{
				{Title: "Fonte", Layout: VBox{}, Children: []Widget{
					Composite{Layout: Grid{Columns: 2}, Children: []Widget{
						Label{Text: "Fonte:"}, ComboBox{AssignTo: &fontCombo, Model: fonts},
						Label{Text: "Tamanho:"}, NumberEdit{AssignTo: &sizeEdit, MinValue: 4, MaxValue: 72, Decimals: 1},
					}},
					Composite{Layout: HBox{}, Children: []Widget{
						CheckBox{AssignTo: &boldCB, Text: "Negrito"},
						CheckBox{AssignTo: &italicCB, Text: "It\u00e1lico"},
						CheckBox{AssignTo: &underlineCB, Text: "Sublinhado"},
					}},
					Label{Text: "Texto:"}, TextEdit{AssignTo: &textEdit},
				}},
				{Title: "Objeto", Layout: Grid{Columns: 2}, Children: []Widget{
					Label{Text: "E (mm):"}, NumberEdit{AssignTo: &xEdit, MinValue: 0, MaxValue: 500, Decimals: 2},
					Label{Text: "T (mm):"}, NumberEdit{AssignTo: &yEdit, MinValue: 0, MaxValue: 500, Decimals: 2},
					Label{Text: "L (mm):"}, NumberEdit{AssignTo: &wEdit, MinValue: 0.1, MaxValue: 500, Decimals: 2},
					Label{Text: "A (mm):"}, NumberEdit{AssignTo: &hEdit, MinValue: 0.1, MaxValue: 500, Decimals: 2},
				}},
			}},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "OK", OnClicked: func() {
					a.applyProps(el, fontCombo, sizeEdit, xEdit, yEdit, wEdit, hEdit, boldCB, italicCB, underlineCB, textEdit)
					dlg.Accept()
				}},
				PushButton{Text: "Cancelar", OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}.Create(a.mainWindow)
	for i, f := range fonts {
		if f == el.FontName {
			fontCombo.SetCurrentIndex(i)
			break
		}
	}
	if fontCombo.CurrentIndex() < 0 {
		fontCombo.SetCurrentIndex(0)
	}
	fontSize := el.FontSize
	if fontSize < 4 {
		fontSize = 8
	}
	sizeEdit.SetValue(fontSize)
	xEdit.SetValue(el.XMM)
	yEdit.SetValue(el.YMM)
	wEdit.SetValue(el.WidthMM)
	hEdit.SetValue(el.HeightMM)
	boldCB.SetChecked(el.Bold)
	italicCB.SetChecked(el.Italic)
	underlineCB.SetChecked(el.Underline)
	if el.Text != "" {
		textEdit.SetText(el.Text)
	}
	dlg.Run()
	dlg.Dispose()
}

func (a *App) applyProps(el *LabelElement, fontCombo *walk.ComboBox, sizeEdit, xEdit, yEdit, wEdit, hEdit *walk.NumberEdit, boldCB, italicCB, underlineCB *walk.CheckBox, textEdit *walk.TextEdit) {
	a.pushUndo()
	oldText, oldFont, oldSize := el.Text, el.FontName, el.FontSize
	oldBold, oldItalic, oldUnderline := el.Bold, el.Italic, el.Underline
	if fontCombo.CurrentIndex() >= 0 {
		el.FontName = fontCombo.Text()
	}
	if el.Type == "text" {
		el.FontSize = sizeEdit.Value()
	}
	el.XMM = xEdit.Value()
	el.YMM = yEdit.Value()
	el.WidthMM = wEdit.Value()
	el.HeightMM = hEdit.Value()
	a.clampElementToLabel(el)
	el.Bold = boldCB.Checked()
	el.Italic = italicCB.Checked()
	el.Underline = underlineCB.Checked()
	if el.Type == "text" {
		el.Text = textEdit.Text()
		if oldText != el.Text || oldFont != el.FontName || oldSize != el.FontSize || oldBold != el.Bold || oldItalic != el.Italic || oldUnderline != el.Underline {
			syncEditableTextPayload(el)
		}
	}
	a.invalidateCanvas()
	a.updateStatus()
}

func syncEditableTextPayload(el *LabelElement) {
	if el == nil || el.Type != "text" {
		return
	}
	el.RTFRaw = ""
	el.StyleByte = styleByteFromBools(el.Bold, el.Italic, el.Underline)
	el.PayloadRaw = base64.StdEncoding.EncodeToString(render.CadMapaANSIBytes(el.Text))
}

func styleByteFromBools(bold, italic, underline bool) byte {
	var style byte
	if bold {
		style |= 0x01
	}
	if italic {
		style |= 0x02
	}
	if underline {
		style |= 0x04
	}
	return style
}

func (a *App) invalidateCanvas() {
	if a.canvas != nil {
		a.canvas.Invalidate()
	}
}

func (a *App) updateStatus() {
	if a.mainWindow == nil {
		return
	}
	sb := a.mainWindow.StatusBar()
	if sb == nil {
		return
	}
	items := sb.Items()
	if items == nil || items.Len() == 0 {
		return
	}
	item := items.At(0)
	if a.selectedIdx >= 0 && a.selectedIdx < len(a.elements) {
		el := a.elements[a.selectedIdx]
		item.SetText(fmt.Sprintf("E: %.2f  T: %.2f  L: %.2f  A: %.2f cm", el.XMM/10, el.YMM/10, el.WidthMM/10, el.HeightMM/10))
	} else if a.persistenceStatus != "" {
		item.SetText(a.persistenceStatus)
	} else if a.currentLayout != nil {
		item.SetText(statusTipoModelo(a.currentLayoutType, a.currentTemplate, a.currentLayout.Name))
	} else {
		item.SetText("MasterPrint 3.0 - Arquivo \u2192 Abrir")
	}
}

func (a *App) paintCanvas(canvas *walk.Canvas, bounds walk.Rectangle) error {
	if a.canvas != nil {
		bounds = a.canvas.ClientBoundsPixels()
	}
	canvas.FillRectanglePixels(grayBrush, bounds)
	if a.currentLayout == nil {
		return nil
	}

	canvas.FillRectanglePixels(whiteBrush, walk.Rectangle{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: rulerSize})
	canvas.FillRectanglePixels(whiteBrush, walk.Rectangle{X: bounds.X, Y: bounds.Y, Width: rulerSize, Height: bounds.Height})

	blackPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
	canvas.DrawLinePixels(blackPen, walk.Point{X: bounds.X, Y: bounds.Y + rulerSize}, walk.Point{X: bounds.X + bounds.Width, Y: bounds.Y + rulerSize})
	canvas.DrawLinePixels(blackPen, walk.Point{X: bounds.X + rulerSize, Y: bounds.Y}, walk.Point{X: bounds.X + rulerSize, Y: bounds.Y + bounds.Height})
	blackPen.Dispose()

	tickPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
	rulerFont, _ := walk.NewFont("Arial", 7, 0)
	dpi := float64(a.dpi)
	if dpi <= 0 {
		dpi = 96
	}
	pxPerMm := dpi / 25.4 * a.zoom

	for mm := 0; ; mm++ {
		px := int(float64(mm)*pxPerMm) + rulerSize + bounds.X - a.scrollX
		if px > bounds.X+bounds.Width {
			break
		}
		if px < bounds.X+rulerSize {
			continue
		}
		tl := 3
		if mm%5 == 0 {
			tl = 5
		}
		if mm%10 == 0 {
			tl = 8
			cm := mm / 10
			canvas.DrawTextPixels(fmt.Sprintf("%d", cm), rulerFont, walk.RGB(0, 0, 0),
				walk.Rectangle{X: px - 8, Y: bounds.Y + rulerSize - tl - 12, Width: 16, Height: 12}, walk.TextCenter)
		}
		canvas.DrawLinePixels(tickPen, walk.Point{X: px, Y: bounds.Y + rulerSize}, walk.Point{X: px, Y: bounds.Y + rulerSize - tl})
	}
	for mm := 0; ; mm++ {
		py := int(float64(mm)*pxPerMm) + rulerSize + bounds.Y - a.scrollY
		if py > bounds.Y+bounds.Height {
			break
		}
		if py < bounds.Y+rulerSize {
			continue
		}
		tl := 3
		if mm%5 == 0 {
			tl = 5
		}
		if mm%10 == 0 {
			tl = 8
			cm := mm / 10
			canvas.DrawTextPixels(fmt.Sprintf("%d", cm), rulerFont, walk.RGB(0, 0, 0),
				walk.Rectangle{X: bounds.X + rulerSize - tl - 16, Y: py - 5, Width: 14, Height: 12}, walk.TextCenter)
		}
		canvas.DrawLinePixels(tickPen, walk.Point{X: bounds.X + rulerSize, Y: py}, walk.Point{X: bounds.X + rulerSize - tl, Y: py})
	}
	tickPen.Dispose()
	rulerFont.Dispose()

	lw, lh := a.mmToPxLenX(a.renderW()), a.mmToPxLenY(a.renderH())
	ox, oy := bounds.X+rulerSize+10, bounds.Y+rulerSize+10

	cr := 6
	canvas.FillRectanglePixels(shadowBrush, walk.Rectangle{X: ox + cr + 4, Y: oy + cr + 4, Width: lw - 2*cr, Height: lh - 2*cr})
	canvas.FillRectanglePixels(shadowBrush, walk.Rectangle{X: ox + cr + 4, Y: oy + 4, Width: lw - 2*cr, Height: lh - 2*cr + 2*(cr+4)})
	canvas.FillRectanglePixels(shadowBrush, walk.Rectangle{X: ox + 4, Y: oy + cr + 4, Width: lw - 2*cr + 2*(cr+4), Height: lh - 2*cr})
	shadowCornerBrush, _ := walk.NewSolidColorBrush(walk.RGB(60, 60, 60))
	for _, p := range []walk.Point{
		{X: ox + lw - cr + 4, Y: oy + cr + 4},
		{X: ox + lw - cr + 4, Y: oy + lh - cr + 4},
		{X: ox + cr + 4, Y: oy + lh - cr + 4},
	} {
		canvas.FillEllipsePixels(shadowCornerBrush, walk.Rectangle{X: p.X - cr, Y: p.Y - cr, Width: 2 * cr, Height: 2 * cr})
	}
	shadowCornerBrush.Dispose()

	canvas.FillRectanglePixels(whiteBrush, walk.Rectangle{X: ox + cr, Y: oy, Width: lw - 2*cr, Height: lh})
	canvas.FillRectanglePixels(whiteBrush, walk.Rectangle{X: ox, Y: oy + cr, Width: lw, Height: lh - 2*cr})
	cornerBrush := whiteBrush
	for _, p := range []walk.Point{
		{X: ox + cr, Y: oy + cr},
		{X: ox + lw - cr, Y: oy + cr},
		{X: ox + cr, Y: oy + lh - cr},
		{X: ox + lw - cr, Y: oy + lh - cr},
	} {
		canvas.FillEllipsePixels(cornerBrush, walk.Rectangle{X: p.X - cr, Y: p.Y - cr, Width: 2 * cr, Height: 2 * cr})
	}

	bp, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
	canvas.DrawRectanglePixels(bp, walk.Rectangle{X: ox, Y: oy, Width: lw, Height: lh})
	bp.Dispose()

	if a.showGrid {
		gridPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(200, 200, 200))
		defer gridPen.Dispose()
		gridMM := 5.0
		for mm := gridMM; mm < a.renderW(); mm += gridMM {
			px := ox + a.mmToPxX(mm)
			if px > ox+lw {
				break
			}
			canvas.DrawLinePixels(gridPen, walk.Point{X: px, Y: oy}, walk.Point{X: px, Y: oy + lh})
		}
		for mm := gridMM; mm < a.renderH(); mm += gridMM {
			py := oy + a.mmToPxY(mm)
			if py > oy+lh {
				break
			}
			canvas.DrawLinePixels(gridPen, walk.Point{X: ox, Y: py}, walk.Point{X: ox + lw, Y: py})
		}
	}

	if a.scrollX != 0 || a.scrollY != 0 {
		indicatorFont, _ := walk.NewFont("Arial", 7, 0)
		indicatorBrush, _ := walk.NewSolidColorBrush(walk.RGB(255, 255, 220))
		indicatorPen, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(180, 180, 180))
		ir := walk.Rectangle{X: bounds.X + rulerSize + 2, Y: bounds.Y + bounds.Height - 16, Width: 80, Height: 14}
		canvas.FillRectanglePixels(indicatorBrush, ir)
		canvas.DrawRectanglePixels(indicatorPen, ir)
		canvas.DrawTextPixels(fmt.Sprintf("Pan %d,%d", -a.scrollX, -a.scrollY), indicatorFont, walk.RGB(80, 80, 80), ir, walk.TextCenter|walk.TextVCenter)
		indicatorFont.Dispose()
		indicatorBrush.Dispose()
		indicatorPen.Dispose()
	}

	for i, el := range a.elements {
		rx, ry := el.XMM, el.YMM
		rw, rh := el.WidthMM, el.HeightMM
		ex, ey := ox+a.mmToPxX(rx), oy+a.mmToPxY(ry)
		ew, eh := a.mmToPxLenX(rw), a.mmToPxLenY(rh)
		r := walk.Rectangle{X: ex, Y: ey, Width: ew, Height: eh}

		if el.Type == "text" {
			drawn := false
			if el.RTFRaw != "" && r.Width > 0 && r.Height > 0 {
				if raw, err := base64.StdEncoding.DecodeString(el.RTFRaw); err == nil && len(raw) > 0 {
					if err := print.FormatRTFToHDC(uintptr(canvas.HDC()), raw, r.X, r.Y, r.X+r.Width, r.Y+r.Height); err == nil {
						drawn = true
					}
				}
			}
			if !drawn {
				screenDPI := float64(a.dpi)
				if screenDPI <= 0 {
					screenDPI = 96
				}
				fs := int(el.FontSize * screenDPI / 72.0 * a.zoom)
				if el.HeightMM > 0 {
					fs = a.mmToPxLenY(el.HeightMM)
				}
				if fs < 4 {
					fs = 4
				}
				st := walk.FontStyle(0)
				if el.Bold {
					st |= walk.FontBold
				}
				if el.Italic {
					st |= walk.FontItalic
				}
				if el.Underline {
					st |= walk.FontUnderline
				}
				font, err := walk.NewFont(el.FontName, fs, st)
				if err != nil {
					font, _ = walk.NewFont("Arial", fs, st)
				}
				fg := walk.RGB(0, 0, 0)
				if el.Color != nil {
					cr, cg, cb, _ := el.Color.RGBA()
					fg = walk.RGB(byte(cr>>8), byte(cg>>8), byte(cb>>8))
				}
				flags := drawTextFlags(el.Align)
				canvas.DrawTextPixels(el.Text, font, fg, r, flags)
				font.Dispose()
			}
		} else if el.Type == "image" || el.Type == "preview" {
			if err := a.drawWMFElement(canvas, r, el); err != nil {
				img, imgErr := walk.NewImageFromFile(el.ImagePath)
				if imgErr != nil {
					canvas.FillRectanglePixels(darkBrush, r)
				} else {
					canvas.DrawImageStretchedPixels(img, r)
					img.Dispose()
				}
			}
		} else if el.Type == "line" {
			p, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
			canvas.DrawLinePixels(p, walk.Point{X: ex, Y: ey}, walk.Point{X: ex + ew, Y: ey + eh})
			p.Dispose()
		} else if el.Type == "rect" {
			p, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
			canvas.DrawRectanglePixels(p, r)
			p.Dispose()
		} else if el.Type == "ellipse" {
			p, _ := walk.NewCosmeticPen(walk.PenSolid, walk.RGB(0, 0, 0))
			canvas.DrawEllipsePixels(p, r)
			p.Dispose()
		}

		if i == a.selectedIdx {
			dp, _ := walk.NewCosmeticPen(walk.PenDash, walk.RGB(0, 0, 0))
			canvas.DrawRectanglePixels(dp, r)
			dp.Dispose()
			hs := 4
			for _, h := range []walk.Point{
				{X: ex - hs, Y: ey - hs}, {X: ex + ew/2 - hs, Y: ey - hs}, {X: ex + ew - hs, Y: ey - hs},
				{X: ex - hs, Y: ey + eh/2 - hs}, {X: ex + ew - hs, Y: ey + eh/2 - hs},
				{X: ex - hs, Y: ey + eh - hs}, {X: ex + ew/2 - hs, Y: ey + eh - hs}, {X: ex + ew - hs, Y: ey + eh - hs},
			} {
				canvas.FillRectanglePixels(blackBrush, walk.Rectangle{X: h.X, Y: h.Y, Width: hs * 2, Height: hs * 2})
			}
		}
	}
	return nil
}

func (a *App) canvasMouseDown(x, y int, button walk.MouseButton) {
	log.Printf("mouseDown: x=%d y=%d button=%v tool=%s layout=%v", x, y, button, a.tool, a.currentLayout != nil)
	if a.currentLayout == nil {
		return
	}
	if button == walk.RightButton {
		a.canvasSelectAt(x, y)
		a.invalidateCanvas()
		a.updateStatus()
		a.showCanvasPopup(x, y)
		return
	}
	if button == walk.MiddleButton {
		a.panning = true
		a.panStart = image.Point{X: x, Y: y}
		return
	}
	if button != walk.LeftButton {
		return
	}
	labelMargin := 10
	cx, cy := x-rulerSize-labelMargin, y-rulerSize-labelMargin
	lw, lh := a.mmToPxLenX(a.renderW()), a.mmToPxLenY(a.renderH())
	insideLabel := cx >= 0 && cy >= 0 && cx <= lw && cy <= lh

	if a.tool == "select" {
		clicked := -1
		for i := len(a.elements) - 1; i >= 0; i-- {
			el := &a.elements[i]
			if el.Type == "preview" {
				continue
			}
			rx, ry := el.XMM, el.YMM
			rw, rh := el.WidthMM, el.HeightMM
			ex, ey, ew, eh := a.mmToPxX(rx), a.mmToPxY(ry), a.mmToPxLenX(rw), a.mmToPxLenY(rh)
			if cx >= ex-4 && cx <= ex+ew+4 && cy >= ey-4 && cy <= ey+eh+4 {
				clicked = i
				break
			}
		}
		a.selectedIdx = clicked
		if clicked >= 0 {
			isDoubleClick := time.Since(a.lastClickAt) < 450*time.Millisecond && absInt(x-a.lastClickPoint.X) <= 3 && absInt(y-a.lastClickPoint.Y) <= 3
			a.lastClickAt = time.Now()
			a.lastClickPoint = image.Point{X: x, Y: y}
			if isDoubleClick {
				a.invalidateCanvas()
				a.updateStatus()
				a.onProperties()
				return
			}
			el := &a.elements[clicked]
			a.dragging = true
			a.dragUndoPending = true
			a.dragStart = image.Point{X: x, Y: y}
			a.dragOrigX, a.dragOrigY = el.XMM, el.YMM
			a.dragOrigW, a.dragOrigH = el.WidthMM, el.HeightMM
			a.dragHandle = a.hitHandle(cx, cy, el)
		} else {
			a.lastClickAt = time.Time{}
		}
	} else if a.tool == "zoom" {
		if button == walk.LeftButton {
			a.zoom = math.Min(4.0, a.zoom*1.25)
		} else {
			a.zoom = math.Max(0.25, a.zoom/1.25)
		}
		a.invalidateCanvas()
		a.updateStatus()
		return
	} else if a.tool == "text" {
		if !insideLabel {
			a.setTool("select")
			a.invalidateCanvas()
			return
		}
		rxmm := math.Round(a.pxToMmX(cx)*100) / 100
		rymm := math.Round(a.pxToMmY(cy)*100) / 100
		pxmm, pymm := rxmm, rymm
		physW := a.renderW()
		a.pushUndo()
		el := LabelElement{
			ID: nextElemID, Type: "text",
			XMM: pxmm, YMM: pymm,
			WidthMM: math.Max(2, physW-pxmm), HeightMM: 4, Text: "Texto", FontName: a.defaultFontName, FontSize: a.defaultFontSize, Color: a.defaultTextColor,
		}
		a.clampElementToLabel(&el)
		nextElemID++
		a.elements = append(a.elements, el)
		a.selectedIdx = len(a.elements) - 1
		a.setTool("select")
	} else if a.tool == "line" || a.tool == "rect" || a.tool == "ellipse" {
		if !insideLabel {
			a.setTool("select")
			a.invalidateCanvas()
			return
		}
		rxmm := math.Round(a.pxToMmX(cx)*100) / 100
		rymm := math.Round(a.pxToMmY(cy)*100) / 100
		pxmm, pymm := rxmm, rymm
		a.pushUndo()
		el := LabelElement{
			ID: nextElemID, Type: a.tool,
			XMM: pxmm, YMM: pymm,
			WidthMM: 10, HeightMM: 6, Color: color.Black,
		}
		a.clampElementToLabel(&el)
		nextElemID++
		a.elements = append(a.elements, el)
		a.selectedIdx = len(a.elements) - 1
		a.setTool("select")
	}
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) canvasSelectAt(x, y int) {
	labelMargin := 10
	cx, cy := x-rulerSize-labelMargin, y-rulerSize-labelMargin
	clicked := -1
	for i := len(a.elements) - 1; i >= 0; i-- {
		el := &a.elements[i]
		if el.Type == "preview" {
			continue
		}
		ex, ey := a.mmToPxX(el.XMM), a.mmToPxY(el.YMM)
		ew, eh := a.mmToPxLenX(el.WidthMM), a.mmToPxLenY(el.HeightMM)
		if cx >= ex-4 && cx <= ex+ew+4 && cy >= ey-4 && cy <= ey+eh+4 {
			clicked = i
			break
		}
	}
	a.selectedIdx = clicked
}

func (a *App) hitHandle(cx, cy int, el *LabelElement) string {
	rx, ry := el.XMM, el.YMM
	rw, rh := el.WidthMM, el.HeightMM
	ex, ey := a.mmToPxX(rx), a.mmToPxY(ry)
	ew, eh := a.mmToPxLenX(rw), a.mmToPxLenY(rh)
	hs := 6
	type hp struct {
		n      string
		cx, cy int
	}
	for _, p := range []hp{
		{"nw", ex, ey}, {"n", ex + ew/2, ey}, {"ne", ex + ew, ey},
		{"w", ex, ey + eh/2}, {"e", ex + ew, ey + eh/2},
		{"sw", ex, ey + eh}, {"s", ex + ew/2, ey + eh}, {"se", ex + ew, ey + eh},
	} {
		if cx >= p.cx-hs && cx <= p.cx+hs && cy >= p.cy-hs && cy <= p.cy+hs {
			return p.n
		}
	}
	return ""
}

func (a *App) canvasMouseMove(x, y int, button walk.MouseButton) {
	if a.panning {
		dx := x - a.panStart.X
		dy := y - a.panStart.Y
		a.scrollX += dx
		a.scrollY += dy
		a.panStart = image.Point{X: x, Y: y}
		a.invalidateCanvas()
		return
	}
	if !a.dragging || a.selectedIdx < 0 {
		return
	}
	rdx := float64(x-a.dragStart.X) / (float64(a.dpi)/25.4*a.zoom + 1e-9)
	rdy := float64(y-a.dragStart.Y) / (float64(a.dpi)/25.4*a.zoom + 1e-9)
	pdx, pdy := rdx, rdy
	el := &a.elements[a.selectedIdx]
	if a.dragUndoPending && (math.Abs(rdx) > 0.01 || math.Abs(rdy) > 0.01 || a.dragHandle != "") {
		a.pushUndo()
		a.dragUndoPending = false
	}
	if h := a.dragHandle; h != "" {
		nw, nh, nx, ny := a.dragOrigW, a.dragOrigH, a.dragOrigX, a.dragOrigY
		if strings.Contains(h, "e") {
			nw = math.Max(1, a.dragOrigW+pdx)
		}
		if strings.Contains(h, "w") {
			nw = math.Max(1, a.dragOrigW-pdx)
			nx = a.dragOrigX + pdx
		}
		if strings.Contains(h, "s") {
			nh = math.Max(1, a.dragOrigH+pdy)
		}
		if strings.Contains(h, "n") {
			nh = math.Max(1, a.dragOrigH-pdy)
			ny = a.dragOrigY + pdy
		}
		el.XMM, el.YMM, el.WidthMM, el.HeightMM = nx, ny, nw, nh
	} else {
		el.XMM, el.YMM = a.dragOrigX+pdx, a.dragOrigY+pdy
	}
	a.clampElementToLabel(el)
	a.invalidateCanvas()
	a.updateStatus()
}

func (a *App) clampElementToLabel(el *LabelElement) {
	if a.currentLayout == nil || el == nil {
		return
	}
	designW, designH := a.renderW(), a.renderH()
	if el.WidthMM > designW {
		el.WidthMM = designW
	}
	if el.HeightMM > designH {
		el.HeightMM = designH
	}
	el.XMM = math.Max(0, math.Min(el.XMM, designW-el.WidthMM))
	el.YMM = math.Max(0, math.Min(el.YMM, designH-el.HeightMM))
}

func (a *App) canvasMouseUp(x, y int, button walk.MouseButton) {
	if button == walk.MiddleButton {
		a.panning = false
	}
	a.dragging = false
	a.dragUndoPending = false
}

func main() {
	app := NewApp()
	app.loadAllLayouts()
	if err := app.run(); err != nil {
		log.Fatal(err)
	}
}
