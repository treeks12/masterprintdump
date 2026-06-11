package uicheck

import (
	"bufio"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestMapaRiscParity(t *testing.T) {
	main := readRepoFile(t, "main.go")
	objects := extractSliceIDs(t, main, "objectsAndDB")
	wantOrder := []string{"barcode", "image", "ole", "mapaRisc", "fileMan"}
	if !containsSubsequence(objects, wantOrder) {
		t.Fatalf("objects toolbar does not contain %v in order: %#v", wantOrder, objects)
	}
	if !strings.Contains(main, `"mapaRisc": 321`) {
		t.Fatalf("decodedLeft missing mapaRisc=321")
	}
	assertPNGSize(t, filepath.Join(repoRoot(), "assets", "cadmapa_glyphs", "mapaRisc.png"), 16, 16)

	dfm := readFileIfExists(t, filepath.Join(`C:\Users\HB\Projects\paulimaq-reverse\ciabrafe\analysis`, "TLAYOUTDESKTOP_decoded.txt"))
	if dfm != "" && (!strings.Contains(dfm, "TTBXItem btnMapaRisc") || !strings.Contains(dfm, "Left = 321")) {
		t.Fatalf("decoded TLayoutDesktop does not confirm btnMapaRisc at Left=321")
	}
}

func TestToolbarObjectOrderParity(t *testing.T) {
	main := readRepoFile(t, "main.go")
	objects := extractSliceIDs(t, main, "objectsAndDB")
	wantPrefix := []string{"select", "zoom", "line", "roundRect", "rect", "ellipse", "simpleText", "text", "artText", "barcode", "image", "ole", "mapaRisc", "fileMan"}
	if len(objects) < len(wantPrefix) || !reflect.DeepEqual(objects[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("object toolbar prefix\ngot  %#v\nwant %#v", objects[:min(len(objects), len(wantPrefix))], wantPrefix)
	}
}

func TestZoomComboStrings(t *testing.T) {
	main := readRepoFile(t, "main.go")
	levels := extractZoomLevels(t, main)
	labels := make([]string, 0, len(levels)+2)
	for _, level := range levels {
		labels = append(labels, level)
	}
	if strings.Contains(main, `label: "Página Inteira"`) {
		labels = append(labels, "Página Inteira")
	}
	if strings.Contains(main, `label: "Largura da Página"`) {
		labels = append(labels, "Largura da Página")
	}
	want := []string{"25%", "50%", "75%", "100%", "150%", "200%", "300%", "400%", "Página Inteira", "Largura da Página"}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("zoom labels\ngot  %#v\nwant %#v", labels, want)
	}
}

func TestGlyphManifest(t *testing.T) {
	main := readRepoFile(t, "main.go")
	loaded := extractLoadedGlyphIDs(t, main)
	manifest := readManifest(t)
	assetDir := filepath.Join(repoRoot(), "assets", "cadmapa_glyphs")
	for id, file := range manifest {
		if file == "" {
			t.Fatalf("manifest entry %s has empty file", id)
		}
		assertPNGSize(t, filepath.Join(assetDir, file), 16, 16)
	}
	for _, id := range loaded {
		if _, ok := manifest[id]; !ok {
			t.Fatalf("loaded glyph %q missing from manifest", id)
		}
	}
}

func TestTopLevelMenuCaptions(t *testing.T) {
	main := readRepoFile(t, "main.go")
	got := extractMenuHotspotTexts(t, main)
	want := []string{"Arquivo", "Editar", "Objeto", "Banco de Dados", "Opções", "Ajuda ?"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("top menu captions\ngot  %#v\nwant %#v", got, want)
	}
}

func TestArquivoEditarMenuCaptions(t *testing.T) {
	main := readRepoFile(t, "main.go")
	if got, want := extractMenuItemLabels(t, main, "arquivoMenuItems"), []string{"Novo", "Abrir", "Salvar", "Salvar Como...", "Salvar Como Modelo...", "Reabrir", "Exportar...", "Configurar Documento", "Imprimir", "Sair"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Arquivo menu\ngot  %#v\nwant %#v", got, want)
	}
	if got, want := extractMenuItemLabels(t, main, "editarMenuItems"), []string{"Recortar", "Copiar", "Colar", "Apagar"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Editar menu\ngot  %#v\nwant %#v", got, want)
	}
}

func TestObjetoPopupMenuCaptions(t *testing.T) {
	main := readRepoFile(t, "main.go")
	want := []string{"Borda", "Preenchimento", "Fonte", "Enviar Para Trás", "Trazer Para Frente", "A&grupar", "Desagr&upar", "&Alinhar", "Escalonar", "Propriedades"}
	if got := extractMenuItemLabels(t, main, "objetoMenuItems"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Objeto menu\ngot  %#v\nwant %#v", got, want)
	}
	if !strings.Contains(main, "func (a *App) popupMenu1Items() []menuItemAction") || !strings.Contains(main, "return a.objetoMenuItems()") {
		t.Fatalf("PopupMenu1 must reuse Objecto1 captions/order")
	}
	dfm := readFileIfExists(t, filepath.Join(`C:\Users\HB\Projects\paulimaq-reverse\ciabrafe\analysis`, "TLAYOUTDESKTOP_decoded.txt"))
	if dfm == "" {
		return
	}
	for _, caption := range want {
		if !strings.Contains(dfm, `Caption = "`+caption+`"`) {
			t.Fatalf("decoded TLayoutDesktop missing caption %q", caption)
		}
	}
}

func extractSliceIDs(t *testing.T, src, name string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(name) + ` := \[\]ToolbarButton\{(.*?)\n\t\}`)
	m := re.FindStringSubmatch(src)
	if len(m) != 2 {
		t.Fatalf("could not find toolbar slice %s", name)
	}
	idRe := regexp.MustCompile(`ID:\s*"([^"]+)"`)
	matches := idRe.FindAllStringSubmatch(m[1], -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match[1])
	}
	return out
}

