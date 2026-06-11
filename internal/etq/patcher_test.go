package etq

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func loadLunelliDoc(t *testing.T) (*ETQFile, []byte) {
	t.Helper()
	path := filepath.Join(installedPaulimaqRoot(), "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc, err := ParseETQ(path)
	if err != nil {
		t.Fatal(err)
	}
	return doc, data
}

func TestPatchTextPositionLunelliAnchor(t *testing.T) {
	const off = 0x15d3
	doc, original := loadLunelliDoc(t)
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PatchTextPosition(off, 2100, 400); err != nil {
		t.Fatal(err)
	}
	patched := p.Bytes()
	if binary.LittleEndian.Uint32(patched[off+16:off+20]) != 2100 || binary.LittleEndian.Uint32(patched[off+20:off+24]) != 400 {
		t.Fatalf("position bytes not patched at %#x", off)
	}
	if !bytes.Equal(patched[off:off+16], original[off:off+16]) || !bytes.Equal(patched[off+24:off+40], original[off+24:off+40]) {
		t.Fatal("patch touched bytes outside text x/y fields")
	}
}

func TestPatchTextPayloadSameLengthLunelliAnchor(t *testing.T) {
	const off = 0x15d3
	doc, original := loadLunelliDoc(t)
	tln := int(binary.LittleEndian.Uint16(original[off+38 : off+40]))
	if tln != 11 {
		t.Fatalf("unexpected anchor tln=%d want 11", tln)
	}
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PatchTextPayloadLatin1(off, "71% ALGODÃO"); err != nil {
		t.Fatal(err)
	}
	patched := p.Bytes()
	if !bytes.Equal(patched[off+16:off+40], original[off+16:off+40]) {
		t.Fatal("payload patch changed position or length fields")
	}
	if got := decodeLatin1(bytes.TrimRight(patched[off+40:off+40+tln], "\x00")); got != "71% ALGODÃO" {
		t.Fatalf("patched text=%q want %q", got, "71% ALGODÃO")
	}
}

func TestPatchWMFRectCloroxAnchor(t *testing.T) {
	const off = 0x111
	const preOff = off - wmfPreBlockSize
	doc, original := loadLunelliDoc(t)
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PatchWMFRect(off, 445, 445, 530, 2700); err != nil {
		t.Fatal(err)
	}
	patched := p.Bytes()
	if binary.LittleEndian.Uint32(patched[off+16:off+20]) != 445 || binary.LittleEndian.Uint32(patched[off+20:off+24]) != 445 {
		t.Fatalf("head rect should remain unchanged at %#x", off)
	}
	if binary.LittleEndian.Uint32(patched[preOff+4:preOff+8]) != 530 || binary.LittleEndian.Uint32(patched[preOff+8:preOff+12]) != 2700 {
		t.Fatalf("pre-block rect not patched at %#x", preOff)
	}
	if bytes.Equal(patched, original) {
		t.Fatal("patch changed nothing")
	}
}

func TestPatchTextPositionUpdatesChainPredecessor(t *testing.T) {
	const off = 0x15d3
	const predNextXOff = 0x158c
	const predNextYOff = 0x1590
	doc, original := loadLunelliDoc(t)
	if binary.LittleEndian.Uint32(original[predNextXOff:predNextXOff+4]) != 2056 || binary.LittleEndian.Uint32(original[predNextYOff:predNextYOff+4]) != 353 {
		t.Fatalf("unexpected predecessor next anchor")
	}
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PatchTextPosition(off, 2100, 400); err != nil {
		t.Fatal(err)
	}
	patched := p.Bytes()
	if binary.LittleEndian.Uint32(patched[predNextXOff:predNextXOff+4]) != 2100 || binary.LittleEndian.Uint32(patched[predNextYOff:predNextYOff+4]) != 400 {
		t.Fatalf("predecessor nextX/nextY not relinked")
	}
}

func TestPatchWMFRectRefusesDuplicateChainKey(t *testing.T) {
	doc, _ := loadLunelliDoc(t)
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	err = p.PatchWMFRect(0x111, 450, 451, 529, 2696)
	if !errors.Is(err, ErrAmbiguousChain) {
		t.Fatalf("err=%v want ErrAmbiguousChain", err)
	}
}

func TestPatchTextPositionRefusesNewKeyCollision(t *testing.T) {
	doc, original := loadLunelliDoc(t)
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	collideX := binary.LittleEndian.Uint32(original[0x154b+16 : 0x154b+20])
	collideY := binary.LittleEndian.Uint32(original[0x154b+20 : 0x154b+24])
	err = p.PatchTextPosition(0x15d3, collideX, collideY)
	if !errors.Is(err, ErrAmbiguousChain) {
		t.Fatalf("err=%v want ErrAmbiguousChain", err)
	}
}

func TestSavePatchedETQRoundTripLunelliAnchor(t *testing.T) {
	const off = 0x15d3
	doc, _ := loadLunelliDoc(t)
	dst := filepath.Join(t.TempDir(), "patched.ETQ")
	if err := SavePatchedETQ(doc, dst, func(p *Patcher) error { return p.PatchTextPosition(off, 2100, 400) }); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(saved[off+16:off+20]) != 2100 || binary.LittleEndian.Uint32(saved[off+20:off+24]) != 400 {
		t.Fatalf("round-trip position not persisted")
	}
}

func TestPatcherRefusesVariableLengthText(t *testing.T) {
	const off = 0x15d3
	doc, _ := loadLunelliDoc(t)
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	err = p.PatchTextPayload(off, []byte("too short"))
	if !errors.Is(err, ErrVariableLengthText) {
		t.Fatalf("err=%v want ErrVariableLengthText", err)
	}
}

func TestPatcherRefusesMissingOrUnknownOffsets(t *testing.T) {
	doc, _ := loadLunelliDoc(t)
	p, err := NewPatcher(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.PatchTextPosition(0, 1, 1); !errors.Is(err, ErrMissingFileOffset) {
		t.Fatalf("missing offset err=%v", err)
	}
	if err := p.PatchTextPosition(0x713b, 1, 1); !errors.Is(err, ErrUnknownObject) {
		t.Fatalf("unknown text offset err=%v", err)
	}
	if err := p.PatchWMFRect(0x15d3, 1, 1, 1, 1); !errors.Is(err, ErrUnknownObject) {
		t.Fatalf("wrong object offset err=%v", err)
	}
}

func TestPatcherRefusesDocumentsWithUnknownObjects(t *testing.T) {
	path := filepath.Join(installedPaulimaqRoot(), "ARQUIVOS", "SLIM.ETQ")
	doc, err := ParseETQ(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc.UnknownObjects = append(doc.UnknownObjects, ETQUnknownObject{Offset: 0x1234, Flags: 0, Tag: 9, Kind: "unsupported"})
	_, err = NewPatcher(doc)
	if !errors.Is(err, ErrUnknownObjects) {
		t.Fatalf("NewPatcher err=%v want ErrUnknownObjects", err)
	}
}
