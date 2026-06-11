//go:build windows

package render

import "testing"

func TestCadMapaGDIStyleBitmask(t *testing.T) {
	bold, italic, underline, strikeout := CadMapaGDIStyle(0x0f)
	if !bold || !italic || !underline || !strikeout {
		t.Fatalf("style bitmask not decoded: %v %v %v %v", bold, italic, underline, strikeout)
	}
}

func TestCadMapaFitCharWidthLunelli72(t *testing.T) {
	got := CadMapaFitCharWidthPx(0, "72% ALGODÃO", 137, 6, "Arial", 5, 0)
	if got != 9 {
		t.Fatalf("fit width=%d want 9", got)
	}
}

func TestCadMapaFitCharWidthEmpty(t *testing.T) {
	if got := CadMapaFitCharWidthPx(0, "", 137, 6, "Arial", 5, 0); got != 0 {
		t.Fatalf("empty fit width=%d want 0", got)
	}
}
