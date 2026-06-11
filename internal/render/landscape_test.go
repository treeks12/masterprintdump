package render

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"masterprint-native/internal/etq"
	"masterprint-native/internal/model"
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

func nearMM(got, want float64) bool {
	return math.Abs(got-want) < 0.02
}

func TestLandscapeDesignSizeSwapsLNT2Aspect(t *testing.T) {
	layout := lnt2LandscapeLayout()
	w, h := LandscapeDesignSize(layout)
	if !nearMM(w, 55.5) || !nearMM(h, 25) {
		t.Fatalf("design size=(%.2f,%.2f) want (55.5,25)", w, h)
	}
}

func TestPhysicalCellSizeKeepsRawLNT2Dimensions(t *testing.T) {
	layout := lnt2LandscapeLayout()
	w, h := PhysicalCellSize(layout)
	if !nearMM(w, 25) || !nearMM(h, 55.5) {
		t.Fatalf("cell size=(%.2f,%.2f) want (25,55.5)", w, h)
	}
}

func TestMapDesignRectPortraitIdentity(t *testing.T) {
	layout := model.LayoutDefinition{WidthMM: 70, HeightMM: 37, Landscape: 0}
	design := RectMM{X: 10, Y: 20, Width: 30, Height: 4}
	got := MapDesignRectToPhysicalCell(layout, design)
	if got != design {
		t.Fatalf("portrait map changed rect: %#v", got)
	}
}

func TestMapDesignRectCanelado72AlgodaoGolden(t *testing.T) {
	layout := lnt2LandscapeLayout()
	design := RectMM{X: 20.56, Y: 3.53, Width: 36.37, Height: 1.61}
	got := MapDesignRectToPhysicalCell(layout, design)
	want := RectMM{X: 3.53, Y: 20.56, Width: 1.61, Height: 36.37}
	if !nearMM(got.X, want.X) || !nearMM(got.Y, want.Y) || !nearMM(got.Width, want.Width) || !nearMM(got.Height, want.Height) {
		t.Fatalf("72%% ALGODÃO phys=%#v want %#v", got, want)
	}
}

func TestMapDesignRectCaneladoCloroxGolden(t *testing.T) {
	layout := lnt2LandscapeLayout()
	design := RectMM{X: 5.29, Y: 26.96, Width: 4.45, Height: 4.45}
	got := MapDesignRectToPhysicalCell(layout, design)
	want := RectMM{X: 26.96, Y: 5.29, Width: 4.45, Height: 4.45}
	if !nearMM(got.X, want.X) || !nearMM(got.Y, want.Y) || !nearMM(got.Width, want.Width) || !nearMM(got.Height, want.Height) {
		t.Fatalf("clorox phys=%#v want %#v", got, want)
	}
	cellW, _ := PhysicalCellSize(layout)
	if got.X <= cellW+0.02 {
		t.Fatalf("expected documented phys_x overflow > %.2f, got %.2f", cellW, got.X)
	}
}

func TestLandscapeDesignExtentAuditCanelado(t *testing.T) {
	doc := loadLandscapeETQ(t, "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	minX, minY, maxX, maxY := designExtents(doc)
	if !nearMM(maxX, 56.93) {
		t.Fatalf("Canelado max design X=%.2f want ~56.93 (confirms HeightMM-wide design space)", maxX)
	}
	if minX > 2 || minY > 3 {
		t.Fatalf("unexpected Canelado min design extent (%.2f,%.2f)", minX, minY)
	}
	if maxY < 30 {
		t.Fatalf("Canelado max design Y=%.2f; exceeds swapped design height 25mm", maxY)
	}
	designW, designH := LandscapeDesignSize(lnt2LandscapeLayout())
	if !nearMM(designW, 55.5) || !nearMM(designH, 25) {
		t.Fatalf("design size=(%.2f,%.2f)", designW, designH)
	}
	if maxX > designW+2 {
		t.Fatalf("design X extent %.2f far beyond design width %.2f", maxX, designW)
	}
}

func TestLandscapeDesignExtentAuditADAR(t *testing.T) {
	doc := loadLandscapeETQ(t, "ADAR SOFA CANELADO.ETQ")
	_, _, maxX, maxY := designExtents(doc)
	if maxX < 45 || maxX > 55 {
		t.Fatalf("ADAR max design X=%.2f want ~52-55 (HeightMM-wide space)", maxX)
	}
	if maxY < 30 {
		t.Fatalf("ADAR max design Y=%.2f; exceeds swapped design height 25mm", maxY)
	}
}

func TestLandscapePhysicalFitAuditKnownOverflows(t *testing.T) {
	layout := lnt2LandscapeLayout()
	cellW, cellH := PhysicalCellSize(layout)

	type caseSpec struct {
		name   string
		design RectMM
	}
	cases := []caseSpec{
		{name: "72% ALGODÃO", design: RectMM{X: 20.56, Y: 3.53, Width: 36.37, Height: 1.61}},
		{name: "clorox", design: RectMM{X: 5.29, Y: 26.96, Width: 4.45, Height: 4.45}},
		{name: "seco-w", design: RectMM{X: 1.68, Y: 31.73, Width: 3.93, Height: 4.26}},
	}
	overflows := 0
	for _, tc := range cases {
		phys := MapDesignRectToPhysicalCell(layout, tc.design)
		if phys.FitsPhysicalCell(cellW, cellH) {
			continue
		}
		overflows++
		t.Logf("%s design=%#v phys=%#v exceeds cell %.1fx%.1f", tc.name, tc.design, phys, cellW, cellH)
	}
	if overflows == 0 {
		t.Fatal("expected documented fit overflows for landscape transpose formula")
	}
}

func loadLandscapeETQ(t *testing.T, name string) *etq.ETQFile {
	t.Helper()
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc, err := etq.ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func designExtents(doc *etq.ETQFile) (minX, minY, maxX, maxY float64) {
	minX, minY = 1e9, 1e9
	for _, el := range doc.TextElements {
		minX, minY, maxX, maxY = bumpExtents(minX, minY, maxX, maxY, el.XMM, el.YMM, el.WidthMM, el.HeightMM)
	}
	for _, el := range doc.WMFElements {
		minX, minY, maxX, maxY = bumpExtents(minX, minY, maxX, maxY, el.XMM, el.YMM, el.WidthMM, el.HeightMM)
	}
	return minX, minY, maxX, maxY
}

func bumpExtents(minX, minY, maxX, maxY, x, y, w, h float64) (float64, float64, float64, float64) {
	if x < minX {
		minX = x
	}
	if y < minY {
		minY = y
	}
	if x+w > maxX {
		maxX = x + w
	}
	if y+h > maxY {
		maxY = y + h
	}
	return minX, minY, maxX, maxY
}