func extractMenuHotspotTexts(t *testing.T, src string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?m)items := \[\]MenuHotspot\{([^\n]+)\}`)
	m := re.FindStringSubmatch(src)
	if len(m) != 2 {
		t.Fatalf("could not find menuHotspots literals")
	}
	textRe := regexp.MustCompile(`\{"[^"]+",\s*"([^"]+)"`)
	matches := textRe.FindAllStringSubmatch(m[1], -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match[1])
	}
	return out
}

func extractMenuItemLabels(t *testing.T, src, name string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?s)func \(a \*App\) ` + regexp.QuoteMeta(name) + `\(\) \[\]menuItemAction \{\s*return \[\]menuItemAction\{(.*?)\n\t\}`)
	m := re.FindStringSubmatch(src)
	if len(m) != 2 {
		t.Fatalf("could not find %s", name)
	}
	labelRe := regexp.MustCompile(`label:\s*"([^"]+)"`)
	matches := labelRe.FindAllStringSubmatch(m[1], -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match[1])
	}
	return out
}

func extractZoomLevels(t *testing.T, src string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?s)func toolbarZoomLevels\(\) \[\]float64 \{\s*return \[\]float64\{([^}]*)\}`)
	m := re.FindStringSubmatch(src)
	if len(m) != 2 {
		t.Fatalf("could not find toolbarZoomLevels")
	}
	parts := strings.Split(m[1], ",")
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch part {
		case "0.25":
			out = append(out, "25%")
		case "0.5":
			out = append(out, "50%")
		case "0.75":
			out = append(out, "75%")
		case "1.0":
			out = append(out, "100%")
		case "1.5":
			out = append(out, "150%")
		case "2.0":
			out = append(out, "200%")
		case "3.0":
			out = append(out, "300%")
		case "4.0":
			out = append(out, "400%")
		default:
			out = append(out, part)
		}
	}
	return out
}

func extractLoadedGlyphIDs(t *testing.T, src string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?s)for _, id := range \[\]string\{(.*?)\}`)
	m := re.FindStringSubmatch(src)
	if len(m) != 2 {
		t.Fatalf("could not find loadToolbarGlyphs id list")
	}
	idRe := regexp.MustCompile(`"([^"]+)"`)
	matches := idRe.FindAllStringSubmatch(m[1], -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match[1])
	}
	return out
}

func readManifest(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join(repoRoot(), "assets", "cadmapa_glyphs", "manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	manifest := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) != 4 {
			t.Fatalf("bad manifest row: %q", scanner.Text())
		}
		id := strings.TrimPrefix(strings.TrimSpace(fields[0]), "\ufeff")
		manifest[id] = strings.TrimSpace(fields[3])
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func assertPNGSize(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if cfg.Width != w || cfg.Height != h {
		t.Fatalf("%s size=%dx%d want %dx%d", path, cfg.Width, cfg.Height, w, h)
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	return readRequiredFile(t, filepath.Join(repoRoot(), rel))
}

func readRequiredFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func readFileIfExists(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Clean(filepath.Join("..", ".."))
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func containsSubsequence(haystack, needle []string) bool {
	if len(needle) == 0 {
		return true
	}
	pos := 0
	for _, item := range haystack {
		if item == needle[pos] {
			pos++
			if pos == len(needle) {
				return true
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
