package render

import "masterprint-native/internal/model"

// RectMM is a rectangle in millimeters.
type RectMM struct {
	X, Y, Width, Height float64
}

// LandscapeDesignSize returns the swapped-aspect design canvas size used by
// CadMapa FUN_00464b68 for landscape layouts. ETQ object coordinates live in
// this space; physical sheet cells keep raw INF WidthMM/HeightMM.
func LandscapeDesignSize(layout model.LayoutDefinition) (widthMM, heightMM float64) {
	if layout.Landscape != 1 {
		return layout.WidthMM, layout.HeightMM
	}
	return layout.HeightMM, layout.WidthMM
}

// PhysicalCellSize returns the raw INF cell dimensions. printlayout.Cells uses
// these values unchanged for landscape and portrait.
func PhysicalCellSize(layout model.LayoutDefinition) (widthMM, heightMM float64) {
	return layout.WidthMM, layout.HeightMM
}

// MapDesignRectToPhysicalCell maps an ETQ design rectangle into physical cell
// coordinates. Portrait layouts use identity. Landscape layouts apply the
// 90-degree axis swap implemented by CadMapa FUN_00488aec / FUN_00484f7c after
// FUN_00464b68 letterboxes the swapped design aspect (HeightMM x WidthMM).
//
// Explicit math equivalent (GDI Y-down, 90° clockwise):
//
//	phys_x     = design_y
//	phys_y     = design_x
//	phys_width  = design_height
//	phys_height = design_width
func MapDesignRectToPhysicalCell(layout model.LayoutDefinition, design RectMM) RectMM {
	if layout.Landscape != 1 {
		return design
	}
	return RectMM{
		X:      design.Y,
		Y:      design.X,
		Width:  design.Height,
		Height: design.Width,
	}
}

// FitsPhysicalCell reports whether rect lies inside a physical cell of the given size.
func (r RectMM) FitsPhysicalCell(cellW, cellH float64) bool {
	return r.X >= 0 && r.Y >= 0 && r.X+r.Width <= cellW+0.001 && r.Y+r.Height <= cellH+0.001
}
