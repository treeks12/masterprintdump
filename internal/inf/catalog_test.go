package inf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCatalogsRejectsWindowsINF(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "driver.inf"), []byte("[Version]\nSignature=$Windows NT$\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "layout.ini"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	line := fixedRecordForCatalogTest([]fieldValue{{name: "LNT-2", width: 35}, {name: "8", width: 11}, {name: "25,00", width: 22}, {name: "55,50", width: 22}, {name: "5,00", width: 22}, {name: "11,00", width: 22}, {name: "0,00", width: 22}, {name: "0,00", width: 22}, {name: "1", width: 2}, {name: "0", width: 2}, {name: "0", width: 1}})
	if err := os.WriteFile(filepath.Join(dir, "etiqueta.inf"), []byte("0000Etiq. para Composicoes\n"+line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	catalogs, err := LoadCatalogs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogs) != 1 || len(catalogs["etiqueta"]) != 1 {
		t.Fatalf("catalogs=%#v", catalogs)
	}
	if _, ok := catalogs["driver"]; ok {
		t.Fatalf("Windows INF should not be loaded: %#v", catalogs)
	}
}

func TestLoadInstalledCatalogs(t *testing.T) {
	dir := `C:\Program Files (x86)\paulimaq`
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("paulimaq install not found: %v", err)
	}
	catalogs, err := LoadCatalogs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogs) != 22 {
		t.Fatalf("catalogs=%d want 22: %#v", len(catalogs), catalogs)
	}
	for key, want := range map[string]int{
		"etiqueta":   11,
		"tag":        25,
		"tag2":       22,
		"tag3":       3,
		"joia":       13,
		"sapato":     2,
		"invite":     11,
		"fixbands":   5,
		"etiqueta_m": 9,
		"etiqueta_r": 9,
	} {
		if got := len(catalogs[key]); got != want {
			t.Fatalf("catalog %s count=%d want %d", key, got, want)
		}
	}
}

func fixedRecordForCatalogTest(fields []fieldValue) string {
	return fixedRecord(fields)
}
