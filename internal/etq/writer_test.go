package etq

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCanWriteAllowsCleanETQ(t *testing.T) {
	path := filepath.Join(installedPaulimaqRoot(), "ARQUIVOS", "SLIM.ETQ")
	doc, err := ParseETQ(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	if !CanWrite(doc) {
		t.Fatalf("expected writable ETQ, unknown=%#v", doc.UnknownObjects)
	}
}

func TestSaveETQRefusesUnknownObjects(t *testing.T) {
	doc := &ETQFile{FilePath: filepath.Join(installedPaulimaqRoot(), "ARQUIVOS", "SLIM.ETQ"), UnknownObjects: []ETQUnknownObject{{Offset: 0x1234, Flags: 0x10, Tag: 9, Kind: "unsupported"}}}
	if CanWrite(doc) {
		t.Fatalf("document with unknown objects must refuse write")
	}
	err := SaveETQ(doc, filepath.Join(t.TempDir(), "out.ETQ"))
	if !errors.Is(err, ErrUnknownObjects) {
		t.Fatalf("SaveETQ err=%v want ErrUnknownObjects", err)
	}
}

func TestSaveETQNoOpByteIdentical(t *testing.T) {
	path := filepath.Join(installedPaulimaqRoot(), "ARQUIVOS", "SLIM.ETQ")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc, err := ParseETQ(path)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "copy.ETQ")
	if err := SaveETQ(doc, dst); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, saved) {
		t.Fatalf("no-op save changed bytes: len %d -> %d", len(original), len(saved))
	}
}
