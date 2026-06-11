package etq

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFERecordFieldsLunelliText(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	recs := extractTextRecords(data)
	var rec *etqTextRecord
	for i := range recs {
		if recs[i].Offset == 5587 {
			rec = &recs[i]
			break
		}
	}
	if rec == nil {
		t.Fatalf("expected text record at offset 5587")
	}
	if rec.Text != "72% ALGODÃO" || rec.RawX != 2056 || rec.RawY != 353 || rec.RectHeight != 161 || rec.RectWidth != 3637 || rec.Align != 0 || rec.FontStyle != 5 {
		t.Fatalf("unexpected decoded record: %#v", *rec)
	}
	doc, err := ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, txt := range doc.TextElements {
		if txt.Text == "72% ALGODÃO" {
			found = true
			if !near(txt.XMM, 20.56) || !near(txt.YMM, 3.53) || !near(txt.HeightMM, 1.61) || !near(txt.WidthMM, 36.37) || txt.Align != "left" || txt.FEFlags != 0 || txt.FETag != 1 || txt.StyleByte != 5 || txt.NextX != 2238 || txt.NextY != 419 {
				t.Fatalf("unexpected text element geometry: %#v", txt)
			}
			if !txt.Bold || txt.Italic || !txt.Underline {
				t.Fatalf("style byte 5 not decoded as bold+underline: %#v", txt)
			}
			if decodeLatin1(bytes.TrimRight(txt.PayloadRaw, "\x00")) != txt.Text {
				t.Fatalf("payload raw did not preserve text bytes: %q vs %q", decodeLatin1(txt.PayloadRaw), txt.Text)
			}
		}
	}
	if !found {
		t.Fatalf("missing parsed text element")
	}
}

func TestFEWMFPreBlockLunelliClorox(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 273
	preOff := off - 83
	if binary.LittleEndian.Uint32(data[off+8:off+12]) != 0x80000008 || binary.LittleEndian.Uint32(data[off+12:off+16]) != 0 {
		t.Fatalf("unexpected WMF tag/flags")
	}
	if binary.LittleEndian.Uint32(data[off+16:off+20]) != 445 || binary.LittleEndian.Uint32(data[off+20:off+24]) != 445 {
		t.Fatalf("unexpected WMF head dimensions")
	}
	if binary.LittleEndian.Uint32(data[preOff:preOff+4]) != 0xffffffff || binary.LittleEndian.Uint32(data[preOff+4:preOff+8]) != 529 || binary.LittleEndian.Uint32(data[preOff+8:preOff+12]) != 2696 {
		t.Fatalf("unexpected WMF pre-block")
	}
	doc, err := ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, sym := range doc.WMFElements {
		if sym.FilePath == "clorox.wmf" {
			if !near(sym.XMM, 5.29) || !near(sym.YMM, 26.96) || !near(sym.WidthMM, 4.45) || !near(sym.HeightMM, 4.45) || sym.StyleByte != 6 || sym.NextX != 445 || sym.NextY != 445 {
				t.Fatalf("unexpected WMF geometry: %#v", sym)
			}
			blob, ok := wmfBlobAt(data, off)
			if !ok {
				t.Fatalf("expected WMF blob at offset %#x", off)
			}
			if !bytes.Equal(sym.Embedded, blob) {
				t.Fatalf("embedded WMF blob was not preserved")
			}
			if len(sym.PreBlock) != 83 || !bytes.Equal(sym.PreBlock[:4], []byte{0xff, 0xff, 0xff, 0xff}) {
				t.Fatalf("WMF pre-block was not preserved: len=%d", len(sym.PreBlock))
			}
			if !bytes.Equal(sym.PreBlock, data[preOff:off]) {
				t.Fatalf("WMF pre-block bytes differ from source")
			}
			return
		}
	}
	t.Fatalf("missing clorox.wmf in %#v", doc.WMFElements)
}

