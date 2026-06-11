package etq

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

var (
	ErrMissingFileOffset  = errors.New("etq: missing file offset")
	ErrUnknownObject      = errors.New("etq: unknown object at file offset")
	ErrVariableLengthText = errors.New("etq: variable-length text payload not supported")
	ErrInvalidPatch       = errors.New("etq: invalid patch")
	ErrAmbiguousChain     = errors.New("etq: ambiguous chain coordinate target")
)

const wmfPreBlockSize = 83

type Patcher struct {
	doc  *ETQFile
	data []byte
}

func NewPatcher(doc *ETQFile) (*Patcher, error) {
	if doc == nil {
		return nil, fmt.Errorf("new patcher: nil document")
	}
	if len(doc.UnknownObjects) > 0 {
		return nil, ErrUnknownObjects
	}
	if doc.FilePath == "" {
		return nil, fmt.Errorf("new patcher: missing source path")
	}
	data, err := os.ReadFile(doc.FilePath)
	if err != nil {
		return nil, fmt.Errorf("new patcher: read source: %w", err)
	}
	return &Patcher{doc: doc, data: append([]byte(nil), data...)}, nil
}

func SavePatchedETQ(doc *ETQFile, dstPath string, patch func(*Patcher) error) error {
	p, err := NewPatcher(doc)
	if err != nil {
		return err
	}
	if patch != nil {
		if err := patch(p); err != nil {
			return err
		}
	}
	return p.WriteTo(dstPath)
}

func (p *Patcher) Bytes() []byte {
	return append([]byte(nil), p.data...)
}

func (p *Patcher) WriteTo(dstPath string) error {
	if err := os.WriteFile(dstPath, p.data, 0644); err != nil {
		return fmt.Errorf("write patched ETQ: %w", err)
	}
	return nil
}

func (p *Patcher) PatchTextPosition(fileOffset int, rawX, rawY uint32) error {
	if fileOffset <= 0 {
		return ErrMissingFileOffset
	}
	if !p.knownTextOffset(fileOffset) {
		return ErrUnknownObject
	}
	if err := validateTextFE(p.data, fileOffset); err != nil {
		return err
	}
	oldX := binary.LittleEndian.Uint32(p.data[fileOffset+16 : fileOffset+20])
	oldY := binary.LittleEndian.Uint32(p.data[fileOffset+20 : fileOffset+24])
	if oldX != rawX || oldY != rawY {
		if err := p.relinkChainPredecessor(chainKey{x: oldX, y: oldY}, chainKey{x: rawX, y: rawY}); err != nil {
			return err
		}
	}
	binary.LittleEndian.PutUint32(p.data[fileOffset+16:fileOffset+20], rawX)
	binary.LittleEndian.PutUint32(p.data[fileOffset+20:fileOffset+24], rawY)
	return nil
}

func (p *Patcher) PatchTextPayload(fileOffset int, payload []byte) error {
	if fileOffset <= 0 {
		return ErrMissingFileOffset
	}
	if !p.knownTextOffset(fileOffset) {
		return ErrUnknownObject
	}
	if err := validateTextFE(p.data, fileOffset); err != nil {
		return err
	}
	tln := int(binary.LittleEndian.Uint16(p.data[fileOffset+38 : fileOffset+40]))
	if len(payload) != tln {
		return ErrVariableLengthText
	}
	copy(p.data[fileOffset+40:fileOffset+40+tln], payload)
	return nil
}

func (p *Patcher) PatchTextPayloadLatin1(fileOffset int, text string) error {
	payload, err := encodeLatin1Payload(text)
	if err != nil {
		return err
	}
	return p.PatchTextPayload(fileOffset, payload)
}

