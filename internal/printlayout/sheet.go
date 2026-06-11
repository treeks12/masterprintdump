package printlayout

import "masterprint-native/internal/model"

type Cell struct {
	Col      int
	Row      int
	XMM      float64
	YMM      float64
	WidthMM  float64
	HeightMM float64
}

type Sheet struct {
	PageWidthMM  float64
	PageHeightMM float64
	MaxLabels    int
	Overrides    map[int]int
}

func SheetForPage(layout model.LayoutDefinition, pageWidthMM, pageHeightMM float64, page model.PrintPage) Sheet {
	if layout.Landscape == 1 && pageWidthMM < pageHeightMM {
		pageWidthMM, pageHeightMM = pageHeightMM, pageWidthMM
	}
	return Sheet{
		PageWidthMM:  pageWidthMM,
		PageHeightMM: pageHeightMM,
		MaxLabels:    page.LabelsPerPage,
		Overrides:    page.Overrides,
	}
}

func Cells(layout model.LayoutDefinition, sheet Sheet) []Cell {
	cols := layout.NumCol
	if cols <= 0 {
		cols = 1
	}
	rows := layout.NumRow
	if rows <= 0 {
		rows = autoRows(layout, sheet)
	}
	if rows <= 0 {
		rows = 1
	}

	var cells []Cell
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			if !allowedByOverride(col, row, sheet.Overrides) {
				continue
			}
			cells = append(cells, Cell{
				Col:      col,
				Row:      row,
				XMM:      layout.MarginLeft + float64(col)*(layout.WidthMM+layout.SpacingCol),
				YMM:      layout.MarginTop + float64(row)*(layout.HeightMM+layout.SpacingRow),
				WidthMM:  layout.WidthMM,
				HeightMM: layout.HeightMM,
			})
			if sheet.MaxLabels > 0 && len(cells) >= sheet.MaxLabels {
				return cells
			}
		}
	}
	return cells
}

func PageCount(totalLabels, labelsPerPage int) int {
	if labelsPerPage <= 0 {
		return 0
	}
	if totalLabels <= 0 {
		return 1
	}
	return (totalLabels + labelsPerPage - 1) / labelsPerPage
}

func CellsForPage(cells []Cell, totalLabels, pageIndex int) []Cell {
	if len(cells) == 0 || pageIndex < 0 {
		return nil
	}
	if totalLabels <= 0 {
		if pageIndex == 0 {
			return cells
		}
		return nil
	}
	start := pageIndex * len(cells)
	if start >= totalLabels {
		return nil
	}
	end := start + len(cells)
	if end > totalLabels {
		end = totalLabels
	}
	return cells[:end-start]
}

func autoRows(layout model.LayoutDefinition, sheet Sheet) int {
	pageH := sheet.PageHeightMM
	if pageH <= 0 || layout.HeightMM <= 0 {
		return 1
	}
	rows := 0
	step := layout.HeightMM + layout.SpacingRow
	if step <= 0 {
		step = layout.HeightMM
	}
	for float64(rows)*step+layout.MarginTop+layout.HeightMM <= pageH+0.0001 {
		rows++
	}
	return rows
}

func allowedByOverride(col, row int, overrides map[int]int) bool {
	if len(overrides) == 0 {
		return true
	}
	rowCount, ok := overrides[col]
	return ok && row < rowCount
}