func TestFEWMFNonSquareFavero(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "FAVERO.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x114f
	preOff := off - 83
	if binary.LittleEndian.Uint32(data[off+8:off+12]) != 0x80000008 || binary.LittleEndian.Uint32(data[off+12:off+16]) != 0 {
		t.Fatalf("unexpected WMF tag/flags")
	}
	if binary.LittleEndian.Uint32(data[off+16:off+20]) != 496 || binary.LittleEndian.Uint32(data[off+20:off+24]) != 639 {
		t.Fatalf("unexpected non-square WMF dimensions")
	}
	if binary.LittleEndian.Uint32(data[preOff:preOff+4]) != 0xffffffff || binary.LittleEndian.Uint32(data[preOff+4:preOff+8]) != 137 || binary.LittleEndian.Uint32(data[preOff+8:preOff+12]) != 2988 {
		t.Fatalf("unexpected WMF pre-block")
	}
	doc, err := ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, sym := range doc.WMFElements {
		if sym.FilePath == "secox.wmf" {
			if !near(sym.XMM, 1.37) || !near(sym.YMM, 29.88) || !near(sym.WidthMM, 4.96) || !near(sym.HeightMM, 6.39) || sym.WidthMM == sym.HeightMM {
				t.Fatalf("unexpected secox.wmf geometry: %#v", sym)
			}
			return
		}
	}
	t.Fatalf("missing secox.wmf in %#v", doc.WMFElements)
}

func TestFEChainNextFieldOffsetsLunelliText(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x15d3
	tln := int(binary.LittleEndian.Uint16(data[off+38 : off+40]))
	afterTerminator := off + 40 + tln + 4
	if binary.LittleEndian.Uint32(data[off+40+tln:off+40+tln+4]) != 0xffffffff {
		t.Fatalf("missing text terminator")
	}
	if binary.LittleEndian.Uint32(data[afterTerminator+8:afterTerminator+12]) != 2238 || binary.LittleEndian.Uint32(data[afterTerminator+12:afterTerminator+16]) != 419 {
		t.Fatalf("unexpected text next coordinates")
	}
}

func TestFEChainNextFieldOffsetsLunelliWMF(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x046d
	end, ok := wmfBlobEnd(data, off)
	if !ok {
		t.Fatalf("expected WMF blob at offset %#x", off)
	}
	if binary.LittleEndian.Uint32(data[end:end+4]) != 0xffffffff {
		t.Fatalf("missing WMF post-block terminator")
	}
	if binary.LittleEndian.Uint32(data[end+12:end+16]) != 432 || binary.LittleEndian.Uint32(data[end+16:end+20]) != 432 {
		t.Fatalf("unexpected WMF post-block next coordinates")
	}
}

