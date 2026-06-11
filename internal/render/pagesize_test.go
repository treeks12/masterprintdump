package render

import (
	"math"
	"testing"
)

func TestCadMapaDeviceCapIndicesMatchDecompile(t *testing.T) {
	if DeviceCapHorzRes != 8 || DeviceCapVertRes != 10 || DeviceCapHorzSize != 4 || DeviceCapVertSize != 6 {
		t.Fatalf("device cap constants drifted from CadMapa anchors")
	}
}

func TestCadMapaPxPerMmFUN00521c3c(t *testing.T) {
	got := CadMapaPxPerMm(793, 210)
	want := 793.0 / 210.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("CadMapaPxPerMm=%v want %v", got, want)
	}
}

func TestCadMapaPageSizeMMUsesHorzSizeVertSize(t *testing.T) {
	w, h := CadMapaPageSizeMM(CadMapaDeviceCaps{HorzRes: 793, VertRes: 1122, HorzSize: 210, VertSize: 297, DPIX: 96, DPIY: 96})
	if w != 210 || h != 297 {
		t.Fatalf("CadMapaPageSizeMM=%.0fx%.0f want 210x297", w, h)
	}
}

func TestCadMapaPageSizeMMHorzResFallback(t *testing.T) {
	w, h := CadMapaPageSizeMM(CadMapaDeviceCaps{HorzRes: 800, VertRes: 1200, DPIX: 203, DPIY: 203})
	wantW := 800.0 * 25.4 / 203.0
	wantH := 1200.0 * 25.4 / 203.0
	if math.Abs(w-wantW) > 0.01 || math.Abs(h-wantH) > 0.01 {
		t.Fatalf("CadMapaPageSizeMM=%vx%v want %vx%v", w, h, wantW, wantH)
	}
}

func TestPageSizePhysicalDiffersFromCadMapaPath(t *testing.T) {
	physW, _ := PageSizeFromPhysicalPx(850, 1200, 96, 96)
	cadW, _ := CadMapaPageSizeMM(CadMapaDeviceCaps{HorzRes: 793, VertRes: 1122, HorzSize: 210, VertSize: 297, DPIX: 96, DPIY: 96})
	if physW <= cadW {
		t.Fatalf("physical width %.2f should exceed CadMapa printable %.2f on this fixture", physW, cadW)
	}
}

func TestCadMapaPageSizeMMZeroSafe(t *testing.T) {
	w, h := CadMapaPageSizeMM(CadMapaDeviceCaps{})
	if w != 0 || h != 0 {
		t.Fatalf("zero caps should return 0x0, got %vx%v", w, h)
	}
}
