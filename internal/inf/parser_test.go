package inf

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseLayoutINILabelRecEtiPag(t *testing.T) {
	path := writeTempFile(t, "layout.ini", []byte(`[LabelRecEtiPag]

FileCode=0
Nome=35
NumCol=11
TamHoriz=22
TamVert=22
MargEsq=22
MargSup=22
EspEntrCol=22
EspEntrLin=22
Paisa=2
Matri=2
Pagina=1
`))
	recs, err := ParseLayoutINI(path)
	if err != nil {
		t.Fatal(err)
	}
	rec, ok := recs["LabelRecEtiPag"]
	if !ok {
		t.Fatalf("missing LabelRecEtiPag in %#v", recs)
	}
	if !reflect.DeepEqual(rec.FileCodes, []string{"0"}) {
		t.Fatalf("FileCodes=%#v want [0]", rec.FileCodes)
	}
	if len(rec.Fields) != 11 {
		t.Fatalf("fields=%d want 11", len(rec.Fields))
	}
	if rec.RecordSize() != 183 {
		t.Fatalf("RecordSize=%d want 183", rec.RecordSize())
	}
	if off, ok := rec.FieldOffset("Nome"); !ok || off != 0 || rec.Fields[0].Size != 35 || rec.Fields[0].IsNum {
		t.Fatalf("unexpected Nome field: off=%d ok=%v field=%#v", off, ok, rec.Fields[0])
	}
	if off, ok := rec.FieldOffset("NumCol"); !ok || off != 35 || rec.Fields[1].Size != 11 || !rec.Fields[1].IsNum {
		t.Fatalf("unexpected NumCol field: off=%d ok=%v field=%#v", off, ok, rec.Fields[1])
	}
	if off, ok := rec.FieldOffset("Pagina"); !ok || off != 182 || rec.Fields[10].Size != 1 || !rec.Fields[10].IsNum {
		t.Fatalf("unexpected Pagina field: off=%d ok=%v field=%#v", off, ok, rec.Fields[10])
	}
}

func TestParsePageOverrideComposicoes(t *testing.T) {
	section := string([]byte{'E', 't', 'i', 'q', '.', ' ', 'p', 'a', 'r', 'a', ' ', 'C', 'o', 'm', 'p', 'o', 's', 'i', 0xe7, 0xf5, 'e', 's'})
	content := []byte("[" + section + "]\n0=6\n1=6\n2=5\n3=5\n4=4\n5=5\n6=5\n7=4\n8=4\n")
	path := writeTempFile(t, "pageovrr.ini", content)
	pages, err := ParsePageOverride(path)
	if err != nil {
		t.Fatal(err)
	}
	page, ok := pages[section]
	if !ok {
		t.Fatalf("missing section %q in %#v", section, pages)
	}
	want := map[int]int{0: 6, 1: 6, 2: 5, 3: 5, 4: 4, 5: 5, 6: 5, 7: 4, 8: 4}
	if !reflect.DeepEqual(page.Overrides, want) {
		t.Fatalf("Overrides=%#v want %#v", page.Overrides, want)
	}
	if page.LabelsPerPage != 6 {
		t.Fatalf("LabelsPerPage=%d want 6", page.LabelsPerPage)
	}
}

func TestParseFileLNT2(t *testing.T) {
	line := fmt.Sprintf("%-35s%s\n", "LNT-2", "8 25,00 55,50 05,00 11,00 0,00 0,00 0 0 1")
	path := writeTempFile(t, "etiqueta.inf", []byte("0000Etiq. para Composicoes\n"+line))
	layouts, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) != 1 {
		t.Fatalf("layouts=%d want 1: %#v", len(layouts), layouts)
	}
	l := layouts[0]
	if l.Name != "LNT-2" || l.NumCol != 8 || !nearFloat(l.WidthMM, 25.00) || !nearFloat(l.HeightMM, 55.50) || !nearFloat(l.MarginLeft, 5.00) || !nearFloat(l.MarginTop, 11.00) || !nearFloat(l.SpacingCol, 0.00) || !nearFloat(l.SpacingRow, 0.00) || l.Flip != 0 || l.Rotation != 0 || l.Landscape != 1 {
		t.Fatalf("unexpected LNT-2 layout: %#v", l)
	}
}

