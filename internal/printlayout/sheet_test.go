package printlayout

import (
	"math"
	"testing"

	"masterprint-native/internal/model"
)

func TestLNT2CellOriginsRowMajor(t *testing.T) {
	cells := Cells(lnt2Layout(), Sheet{PageWidthMM: 297, PageHeightMM: 210})
	if len(cells) != 24 {
		t.Fatalf("cells=%d want 24", len(cells))
	}
	want := []Cell{
		{Col: 0, Row: 0, XMM: 5.00, YMM: 11.00, WidthMM: 25, HeightMM: 55.5},
		{Col: 1, Row: 0, XMM: 30.00, YMM: 11.00, WidthMM: 25, HeightMM: 55.5},
		{Col: 7, Row: 0, XMM: 180.00, YMM: 11.00, WidthMM: 25, HeightMM: 55.5},
		{Col: 0, Row: 1, XMM: 5.00, YMM: 66.50, WidthMM: 25, HeightMM: 55.5},
		{Col: 0, Row: 2, XMM: 5.00, YMM: 122.00, WidthMM: 25, HeightMM: 55.5},
	}
	idx := []int{0, 1, 7, 8, 16}
	for i, wantCell := range want {
		assertCell(t, cells[idx[i]], wantCell)
	}
}

func TestLNT2MaxLabelsStopsInRowMajorOrder(t *testing.T) {
	cells := Cells(lnt2Layout(), Sheet{PageWidthMM: 297, PageHeightMM: 210, MaxLabels: 6})
	if len(cells) != 6 {
		t.Fatalf("cells=%d want 6", len(cells))
	}
	assertCell(t, cells[5], Cell{Col: 5, Row: 0, XMM: 130, YMM: 11, WidthMM: 25, HeightMM: 55.5})
}

func TestLNT2PageOverrideWhitelist(t *testing.T) {
	cells := Cells(lnt2Layout(), Sheet{PageWidthMM: 297, PageHeightMM: 210, Overrides: map[int]int{0: 1, 2: 2}})
	if len(cells) != 3 {
		t.Fatalf("cells=%d want 3: %#v", len(cells), cells)
	}
	want := []Cell{
		{Col: 0, Row: 0, XMM: 5, YMM: 11, WidthMM: 25, HeightMM: 55.5},
		{Col: 2, Row: 0, XMM: 55, YMM: 11, WidthMM: 25, HeightMM: 55.5},
		{Col: 2, Row: 1, XMM: 55, YMM: 66.5, WidthMM: 25, HeightMM: 55.5},
	}
	for i := range want {
		assertCell(t, cells[i], want[i])
	}
}

func TestFixedNumRowDoesNotAutoExtend(t *testing.T) {
	l := lnt2Layout()
	l.NumRow = 2
	cells := Cells(l, Sheet{PageWidthMM: 297, PageHeightMM: 210})
	if len(cells) != 16 {
		t.Fatalf("cells=%d want 16", len(cells))
	}
	assertCell(t, cells[15], Cell{Col: 7, Row: 1, XMM: 180, YMM: 66.5, WidthMM: 25, HeightMM: 55.5})
}

func TestSheetForPageLandscapeSwapsDimensions(t *testing.T) {
	page := model.PrintPage{LabelsPerPage: 6, Overrides: map[int]int{0: 2}}
	sheet := SheetForPage(lnt2Layout(), 210, 297, page)
	if !near(sheet.PageWidthMM, 297) || !near(sheet.PageHeightMM, 210) || sheet.MaxLabels != 6 || sheet.Overrides[0] != 2 {
		t.Fatalf("unexpected sheet: %#v", sheet)
	}
}

func TestSheetForPageLandscapeKeepsAlreadyLandscapeDimensions(t *testing.T) {
	page := model.PrintPage{LabelsPerPage: 6}
	sheet := SheetForPage(lnt2Layout(), 297, 210, page)
	if !near(sheet.PageWidthMM, 297) || !near(sheet.PageHeightMM, 210) || sheet.MaxLabels != 6 {
		t.Fatalf("unexpected sheet: %#v", sheet)
	}
}

func TestSheetForPagePortraitKeepsDimensions(t *testing.T) {
	l := lnt2Layout()
	l.Landscape = 0
	sheet := SheetForPage(l, 210, 297, model.PrintPage{})
	if !near(sheet.PageWidthMM, 210) || !near(sheet.PageHeightMM, 297) {
		t.Fatalf("unexpected portrait sheet: %#v", sheet)
	}
}

func TestPageCount(t *testing.T) {
	for _, tc := range []struct {
		total   int
		perPage int
		want    int
	}{
		{total: 0, perPage: 6, want: 1},
		{total: 5, perPage: 6, want: 1},
		{total: 6, perPage: 6, want: 1},
		{total: 7, perPage: 6, want: 2},
		{total: 12, perPage: 6, want: 2},
		{total: 13, perPage: 6, want: 3},
	} {
		if got := PageCount(tc.total, tc.perPage); got != tc.want {
			t.Fatalf("PageCount(%d,%d)=%d want %d", tc.total, tc.perPage, got, tc.want)
		}
	}
}

func TestCellsForPage(t *testing.T) {
	cells := Cells(lnt2Layout(), Sheet{PageWidthMM: 297, PageHeightMM: 210, MaxLabels: 6})
	if len(cells) != 6 {
		t.Fatalf("fixture cells=%d want 6", len(cells))
	}
	if got := CellsForPage(cells, 7, 0); len(got) != 6 {
		t.Fatalf("page 0 cells=%d want 6", len(got))
	}
	if got := CellsForPage(cells, 7, 1); len(got) != 1 || got[0].Col != 0 || got[0].Row != 0 {
		t.Fatalf("page 1 cells=%#v want first single cell", got)
	}
	if got := CellsForPage(cells, 0, 0); len(got) != 6 {
		t.Fatalf("default page cells=%d want 6", len(got))
	}
	if got := CellsForPage(cells, 0, 1); len(got) != 0 {
		t.Fatalf("default second page cells=%d want 0", len(got))
	}
}

func lnt2Layout() model.LayoutDefinition {
	return model.LayoutDefinition{
		Name:       "LNT-2",
		NumCol:     8,
		WidthMM:    25.00,
		HeightMM:   55.50,
		MarginLeft: 5.00,
		MarginTop:  11.00,
		SpacingCol: 0.00,
		SpacingRow: 0.00,
		Landscape:  1,
	}
}

func assertCell(t *testing.T, got, want Cell) {
	t.Helper()
	if got.Col != want.Col || got.Row != want.Row || !near(got.XMM, want.XMM) || !near(got.YMM, want.YMM) || !near(got.WidthMM, want.WidthMM) || !near(got.HeightMM, want.HeightMM) {
		t.Fatalf("cell=%#v want %#v", got, want)
	}
}

func near(got, want float64) bool {
	return math.Abs(got-want) < 0.001
}
