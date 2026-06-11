package print

import (
	"math"
	"testing"

	"masterprint-native/internal/model"
	"masterprint-native/internal/render"
)

func lnt2LandscapeLayout() model.LayoutDefinition {
	return model.LayoutDefinition{
		Name:       "LNT-2",
		WidthMM:    25,
		HeightMM:   55.5,
		Landscape:  1,
		MarginLeft: 5,
		MarginTop:  11,
		NumCol:     8,
	}
}

func nearPx(got, want int) bool {
	return got == want
}

func TestNewLandscapeCellTransformPortraitRejected(t *testing.T) {
	_, ok := NewLandscapeCellTransform(model.LayoutDefinition{WidthMM: 70, HeightMM: 37, Landscape: 0})
	if ok {
		t.Fatal("portrait layout should not produce landscape transform")
	}
}

func TestLandscapeCellTransformLNT2Dimensions(t *testing.T) {
	tr, ok := NewLandscapeCellTransform(lnt2LandscapeLayout())
	if !ok {
		t.Fatal("expected landscape transform")
	}
	if !nearMM(tr.DesignWMM, 55.5) || !nearMM(tr.DesignHMM, 25) {
		t.Fatalf("design size=(%.2f,%.2f) want (55.5,25)", tr.DesignWMM, tr.DesignHMM)
	}
	if !nearMM(tr.CellWMM, 25) || !nearMM(tr.CellHMM, 55.5) {
		t.Fatalf("cell size=(%.2f,%.2f) want (25,55.5)", tr.CellWMM, tr.CellHMM)
	}
}

func TestMapDesignPointToCellPxLunelliGolden(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	px, py := tr.MapDesignPointToCellPx(20.56, 3.53, 96, 96)
	if !nearPx(px, 13) || !nearPx(py, 78) {
		t.Fatalf("cell px=(%d,%d) want (13,78) from px=round(y*s), py=round(x*s)", px, py)
	}
}

func TestMapDesignPointToPagePxLunelliWithCellOrigin(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	px, py := tr.MapDesignPointToPagePx(5, 11, 20.56, 3.53, 96, 96)
	wantX := render.Mm100ToPx(render.MmFloatTo100(5+3.53), 96)
	wantY := render.Mm100ToPx(render.MmFloatTo100(11+20.56), 96)
	if !nearPx(px, wantX) || !nearPx(py, wantY) {
		t.Fatalf("page px=(%d,%d) want (%d,%d)", px, py, wantX, wantY)
	}
}

func TestMapDesignRectToPagePxLunelliGolden(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	design := render.RectMM{X: 20.56, Y: 3.53, Width: 36.37, Height: 1.61}
	left, top, right, bottom := tr.MapDesignRectToPagePx(0, 0, design, 96, 96)
	if left != 13 || top != 78 || right != 19 || bottom != 215 {
		t.Fatalf("landscape rect=(%d,%d,%d,%d) want (13,78,19,215)", left, top, right, bottom)
	}
}

func TestMapDesignRectToPagePxCloroxTransposed(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	design := render.RectMM{X: 5.29, Y: 26.96, Width: 4.45, Height: 4.45}
	left, top, right, bottom := tr.MapDesignRectToPagePx(0, 0, design, 96, 96)
	want := render.RectMM{X: 26.96, Y: 5.29, Width: 4.45, Height: 4.45}
	wantLeft := render.Mm100ToPx(render.MmFloatTo100(want.X), 96)
	wantTop := render.Mm100ToPx(render.MmFloatTo100(want.Y), 96)
	wantRight := wantLeft + render.Mm100ToPx(render.MmFloatTo100(want.Width), 96)
	wantBottom := wantTop + render.Mm100ToPx(render.MmFloatTo100(want.Height), 96)
	if left != wantLeft || top != wantTop || right != wantRight || bottom != wantBottom {
		t.Fatalf("clorox page rect=(%d,%d,%d,%d) want (%d,%d,%d,%d)", left, top, right, bottom, wantLeft, wantTop, wantRight, wantBottom)
	}
}

