//go:build windows

package main

import (
	"os"
	"strings"
	"testing"

	"masterprint-native/internal/print"
)

var printSmokeETQFiles = []string{
	"Canelado algodão (Classic Wave Ramado) lunelli.ETQ",
	"ADAR SOFA CANELADO.ETQ",
}

func requirePrintSmoke(t *testing.T) string {
	t.Helper()
	if !envFlag("MASTERPRINT_PRINT_SMOKE") {
		t.Skip("print smoke disabled: set MASTERPRINT_PRINT_SMOKE=1 and MASTERPRINT_PRINT_PRINTER")
	}
	printer := strings.TrimSpace(os.Getenv("MASTERPRINT_PRINT_PRINTER"))
	if printer == "" {
		t.Skip("print smoke disabled: MASTERPRINT_PRINT_PRINTER is not set")
	}
	return printer
}

func TestPrintSmokeRealPrinter(t *testing.T) {
	printer := requirePrintSmoke(t)

	printers, err := print.EnumPrinters()
	if err != nil {
		t.Fatalf("EnumPrinters: %v", err)
	}
	found := false
	for _, name := range printers {
		if strings.EqualFold(name, printer) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("printer %q not found among installed printers", printer)
	}

	for _, name := range printSmokeETQFiles {
		name := name
		t.Run(name, func(t *testing.T) {
			path := installedETQPath(t, name)
			a := testAppWithInstalledLayouts(t)
			if err := a.loadETQDocument(path); err != nil {
				t.Fatalf("loadETQDocument: %v", err)
			}
			if a.currentLayout == nil {
				t.Fatal("document loaded without layout")
			}
			a.currentPrinter = printer
			if err := a.printDocument(printer, 1); err != nil {
				t.Fatalf("printDocument: %v", err)
			}
		})
	}
}
