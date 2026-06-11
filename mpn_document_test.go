//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"masterprint-native/internal/model"
)

func TestMPNDocumentRoundTrip(t *testing.T) {
	a := NewApp()
	a.layouts["etiqueta"] = []model.LayoutDefinition{{Name: "LNT-2", WidthMM: 70, HeightMM: 37, NumCol: 1}}
	a.currentLayout = &a.layouts["etiqueta"][0]
	a.currentLayoutType = "Etiq. para Composições em Folhas"
	a.currentTemplate = "LNT-2"
	a.currentPrinter = "Test Printer"
	a.elements = []LabelElement{{ID: 1, Type: "text", Text: "ABC", XMM: 1, YMM: 2, WidthMM: 10, HeightMM: 4, FontName: "Arial", FontSize: 8}}

	path := filepath.Join(t.TempDir(), "novo.mpn")
	if err := a.saveMPNDocument(path); err != nil {
		t.Fatalf("saveMPNDocument: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw savedDocument
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.SchemaVersion != mpnSchemaVersion || raw.DocumentKind != mpnDocumentKind || raw.LayoutName != "LNT-2" || len(raw.Elements) != 1 {
		t.Fatalf("saved document metadata/payload: %+v", raw)
	}

	b := NewApp()
	b.layouts["etiqueta"] = a.layouts["etiqueta"]
	if err := b.loadMPNDocument(path); err != nil {
		t.Fatalf("loadMPNDocument: %v", err)
	}
	if b.etqSourcePath != "" || b.currentDocPath != path || b.currentLayout == nil || b.currentLayout.Name != "LNT-2" {
		t.Fatalf("bad loaded state: source=%q path=%q layout=%v", b.etqSourcePath, b.currentDocPath, b.currentLayout)
	}
	if len(b.elements) != 1 || b.elements[0].Text != "ABC" || b.currentPrinter != "Test Printer" {
		t.Fatalf("bad loaded payload: printer=%q elements=%+v", b.currentPrinter, b.elements)
	}
}

func TestMPNDocumentShapeRoundTrip(t *testing.T) {
	a := NewApp()
	a.layouts["etiqueta"] = []model.LayoutDefinition{{Name: "LNT-2", WidthMM: 70, HeightMM: 37, NumCol: 1}}
	a.currentLayout = &a.layouts["etiqueta"][0]
	a.currentLayoutType = "Etiq. para Composições em Folhas"
	a.currentTemplate = "LNT-2"
	a.elements = []LabelElement{
		{ID: 1, Type: "line", XMM: 1, YMM: 2, WidthMM: 20, HeightMM: 0.5},
		{ID: 2, Type: "rect", XMM: 3, YMM: 4, WidthMM: 10, HeightMM: 6},
		{ID: 3, Type: "ellipse", XMM: 5, YMM: 6, WidthMM: 8, HeightMM: 8},
	}
	path := filepath.Join(t.TempDir(), "shapes.mpn")
	if err := a.saveMPNDocument(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	b.layouts["etiqueta"] = a.layouts["etiqueta"]
	if err := b.loadMPNDocument(path); err != nil {
		t.Fatal(err)
	}
	if len(b.elements) != 3 {
		t.Fatalf("elements=%#v", b.elements)
	}
	for i, want := range a.elements {
		got := b.elements[i]
		if got.Type != want.Type || got.XMM != want.XMM || got.YMM != want.YMM || got.WidthMM != want.WidthMM || got.HeightMM != want.HeightMM {
			t.Fatalf("element %d got=%#v want=%#v", i, got, want)
		}
	}
}

func TestLoadMPNRejectsMissingLayoutAndWrongKind(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.mpn")
	data, _ := json.Marshal(savedDocument{SchemaVersion: mpnSchemaVersion, DocumentKind: mpnDocumentKind, LayoutName: "MISSING", LayoutType: "Etiq. para Composições em Folhas"})
	if err := os.WriteFile(missing, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := a.loadMPNDocument(missing); err == nil {
		t.Fatal("expected missing layout error")
	}

	badKind := filepath.Join(dir, "bad.mpn")
	data, _ = json.Marshal(savedDocument{SchemaVersion: 1, LayoutName: "LNT-2", LayoutType: "etiqueta"})
	if err := os.WriteFile(badKind, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := a.loadMPNDocument(badKind); err == nil {
		t.Fatal("expected missing documentKind error")
	}
}

func TestMPNHelpers(t *testing.T) {
	if !isMPNDocument(`C:\tmp\a.mpn`) || !isMPNDocument(`C:\tmp\a.MPN`) || isMPNDocument(`C:\tmp\a.ETQ`) {
		t.Fatal("isMPNDocument mismatch")
	}
	cases := map[string]string{`C:\tmp\nova`: `C:\tmp\nova.mpn`, `C:\tmp\nova.mpn`: `C:\tmp\nova.mpn`, `C:\tmp\nova.MPN`: `C:\tmp\nova.MPN`}
	for in, want := range cases {
		if got := ensureMPNExtension(in); got != want {
			t.Fatalf("ensureMPNExtension(%q)=%q want %q", in, got, want)
		}
	}
	a := NewApp()
	if a.hasETQSource() {
		t.Fatal("new app should not have ETQ source")
	}
	a.etqSourcePath = `C:\tmp\base.ETQ`
	if !a.hasETQSource() {
		t.Fatal("expected ETQ source")
	}
}
