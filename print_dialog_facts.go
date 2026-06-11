//go:build windows

package main

import (
	"fmt"

	"masterprint-native/internal/model"
)

type PrintDialogInput struct {
	Layout       model.LayoutDefinition
	TemplateName string
	LayoutType   string
	PrinterName  string
	UnknownCount int
	Copies       int
}

type PrintDialogFacts struct {
	DocumentStatus  string
	LabelSizeMM     string
	ColumnCount     string
	Orientation     string
	Copies          int
	CalculatedTotal int
	PrinterName     string
	UnknownClause   string
}

func printDialogDocumentStatus(templateName, layoutType string, layout model.LayoutDefinition) string {
	return statusTipoModelo(layoutType, templateName, layout.Name)
}

func printDialogLabelSizeMM(layout model.LayoutDefinition) string {
	return fmt.Sprintf("%.1f x %.1f mm", layout.WidthMM, layout.HeightMM)
}

func printDialogColumnCount(layout model.LayoutDefinition) string {
	if layout.NumCol <= 0 {
		return ""
	}
	return fmt.Sprintf("Colunas: %d", layout.NumCol)
}

func printDialogOrientation(layout model.LayoutDefinition) string {
	if layout.Landscape == 1 {
		return "Orientacao: Paisagem"
	}
	return "Orientacao: Retrato"
}

func printDialogUnknownClause(count int) string {
	return unknownObjectsStatusClause(count)
}

func buildPrintDialogFacts(in PrintDialogInput) PrintDialogFacts {
	copies := in.Copies
	if copies < 1 {
		copies = defaultPrintCopies()
	}
	return PrintDialogFacts{
		DocumentStatus:  printDialogDocumentStatus(in.TemplateName, in.LayoutType, in.Layout),
		LabelSizeMM:     printDialogLabelSizeMM(in.Layout),
		ColumnCount:     printDialogColumnCount(in.Layout),
		Orientation:     printDialogOrientation(in.Layout),
		Copies:          copies,
		CalculatedTotal: totalPrintLabels(in.Layout, copies),
		PrinterName:     in.PrinterName,
		UnknownClause:   printDialogUnknownClause(in.UnknownCount),
	}
}

func (f PrintDialogFacts) SummaryLines() []string {
	lines := []string{
		f.DocumentStatus,
		"Etiqueta: " + f.LabelSizeMM,
	}
	if f.ColumnCount != "" {
		lines = append(lines, f.ColumnCount)
	}
	lines = append(lines,
		f.Orientation,
		fmt.Sprintf("Copias: %d", f.Copies),
		fmt.Sprintf("Total calculado: %d", f.CalculatedTotal),
	)
	if f.PrinterName != "" {
		lines = append(lines, "Impressora: "+f.PrinterName)
	}
	if f.UnknownClause != "" {
		lines = append(lines, f.UnknownClause)
	}
	return lines
}
