//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"masterprint-native/internal/etq"
	"masterprint-native/internal/model"

	"github.com/tailscale/walk"
)

func TestLayoutCategoryFromHeaderExactCatalogs(t *testing.T) {
	cases := map[string]string{
		"Etiq. para Composições em Folhas":      "etiqueta",
		"Etiq. para Composições em Formulários": "etiqueta_m",
		"Etiq. para Composições em Rolo":        "etiqueta_r",
		"TAG'S em Folhas e Formulários":         "tag",
		"Etiq. Ades. Fast Label (Padrão)":       "tag2",
		"Pauli - Tab":                           "tag3",
		"Etiquetas para Jóias":                  "joia",
		"Etiq. para Caixas de Calçados":         "sapato",
		"Cartões de Visita - PRINT CARD":        "cartao",
		"Convites - PRINT INVITE":               "invite",
		"Etiq. Box para CD - PRINT CD FACE":     "box",
		"Caixa de Cartões - PRINT BOX":          "caixa",
		"Etiq. para Plantas":                    "plantas",
		"Pulseiras Bands":                       "fixbands",
		"Etiquetas para CD - CD Center":         "cdcenter",
		"Etiquetas para CD - CD FAST LABEL 2":   "cdfastlab",
		"Etiquetas para CD - PRINT CD LABEL 2":  "cdlab",
		"Etiquetas para CD - Mini CD":           "minicd",
		"Etiquetas para CD - PRINT CD LABEL 3":  "ncd",
		"Print CD Cards":                        "pcd",
		"Photo Quality Álbum":                   "photoa4",
	}
	for header, want := range cases {
		t.Run(header, func(t *testing.T) {
			if got := layoutCategoryFromHeader(header); got != want {
				t.Fatalf("layoutCategoryFromHeader(%q)=%q want %q", header, got, want)
			}
		})
	}
}

func TestWMFIntrinsicSizeFromAldusHeader(t *testing.T) {
	path := `C:\Program Files (x86)\paulimaq\CLIPART\Símbolos\clorox.wmf`
	if _, err := os.Stat(path); err != nil {
		t.Skipf("clipart not installed: %v", err)
	}
	w, h := wmfIntrinsicSizeMM(path)
	if !nearTest(w, 64.69) || !nearTest(h, 64.92) {
		t.Fatalf("clorox intrinsic size=%.2fx%.2f want 64.69x64.92", w, h)
	}
}

func TestWMFSymbolFromElementPreservesEmbeddedAndRect(t *testing.T) {
	embedded := []byte{0xd7, 0xcd, 0xc6, 0x9a, 0x10, 0x20}
	el := LabelElement{Type: "image", FileOffset: 273, WMFRaw: base64.StdEncoding.EncodeToString(embedded), ImagePath: `C:\clipart\clorox.wmf`, StyleByte: 6, NextX: 445, NextY: 445, XMM: 5.29, YMM: 26.96, WidthMM: 4.45, HeightMM: 4.45}
	sym := wmfSymbolFromElement(el)
	if string(sym.Embedded) != string(embedded) {
		t.Fatalf("embedded=%#v", sym.Embedded)
	}
	if sym.FilePath != el.ImagePath || sym.XMM != el.XMM || sym.YMM != el.YMM || sym.WidthMM != el.WidthMM || sym.HeightMM != el.HeightMM || sym.StyleByte != 6 || sym.NextX != 445 || sym.NextY != 445 || sym.FileOffset != 273 {
		t.Fatalf("metadata not preserved: %#v", sym)
	}
}

func TestWMFBytesFromElementPrefersEmbeddedAndFallsBackToFile(t *testing.T) {
	embedded := []byte{0xd7, 0xcd, 0xc6, 0x9a, 0x10, 0x20}
	got, err := wmfBytesFromElement(LabelElement{Type: "image", WMFRaw: base64.StdEncoding.EncodeToString(embedded), ImagePath: filepath.Join(t.TempDir(), "missing.wmf")})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, embedded) {
		t.Fatalf("embedded bytes=%#v", got)
	}

	path := filepath.Join(t.TempDir(), "symbol.wmf")
	want := []byte{1, 2, 3, 4}
	if err := os.WriteFile(path, want, 0644); err != nil {
		t.Fatal(err)
	}
	got, err = wmfBytesFromElement(LabelElement{Type: "image", ImagePath: path})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("file bytes=%#v", got)
	}
}