func TestParseETQWMFBodyHashResolverOverridesSizeFallback(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x111
	blob, ok := wmfBlobAt(data, off)
	if !ok {
		t.Fatalf("expected WMF blob at offset %#x", off)
	}

	tmp := t.TempDir()
	arquivosDir := filepath.Join(tmp, "ARQUIVOS")
	clipartDir := filepath.Join(tmp, "CLIPART", "Símbolos")
	if err := os.MkdirAll(arquivosDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clipartDir, 0755); err != nil {
		t.Fatal(err)
	}
	etqPath := filepath.Join(arquivosDir, "sample.ETQ")
	if err := os.WriteFile(etqPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clipartDir, "sha-only.wmf"), blob, 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseETQ(etqPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, sym := range doc.WMFElements {
		if sym.FileOffset == off {
			if sym.FilePath != "sha-only.wmf" {
				t.Fatalf("expected body-hash resolver to override size fallback, got %q", sym.FilePath)
			}
			return
		}
	}
	t.Fatalf("missing WMF element at offset %#x in %#v", off, doc.WMFElements)
}

func TestFEWMFUnknownResolverKeepsEmbeddedBlob(t *testing.T) {
	const off = 83
	data := make([]byte, off+49+32+21)
	binary.LittleEndian.PutUint32(data[0:4], 0xffffffff)
	binary.LittleEndian.PutUint32(data[4:8], 529)
	binary.LittleEndian.PutUint32(data[8:12], 2696)
	copy(data[off:off+4], feMarker)
	binary.LittleEndian.PutUint32(data[off+8:off+12], 0x80000008)
	binary.LittleEndian.PutUint32(data[off+12:off+16], 0)
	binary.LittleEndian.PutUint32(data[off+16:off+20], 445)
	binary.LittleEndian.PutUint32(data[off+20:off+24], 445)
	aldusOff := off + 49
	copy(data[aldusOff:aldusOff+4], []byte{0xd7, 0xcd, 0xc6, 0x9a})
	std := aldusOff + 22
	binary.LittleEndian.PutUint32(data[std+6:std+10], 5)
	end := aldusOff + 32
	binary.LittleEndian.PutUint32(data[end:end+4], 0xffffffff)
	binary.LittleEndian.PutUint32(data[end+12:end+16], 445)
	binary.LittleEndian.PutUint32(data[end+16:end+20], 445)
	data[end+20] = 6

	syms := extractEmbeddedWMFSymbolsWithResolver(data, func(_ []byte, _ int) (string, bool) { return "", false })
	if len(syms) != 1 {
		t.Fatalf("expected embedded WMF without name, got %#v", syms)
	}
	if syms[0].FilePath != "" || len(syms[0].Embedded) != 32 || syms[0].FileOffset != off {
		t.Fatalf("unexpected symbol: %#v", syms[0])
	}
}

func TestFEChainDuplicateWMFKeyRequiresFileOffsetFallback(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	chain := analyzeFEChainForTest(data)
	if chain.useChain {
		t.Fatalf("duplicate WMF dimension keys must keep FileOffset fallback")
	}
	if got := chain.duplicateOffsets(445, 445); !reflect.DeepEqual(got, []int{0x111, 0x46d}) {
		t.Fatalf("unexpected duplicate WMF key offsets: %#v", got)
	}
	if got := chain.resolveNext(0x114f); got != 0x144b {
		t.Fatalf("expected unique edge 0x114f -> 0x144b, got %#x", got)
	}
	if !reflect.DeepEqual(chain.displayOffsets[:4], []int{0x111, 0x46d, 0x114f, 0x144b}) {
		t.Fatalf("expected display fallback to start in FileOffset order, got %#v", chain.displayOffsets[:4])
	}
}

func TestFEChainHiddenTextNodeSlimParticipatesButIsNotDisplayed(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "SLIM.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const hidden = 0x15d1
	if binary.LittleEndian.Uint16(data[hidden+38:hidden+40]) != 0 {
		t.Fatalf("expected hidden node tln=0")
	}
	chain := analyzeFEChainForTest(data)
	if _, ok := chain.nodeByOffset[hidden]; !ok {
		t.Fatalf("hidden tln=0 node must be retained in chain graph")
	}
	for _, off := range chain.displayOffsets {
		if off == hidden {
			t.Fatalf("hidden tln=0 node must not be displayed")
		}
	}
	if got := chain.resolveNext(0x154b); got != hidden {
		t.Fatalf("expected 0x154b -> hidden node, got %#x", got)
	}
	if got := chain.resolveNext(hidden); got != 0x164c {
		t.Fatalf("expected hidden node -> 0x164c, got %#x", got)
	}
}

func TestFEChainCaracolDuplicateTextKeysRequireFallback(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "CARACOL TULE.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	chain := analyzeFEChainForTest(data)
	if chain.useChain {
		t.Fatalf("duplicate text keys must keep FileOffset fallback")
	}
	if got := chain.duplicateOffsets(2169, 426); !reflect.DeepEqual(got, []int{0x5391, 0x652b}) {
		t.Fatalf("unexpected duplicate text key offsets: %#v", got)
	}
	var elastano []int
	for _, off := range chain.displayOffsets {
		if chain.nodeByOffset[off].text == "4% ELASTANO" {
			elastano = append(elastano, off)
		}
	}
	if !reflect.DeepEqual(elastano, []int{0x5417, 0x65b3}) {
		t.Fatalf("expected both 4%% ELASTANO records in FileOffset order, got %#v", elastano)
	}
}

func TestFEChainADARFalsePositiveIsExcludedFromGraph(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "ADAR SOFA CANELADO.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	chain := analyzeFEChainForTest(data)
	if _, ok := chain.nodeByOffset[0x713b]; ok {
		t.Fatalf("false-positive FE inside WMF must not enter chain graph")
	}
	if chain.useChain {
		t.Fatalf("ambiguous ADAR chain must keep FileOffset fallback")
	}
}

func TestFEFalsePositiveADARIsIgnored(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "ADAR SOFA CANELADO.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x713b
	if binary.LittleEndian.Uint32(data[off:off+4]) != 0xfffffffe || binary.LittleEndian.Uint32(data[off+8:off+12]) != 0 || binary.LittleEndian.Uint32(data[off+12:off+16]) != 1 {
		t.Fatalf("unexpected false-positive FE shape")
	}
	if binary.LittleEndian.Uint32(data[off+16:off+20]) != 476 || binary.LittleEndian.Uint32(data[off+20:off+24]) != 489 || binary.LittleEndian.Uint16(data[off+38:off+40]) != 1 || data[off+40] != 'M' {
		t.Fatalf("unexpected false-positive fields")
	}
	doc, err := ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, txt := range doc.TextElements {
		if txt.Text == "M" || (near(txt.XMM, 4.76) && near(txt.YMM, 4.89)) {
			t.Fatalf("false-positive FE became text: %#v", txt)
		}
	}
}

func TestTag0Flags0TextLikeFEIsPromoted(t *testing.T) {
	cases := map[string]map[int]string{
		"ADAR SOFA CANELADO.ETQ":            {0x72c: "HB GIRLS"},
		"RIBANA CANELADO NEON - 98% 2%.ETQ": {0x3d0: "ÚNICO", 0x844: "HB GIRLS"},
		"SUEDE ERRADO.ETQ":                  {0x54b: "ÚNICO", 0x9bf: "HB GIRLS"},
		"malha tricot everest makro.ETQ":    {0x54b: "ÚNICO", 0x9bf: "HB GIRLS"},
		"tuly.ETQ":                          {0x26a: "ÚNICO", 0x6de: "HB GIRLS"},
	}
	for name, want := range cases {
		name, want := name, want
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("sample ETQ not installed: %v", err)
			}
			doc, err := ParseETQData(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.UnknownObjects) != 0 {
				t.Fatalf("tag0 text-like records should not remain unknown: %#v", doc.UnknownObjects)
			}
			seen := map[int]modelTextForTest{}
			for _, txt := range doc.TextElements {
				if _, ok := want[txt.FileOffset]; ok {
					seen[txt.FileOffset] = modelTextForTest{text: txt.Text, flags: txt.FEFlags, tag: txt.FETag}
				}
			}
			for off, text := range want {
				got, ok := seen[off]
				if !ok || got.text != text || got.flags != 0 || got.tag != 0 {
					t.Fatalf("offset %#x promoted=%#v want text=%q flags=0 tag=0", off, got, text)
				}
			}
		})
	}
}