func TestParseInstalledEtiquetaInfLNT2(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "etiqueta.inf")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("installed etiqueta.inf not found: %v", err)
	}
	layouts, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) != 11 {
		t.Fatalf("layouts=%d want 11", len(layouts))
	}
	if layouts[2].Name != "LNT-2" || layouts[2].NumCol != 8 || !nearFloat(layouts[2].WidthMM, 25.00) || !nearFloat(layouts[2].HeightMM, 55.50) {
		t.Fatalf("unexpected installed LNT-2 layout: %#v", layouts[2])
	}
}

func TestParseFileWithLayoutNumColMatrix(t *testing.T) {
	dir := t.TempDir()
	writeFileAt(t, filepath.Join(dir, "layout.ini"), []byte(`[LabelRecEtiPag]
FileCode=0
Nome=35
NumCol=11
TamHoriz=22
TamVert=22
MargEsq=22
MargSup=22
EspEntrCol=22
EspEntrLin=22
Paisa=2
Matri=2
Pagina=1
`))
	rec := fixedRecord([]fieldValue{
		{name: "MATRIX", width: 35},
		{name: "3x2", width: 11},
		{name: "25,00", width: 22},
		{name: "55,50", width: 22},
		{name: "05,00", width: 22},
		{name: "11,00", width: 22},
		{name: "0,00", width: 22},
		{name: "0,00", width: 22},
		{name: "1", width: 2},
		{name: "0", width: 2},
		{name: "0", width: 1},
	})
	infPath := filepath.Join(dir, "etiqueta.inf")
	writeFileAt(t, infPath, []byte("0000Etiq. para Composicoes\n"+rec+"\n"))
	layouts, err := ParseFile(infPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) != 1 {
		t.Fatalf("layouts=%d want 1", len(layouts))
	}
	if layouts[0].NumCol != 3 || layouts[0].CopiesPerColumn != 2 || layouts[0].NumRow != 0 || layouts[0].Landscape != 1 {
		t.Fatalf("unexpected matrix layout: %#v", layouts[0])
	}
}

func TestParseFileWithLayoutExtendedTagFields(t *testing.T) {
	dir := t.TempDir()
	writeFileAt(t, filepath.Join(dir, "layout.ini"), []byte(`[LabelRecTag]
FileCode=1,18,20,15,51
Nome=35
NumCol=11
TamHoriz=22
TamVert=22
MargEsq=22
MargSup=22
EspEntrCol=22
EspEntrLin=22
Paisa=2
Matri=2
PosXFuro=22
PosYFuro=22
PosPicote1=22
PosPicote2=22
`))
	rec := fixedRecord([]fieldValue{
		{name: "TAG-FC04", width: 35},
		{name: "7", width: 11},
		{name: "30,00", width: 22},
		{name: "38,00", width: 22},
		{name: "05,00", width: 22},
		{name: "10,00", width: 22},
		{name: "2,00", width: 22},
		{name: "3,00", width: 22},
		{name: "1", width: 2},
		{name: "0", width: 2},
		{name: "15,00", width: 22},
		{name: "8,00", width: 22},
		{name: "33,00", width: 22},
		{name: "36,00", width: 22},
	})
	infPath := filepath.Join(dir, "tag.inf")
	writeFileAt(t, infPath, []byte("0101TAG'S em Folhas e Formularios\n"+rec+"\n"))
	layouts, err := ParseFile(infPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) != 1 {
		t.Fatalf("layouts=%d want 1", len(layouts))
	}
	l := layouts[0]
	if l.TypeCode != "1" || l.Name != "TAG-FC04" || l.Landscape != 1 || !nearFloat(parseFloat(l.Fields["PosXFuro"]), 15.00) || !nearFloat(parseFloat(l.Fields["PosPicote1"]), 33.00) {
		t.Fatalf("unexpected tag layout: %#v", l)
	}
}