func TestPrintModelFromElementsPreservesDocumentOrder(t *testing.T) {
	elements := []LabelElement{
		{Type: "image", FileOffset: 0x20, WMFRaw: base64.StdEncoding.EncodeToString([]byte{1, 2, 3}), XMM: 1},
		{Type: "rect", XMM: 1, YMM: 2, WidthMM: 3, HeightMM: 4},
		{Type: "text", FileOffset: 0x30, Text: "ABC", PayloadRaw: base64.StdEncoding.EncodeToString([]byte("ABC")), RTFRaw: base64.StdEncoding.EncodeToString([]byte("{\\rtf1 ABC}")), XMM: 2},
		{Type: "line", XMM: 4, YMM: 5, WidthMM: 6, HeightMM: 7},
		{Type: "image", FileOffset: 0x40, WMFRaw: base64.StdEncoding.EncodeToString([]byte{4, 5, 6}), XMM: 3},
		{Type: "ellipse", XMM: 8, YMM: 9, WidthMM: 10, HeightMM: 11},
	}
	texts, wmfs, objects := printModelFromElements(elements)
	if len(texts) != 1 || len(wmfs) != 2 || len(objects) != 6 {
		t.Fatalf("texts=%d wmfs=%d objects=%d", len(texts), len(wmfs), len(objects))
	}
	if objects[0].Type != "image" || objects[1].Type != "rect" || objects[2].Type != "text" || objects[3].Type != "line" || objects[4].Type != "image" || objects[5].Type != "ellipse" {
		t.Fatalf("print order not preserved: %#v", objects)
	}
	if objects[2].Text.Text != "ABC" || string(objects[2].Text.PayloadRaw) != "ABC" || string(objects[2].Text.RTFRaw) != "{\\rtf1 ABC}" || len(objects[0].WMF.Embedded) != 3 || len(objects[4].WMF.Embedded) != 3 || objects[1].Shape.WidthMM != 3 || objects[5].Shape.HeightMM != 11 {
		t.Fatalf("print payload not preserved: %#v", objects)
	}
}

func TestPrintCopiesAndMatrixTotals(t *testing.T) {
	if got := defaultPrintCopies(); got != 1 {
		t.Fatalf("LNT-2 default copies=%d want 1", got)
	}
	if got := totalPrintLabels(model.LayoutDefinition{Name: "MATRIX", NumCol: 3, CopiesPerColumn: 2}, 1); got != 2 {
		t.Fatalf("matrix total labels=%d want 2", got)
	}
	if got := totalPrintLabels(model.LayoutDefinition{Name: "MATRIX", NumCol: 3, CopiesPerColumn: 2}, 4); got != 8 {
		t.Fatalf("matrix total labels=%d want 8", got)
	}
}

func TestSyncEditableTextPayloadClearsRTFAndUpdatesStyle(t *testing.T) {
	el := LabelElement{Type: "text", Text: "ÚNICO", PayloadRaw: "old", RTFRaw: base64.StdEncoding.EncodeToString([]byte(`{\rtf1 OLD}`)), Bold: true, Underline: true}
	syncEditableTextPayload(&el)
	if el.RTFRaw != "" {
		t.Fatalf("RTFRaw not cleared: %q", el.RTFRaw)
	}
	if el.StyleByte != 5 {
		t.Fatalf("styleByte=%d want 5", el.StyleByte)
	}
	raw, err := base64.StdEncoding.DecodeString(el.PayloadRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte{0xda, 'N', 'I', 'C', 'O'}) {
		t.Fatalf("payload=%#v", raw)
	}
}

func TestFillMissingRawPayloadsPreservesETQBaseline(t *testing.T) {
	doc := savedDocument{Elements: []savedElement{{Type: "text", FileOffset: 0x10, Text: "BASE", FontName: "Arial", FontSize: 8, Bold: true, Underline: true, Align: "left", StyleByte: 5}, {Type: "image", FileOffset: 0x20, ImagePath: `C:\clip\a.wmf`}}}
	baseline := map[int]LabelElement{
		0x10: {Type: "text", Text: "BASE", FontName: "Arial", FontSize: 8, Bold: true, Underline: true, Align: "left", StyleByte: 5, PayloadRaw: "payload", RTFRaw: "rtf"},
		0x20: {Type: "image", ImagePath: `C:\clip\a.wmf`, WMFRaw: "wmf", WMFPreRaw: "pre"},
	}
	fillMissingRawPayloads(&doc, baseline)
	if doc.Elements[0].PayloadRaw != "payload" || doc.Elements[0].RTFRaw != "rtf" {
		t.Fatalf("text raw payloads not preserved: %+v", doc.Elements[0])
	}
	if doc.Elements[1].WMFRaw != "wmf" || doc.Elements[1].WMFPreRaw != "pre" {
		t.Fatalf("wmf raw payloads not preserved: %+v", doc.Elements[1])
	}
}