type modelTextForTest struct {
	text  string
	flags uint32
	tag   uint32
}

func TestInvalidFEIsNotAccountedAsUnknown(t *testing.T) {
	data := make([]byte, 80)
	copy(data[8:], feMarker)
	binary.LittleEndian.PutUint32(data[16:20], 0)
	binary.LittleEndian.PutUint32(data[20:24], 0)
	binary.LittleEndian.PutUint16(data[46:48], 3)
	copy(data[48:51], []byte{0, 1, 2})
	copy(data[51:55], []byte{0xff, 0xff, 0xff, 0xff})
	doc, err := ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.UnknownObjects) != 0 {
		t.Fatalf("invalid FE should not be accounted as unknown: %#v", doc.UnknownObjects)
	}
}

func TestFEEmptyTextNodeSlimIsSkipped(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "SLIM.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x15d1
	postOff := off + 40 + 4
	if binary.LittleEndian.Uint32(data[off+8:off+12]) != 0 || binary.LittleEndian.Uint32(data[off+12:off+16]) != 1 || binary.LittleEndian.Uint16(data[off+38:off+40]) != 0 {
		t.Fatalf("unexpected empty node shape")
	}
	if binary.LittleEndian.Uint32(data[off+40:off+44]) != 0xffffffff || binary.LittleEndian.Uint32(data[postOff+8:postOff+12]) != 2169 || binary.LittleEndian.Uint32(data[postOff+12:postOff+16]) != 419 {
		t.Fatalf("unexpected empty node post fields")
	}
	for _, rec := range extractTextRecords(data) {
		if rec.Offset == off || rec.Text == "" {
			t.Fatalf("empty node should not become a text record: %#v", rec)
		}
	}
}