func TestDesignRectFitsCellKnownOverflows(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	cases := []struct {
		name   string
		design render.RectMM
		fits   bool
	}{
		{name: "72% ALGODÃO", design: render.RectMM{X: 20.56, Y: 3.53, Width: 36.37, Height: 1.61}, fits: false},
		{name: "clorox", design: render.RectMM{X: 5.29, Y: 26.96, Width: 4.45, Height: 4.45}, fits: false},
		{name: "inside", design: render.RectMM{X: 10, Y: 5, Width: 10, Height: 2}, fits: true},
	}
	for _, tc := range cases {
		got := tr.DesignRectFitsCell(tc.design)
		if got != tc.fits {
			t.Fatalf("%s fit=%v want %v", tc.name, got, tc.fits)
		}
	}
}

func TestLandscapeGDIAnisotropicMappingMatchesTransposedLogicalCoords(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	m := tr.LandscapeGDIAnisotropicMapping(0, 0, 96, 96)

	cases := []struct {
		name    string
		designX float64
		designY float64
		logical render.RectMM
	}{
		{name: "lunelli", designX: 20.56, designY: 3.53, logical: render.RectMM{X: 3.53, Y: 20.56}},
		{name: "clorox", designX: 5.29, designY: 26.96, logical: render.RectMM{X: 26.96, Y: 5.29}},
	}
	for _, tc := range cases {
		wantX, wantY := tr.MapDesignPointToCellPx(tc.designX, tc.designY, 96, 96)
		logicalX := render.MmFloatTo100(tc.logical.X)
		logicalY := render.MmFloatTo100(tc.logical.Y)
		gotX, gotY := SimulateGDIAnisotropicMap(m, logicalX, logicalY)
		if pxDrift(gotX, wantX) > 1 || pxDrift(gotY, wantY) > 1 {
			t.Fatalf("%s GDI map=(%d,%d) want (%d,%d); logical must be (designY,designX) in 0.01mm", tc.name, gotX, gotY, wantX, wantY)
		}
	}
}

func TestLandscapeGDIAnisotropicPageOffsetRoundingDrift(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	m := tr.LandscapeGDIAnisotropicMapping(5, 11, 96, 96)
	wantX, wantY := tr.MapDesignPointToPagePx(5, 11, 20.56, 3.53, 96, 96)
	gotX, gotY := SimulateGDIAnisotropicMap(m, render.MmFloatTo100(3.53), render.MmFloatTo100(20.56))
	if gotX != wantX {
		t.Fatalf("page X GDI map=%d want %d", gotX, wantX)
	}
	if gotY-wantY > 1 || wantY-gotY > 1 {
		t.Fatalf("page Y GDI map=%d want %d; two-step MulDiv can drift 1px from single-step mm rounding", gotY, wantY)
	}
}

func TestLandscapeGDIAnisotropicMappingDoesNotUseDesignXYDirectly(t *testing.T) {
	tr, _ := NewLandscapeCellTransform(lnt2LandscapeLayout())
	m := tr.LandscapeGDIAnisotropicMapping(0, 0, 96, 96)
	wantX, wantY := tr.MapDesignPointToPagePx(0, 0, 20.56, 3.53, 96, 96)
	gotX, gotY := SimulateGDIAnisotropicMap(m, render.MmFloatTo100(20.56), render.MmFloatTo100(3.53))
	if gotX == wantX && gotY == wantY {
		t.Fatal("unswapped logical (designX,designY) should not match proven landscape placement")
	}
}

func TestLandscapePrintTransformBlockersDocumented(t *testing.T) {
	if len(LandscapePrintTransformBlockers) < 3 {
		t.Fatal("expected documented blockers for deferred print wiring")
	}
}

func nearMM(got, want float64) bool {
	return math.Abs(got-want) < 0.02
}

func pxDrift(got, want int) int {
	d := got - want
	if d < 0 {
		return -d
	}
	return d
}
