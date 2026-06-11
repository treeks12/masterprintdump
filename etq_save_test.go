//go:build windows

package main

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"masterprint-native/internal/etq"
)

func TestPreflightETQSaveRejectsAddDeleteAndTypeChange(t *testing.T) {
	base := []etqElementSnapshot{{FileOffset: 0x100, Type: "text", Text: "ABC"}}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x100, Type: "text", Text: "ABC"}, {Type: "text", Text: "NEW"}}); !errors.Is(err, errETQSaveStructural) {
		t.Fatalf("add err=%v want structural", err)
	}
	if err := preflightETQSave(base, nil); !errors.Is(err, errETQSaveStructural) {
		t.Fatalf("delete err=%v want structural", err)
	}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x100, Type: "image"}}); !errors.Is(err, errETQSaveStructural) {
		t.Fatalf("type err=%v want structural", err)
	}
}

func TestPreflightETQSaveTextRules(t *testing.T) {
	base := []etqElementSnapshot{{FileOffset: 0x100, Type: "text", Text: "ABC", FontName: "Arial", WidthMM: 10, HeightMM: 2}}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x100, Type: "text", Text: "AB", FontName: "Arial", WidthMM: 10, HeightMM: 2}}); !errors.Is(err, errETQSaveVariableText) {
		t.Fatalf("variable text err=%v want variable", err)
	}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x100, Type: "text", Text: "DEF", FontName: "Arial", WidthMM: 10, HeightMM: 2}}); err != nil {
		t.Fatalf("same-length text unexpectedly refused: %v", err)
	}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x100, Type: "text", Text: "ABC", FontName: "Tahoma", WidthMM: 10, HeightMM: 2}}); !errors.Is(err, errETQSaveUnsupported) {
		t.Fatalf("font err=%v want unsupported", err)
	}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x100, Type: "text", Text: "ABC", FontName: "Arial", WidthMM: 11, HeightMM: 2}}); !errors.Is(err, errETQSaveUnsupported) {
		t.Fatalf("text width err=%v want unsupported", err)
	}
}

func TestPreflightETQSaveRefusesRTFTextPatch(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte(`{\rtf1\ansi ABC}`))
	base := []etqElementSnapshot{{FileOffset: 0x100, Type: "text", PayloadRaw: raw, Text: "ABC"}}
	err := preflightETQSave(base, []LabelElement{{FileOffset: 0x100, Type: "text", Text: "DEF"}})
	if !errors.Is(err, errETQSaveUnsupported) {
		t.Fatalf("RTF text err=%v want unsupported", err)
	}
}

func TestPreflightETQSaveImageRules(t *testing.T) {
	base := []etqElementSnapshot{{FileOffset: 0x200, Type: "image", ImagePath: "a.wmf", SymbolName: "a", XMM: 1, YMM: 2, WidthMM: 3, HeightMM: 4}}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x200, Type: "image", ImagePath: "a.wmf", SymbolName: "a", XMM: 2, YMM: 3, WidthMM: 4, HeightMM: 5}}); err != nil {
		t.Fatalf("image rect unexpectedly refused: %v", err)
	}
	if err := preflightETQSave(base, []LabelElement{{FileOffset: 0x200, Type: "image", ImagePath: "b.wmf", SymbolName: "b", XMM: 1, YMM: 2, WidthMM: 3, HeightMM: 4}}); !errors.Is(err, errETQSaveUnsupported) {
		t.Fatalf("image swap err=%v want unsupported", err)
	}
}

func TestMaybeSaveETQSkipsWithoutFlag(t *testing.T) {
	t.Setenv("MASTERPRINT_ETQ_SAVE", "")
	a := NewApp()
	applied, err := a.maybeSaveETQ(filepath.Join(t.TempDir(), "out.ETQ"))
	if err != nil {
		t.Fatalf("unexpected err=%v", err)
	}
	if applied {
		t.Fatal("ETQ save should not apply without MASTERPRINT_ETQ_SAVE")
	}
}

func TestMMToETQRaw(t *testing.T) {
	if got := mmToETQRaw(20.56); got != 2056 {
		t.Fatalf("mmToETQRaw=%d want 2056", got)
	}
}

func TestSaveETQPatchedPositionIntegration(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	doc, err := etq.ParseETQ(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x15d3
	text, ok := findTextAnchor(doc, off)
	if !ok {
		t.Fatalf("text anchor not found at %#x", off)
	}
	a := NewApp()
	a.etqSourcePath = path
	a.etqBaseline = []etqElementSnapshot{{FileOffset: off, Type: "text", PayloadRaw: base64.StdEncoding.EncodeToString(text.PayloadRaw), XMM: text.XMM, YMM: text.YMM, WidthMM: text.WidthMM, HeightMM: text.HeightMM, Text: text.Text, FontName: text.FontName, FontSize: text.FontSize, Bold: text.Bold, Italic: text.Italic, Underline: text.Underline, Align: text.Align}}
	a.elements = []LabelElement{{FileOffset: off, Type: "text", XMM: text.XMM + 0.01, YMM: text.YMM + 0.02, WidthMM: text.WidthMM, HeightMM: text.HeightMM, Text: text.Text, FontName: text.FontName, FontSize: text.FontSize, Bold: text.Bold, Italic: text.Italic, Underline: text.Underline, Align: text.Align}}
	dst := filepath.Join(t.TempDir(), "moved.ETQ")
	if err := a.saveETQPatched(dst); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	wantX := mmToETQRaw(text.XMM + 0.01)
	wantY := mmToETQRaw(text.YMM + 0.02)
	if binary.LittleEndian.Uint32(saved[off+16:off+20]) != wantX || binary.LittleEndian.Uint32(saved[off+20:off+24]) != wantY {
		t.Fatalf("position not patched")
	}
}

func findTextAnchor(doc *etq.ETQFile, offset int) (etqTextAnchor, bool) {
	for _, text := range doc.TextElements {
		if text.FileOffset == offset {
			return etqTextAnchor{PayloadRaw: text.PayloadRaw, XMM: text.XMM, YMM: text.YMM, WidthMM: text.WidthMM, HeightMM: text.HeightMM, Text: text.Text, FontName: text.FontName, FontSize: text.FontSize, Bold: text.Bold, Italic: text.Italic, Underline: text.Underline, Align: text.Align}, true
		}
	}
	return etqTextAnchor{}, false
}

type etqTextAnchor struct {
	PayloadRaw []byte
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
}
