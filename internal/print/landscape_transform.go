package print

import (
	"masterprint-native/internal/model"
	"masterprint-native/internal/render"
)

// LandscapeCellTransform maps ETQ design coordinates into a physical print cell
// for landscape layouts. Canvas design bounds stay in swapped-aspect design space
// (CadMapa FUN_00464b68); printlayout grid cells keep raw INF WidthMM/HeightMM.
//
// Proven placement math (FUN_00488aec / FUN_00484f7c effective result):
//
//	phys_x_mm = design_y
//	phys_y_mm = design_x
//	px_cell   = round(phys_x_mm * dpi / 25.4)
//	py_cell   = round(phys_y_mm * dpi / 25.4)
//
// Text glyph rotation and WMF play orientation under per-cell GDI mapping are
// not proven here; see LandscapePrintTransformBlockers.
type LandscapeCellTransform struct {
	Layout    model.LayoutDefinition
	DesignWMM float64
	DesignHMM float64
	CellWMM   float64
	CellHMM   float64
}

// LandscapePrintTransformBlockers documents why landscape print wiring is deferred.
var LandscapePrintTransformBlockers = []string{
	"FUN_00484f7c / FUN_00488aec SetWindowOrgEx/SetViewportExtEx parameterization is not decompiled; only the effective axis swap is proven.",
	"Per-cell MM_ANISOTROPIC scale+swap maps transposed pixel positions but does not rotate text along design +X; CreateFont escapement would invent behavior.",
	"WMF PlayEnhMetaFile destination rects are device-space; interaction with a landscape world transform is unverified (EnumEnhMetaFile audit deferred).",
	"RTF EM_FORMATRANGE resets MM_TEXT and uses twips rects derived from device pixels; landscape cell scope is unverified.",
	"MM_ANISOTROPIC two-step MulDiv can drift 1px from single-step round((cell+phys)*dpi/25.4) at nonzero sheet offsets.",
}

// NewLandscapeCellTransform returns a transform for landscape layouts.
func NewLandscapeCellTransform(layout model.LayoutDefinition) (LandscapeCellTransform, bool) {
	if layout.Landscape != 1 {
		return LandscapeCellTransform{}, false
	}
	designW, designH := render.LandscapeDesignSize(layout)
	cellW, cellH := render.PhysicalCellSize(layout)
	return LandscapeCellTransform{
		Layout:    layout,
		DesignWMM: designW,
		DesignHMM: designH,
		CellWMM:   cellW,
		CellHMM:   cellH,
	}, true
}

// MapDesignPointToCellMM maps a design-space point to millimeters inside the cell.
func (t LandscapeCellTransform) MapDesignPointToCellMM(designX, designY float64) (cellXMM, cellYMM float64) {
	r := render.MapDesignRectToPhysicalCell(t.Layout, render.RectMM{X: designX, Y: designY})
	return r.X, r.Y
}

// MapDesignPointToCellPx maps a design-space point to device pixels relative to the
// cell origin using px_cell=round(y*s), py_cell=round(x*s).
func (t LandscapeCellTransform) MapDesignPointToCellPx(designX, designY float64, dpiX, dpiY int) (px, py int) {
	cellXMM, cellYMM := t.MapDesignPointToCellMM(designX, designY)
	px = render.Mm100ToPx(render.MmFloatTo100(cellXMM), dpiX)
	py = render.Mm100ToPx(render.MmFloatTo100(cellYMM), dpiY)
	return px, py
}

// MapDesignPointToPagePx maps a design-space point to absolute page device pixels.
func (t LandscapeCellTransform) MapDesignPointToPagePx(sheetCellXMM, sheetCellYMM, designX, designY float64, dpiX, dpiY int) (px, py int) {
	cellXMM, cellYMM := t.MapDesignPointToCellMM(designX, designY)
	px = render.Mm100ToPx(render.MmFloatTo100(sheetCellXMM+cellXMM), dpiX)
	py = render.Mm100ToPx(render.MmFloatTo100(sheetCellYMM+cellYMM), dpiY)
	return px, py
}

