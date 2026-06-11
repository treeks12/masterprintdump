//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var canonicalETQRoundTripFiles = []string{
	"Canelado algodão (Classic Wave Ramado) lunelli.ETQ",
	"ADAR SOFA CANELADO.ETQ",
	"FAVERO.ETQ",
}

type etqRoundTripSnapshot struct {
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
	Align      string
	ImagePath  string
	SymbolName string
}

type etqRoundTripDocumentSnapshot struct {
	PrinterName    string
	LayoutType     string
	TemplateName   string
	LayoutName     string
	UnknownObjects []savedUnknownObject
	Elements       []etqRoundTripSnapshot
}

func requireRoundTripCorpus() bool {
	return envFlag("MASTERPRINT_REQUIRE_CORPUS")
}

func installedETQPath(t *testing.T, name string) string {
	t.Helper()
	root := `C:\Program Files (x86)\paulimaq`
	if custom := os.Getenv("MASTERPRINT_DATA"); custom != "" {
		root = custom
	}
	path := filepath.Join(root, "ARQUIVOS", name)
	if _, err := os.Stat(path); err != nil {
		if requireRoundTripCorpus() {
			t.Fatalf("required sample ETQ not installed: %s: %v", path, err)
		}
		t.Skipf("sample ETQ not installed: %v", err)
	}
	return path
}

func isolatedETQPath(t *testing.T, source string) string {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "paulimaq", "ARQUIVOS")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filepath.Base(source))
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func testAppWithInstalledLayouts(t *testing.T) *App {
	t.Helper()
	a := NewApp()
	a.loadAllLayouts()
	if len(a.layouts["etiqueta"]) == 0 {
		if requireRoundTripCorpus() {
			t.Fatal("required installed layout catalogs not available")
		}
		t.Skip("installed layout catalogs not available")
	}
	return a
}

func snapshotRoundTripDocument(a *App) etqRoundTripDocumentSnapshot {
	var layoutName string
	if a.currentLayout != nil {
		layoutName = a.currentLayout.Name
	}
	unknown := make([]savedUnknownObject, 0, len(a.unknownObjects))
	for _, obj := range a.unknownObjects {
		unknown = append(unknown, savedUnknownObject{Offset: obj.Offset, Flags: obj.Flags, Tag: obj.Tag, Kind: obj.Kind})
	}
	return etqRoundTripDocumentSnapshot{
		PrinterName:    a.currentPrinter,
		LayoutType:     a.currentLayoutType,
		TemplateName:   a.currentTemplate,
		LayoutName:     layoutName,
		UnknownObjects: unknown,
		Elements:       snapshotRoundTripElements(a.elements),
	}
}

func snapshotRoundTripElements(elements []LabelElement) []etqRoundTripSnapshot {
	out := make([]etqRoundTripSnapshot, 0, len(elements))
	for _, el := range elements {
		if el.Type == "preview" {
			continue
		}
		out = append(out, etqRoundTripSnapshot{
			Type: el.Type, FileOffset: el.FileOffset, FEFlags: el.FEFlags, FETag: el.FETag,
			PayloadRaw: el.PayloadRaw, RTFRaw: el.RTFRaw, WMFRaw: el.WMFRaw, WMFPreRaw: el.WMFPreRaw,
			StyleByte: el.StyleByte, NextX: el.NextX, NextY: el.NextY,
			XMM: el.XMM, YMM: el.YMM, WidthMM: el.WidthMM, HeightMM: el.HeightMM,
			Text: el.Text, FontName: el.FontName, FontSize: el.FontSize,
			Bold: el.Bold, Italic: el.Italic, Underline: el.Underline, Align: el.Align,
			ImagePath: el.ImagePath, SymbolName: el.SymbolName,
		})
	}
	return out
}

func assertRoundTripDocumentsEqual(t *testing.T, want, got etqRoundTripDocumentSnapshot) {
	t.Helper()
	if want.PrinterName != got.PrinterName || want.LayoutType != got.LayoutType || want.TemplateName != got.TemplateName || want.LayoutName != got.LayoutName {
		t.Fatalf("metadata: got printer=%q layoutType=%q template=%q layout=%q want printer=%q layoutType=%q template=%q layout=%q", got.PrinterName, got.LayoutType, got.TemplateName, got.LayoutName, want.PrinterName, want.LayoutType, want.TemplateName, want.LayoutName)
	}
	if !reflect.DeepEqual(want.UnknownObjects, got.UnknownObjects) {
		t.Fatalf("unknownObjects: got=%+v want=%+v", got.UnknownObjects, want.UnknownObjects)
	}
	assertRoundTripElementsEqual(t, want.Elements, got.Elements)
}

func assertSidecarCopiesWritten(t *testing.T, etqPath string) {
	t.Helper()
	var first []byte
	for i, path := range sidecarPaths(etqPath) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("sidecar copy %s not written: %v", path, err)
		}
		if i == 0 {
			first = data
			continue
		}
		if !bytes.Equal(first, data) {
			t.Fatalf("sidecar copy %s differs from first copy", path)
		}
	}
}