func TestFETruncatedEOFAlgodaoIsSkipped(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "ALGODÃO.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	const off = 0x5686
	tln := int(binary.LittleEndian.Uint16(data[off+38 : off+40]))
	if binary.LittleEndian.Uint32(data[off:off+4]) != 0xfffffffe || tln != 24 || off+40+tln != len(data) {
		t.Fatalf("unexpected truncated EOF shape")
	}
	doc, err := ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, txt := range doc.TextElements {
		if strings.Contains(txt.Text, "A.n.n Silva") || strings.Contains(txt.Text, "Confec") {
			t.Fatalf("truncated EOF text should be skipped: %#v", txt)
		}
	}
}

func TestFENoDedupeByContentText(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "CARACOL TULE.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc, err := ParseETQData(data)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, txt := range doc.TextElements {
		if txt.Text == "4% ELASTANO" {
			count++
			if !near(txt.XMM, 21.89) || !near(txt.YMM, 4.59) {
				t.Fatalf("unexpected duplicate text position: %#v", txt)
			}
		}
	}
	if count != 2 {
		t.Fatalf("expected two 4%% ELASTANO text objects, got %d in %#v", count, doc.TextElements)
	}
}

func TestRtfHasBold(t *testing.T) {
	cases := []struct {
		name string
		rtf  string
		want bool
	}{
		{"colortbl blue only", `{\colortbl ;\red0\green0\blue0;}\pard\fs16 FEITO`, false},
		{"bold off only", `{\colortbl ;\red0\green0\blue0;}\pard\b0\fs16 FEITO`, false},
		{"bullet", `\pard\bullet item`, false},
		{"border", `\pard\brdrw10`, false},
		{"bold plain", `\cf1\b\f0\fs16 FEITO NO BRASIL\par \pard\b0`, true},
		{"bold explicit one", `\pard\b1 texto`, true},
		{"bold then off", `\pard\qc\b\fs22\'daNICO\b0\f1\fs20`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rtfHasBold(tc.rtf); got != tc.want {
				t.Fatalf("rtfHasBold()=%v want %v", got, tc.want)
			}
		})
	}
}

func near(got, want float64) bool {
	return math.Abs(got-want) < 0.001
}

func wmfBlobEnd(data []byte, feOff int) (int, bool) {
	aldusOff := feOff + 49
	if aldusOff+32 > len(data) || binary.LittleEndian.Uint32(data[aldusOff:aldusOff+4]) != 0x9ac6cdd7 {
		return 0, false
	}
	std := aldusOff + 22
	words := int(binary.LittleEndian.Uint32(data[std+6 : std+10]))
	end := aldusOff + 22 + words*2
	return end, end <= len(data)
}

func wmfBlobAt(data []byte, feOff int) ([]byte, bool) {
	end, ok := wmfBlobEnd(data, feOff)
	if !ok {
		return nil, false
	}
	aldusOff := feOff + 49
	return data[aldusOff:end], true
}

type feChainKey struct {
	x uint32
	y uint32
}

type feChainNode struct {
	offset  int
	key     feChainKey
	next    feChainKey
	display bool
	text    string
}

type feChainAnalysis struct {
	useChain       bool
	nodes          []feChainNode
	nodeByOffset   map[int]feChainNode
	displayOffsets []int
	byKey          map[feChainKey][]feChainNode
}