func TestFillMissingRawPayloadsDoesNotRestoreChangedTextRTF(t *testing.T) {
	doc := savedDocument{Elements: []savedElement{{Type: "text", FileOffset: 0x10, Text: "EDIT"}, {Type: "image", FileOffset: 0x20, ImagePath: `C:\clip\b.wmf`}}}
	baseline := map[int]LabelElement{
		0x10: {Type: "text", Text: "BASE", PayloadRaw: "payload", RTFRaw: "rtf"},
		0x20: {Type: "image", ImagePath: `C:\clip\a.wmf`, WMFRaw: "wmf", WMFPreRaw: "pre"},
	}
	fillMissingRawPayloads(&doc, baseline)
	if doc.Elements[0].PayloadRaw != "" || doc.Elements[0].RTFRaw != "" {
		t.Fatalf("changed text should not restore old raw payloads: %+v", doc.Elements[0])
	}
	if doc.Elements[1].WMFRaw != "" || doc.Elements[1].WMFPreRaw != "" {
		t.Fatalf("changed image should not restore old raw payloads: %+v", doc.Elements[1])
	}
}

func TestFillMissingRawPayloadsDoesNotRestoreClearedTextRTF(t *testing.T) {
	doc := savedDocument{Elements: []savedElement{{Type: "text", FileOffset: 0x10, Text: "", FontName: "Arial", FontSize: 8, Bold: true, Align: "left", StyleByte: 1}}}
	baseline := map[int]LabelElement{0x10: {Type: "text", Text: "BASE", FontName: "Arial", FontSize: 8, Bold: true, Align: "left", StyleByte: 1, PayloadRaw: "payload", RTFRaw: "rtf"}}
	fillMissingRawPayloads(&doc, baseline)
	if doc.Elements[0].PayloadRaw != "" || doc.Elements[0].RTFRaw != "" {
		t.Fatalf("cleared text should not restore old raw payloads: %+v", doc.Elements[0])
	}
}

func TestFillMissingRawPayloadsDoesNotRestoreChangedStyleRTF(t *testing.T) {
	doc := savedDocument{Elements: []savedElement{{Type: "text", FileOffset: 0x10, Text: "BASE", FontName: "Arial", FontSize: 8, Bold: false, Align: "left", StyleByte: 0}}}
	baseline := map[int]LabelElement{0x10: {Type: "text", Text: "BASE", FontName: "Arial", FontSize: 8, Bold: true, Align: "left", StyleByte: 1, PayloadRaw: "payload", RTFRaw: "rtf"}}
	fillMissingRawPayloads(&doc, baseline)
	if doc.Elements[0].PayloadRaw != "" || doc.Elements[0].RTFRaw != "" {
		t.Fatalf("changed style should not restore old raw payloads: %+v", doc.Elements[0])
	}
}

func TestSelectWithoutMoveDoesNotDirty(t *testing.T) {
	a := NewApp()
	a.currentLayout = &model.LayoutDefinition{Name: "LNT-2", WidthMM: 25, HeightMM: 55}
	a.elements = []LabelElement{{ID: 1, Type: "text", XMM: 5, YMM: 5, WidthMM: 10, HeightMM: 4, Text: "ABC"}}
	a.selectedIdx = 0
	a.dragging = true
	a.dragUndoPending = true
	a.canvasMouseUp(0, 0, walk.LeftButton)
	if a.isDocumentDirty() {
		t.Fatal("select-only mouse up should not dirty document")
	}
}

