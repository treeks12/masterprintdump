package etq

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type clipartFile struct {
	name string
	size int
	hash [32]byte
}

func TestClipartCatalogUniqueBodyHashes(t *testing.T) {
	catalog := readInstalledClipartCatalog(t)
	if len(catalog) != 49 {
		t.Fatalf("clipart count=%d want 49", len(catalog))
	}
	byHash := map[[32]byte][]string{}
	for _, file := range catalog {
		byHash[file.hash] = append(byHash[file.hash], file.name)
	}
	if len(byHash) != 49 {
		t.Fatalf("unique body hashes=%d want 49", len(byHash))
	}
	for hash, names := range byHash {
		if len(names) != 1 {
			t.Fatalf("hash collision %x: %#v", hash, names)
		}
	}
	assertClipartHashPrefix(t, catalog, "cloro.wmf", "7d1d692fcee32f5c")
	assertClipartHashPrefix(t, catalog, "secah.wmf", "9230d05c03b130c9")
	assertClipartHashPrefix(t, catalog, "clorox.wmf", "7705481c7c0e30aa")
}

func TestEmbeddedWMFBySizeBijectionWithDisk(t *testing.T) {
	catalog := readInstalledClipartCatalog(t)
	if len(embeddedWMFBySize) != 49 {
		t.Fatalf("embeddedWMFBySize entries=%d want 49", len(embeddedWMFBySize))
	}
	diskBySize := map[int]string{}
	for _, file := range catalog {
		if prior, ok := diskBySize[file.size]; ok {
			t.Fatalf("duplicate disk WMF size %d: %s and %s", file.size, prior, file.name)
		}
		diskBySize[file.size] = file.name
	}
	for size, name := range embeddedWMFBySize {
		if got := diskBySize[size]; got != name {
			t.Fatalf("size map mismatch for %d: got disk %q want map %q", size, got, name)
		}
	}
	for size, name := range diskBySize {
		if got := embeddedWMFBySize[size]; got != name {
			t.Fatalf("disk WMF missing from size map: size=%d name=%q got=%q", size, name, got)
		}
	}
}

func TestLunelliEmbeddedBlobsMatchClipartSHA(t *testing.T) {
	etqPath := filepath.Join(installedPaulimaqRoot(), "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(etqPath)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	clipDir := installedClipartDir(t)
	doc, err := ParseETQ(etqPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		off  int
		size int
		name string
	}{
		{off: 0x0111, size: 728, name: "clorox.wmf"},
		{off: 0x046d, size: 3166, name: "tamborx.wmf"},
		{off: 0x114f, size: 632, name: "secah.wmf"},
		{off: 0x17f3, size: 8558, name: "lav-30.wmf"},
		{off: 0x39e5, size: 5322, name: "ferro-.wmf"},
		{off: 0x4f33, size: 3262, name: "seco-w.wmf"},
	}
	if len(doc.WMFElements) != len(want) {
		t.Fatalf("WMF count=%d want %d: %#v", len(doc.WMFElements), len(want), doc.WMFElements)
	}
	for _, tc := range want {
		t.Run(tc.name, func(t *testing.T) {
			blob, ok := wmfBlobAt(data, tc.off)
			if !ok {
				t.Fatalf("missing WMF blob at %#x", tc.off)
			}
			if len(blob) != tc.size {
				t.Fatalf("blob size=%d want %d", len(blob), tc.size)
			}
			disk, err := os.ReadFile(filepath.Join(clipDir, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			if len(disk) != len(blob) {
				t.Fatalf("disk/blob size mismatch: %d/%d", len(disk), len(blob))
			}
			if got, want := wmfBodyHash(blob), wmfBodyHash(disk); got != want {
				t.Fatalf("body SHA mismatch: got %x want %x", got, want)
			}
			sym := findWMFByOffset(t, doc, tc.off)
			if sym.FilePath != tc.name {
				t.Fatalf("resolved name=%q want %q", sym.FilePath, tc.name)
			}
		})
	}
}

func TestWMFResolverMissingClipartDirFallsBackToSize(t *testing.T) {
	etqPath := filepath.Join(installedPaulimaqRoot(), "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(etqPath)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	tmp := t.TempDir()
	arquivos := filepath.Join(tmp, "ARQUIVOS")
	if err := os.MkdirAll(arquivos, 0755); err != nil {
		t.Fatal(err)
	}
	tmpETQ := filepath.Join(arquivos, "sample.ETQ")
	if err := os.WriteFile(tmpETQ, data, 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := ParseETQ(tmpETQ)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, sym := range doc.WMFElements {
		seen[sym.FilePath] = true
	}
	for _, name := range []string{"clorox.wmf", "tamborx.wmf", "secah.wmf", "lav-30.wmf", "ferro-.wmf", "seco-w.wmf"} {
		if !seen[name] {
			t.Fatalf("missing %s through size fallback: %#v", name, doc.WMFElements)
		}
	}
}

func readInstalledClipartCatalog(t *testing.T) []clipartFile {
	t.Helper()
	dir := installedClipartDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("clipart catalog not installed: %v", err)
	}
	var out []clipartFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".wmf") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(data) < 22 {
			t.Fatalf("clipart %s too short: %d bytes", entry.Name(), len(data))
		}
		out = append(out, clipartFile{name: entry.Name(), size: len(data), hash: wmfBodyHash(data)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func installedClipartDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(installedPaulimaqRoot(), "CLIPART", "Símbolos")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Skipf("clipart catalog not installed at %s", dir)
	}
	return dir
}

func installedPaulimaqRoot() string {
	return `C:\Program Files (x86)\paulimaq`
}

func wmfBodyHash(data []byte) [32]byte {
	return sha256.Sum256(data[22:])
}

func assertClipartHashPrefix(t *testing.T, catalog []clipartFile, name, prefix string) {
	t.Helper()
	for _, file := range catalog {
		if file.name == name {
			got := strings.ToLower(hexPrefix(file.hash, len(prefix)))
			if got != prefix {
				t.Fatalf("%s hash prefix=%s want %s", name, got, prefix)
			}
			return
		}
	}
	t.Fatalf("missing clipart %s", name)
}

func hexPrefix(hash [32]byte, chars int) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 0, 64)
	for _, b := range hash {
		out = append(out, hexdigits[b>>4], hexdigits[b&0x0f])
	}
	if chars > len(out) {
		chars = len(out)
	}
	return string(out[:chars])
}

func findWMFByOffset(t *testing.T, doc *ETQFile, off int) modelWMFForTest {
	t.Helper()
	for _, sym := range doc.WMFElements {
		if sym.FileOffset == off {
			return modelWMFForTest{FilePath: sym.FilePath}
		}
	}
	t.Fatalf("missing WMF at offset %#x in %#v", off, doc.WMFElements)
	return modelWMFForTest{}
}

type modelWMFForTest struct {
	FilePath string
}