func analyzeFEChainForTest(data []byte) feChainAnalysis {
	chain := feChainAnalysis{
		nodeByOffset: map[int]feChainNode{},
		byKey:        map[feChainKey][]feChainNode{},
	}
	for i := 0; i+48 < len(data); i++ {
		if !bytes.Equal(data[i:i+4], feMarker) {
			continue
		}
		flags := binary.LittleEndian.Uint32(data[i+8 : i+12])
		tag := binary.LittleEndian.Uint32(data[i+12 : i+16])
		switch {
		case tag == 1 && flags == 0:
			node, ok := feTextChainNodeForTest(data, i)
			if ok {
				chain.add(node)
			}
		case tag == 0 && flags == 0x80000008:
			node, ok := feWMFChainNodeForTest(data, i)
			if ok {
				chain.add(node)
			}
		}
	}
	chain.useChain = chain.canUseChain()
	return chain
}

func (c *feChainAnalysis) add(node feChainNode) {
	c.nodes = append(c.nodes, node)
	c.nodeByOffset[node.offset] = node
	c.byKey[node.key] = append(c.byKey[node.key], node)
	if node.display {
		c.displayOffsets = append(c.displayOffsets, node.offset)
	}
}

func (c feChainAnalysis) duplicateOffsets(x, y uint32) []int {
	key := feChainKey{x: x, y: y}
	nodes := c.byKey[key]
	out := make([]int, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node.offset)
	}
	return out
}

func (c feChainAnalysis) resolveNext(offset int) int {
	node, ok := c.nodeByOffset[offset]
	if !ok {
		return 0
	}
	matches := c.byKey[node.next]
	if len(matches) != 1 {
		return 0
	}
	return matches[0].offset
}

func (c feChainAnalysis) canUseChain() bool {
	for _, nodes := range c.byKey {
		if len(nodes) > 1 {
			return false
		}
	}
	if len(c.nodes) == 0 {
		return false
	}
	visited := map[int]bool{}
	displayVisited := 0
	current := c.nodes[0]
	for {
		if visited[current.offset] {
			return false
		}
		visited[current.offset] = true
		if current.display {
			displayVisited++
		}
		next := c.byKey[current.next]
		if len(next) == 0 {
			break
		}
		if len(next) != 1 {
			return false
		}
		current = next[0]
	}
	return displayVisited == len(c.displayOffsets)
}

func feTextChainNodeForTest(data []byte, off int) (feChainNode, bool) {
	if off+40 > len(data) {
		return feChainNode{}, false
	}
	tln := int(binary.LittleEndian.Uint16(data[off+38 : off+40]))
	if tln < 0 || tln > 4096 || off+40+tln+4 > len(data) {
		return feChainNode{}, false
	}
	term := off + 40 + tln
	if !bytes.Equal(data[term:term+4], []byte{0xff, 0xff, 0xff, 0xff}) {
		return feChainNode{}, false
	}
	post := term + 4
	if post+16 > len(data) {
		return feChainNode{}, false
	}
	node := feChainNode{
		offset: off,
		key: feChainKey{
			x: binary.LittleEndian.Uint32(data[off+16 : off+20]),
			y: binary.LittleEndian.Uint32(data[off+20 : off+24]),
		},
		next: feChainKey{
			x: binary.LittleEndian.Uint32(data[post+8 : post+12]),
			y: binary.LittleEndian.Uint32(data[post+12 : post+16]),
		},
	}
	if tln == 0 {
		return node, true
	}
	text, _, _, ok := decodeTextPayload(data[off+40 : off+40+tln])
	if !ok || !isDocumentText(text) {
		return feChainNode{}, false
	}
	node.display = true
	node.text = text
	return node, true
}

func feWMFChainNodeForTest(data []byte, off int) (feChainNode, bool) {
	end, ok := wmfBlobEnd(data, off)
	if !ok || end+20 > len(data) || !bytes.Equal(data[end:end+4], []byte{0xff, 0xff, 0xff, 0xff}) {
		return feChainNode{}, false
	}
	return feChainNode{
		offset:  off,
		display: true,
		key: feChainKey{
			x: binary.LittleEndian.Uint32(data[off+16 : off+20]),
			y: binary.LittleEndian.Uint32(data[off+20 : off+24]),
		},
		next: feChainKey{
			x: binary.LittleEndian.Uint32(data[end+12 : end+16]),
			y: binary.LittleEndian.Uint32(data[end+16 : end+20]),
		},
	}, true
}