func (p *Patcher) PatchWMFRect(fileOffset int, headW, headH, x, y uint32) error {
	if fileOffset <= 0 {
		return ErrMissingFileOffset
	}
	if !p.knownWMFOffset(fileOffset) {
		return ErrUnknownObject
	}
	if err := validateWMFFE(p.data, fileOffset); err != nil {
		return err
	}
	preOff := fileOffset - wmfPreBlockSize
	oldW := binary.LittleEndian.Uint32(p.data[fileOffset+16 : fileOffset+20])
	oldH := binary.LittleEndian.Uint32(p.data[fileOffset+20 : fileOffset+24])
	if oldW != headW || oldH != headH {
		if err := p.relinkChainPredecessor(chainKey{x: oldW, y: oldH}, chainKey{x: headW, y: headH}); err != nil {
			return err
		}
	}
	binary.LittleEndian.PutUint32(p.data[fileOffset+16:fileOffset+20], headW)
	binary.LittleEndian.PutUint32(p.data[fileOffset+20:fileOffset+24], headH)
	binary.LittleEndian.PutUint32(p.data[preOff+4:preOff+8], x)
	binary.LittleEndian.PutUint32(p.data[preOff+8:preOff+12], y)
	return nil
}

func (p *Patcher) knownTextOffset(fileOffset int) bool {
	for _, txt := range p.doc.TextElements {
		if txt.FileOffset == fileOffset {
			return true
		}
	}
	return false
}

func (p *Patcher) knownWMFOffset(fileOffset int) bool {
	for _, sym := range p.doc.WMFElements {
		if sym.FileOffset == fileOffset {
			return true
		}
	}
	return false
}

func validateTextFE(data []byte, off int) error {
	if off+40 > len(data) || !bytes.Equal(data[off:off+4], feMarker) {
		return ErrInvalidPatch
	}
	flags := binary.LittleEndian.Uint32(data[off+8 : off+12])
	tag := binary.LittleEndian.Uint32(data[off+12 : off+16])
	if flags != 0 || (tag != 1 && tag != 0) {
		return ErrInvalidPatch
	}
	tln := int(binary.LittleEndian.Uint16(data[off+38 : off+40]))
	if tln < 1 || tln > 4096 || off+40+tln+4 > len(data) {
		return ErrInvalidPatch
	}
	if !bytes.Equal(data[off+40+tln:off+40+tln+4], []byte{0xff, 0xff, 0xff, 0xff}) {
		return ErrInvalidPatch
	}
	return nil
}

func validateWMFFE(data []byte, off int) error {
	if off+64 > len(data) || !bytes.Equal(data[off:off+4], feMarker) {
		return ErrInvalidPatch
	}
	flags := binary.LittleEndian.Uint32(data[off+8 : off+12])
	tag := binary.LittleEndian.Uint32(data[off+12 : off+16])
	if tag != 0 || flags != 0x80000008 {
		return ErrInvalidPatch
	}
	preOff := off - wmfPreBlockSize
	if preOff < 0 || preOff+12 > len(data) || !bytes.Equal(data[preOff:preOff+4], []byte{0xff, 0xff, 0xff, 0xff}) {
		return ErrInvalidPatch
	}
	aldusOff := off + 49
	if aldusOff+4 > len(data) || !bytes.Equal(data[aldusOff:aldusOff+4], []byte{0xd7, 0xcd, 0xc6, 0x9a}) {
		return ErrInvalidPatch
	}
	return nil
}

func encodeLatin1Payload(text string) ([]byte, error) {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		if r > 0xff {
			return nil, fmt.Errorf("encode latin1 payload: rune out of range: %q", r)
		}
		out = append(out, byte(r))
	}
	return out, nil
}

type chainKey struct {
	x uint32
	y uint32
}

type chainNode struct {
	offset int
	key    chainKey
	next   chainKey
}

type chainIndex struct {
	nodes []chainNode
	byKey map[chainKey][]int
}

func (p *Patcher) relinkChainPredecessor(oldKey, newKey chainKey) error {
	if oldKey == newKey {
		return nil
	}
	idx := buildChainIndex(p.data)
	if len(idx.byKey[oldKey]) > 1 || len(idx.byKey[newKey]) > 0 {
		return ErrAmbiguousChain
	}
	pred, err := idx.uniquePredecessor(oldKey)
	if err != nil || pred == 0 {
		return err
	}
	return patchNodeNextCoords(p.data, pred, newKey.x, newKey.y)
}

func buildChainIndex(data []byte) chainIndex {
	idx := chainIndex{byKey: map[chainKey][]int{}}
	for i := 0; i+48 < len(data); i++ {
		if !bytes.Equal(data[i:i+4], feMarker) {
			continue
		}
		flags := binary.LittleEndian.Uint32(data[i+8 : i+12])
		tag := binary.LittleEndian.Uint32(data[i+12 : i+16])
		switch {
		case flags == 0 && (tag == 1 || tag == 0):
			if node, ok := chainTextNode(data, i); ok {
				idx.add(node)
			}
		case tag == 0 && flags == 0x80000008:
			if node, ok := chainWMFNode(data, i); ok {
				idx.add(node)
			}
		}
	}
	return idx
}

