package print

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"masterprint-native/internal/model"
)

func TestWMFBytesPrefersEmbedded(t *testing.T) {
	embedded := []byte{0xd7, 0xcd, 0xc6, 0x9a, 0x01, 0x02}
	got, err := WMFBytes(model.WMFSymbol{Embedded: embedded, FilePath: filepath.Join(t.TempDir(), "missing.wmf")})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, embedded) {
		t.Fatalf("got %#v want %#v", got, embedded)
	}
}

func TestWMFBytesFallsBackToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sym.wmf")
	want := []byte{0xaa, 0xbb, 0xcc}
	if err := os.WriteFile(path, want, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := WMFBytes(model.WMFSymbol{FilePath: path})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestWMFBytesMissingBoth(t *testing.T) {
	if _, err := WMFBytes(model.WMFSymbol{}); err == nil {
		t.Fatal("expected error")
	}
}
