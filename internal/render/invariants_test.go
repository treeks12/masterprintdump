package render

import "testing"

func TestCadMapaMulDivETQGoldens(t *testing.T) {
	cases := []struct {
		name  string
		mm100 int
		dpi   int
		want  int
	}{
		{name: "lunelli text x", mm100: 2056, dpi: 96, want: 78},
		{name: "lunelli text y", mm100: 353, dpi: 96, want: 13},
		{name: "lunelli text rect width", mm100: 3637, dpi: 96, want: 137},
		{name: "lunelli text rect height", mm100: 161, dpi: 96, want: 6},
		{name: "lunelli text x 120", mm100: 2056, dpi: 120, want: 97},
		{name: "lunelli text rect height 203", mm100: 161, dpi: 203, want: 13},
		{name: "favero wmf x", mm100: 137, dpi: 96, want: 5},
		{name: "favero wmf y", mm100: 2988, dpi: 96, want: 113},
		{name: "favero wmf width", mm100: 496, dpi: 96, want: 19},
		{name: "favero wmf height", mm100: 639, dpi: 96, want: 24},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Mm100ToPx(tc.mm100, tc.dpi); got != tc.want {
				t.Fatalf("Mm100ToPx(%d,%d)=%d want %d", tc.mm100, tc.dpi, got, tc.want)
			}
		})
	}
}

func TestCadMapaDrawTextFlags(t *testing.T) {
	cases := []struct {
		align int16
		rtl   bool
		want  uint32
	}{
		{align: 0, rtl: false, want: 0x810},
		{align: 1, rtl: false, want: 0x811},
		{align: 2, rtl: false, want: 0x812},
		{align: 0, rtl: true, want: 0x910},
		{align: 1, rtl: true, want: 0x911},
		{align: 2, rtl: true, want: 0x912},
		{align: 99, rtl: false, want: 0x810},
	}
	for _, tc := range cases {
		if got := CadMapaDrawTextFlags(tc.align, tc.rtl); got != tc.want {
			t.Fatalf("CadMapaDrawTextFlags(%d,%v)=%#x want %#x", tc.align, tc.rtl, got, tc.want)
		}
	}
}

func TestCadMapaFontHeightFromRectETQOffsets(t *testing.T) {
	cases := []struct {
		name   string
		height int
		dpi    int
		want   int
	}{
		{name: "lunelli 72 algodao 96", height: 161, dpi: 96, want: 6},
		{name: "lunelli 28 poliester 96", height: 213, dpi: 96, want: 8},
		{name: "lunelli feito brasil 96", height: 142, dpi: 96, want: 5},
		{name: "lunelli cnpj 203", height: 271, dpi: 203, want: 22},
		{name: "adar poliester 120", height: 254, dpi: 120, want: 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CadMapaFontHeightPx(tc.height, tc.dpi); got != tc.want {
				t.Fatalf("CadMapaFontHeightPx(%d,%d)=%d want %d", tc.height, tc.dpi, got, tc.want)
			}
		})
	}
}

func TestCadMapaTextRectPxLunelli15d3(t *testing.T) {
	left, top, right, bottom := CadMapaTextRectPx(2056, 353, 3637, 161, 96, 96)
	if left != 78 || top != 13 || right != 215 || bottom != 19 {
		t.Fatalf("text rect=(%d,%d,%d,%d), want (78,13,215,19)", left, top, right, bottom)
	}
}

func TestCadMapaObjectRectPxFromMMLunelli15d3(t *testing.T) {
	left, top, right, bottom := CadMapaObjectRectPxFromMM(20.56, 3.53, 36.37, 1.61, 96, 96)
	if left != 78 || top != 13 || right != 215 || bottom != 19 {
		t.Fatalf("object rect=(%d,%d,%d,%d), want (78,13,215,19)", left, top, right, bottom)
	}
}

func TestCadMapaWMFPlayRectFromMMLunelliClorox(t *testing.T) {
	left, top, right, bottom := CadMapaWMFPlayRectFromMM(5.29, 26.96, 4.45, 4.45, 96, 96)
	if left != 20 || top != 102 || right != 36 || bottom != 118 {
		t.Fatalf("wmf play rect=(%d,%d,%d,%d), want (20,102,36,118)", left, top, right, bottom)
	}
}

func TestCadMapaWMFExternalPlayRectETQOffsets(t *testing.T) {
	cases := []struct {
		name         string
		x, y, w, h   int
		dpi          int
		wantL, wantT int
		wantR, wantB int
	}{
		{name: "lunelli clorox 96", x: 529, y: 2696, w: 445, h: 445, dpi: 96, wantL: 20, wantT: 102, wantR: 36, wantB: 118},
		{name: "lunelli tamborx 96", x: 974, y: 2689, w: 445, h: 445, dpi: 96, wantL: 37, wantT: 102, wantR: 53, wantB: 118},
		{name: "lunelli third wmf 96", x: 1438, y: 2689, w: 432, h: 432, dpi: 96, wantL: 54, wantT: 102, wantR: 69, wantB: 117},
		{name: "favero secox 96", x: 137, y: 2988, w: 496, h: 639, dpi: 96, wantL: 5, wantT: 113, wantR: 23, wantB: 136},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			left, top, right, bottom := CadMapaWMFPlayRectPx(tc.x, tc.y, tc.w, tc.h, tc.dpi, tc.dpi)
			if left != tc.wantL || top != tc.wantT || right != tc.wantR || bottom != tc.wantB {
				t.Fatalf("play rect=(%d,%d,%d,%d), want (%d,%d,%d,%d)", left, top, right, bottom, tc.wantL, tc.wantT, tc.wantR, tc.wantB)
			}
		})
	}
}
