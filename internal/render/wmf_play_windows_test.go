//go:build windows

package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCadMapaEnhMetaFileFromInstalledWMF(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "CLIPART", "Símbolos", "clorox.wmf")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("clipart not installed: %v", err)
	}
	_, _, _, _, metaData, err := ParseWMFBounds(data)
	if err != nil {
		t.Fatal(err)
	}
	hEMF, err := cadMapaEnhMetaFileFromWMF(metaData)
	if err != nil {
		t.Fatal(err)
	}
	if hEMF == 0 {
		t.Fatal("empty enhanced metafile handle")
	}
	procDeleteEnhMetaFileR.Call(hEMF)
}
