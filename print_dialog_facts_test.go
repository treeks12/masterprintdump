//go:build windows

package main

import (
	"strings"
	"testing"

	"masterprint-native/internal/model"
)

func TestPrintDialogColumnCountOmitsZeroOrNegative(t *testing.T) {
	if got := printDialogColumnCount(model.LayoutDefinition{NumCol: 0}); got != "" {
		t.Fatalf("zero=%q", got)
	}
	if got := printDialogColumnCount(model.LayoutDefinition{NumCol: -1}); got != "" {
		t.Fatalf("negative=%q", got)
	}
	if got := printDialogColumnCount(model.LayoutDefinition{NumCol: 8}); got != "Colunas: 8" {
		t.Fatalf("positive=%q", got)
	}
}

func TestPrintDialogOrientationFromLayoutLandscape(t *testing.T) {
	if got := printDialogOrientation(model.LayoutDefinition{Landscape: 1}); got != "Orientacao: Paisagem" {
		t.Fatalf("landscape=%q", got)
	}
	if got := printDialogOrientation(model.LayoutDefinition{Landscape: 0}); got != "Orientacao: Retrato" {
		t.Fatalf("portrait=%q", got)
	}
}

func TestPrintDialogDocumentStatusUsesExistingStatusFormat(t *testing.T) {
	layout := model.LayoutDefinition{Name: "LNT-2"}
	if got := printDialogDocumentStatus("", "", layout); got != "Tipo:     Modelo: LNT-2" {
		t.Fatalf("layout only=%q", got)
	}
	if got := printDialogDocumentStatus("SLIM", "Etiqueta Adesiva", layout); got != "Tipo: Etiqueta Adesiva    Modelo: SLIM" {
		t.Fatalf("template+type=%q", got)
	}
	if got := printDialogDocumentStatus("", "MATRIX", layout); got != "Tipo: MATRIX    Modelo: LNT-2" {
		t.Fatalf("layout+type=%q", got)
	}
}

func TestBuildPrintDialogFactsUsesPrintPathTotals(t *testing.T) {
	layout := model.LayoutDefinition{Name: "LNT-2", WidthMM: 25, HeightMM: 55.5, NumCol: 8, CopiesPerColumn: 2, Landscape: 1}
	facts := buildPrintDialogFacts(PrintDialogInput{
		Layout:       layout,
		TemplateName: "SLIM",
		LayoutType:   "Etiqueta Adesiva",
		PrinterName:  "PDFCreator",
		UnknownCount: 2,
		Copies:       4,
	})
	if facts.DocumentStatus != "Tipo: Etiqueta Adesiva    Modelo: SLIM" {
		t.Fatalf("documentStatus=%q", facts.DocumentStatus)
	}
	if facts.LabelSizeMM != "25.0 x 55.5 mm" {
		t.Fatalf("labelSize=%q", facts.LabelSizeMM)
	}
	if facts.ColumnCount != "Colunas: 8" {
		t.Fatalf("columnCount=%q", facts.ColumnCount)
	}
	if facts.Orientation != "Orientacao: Paisagem" {
		t.Fatalf("orientation=%q", facts.Orientation)
	}
	if facts.Copies != 4 || facts.CalculatedTotal != 8 {
		t.Fatalf("copies=%d total=%d want 4/8", facts.Copies, facts.CalculatedTotal)
	}
	if facts.PrinterName != "PDFCreator" {
		t.Fatalf("printer=%q", facts.PrinterName)
	}
	if facts.UnknownClause != "2 objetos desconhecidos" {
		t.Fatalf("unknown=%q", facts.UnknownClause)
	}
}

func TestBuildPrintDialogFactsDefaultsCopiesToOne(t *testing.T) {
	layout := model.LayoutDefinition{Name: "PLAIN"}
	facts := buildPrintDialogFacts(PrintDialogInput{Layout: layout, Copies: 0})
	if facts.Copies != 1 || facts.CalculatedTotal != 1 {
		t.Fatalf("copies=%d total=%d want 1/1", facts.Copies, facts.CalculatedTotal)
	}
	if facts.ColumnCount != "" {
		t.Fatalf("columnCount=%q want omitted", facts.ColumnCount)
	}
	if facts.Orientation != "Orientacao: Retrato" {
		t.Fatalf("orientation=%q", facts.Orientation)
	}
}

func TestPrintDialogFactsSummaryLines(t *testing.T) {
	facts := buildPrintDialogFacts(PrintDialogInput{
		Layout:      model.LayoutDefinition{Name: "LNT-2", WidthMM: 25, HeightMM: 55.5},
		PrinterName: "HP LaserJet",
		Copies:      1,
	})
	lines := facts.SummaryLines()
	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"Tipo:     Modelo: LNT-2",
		"Etiqueta: 25.0 x 55.5 mm",
		"Orientacao: Retrato",
		"Copias: 1",
		"Total calculado: 1",
		"Impressora: HP LaserJet",
	}, "\n")
	if got != want {
		t.Fatalf("summary:\n%s\nwant:\n%s", got, want)
	}

	facts = buildPrintDialogFacts(PrintDialogInput{
		Layout:       model.LayoutDefinition{Name: "LNT-2", CopiesPerColumn: 3},
		UnknownCount: 1,
		Copies:       2,
	})
	lines = facts.SummaryLines()
	summary := strings.Join(lines, "\n")
	if !strings.Contains(summary, "Orientacao: Retrato") {
		t.Fatalf("orientation missing: %#v", lines)
	}
	if strings.Contains(summary, "Colunas:") {
		t.Fatalf("column count should be omitted: %#v", lines)
	}
	if !strings.Contains(summary, "Total calculado: 6") {
		t.Fatalf("matrix effective labels missing: %#v", lines)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "1 objeto desconhecido") {
		t.Fatalf("unknown warning missing: %#v", lines)
	}
	if strings.Contains(strings.Join(lines, "\n"), "Impressora:") {
		t.Fatalf("printer line should be omitted: %#v", lines)
	}
}