func (idx *chainIndex) add(node chainNode) {
	idx.nodes = append(idx.nodes, node)
	idx.byKey[node.key] = append(idx.byKey[node.key], node.offset)
}

func (idx chainIndex) uniquePredecessor(target chainKey) (int, error) {
	var pred int
	for _, node := range idx.nodes {
		if node.next != target {
			continue
		}
		if pred != 0 {
			return 0, ErrAmbiguousChain
		}
		pred = node.offset
	}
	return pred, nil
}

func chainTextNode(data []byte, off int) (chainNode, bool) {
	if off+40 > len(data) {
		return chainNode{}, false
	}
	tln := int(binary.LittleEndian.Uint16(data[off+38 : off+40]))
	if tln > 4096 || off+40+tln+4 > len(data) {
		return chainNode{}, false
	}
	term := off + 40 + tln
	if !bytes.Equal(data[term:term+4], []byte{0xff, 0xff, 0xff, 0xff}) {
		return chainNode{}, false
	}
	post := term + 4
	if post+16 > len(data) {
		return chainNode{}, false
	}
	node := chainNode{offset: off, key: chainKey{x: binary.LittleEndian.Uint32(data[off+16 : off+20]), y: binary.LittleEndian.Uint32(data[off+20 : off+24])}, next: chainKey{x: binary.LittleEndian.Uint32(data[post+8 : post+12]), y: binary.LittleEndian.Uint32(data[post+12 : post+16])}}
	if tln == 0 {
		return node, true
	}
	text, _, _, ok := decodeTextPayload(data[off+40 : off+40+tln])
	return node, ok && isDocumentText(text)
}

func chainWMFNode(data []byte, off int) (chainNode, bool) {
	end, ok := wmfBlobEndForPatch(data, off)
	if !ok || end+20 > len(data) || !bytes.Equal(data[end:end+4], []byte{0xff, 0xff, 0xff, 0xff}) {
		return chainNode{}, false
	}
	return chainNode{offset: off, key: chainKey{x: binary.LittleEndian.Uint32(data[off+16 : off+20]), y: binary.LittleEndian.Uint32(data[off+20 : off+24])}, next: chainKey{x: binary.LittleEndian.Uint32(data[end+12 : end+16]), y: binary.LittleEndian.Uint32(data[end+16 : end+20])}}, true
}

func patchNodeNextCoords(data []byte, off int, nextX, nextY uint32) error {
	flags := binary.LittleEndian.Uint32(data[off+8 : off+12])
	tag := binary.LittleEndian.Uint32(data[off+12 : off+16])
	switch {
	case flags == 0 && (tag == 1 || tag == 0):
		tln := int(binary.LittleEndian.Uint16(data[off+38 : off+40]))
		post := off + 40 + tln + 4
		if post+16 > len(data) {
			return ErrInvalidPatch
		}
		binary.LittleEndian.PutUint32(data[post+8:post+12], nextX)
		binary.LittleEndian.PutUint32(data[post+12:post+16], nextY)
		return nil
	case tag == 0 && flags == 0x80000008:
		end, ok := wmfBlobEndForPatch(data, off)
		if !ok || end+20 > len(data) {
			return ErrInvalidPatch
		}
		binary.LittleEndian.PutUint32(data[end+12:end+16], nextX)
		binary.LittleEndian.PutUint32(data[end+16:end+20], nextY)
		return nil
	default:
		return ErrInvalidPatch
	}
}

func wmfBlobEndForPatch(data []byte, feOff int) (int, bool) {
	aldusOff := feOff + 49
	if aldusOff+32 > len(data) || binary.LittleEndian.Uint32(data[aldusOff:aldusOff+4]) != 0x9ac6cdd7 {
		return 0, false
	}
	std := aldusOff + 22
	words := int(binary.LittleEndian.Uint32(data[std+6 : std+10]))
	end := aldusOff + 22 + words*2
	return end, end <= len(data)
}
