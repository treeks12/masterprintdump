package inf

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"masterprint-native/internal/model"
)

func LoadCatalogs(dir string) (map[string][]model.LayoutDefinition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string][]model.LayoutDefinition{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".inf") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if !isPaulimaqLayoutINF(path) {
			continue
		}
		layouts, err := ParseFile(path)
		if err != nil || !hasUsableLayout(layouts) {
			continue
		}
		key := strings.ToLower(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		out[key] = layouts
	}
	return out, nil
}

func isPaulimaqLayoutINF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return false
	}
	for _, b := range hdr {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func hasUsableLayout(layouts []model.LayoutDefinition) bool {
	for _, layout := range layouts {
		if layout.Name != "" && layout.WidthMM > 0 && layout.HeightMM > 0 {
			return true
		}
	}
	return false
}