func TestDragMoveDoesDirty(t *testing.T) {
	a := NewApp()
	a.currentLayout = &model.LayoutDefinition{Name: "LNT-2", WidthMM: 25, HeightMM: 55}
	a.dpi = 96
	a.zoom = 1
	a.elements = []LabelElement{{ID: 1, Type: "text", XMM: 5, YMM: 5, WidthMM: 10, HeightMM: 4, Text: "ABC"}}
	a.selectedIdx = 0
	a.dragging = true
	a.dragUndoPending = true
	a.dragStart = image.Point{X: 100, Y: 100}
	a.dragOrigX = 5
	a.dragOrigY = 5
	a.dragOrigW = 10
	a.dragOrigH = 4
	a.canvasMouseMove(130, 100, walk.LeftButton)
	if !a.isDocumentDirty() {
		t.Fatal("real drag should dirty document")
	}
}

func TestSaveSidecarPreservesHeaderMetadata(t *testing.T) {
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata"))
	a := NewApp()
	a.currentLayout = &model.LayoutDefinition{Name: "LNT-2"}
	a.currentPrinter = "Epson Stylus Photo R200"
	a.currentLayoutType = "Etiq. para Composições em Folhas"
	a.currentTemplate = "LNT-2"
	a.unknownObjects = []etq.ETQUnknownObject{{Offset: 0x72c, Flags: 0, Tag: 0, Kind: "text-like"}}
	a.elements = []LabelElement{{ID: 1, Type: "text", FileOffset: 0x100, FEFlags: 0, FETag: 1, Text: "ABC", XMM: 1, YMM: 2, WidthMM: 3, HeightMM: 4}}

	path := filepath.Join(t.TempDir(), "sample.etq")
	if err := a.saveSidecar(path); err != nil {
		t.Fatalf("saveSidecar: %v", err)
	}
	data, err := os.ReadFile(path + ".masterprint-native.json")
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var doc savedDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal sidecar: %v", err)
	}
	if doc.SchemaVersion != 1 || doc.DocumentKind != "" || doc.PrinterName != a.currentPrinter || doc.LayoutType != a.currentLayoutType || doc.TemplateName != a.currentTemplate {
		t.Fatalf("metadata not preserved: %+v", doc)
	}
	if len(doc.UnknownObjects) != 1 || doc.UnknownObjects[0].Offset != 0x72c || doc.UnknownObjects[0].Kind != "text-like" {
		t.Fatalf("unknown objects not preserved: %+v", doc.UnknownObjects)
	}
	if len(doc.Elements) != 1 || doc.Elements[0].FEFlags != 0 || doc.Elements[0].FETag != 1 {
		t.Fatalf("FE tag/flags not preserved: %+v", doc.Elements)
	}
}

func TestSaveSidecarWritesAllWritablePaths(t *testing.T) {
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata"))
	a := NewApp()
	a.currentLayout = &model.LayoutDefinition{Name: "LNT-2"}
	a.currentTemplate = "LNT-2"
	a.elements = []LabelElement{{Type: "text", FileOffset: 0x100, Text: "ABC"}}
	path := filepath.Join(t.TempDir(), "sample.etq")
	if err := a.saveSidecar(path); err != nil {
		t.Fatal(err)
	}
	for _, sidecar := range sidecarPaths(path) {
		if _, err := os.Stat(sidecar); err != nil {
			t.Fatalf("expected sidecar %s: %v", sidecar, err)
		}
	}
}

