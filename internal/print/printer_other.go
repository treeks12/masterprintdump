//go:build !windows

package print

import (
	"fmt"

	"masterprint-native/internal/model"
	"masterprint-native/internal/printlayout"
)

type LabelPrinter struct{}

func MmToPixels(mm float64, dpi int) int {
	return int(mm * float64(dpi) / 25.4)
}

func EnumPrinters() ([]string, error) {
	return nil, fmt.Errorf("nao suportado neste SO")
}

func NewLabelPrinter(printerName string, landscape int) (*LabelPrinter, error) {
	return nil, fmt.Errorf("nao suportado neste SO")
}

func (lp *LabelPrinter) Close() {}

func (lp *LabelPrinter) PrintLabel(docName string, layout model.LayoutDefinition, sheet printlayout.Sheet, copies int, label model.Label) error {
	return fmt.Errorf("nao suportado neste SO")
}

func WMFToPNG(wmfPath string, widthMM, heightMM float64) (string, error) {
	return "", fmt.Errorf("nao suportado neste SO")
}
