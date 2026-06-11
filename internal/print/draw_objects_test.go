//go:build windows

package print

import (
	"testing"

	"masterprint-native/internal/model"
)

func TestDrawPrintObjectsReturnsWMFError(t *testing.T) {
	lp := &LabelPrinter{}
	err := lp.drawPrintObjects(model.LayoutDefinition{}, 0, 0, []model.PrintObject{{Type: "image", WMF: model.WMFSymbol{FileOffset: 0x99}}})
	if err == nil {
		t.Fatal("expected WMF error")
	}
}

func TestLandscapePrintObjectsTransformsRectangles(t *testing.T) {
	tr, ok := NewLandscapeCellTransform(lnt2LandscapeLayout())
	if !ok {
		t.Fatal("expected landscape transform")
	}
	objects := []model.PrintObject{
		{Type: "text", Text: model.TextElement{XMM: 20.56, YMM: 3.53, WidthMM: 36.37, HeightMM: 1.61, Text: "72% ALGODAO"}},
		{Type: "image", WMF: model.WMFSymbol{XMM: 5.29, YMM: 26.96, WidthMM: 4.45, HeightMM: 4.45}},
		{Type: "rect", Shape: model.ShapeElement{XMM: 1, YMM: 2, WidthMM: 3, HeightMM: 4}},
	}
	got := landscapePrintObjects(tr, objects)
	if !nearMM(got[0].Text.XMM, 3.53) || !nearMM(got[0].Text.YMM, 20.56) || !nearMM(got[0].Text.WidthMM, 1.61) || !nearMM(got[0].Text.HeightMM, 36.37) {
		t.Fatalf("text transform=%#v", got[0].Text)
	}
	if !nearMM(got[1].WMF.XMM, 26.96) || !nearMM(got[1].WMF.YMM, 5.29) || !nearMM(got[1].WMF.WidthMM, 4.45) || !nearMM(got[1].WMF.HeightMM, 4.45) {
		t.Fatalf("wmf transform=%#v", got[1].WMF)
	}
	if !nearMM(got[2].Shape.XMM, 2) || !nearMM(got[2].Shape.YMM, 1) || !nearMM(got[2].Shape.WidthMM, 4) || !nearMM(got[2].Shape.HeightMM, 3) {
		t.Fatalf("shape transform=%#v", got[2].Shape)
	}
	if objects[0].Text.XMM != 20.56 {
		t.Fatalf("transform mutated input: %#v", objects[0].Text)
	}
}