func assertRoundTripElementsEqual(t *testing.T, want, got []etqRoundTripSnapshot) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("element count: got %d want %d", len(got), len(want))
	}
	for i := range want {
		w, g := want[i], got[i]
		prefix := fmt.Sprintf("element[%d] offset=%#x type=%s", i, w.FileOffset, w.Type)
		if w.Type != g.Type || w.FileOffset != g.FileOffset || w.FEFlags != g.FEFlags || w.FETag != g.FETag {
			t.Fatalf("%s identity: got=%+v want=%+v", prefix, g, w)
		}
		if w.PayloadRaw != g.PayloadRaw {
			t.Fatalf("%s PayloadRaw: got len=%d want len=%d", prefix, len(g.PayloadRaw), len(w.PayloadRaw))
		}
		if w.RTFRaw != g.RTFRaw {
			t.Fatalf("%s RTFRaw: got len=%d want len=%d", prefix, len(g.RTFRaw), len(w.RTFRaw))
		}
		if w.WMFRaw != g.WMFRaw {
			t.Fatalf("%s WMFRaw: got len=%d want len=%d", prefix, len(g.WMFRaw), len(w.WMFRaw))
		}
		if w.WMFPreRaw != g.WMFPreRaw {
			t.Fatalf("%s WMFPreRaw: got len=%d want len=%d", prefix, len(g.WMFPreRaw), len(w.WMFPreRaw))
		}
		if w.StyleByte != g.StyleByte || w.NextX != g.NextX || w.NextY != g.NextY {
			t.Fatalf("%s style/chain: got style=%#x next=(%d,%d) want style=%#x next=(%d,%d)", prefix, g.StyleByte, g.NextX, g.NextY, w.StyleByte, w.NextX, w.NextY)
		}
		if !nearFloat64(w.XMM, g.XMM) || !nearFloat64(w.YMM, g.YMM) || !nearFloat64(w.WidthMM, g.WidthMM) || !nearFloat64(w.HeightMM, g.HeightMM) {
			t.Fatalf("%s geometry: got=%.4f,%.4f %.4fx%.4f want=%.4f,%.4f %.4fx%.4f", prefix, g.XMM, g.YMM, g.WidthMM, g.HeightMM, w.XMM, w.YMM, w.WidthMM, w.HeightMM)
		}
		if w.Text != g.Text || w.FontName != g.FontName || !nearFloat64(w.FontSize, g.FontSize) || w.Bold != g.Bold || w.Italic != g.Italic || w.Underline != g.Underline || w.Align != g.Align {
			t.Fatalf("%s text/style: got=%q font=%q size=%.2f b=%v i=%v u=%v align=%q want=%q font=%q size=%.2f b=%v i=%v u=%v align=%q", prefix, g.Text, g.FontName, g.FontSize, g.Bold, g.Italic, g.Underline, g.Align, w.Text, w.FontName, w.FontSize, w.Bold, w.Italic, w.Underline, w.Align)
		}
		if w.ImagePath != g.ImagePath || w.SymbolName != g.SymbolName {
			t.Fatalf("%s image: got path=%q symbol=%q want path=%q symbol=%q", prefix, g.ImagePath, g.SymbolName, w.ImagePath, w.SymbolName)
		}
	}
}

func TestCanonicalETQRoundTripHarness(t *testing.T) {
	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "appdata"))
	for _, name := range canonicalETQRoundTripFiles {
		name := name
		t.Run(name, func(t *testing.T) {
			source := installedETQPath(t, name)
			t.Run("sidecar", func(t *testing.T) {
				path := isolatedETQPath(t, source)
				originalETQ, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				a := testAppWithInstalledLayouts(t)
				if err := a.loadETQDocument(path); err != nil {
					t.Fatalf("loadETQDocument: %v", err)
				}
				want := snapshotRoundTripDocument(a)
				if err := a.saveSidecar(path); err != nil {
					t.Fatalf("saveSidecar: %v", err)
				}
				assertSidecarCopiesWritten(t, path)
				afterSaveETQ, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(originalETQ, afterSaveETQ) {
					t.Fatal("sidecar save modified ETQ bytes")
				}
				b := testAppWithInstalledLayouts(t)
				if err := b.loadETQDocument(path); err != nil {
					t.Fatalf("reload ETQ: %v", err)
				}
				assertRoundTripDocumentsEqual(t, want, snapshotRoundTripDocument(b))
			})
			t.Run("mpn", func(t *testing.T) {
				path := isolatedETQPath(t, source)
				originalETQ, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				a := testAppWithInstalledLayouts(t)
				if err := a.loadETQDocument(path); err != nil {
					t.Fatalf("loadETQDocument: %v", err)
				}
				want := snapshotRoundTripDocument(a)
				mpnPath := filepath.Join(t.TempDir(), "roundtrip.mpn")
				if err := a.saveMPNDocument(mpnPath); err != nil {
					t.Fatalf("saveMPNDocument: %v", err)
				}
				afterSaveETQ, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(originalETQ, afterSaveETQ) {
					t.Fatal("MPN save modified ETQ bytes")
				}
				b := testAppWithInstalledLayouts(t)
				if err := b.loadMPNDocument(mpnPath); err != nil {
					t.Fatalf("loadMPNDocument: %v", err)
				}
				assertRoundTripDocumentsEqual(t, want, snapshotRoundTripDocument(b))
			})
		})
	}
}