func TestParseFileWithLayoutSapatoUsesPrimarySizeFields(t *testing.T) {
	dir := t.TempDir()
	writeFileAt(t, filepath.Join(dir, "layout.ini"), []byte(`[LabelRecSap]
FileCode=5
Nome=35
NumCol=11
TamHoriz1=22
TamVert1=22
TamHoriz2=22
TamVert2=22
MargEsq=22
MargSup=22
EspEntrCol=22
EspEntrLin=22
Paisa=2
Matri=2
`))
	rec := fixedRecord([]fieldValue{
		{name: "CS0210 / LJA 272", width: 35},
		{name: "1", width: 11},
		{name: "145,00", width: 22},
		{name: "12,70", width: 22},
		{name: "45,00", width: 22},
		{name: "12,70", width: 22},
		{name: "11,00", width: 22},
		{name: "13,00", width: 22},
		{name: "0,00", width: 22},
		{name: "0,00", width: 22},
		{name: "0", width: 2},
		{name: "0", width: 2},
	})
	infPath := filepath.Join(dir, "sapato.inf")
	writeFileAt(t, infPath, []byte("0505Etiq. para Calçados\n"+rec+"\n"))
	layouts, err := ParseFile(infPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) != 1 {
		t.Fatalf("layouts=%d want 1", len(layouts))
	}
	l := layouts[0]
	if l.TypeCode != "5" || l.Name != "CS0210 / LJA 272" || !nearFloat(l.WidthMM, 145.00) || !nearFloat(l.HeightMM, 12.70) || !nearFloat(l.MarginLeft, 11.00) || !nearFloat(l.MarginTop, 13.00) || l.Fields["TamHoriz2"] != "45,00" {
		t.Fatalf("unexpected sapato layout: %#v", l)
	}
}

func TestParseInstalledSapatoDimensions(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "sapato.inf")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("installed sapato.inf not found: %v", err)
	}
	layouts, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) == 0 {
		t.Fatalf("expected installed sapato layouts")
	}
	if layouts[0].Name != "CS0210 / LJA 272" || !nearFloat(layouts[0].WidthMM, 145.00) || !nearFloat(layouts[0].HeightMM, 12.70) {
		t.Fatalf("unexpected installed sapato layout: %#v", layouts[0])
	}
}

func TestParseLayoutININumericExtendedFields(t *testing.T) {
	path := writeTempFile(t, "layout.ini", []byte(`[LabelRecJoia]
FileCode=2
Nome=35
NumCol=11
TamHoriz=22
TamVert=22
MargEsq=22
MargSup=22
EspEntrCol=22
EspEntrLin=22
Paisa=2
Matri=2
TamLigCentro=11
DobraCentral=11
InicioTira=11
AlturaTira=11
`))
	recs, err := ParseLayoutINI(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := recs["LabelRecJoia"]
	for _, name := range []string{"TamLigCentro", "DobraCentral", "InicioTira", "AlturaTira"} {
		found := false
		for _, field := range rec.Fields {
			if field.Name == name {
				found = true
				if !field.IsNum {
					t.Fatalf("field %s should be numeric", name)
				}
			}
		}
		if !found {
			t.Fatalf("missing field %s", name)
		}
	}
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFileAt(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

type fieldValue struct {
	name  string
	width int
}

func fixedRecord(fields []fieldValue) string {
	out := ""
	for _, field := range fields {
		v := field.name
		if len(v) > field.width {
			v = v[:field.width]
		}
		out += fmt.Sprintf("%-*s", field.width, v)
	}
	return out
}

func nearFloat(got, want float64) bool {
	return math.Abs(got-want) < 0.001
}