// MapDesignRectToCellMM maps a design-space rectangle to millimeters inside the cell.
func (t LandscapeCellTransform) MapDesignRectToCellMM(design render.RectMM) render.RectMM {
	return render.MapDesignRectToPhysicalCell(t.Layout, design)
}

// MapDesignRectToPagePx maps a design-space rectangle to an inclusive device pixel
// bounding box on the page (CadMapa object-rect convention).
func (t LandscapeCellTransform) MapDesignRectToPagePx(sheetCellXMM, sheetCellYMM float64, design render.RectMM, dpiX, dpiY int) (left, top, right, bottom int) {
	phys := t.MapDesignRectToCellMM(design)
	left = render.Mm100ToPx(render.MmFloatTo100(sheetCellXMM+phys.X), dpiX)
	top = render.Mm100ToPx(render.MmFloatTo100(sheetCellYMM+phys.Y), dpiY)
	right = left + render.Mm100ToPx(render.MmFloatTo100(phys.Width), dpiX)
	bottom = top + render.Mm100ToPx(render.MmFloatTo100(phys.Height), dpiY)
	return left, top, right, bottom
}

// DesignRectFitsCell reports whether a design rectangle lies inside the physical cell.
func (t LandscapeCellTransform) DesignRectFitsCell(design render.RectMM) bool {
	phys := t.MapDesignRectToCellMM(design)
	return phys.FitsPhysicalCell(t.CellWMM, t.CellHMM)
}

// GDIAnisotropicMapping holds per-cell MM_ANISOTROPIC parameters derived from
// proven landscape scale factors. SimulateGDIAnisotropicMap models GDI integer
// MulDiv rounding; use tests to compare against MapDesignPointToPagePx.
type GDIAnisotropicMapping struct {
	WindowOrgX, WindowOrgY     int
	WindowExtX, WindowExtY     int
	ViewportOrgX, ViewportOrgY int
	ViewportExtX, ViewportExtY int
}

// LandscapeGDIAnisotropicMapping builds MM_ANISOTROPIC parameters that scale
// design logical units into a cell viewport. Window extents swap design width
// and height so viewport axes align with physical cell width/height.
func (t LandscapeCellTransform) LandscapeGDIAnisotropicMapping(sheetCellXMM, sheetCellYMM float64, dpiX, dpiY int) GDIAnisotropicMapping {
	cellLeft := render.Mm100ToPx(render.MmFloatTo100(sheetCellXMM), dpiX)
	cellTop := render.Mm100ToPx(render.MmFloatTo100(sheetCellYMM), dpiY)
	cellWPx := render.Mm100ToPx(render.MmFloatTo100(t.CellWMM), dpiX)
	cellHPx := render.Mm100ToPx(render.MmFloatTo100(t.CellHMM), dpiY)
	return GDIAnisotropicMapping{
		WindowOrgX:   0,
		WindowOrgY:   0,
		WindowExtX:   render.MmFloatTo100(t.DesignHMM),
		WindowExtY:   render.MmFloatTo100(t.DesignWMM),
		ViewportOrgX: cellLeft,
		ViewportOrgY: cellTop,
		ViewportExtX: cellWPx,
		ViewportExtY: cellHPx,
	}
}

// SimulateGDIAnisotropicMap models GDI MM_ANISOTROPIC device mapping:
//
//	device = viewportOrg + MulDiv(logical - windowOrg, viewportExt, windowExt)
func SimulateGDIAnisotropicMap(m GDIAnisotropicMapping, logicalX, logicalY int) (deviceX, deviceY int) {
	deviceX = m.ViewportOrgX + render.MulDiv(logicalX-m.WindowOrgX, m.ViewportExtX, m.WindowExtX)
	deviceY = m.ViewportOrgY + render.MulDiv(logicalY-m.WindowOrgY, m.ViewportExtY, m.WindowExtY)
	return deviceX, deviceY
}