func TestParseETQEmbeddedWMFRects(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc, err := ParseETQ(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.WMFElements) != 6 {
		t.Fatalf("expected decoded WMF symbols, got %#v", doc.WMFElements)
	}
	seen := map[string]bool{}
	for _, sym := range doc.WMFElements {
		seen[sym.FilePath] = true
		if sym.XMM <= 0 || sym.YMM <= 0 || sym.WidthMM <= 0 || sym.HeightMM <= 0 {
			t.Fatalf("invalid decoded WMF rect: %#v", sym)
		}
	}
	for _, want := range []string{"lav-30.wmf", "clorox.wmf", "tamborx.wmf", "ferro-.wmf", "seco-w.wmf"} {
		if !seen[want] {
			t.Fatalf("missing WMF %s in %#v", want, doc.WMFElements)
		}
	}
}

func TestParseETQTemplateNameFromHeader(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc, err := ParseETQ(path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.TemplateName != "LNT-2" {
		t.Fatalf("expected LNT-2 template from ETQ header, got %q", doc.TemplateName)
	}
	if doc.LayoutType != "Etiq. para Composições em Folhas" {
		t.Fatalf("unexpected layout type %q", doc.LayoutType)
	}
}

func TestParseETQCompositionOrderUsesRawCoordinates(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "Canelado algodão (Classic Wave Ramado) lunelli.ETQ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	recs := extractTextRecords(data)
	var upper, lower *etqTextRecord
	for i := range recs {
		switch recs[i].Offset {
		case 0x15d3:
			upper = &recs[i]
		case 0x154b:
			lower = &recs[i]
		}
	}
	if upper == nil || lower == nil {
		t.Fatalf("expected composition records at fixed offsets")
	}
	if upper.RawY >= lower.RawY {
		t.Fatalf("expected offset 0x15d3 to be visually above 0x154b, got y=%d/%d", upper.RawY, lower.RawY)
	}
}

func TestParseETQRTFTextPayload(t *testing.T) {
	path := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS", "ADAR SOFA CANELADO.ETQ")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("sample ETQ not installed: %v", err)
	}
	doc, err := ParseETQ(path)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	fontSizes := map[string]float64{}
	bold := map[string]bool{}
	rtfRaw := map[string][]byte{}
	for _, txt := range doc.TextElements {
		seen[txt.Text] = true
		fontSizes[txt.Text] = txt.FontSize
		bold[txt.Text] = txt.Bold
		rtfRaw[txt.Text] = txt.RTFRaw
		if strings.Contains(txt.Text, "fonttbl") || strings.Contains(txt.Text, "Arial Narrow;") {
			t.Fatalf("RTF metadata leaked into text: %q", txt.Text)
		}
	}
	for _, want := range []string{"FEITO NO BRASIL", "13.240.553/0001-63", "99% POLIESTER", "1% ELASTANO"} {
		if !seen[want] {
			t.Fatalf("missing RTF/plain text %q in %#v", want, doc.TextElements)
		}
	}
	if !near(fontSizes["13.240.553/0001-63"], 5.74*72.0/25.4) {
		t.Fatalf("expected ETQ RECT height, not RTF \\fs18, got %.2f", fontSizes["13.240.553/0001-63"])
	}
	if !bold["FEITO NO BRASIL"] {
		t.Fatalf("expected RTF bold flag for FEITO NO BRASIL")
	}
	if raw := rtfRaw["FEITO NO BRASIL"]; !bytes.HasPrefix(raw, []byte("{\\rtf")) || !bytes.Contains(raw, []byte("\\b")) {
		t.Fatalf("expected raw RTF payload for FEITO NO BRASIL, got %q", string(raw))
	}
}