func TestLoadSidecarPrefersNewestCandidate(t *testing.T) {
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata"))
	path := filepath.Join(t.TempDir(), "sample.etq")
	paths := sidecarPaths(path)
	oldDoc := savedDocument{SchemaVersion: 1, LayoutName: "LNT-2", Elements: []savedElement{{Type: "text", Text: "OLD"}}}
	newDoc := savedDocument{SchemaVersion: 1, LayoutName: "LNT-2", Elements: []savedElement{{Type: "text", Text: "NEW"}}}
	writeSidecarForTest(t, paths[0], oldDoc)
	writeSidecarForTest(t, paths[1], newDoc)
	oldTime := time.Now().Add(-time.Hour)
	newTime := time.Now()
	if err := os.Chtimes(paths[0], oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(paths[1], newTime, newTime); err != nil {
		t.Fatal(err)
	}

	a := NewApp()
	if !a.loadSidecar(path, "LNT-2") {
		t.Fatal("sidecar did not load")
	}
	if len(a.elements) != 1 || a.elements[0].Text != "NEW" {
		t.Fatalf("loaded elements=%#v", a.elements)
	}
}

func TestLoadSidecarRestoresUnknownObjects(t *testing.T) {
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata"))
	path := filepath.Join(t.TempDir(), "sample.etq")
	doc := savedDocument{SchemaVersion: 1, LayoutName: "LNT-2", UnknownObjects: []savedUnknownObject{{Offset: 0x72c, Flags: 1, Tag: 2, Kind: "opaque"}}, Elements: []savedElement{{Type: "text", Text: "ABC"}}}
	writeSidecarForTest(t, sidecarPaths(path)[0], doc)
	a := NewApp()
	if !a.loadSidecar(path, "LNT-2") {
		t.Fatal("sidecar did not load")
	}
	if len(a.unknownObjects) != 1 || a.unknownObjects[0].Offset != 0x72c || a.unknownObjects[0].Kind != "opaque" {
		t.Fatalf("unknown objects=%#v", a.unknownObjects)
	}
}

func writeSidecarForTest(t *testing.T, path string, doc savedDocument) {
	t.Helper()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAsETQCopiesSourceBinary(t *testing.T) {
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata"))
	dir := t.TempDir()
	src := filepath.Join(dir, "source.ETQ")
	dst := filepath.Join(dir, "copy.ETQ")
	want := []byte("original-etq-bytes")
	if err := os.WriteFile(src, want, 0644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	a.currentLayout = &model.LayoutDefinition{Name: "LNT-2"}
	a.currentTemplate = "LNT-2"
	a.etqSourcePath = src
	a.elements = []LabelElement{{Type: "text", FileOffset: 0x100, Text: "ABC"}}
	applied, err := a.saveAsETQToPath(dst)
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("ETQ patch should not apply without MASTERPRINT_ETQ_SAVE")
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("dst bytes=%q want %q", got, want)
	}
	if a.currentDocPath != dst || a.etqSourcePath != dst || a.isDocumentDirty() {
		t.Fatalf("state not updated after Save As: path=%q source=%q dirty=%v", a.currentDocPath, a.etqSourcePath, a.isDocumentDirty())
	}
	if _, err := os.Stat(dst + ".masterprint-native.json"); err != nil {
		t.Fatalf("sidecar missing: %v", err)
	}
}

func TestDocumentDirtyAndSavedState(t *testing.T) {
	a := NewApp()
	a.currentLayout = &model.LayoutDefinition{Name: "LNT-2"}
	if a.isDocumentDirty() {
		t.Fatal("new app should be clean")
	}
	a.elements = []LabelElement{{Type: "text", FileOffset: 0x100, Text: "ABC"}}
	a.pushUndo()
	if !a.isDocumentDirty() {
		t.Fatal("pushUndo should mark document dirty")
	}
	a.markDocumentSaved()
	if a.isDocumentDirty() {
		t.Fatal("markDocumentSaved should clear dirty state")
	}
	if len(a.etqBaseline) != 1 || a.etqBaseline[0].Text != "ABC" {
		t.Fatalf("baseline not refreshed: %#v", a.etqBaseline)
	}
}

func TestApplySelectedLayoutSetsBlankNativeDocument(t *testing.T) {
	a := NewApp()
	a.layouts["etiqueta"] = []model.LayoutDefinition{{Name: "LNT-2", WidthMM: 25, HeightMM: 55}}
	a.currentDocPath = `C:\old\doc.ETQ`
	a.etqSourcePath = `C:\old\doc.ETQ`
	a.currentPrinter = "Old Printer"
	a.elements = []LabelElement{{Type: "text", Text: "OLD"}}
	a.pushUndo()
	if err := a.applySelectedLayout("etiqueta", 0); err != nil {
		t.Fatal(err)
	}
	if a.currentLayout == nil || a.currentLayout.Name != "LNT-2" || a.currentLayoutType != "Etiq. para Composições em Folhas" || a.currentTemplate != "LNT-2" {
		t.Fatalf("layout state invalid: %#v template=%q type=%q", a.currentLayout, a.currentTemplate, a.currentLayoutType)
	}
	if a.currentDocPath != "" || a.etqSourcePath != "" || a.currentPrinter != "" || len(a.elements) != 0 || a.isDocumentDirty() {
		t.Fatalf("document not reset: path=%q source=%q printer=%q elements=%d dirty=%v", a.currentDocPath, a.etqSourcePath, a.currentPrinter, len(a.elements), a.isDocumentDirty())
	}
}

func TestSaveStatusText(t *testing.T) {
	if got := saveStatusText(true, false); got != "Documento salvo (.mpn)" {
		t.Fatalf("mpn status=%q", got)
	}
	if got := saveStatusText(false, true); got != "Alteracoes salvas no .ETQ (experimental)" {
		t.Fatalf("etq status=%q", got)
	}
	if got := saveStatusText(false, false); got != "Alteracoes salvas no documento auxiliar; o .ETQ original nao foi alterado" {
		t.Fatalf("sidecar status=%q", got)
	}
}

func TestLayoutHeaderFromCategory(t *testing.T) {
	cases := map[string]string{
		"etiqueta":   "Etiq. para Composições em Folhas",
		"etiqueta_m": "Etiq. para Composições em Formulários",
		"tag":        "TAG'S em Folhas e Formulários",
		"joia":       "Etiquetas para Jóias",
	}
	for cat, want := range cases {
		if got := layoutHeaderFromCategory(cat); got != want {
			t.Fatalf("layoutHeaderFromCategory(%q)=%q want %q", cat, got, want)
		}
	}
}

func TestStatusTipoModelo(t *testing.T) {
	got := statusTipoModelo("etiqueta", "LNT-2", "")
	want := "Tipo: Etiq. para Composições em Folhas    Modelo: LNT-2"
	if got != want {
		t.Fatalf("statusTipoModelo=%q want %q", got, want)
	}
	got = statusTipoModelo("Etiq. para Composições em Folhas", "", "LNT-2")
	want = "Tipo: Etiq. para Composições em Folhas    Modelo: LNT-2"
	if got != want {
		t.Fatalf("statusTipoModelo ETQ=%q want %q", got, want)
	}
}

func TestAppRenderDesignBoundsLandscapeLNT2(t *testing.T) {
	layout := model.LayoutDefinition{Name: "LNT-2", WidthMM: 25, HeightMM: 55.5, Landscape: 1}
	app := NewApp()
	app.currentLayout = &layout

	if !nearTest(app.renderW(), 55.5) || !nearTest(app.renderH(), 25) {
		t.Fatalf("render design size=(%.2f,%.2f) want (55.5,25)", app.renderW(), app.renderH())
	}

	el := LabelElement{Type: "text", XMM: 50, YMM: 0, WidthMM: 5, HeightMM: 4}
	app.clampElementToLabel(&el)
	if !nearTest(el.XMM, 50) {
		t.Fatalf("clamp should keep design X=50 in HeightMM-wide canvas, got %.2f", el.XMM)
	}

	app.elements = []LabelElement{{Type: "image", XMM: 1.68, YMM: 31.73, WidthMM: 3.93, HeightMM: 4.26}}
	if app.renderH() < 35.99 {
		t.Fatalf("render height should include imported landscape element extents, got %.2f", app.renderH())
	}

	el2 := LabelElement{Type: "text", XMM: 0, YMM: 30, WidthMM: 5, HeightMM: 4}
	app.clampElementToLabel(&el2)
	if !nearTest(el2.YMM, 30) {
		t.Fatalf("clamp should preserve Y inside imported ETQ extent, got %.2f", el2.YMM)
	}
}

func TestAppRenderDesignBoundsPortraitIdentity(t *testing.T) {
	layout := model.LayoutDefinition{Name: "TEST", WidthMM: 70, HeightMM: 37, Landscape: 0}
	app := NewApp()
	app.currentLayout = &layout
	if !nearTest(app.renderW(), 70) || !nearTest(app.renderH(), 37) {
		t.Fatalf("portrait render size=(%.2f,%.2f) want (70,37)", app.renderW(), app.renderH())
	}
}

func TestEnsureETQExtension(t *testing.T) {
	cases := map[string]string{
		`C:\tmp\nova`:     `C:\tmp\nova.ETQ`,
		`C:\tmp\nova.ETQ`: `C:\tmp\nova.ETQ`,
		`C:\tmp\nova.etq`: `C:\tmp\nova.etq`,
	}
	for in, want := range cases {
		if got := ensureETQExtension(in); got != want {
			t.Fatalf("ensureETQExtension(%q)=%q want %q", in, got, want)
		}
	}
}

func nearTest(got, want float64) bool {
	if got < want {
		return want-got < 0.01
	}
	return got-want < 0.01
}
