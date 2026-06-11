package print

import (
	"fmt"
	"os"

	"masterprint-native/internal/model"
)

func WMFBytes(sym model.WMFSymbol) ([]byte, error) {
	if len(sym.Embedded) > 0 {
		return append([]byte(nil), sym.Embedded...), nil
	}
	if sym.FilePath == "" {
		return nil, fmt.Errorf("WMF sem dados embutidos nem caminho")
	}
	data, err := os.ReadFile(sym.FilePath)
	if err != nil {
		return nil, fmt.Errorf("leitura WMF: %w", err)
	}
	return data, nil
}
