package etq

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"masterprint-native/internal/model"
)

const (
	dfSignature   = 0x30465044
	dfSignatureEx = 0x30465044
)

type DFMReader struct {
	data []byte
	pos  int
}

func NewDFMReader(data []byte) *DFMReader {
	return &DFMReader{data: data, pos: 0}
}

func (r *DFMReader) ReadByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *DFMReader) ReadBytes(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, io.ErrUnexpectedEOF
	}
	result := r.data[r.pos : r.pos+n]
	r.pos += n
	return result, nil
}

func (r *DFMReader) ReadUint8() (uint8, error) {
	b, err := r.ReadByte()
	return uint8(b), err
}

func (r *DFMReader) ReadInt8() (int8, error) {
	b, err := r.ReadByte()
	return int8(b), err
}

func (r *DFMReader) ReadUint16() (uint16, error) {
	data, err := r.ReadBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(data), nil
}

func (r *DFMReader) ReadInt16() (int16, error) {
	v, err := r.ReadUint16()
	return int16(v), err
}

func (r *DFMReader) ReadUint32() (uint32, error) {
	data, err := r.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(data), nil
}

func (r *DFMReader) ReadInt32() (int32, error) {
	v, err := r.ReadUint32()
	return int32(v), err
}

func (r *DFMReader) ReadFloat32() (float32, error) {
	var v float32
	data, err := r.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &v)
	return v, nil
}

func (r *DFMReader) ReadString() (string, error) {
	lenByte, err := r.ReadByte()
	if err != nil {
		return "", err
	}

	length := int(lenByte)

	if length == 0 {
		return "", nil
	}

	if length >= 0xFE {
		hi, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		lo, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		length = int(hi)<<8 | int(lo)
	}

	data, err := r.ReadBytes(length)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *DFMReader) ReadWideString() (string, error) {
	lenByte, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	length := int(lenByte) * 2

	if length == 0 {
		return "", nil
	}

	data, err := r.ReadBytes(length)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			ch := uint16(data[i]) | uint16(data[i+1])<<8
			sb.WriteRune(rune(ch))
		}
	}
	return sb.String(), nil
}

func (r *DFMReader) Remaining() int {
	return len(r.data) - r.pos
}

type ETQFile struct {
	FilePath       string
	PreviewPath    string
	PrinterName    string
	LayoutType     string
	TemplateName   string
	TextElements   []model.TextElement
	WMFElements    []model.WMFSymbol
	UnknownObjects []ETQUnknownObject
	Flags          []byte
}

type ETQUnknownObject struct {
	Offset int
	Flags  uint32
	Tag    uint32
	Kind   string
}

func ParseETQ(path string) (*ETQFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lendo ETQ: %w", err)
	}
	etq, err := parseETQData(data, wmfResolverForETQPath(path))
	if err != nil {
		return nil, err
	}
	etq.FilePath = path
	etq.PreviewPath = findPreviewPath(path)
	return etq, nil
}

func findPreviewPath(path string) string {
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, name := range []string{base + ".png", "resized_" + strings.ToLower(base) + ".png"} {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	want := normalizePreviewName(base)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".png") {
			continue
		}
		name := strings.TrimPrefix(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), "resized_")
		if normalizePreviewName(name) == want {
			return filepath.Join(dir, entry.Name())
		}
	}
	return ""
}

func normalizePreviewName(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"Á", "a", "À", "a", "Ã", "a", "Â", "a", "Ä", "a",
		"é", "e", "ê", "e", "É", "e", "Ê", "e",
		"í", "i", "Í", "i",
		"ó", "o", "õ", "o", "ô", "o", "Ó", "o", "Õ", "o", "Ô", "o",
		"ú", "u", "Ú", "u",
		"ç", "c", "Ç", "c",
	)
	return strings.ToLower(strings.TrimSpace(replacer.Replace(s)))
}

func ParseETQData(data []byte) (*ETQFile, error) {
	return parseETQData(data, resolveEmbeddedWMFBySize)
}

type embeddedWMFResolver func(blob []byte, size int) (string, bool)

func parseETQData(data []byte, resolveWMF embeddedWMFResolver) (*ETQFile, error) {
	etq := &ETQFile{}
	etq.PrinterName = findFirstPrintableAfter(data, 0, []string{"Epson", "HP", "Canon", "Zebra"})
	etq.LayoutType, etq.TemplateName = extractHeaderLayout(data)
	textRecords := extractTextRecords(data)
	etq.TextElements = buildTextElements(textRecords)
	etq.WMFElements = extractEmbeddedWMFSymbolsWithResolver(data, resolveWMF)
	etq.UnknownObjects = extractUnknownObjects(data, textRecords, etq.WMFElements)
	return etq, nil
}

var feMarker = []byte{0xFE, 0xFF, 0xFF, 0xFF}

var rtfFontSizeRe = regexp.MustCompile(`\\fs(\d+)`)
var templateHeaderRe = regexp.MustCompile(`([A-Z]{2,5}-\d+)\s*\((\d+(?:,\d+)?)x(\d+(?:,\d+)?)mm\)`)

func extractHeaderLayout(data []byte) (layoutType, templateName string) {
	limit := len(data)
	if limit > 512 {
		limit = 512
	}
	var stringsFound []string
	for i := 0; i < limit; i++ {
		j := i
		for j < limit && data[j] >= 0x20 && data[j] < 0xff {
			j++
		}
		if j-i >= 4 {
			s := decodeLatin1(data[i:j])
			stringsFound = append(stringsFound, s)
			if m := templateHeaderRe.FindStringSubmatch(s); len(m) == 4 {
				templateName = m[1]
				if len(stringsFound) >= 2 {
					layoutType = stringsFound[len(stringsFound)-2]
				}
				return layoutType, templateName
			}
		}
		i = j
	}
	return "", ""
}

type etqTextRecord struct {
	Offset      int
	Flags       uint32
	Tag         uint32
	Text        string
	RawX        uint32
	RawY        uint32
	FontName    string
	FontStyle   byte
	Align       int16
	RectHeight  uint32
	RectWidth   uint32
	RTFFontSize float64
	RTFBold     bool
	PayloadRaw  []byte
	RTFRaw      []byte
	NextX       uint32
	NextY       uint32
}

func extractTextRecords(data []byte) []etqTextRecord {
	var out []etqTextRecord
	for i := 0; i+48 < len(data); i++ {
		if !bytes.Equal(data[i:i+4], feMarker) {
			continue
		}
		flags := binary.LittleEndian.Uint32(data[i+8 : i+12])
		tag := binary.LittleEndian.Uint32(data[i+12 : i+16])
		if flags != 0 || (tag != 1 && tag != 0) {
			continue
		}
		rawX := binary.LittleEndian.Uint32(data[i+16 : i+20])
		rawY := binary.LittleEndian.Uint32(data[i+20 : i+24])
		if i+40 > len(data) {
			continue
		}
		tln := int(binary.LittleEndian.Uint16(data[i+38 : i+40]))
		if tln < 1 || tln > 4096 || i+40+tln+4 > len(data) {
			continue
		}
		textBytes := data[i+40 : i+40+tln]
		text, rtfFontSize, rtfBold, ok := decodeTextPayload(textBytes)
		if !ok {
			continue
		}
		if !bytes.Equal(data[i+40+tln:i+40+tln+4], []byte{0xff, 0xff, 0xff, 0xff}) {
			continue
		}
		if !isDocumentText(text) {
			continue
		}
		fontName := extractFontName(data, i)
		align := int16(binary.LittleEndian.Uint16(data[i+32 : i+34]))
		if align < 0 || align > 2 {
			align = 0
		}
		fontStyle := extractFontStyle(data, i+40+tln+4)
		var rectHeight, rectWidth, nextX, nextY uint32
		postOff := i + 40 + tln + 4
		if postOff+16 <= len(data) {
			rectHeight = binary.LittleEndian.Uint32(data[postOff : postOff+4])
			rectWidth = binary.LittleEndian.Uint32(data[postOff+4 : postOff+8])
			nextX = binary.LittleEndian.Uint32(data[postOff+8 : postOff+12])
			nextY = binary.LittleEndian.Uint32(data[postOff+12 : postOff+16])
		}
		var rtfRaw []byte
		if bytes.HasPrefix(bytes.TrimRight(textBytes, "\x00"), []byte("{\\rtf")) {
			rtfRaw = append([]byte(nil), textBytes...)
		}
		out = append(out, etqTextRecord{Offset: i, Flags: flags, Tag: tag, Text: text, RawX: rawX, RawY: rawY, FontName: fontName, FontStyle: fontStyle, Align: align, RectHeight: rectHeight, RectWidth: rectWidth, RTFFontSize: rtfFontSize, RTFBold: rtfBold, PayloadRaw: append([]byte(nil), textBytes...), RTFRaw: rtfRaw, NextX: nextX, NextY: nextY})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Offset < out[j].Offset })
	return out
}

func extractUnknownObjects(data []byte, textRecords []etqTextRecord, wmfRecords []model.WMFSymbol) []ETQUnknownObject {
	known := make(map[int]struct{}, len(textRecords)+len(wmfRecords))
	for _, rec := range textRecords {
		known[rec.Offset] = struct{}{}
	}
	for _, rec := range wmfRecords {
		known[rec.FileOffset] = struct{}{}
	}
	var out []ETQUnknownObject
	for i := 0; i+40 < len(data); i++ {
		if !bytes.Equal(data[i:i+4], feMarker) {
			continue
		}
		if _, ok := known[i]; ok {
			continue
		}
		flags := binary.LittleEndian.Uint32(data[i+8 : i+12])
		tag := binary.LittleEndian.Uint32(data[i+12 : i+16])
		if flags == 0 && looksLikeTextPayload(data, i) {
			out = append(out, ETQUnknownObject{Offset: i, Flags: flags, Tag: tag, Kind: "text-like"})
		}
	}
	return out
}

func looksLikeTextPayload(data []byte, fePos int) bool {
	if fePos+40 > len(data) {
		return false
	}
	tln := int(binary.LittleEndian.Uint16(data[fePos+38 : fePos+40]))
	if tln < 1 || tln > 4096 || fePos+40+tln+4 > len(data) {
		return false
	}
	textBytes := data[fePos+40 : fePos+40+tln]
	text, _, _, ok := decodeTextPayload(textBytes)
	if !ok || !isDocumentText(text) {
		return false
	}
	return bytes.Equal(data[fePos+40+tln:fePos+40+tln+4], []byte{0xff, 0xff, 0xff, 0xff})
}

func extractFontName(data []byte, fePos int) string {
	start := fePos - 48
	if start < 0 {
		return "Arial"
	}
	nameStart := start + 3
	if nameStart >= len(data) {
		return "Arial"
	}
	end := nameStart
	for end < fePos && end < len(data) && data[end] != 0 {
		end++
	}
	name := string(data[nameStart:end])
	if len(name) < 2 {
		return "Arial"
	}
	return name
}

func extractFontStyle(data []byte, postOff int) byte {
	if postOff+17 > len(data) {
		return 5
	}
	return data[postOff+16]
}

func buildTextElements(recs []etqTextRecord) []model.TextElement {
	sorted := make([]etqTextRecord, len(recs))
	copy(sorted, recs)
	var out []model.TextElement
	for _, rec := range sorted {
		xMM := float64(rec.RawX) / 100.0
		yMM := float64(rec.RawY) / 100.0
		bold, italic, underline := textStyleFromByte(rec.FontStyle)
		bold = bold || rec.RTFBold
		fontSize := 8.0
		rectHMM := 0.0
		rectWMM := 0.0
		if rec.RectHeight > 0 {
			rectHMM = float64(rec.RectHeight) / 100.0
		}
		if rec.RectWidth > 0 {
			rectWMM = float64(rec.RectWidth) / 100.0
		}
		if rectHMM > 0 && rectHMM <= 8.0 {
			// CadMapa creates text fonts from the object RECT height. The first
			// post-text field behaves as that height for normal text records; the
			// remaining fields include linked-list coordinates and are not a font guess.
			fontSize = rectHMM * 72.0 / 25.4
		}
		out = append(out, model.TextElement{
			FileOffset: rec.Offset,
			FEFlags:    rec.Flags,
			FETag:      rec.Tag,
			Text:       rec.Text,
			PayloadRaw: append([]byte(nil), rec.PayloadRaw...),
			RTFRaw:     append([]byte(nil), rec.RTFRaw...),
			StyleByte:  rec.FontStyle,
			NextX:      rec.NextX,
			NextY:      rec.NextY,
			XMM:        xMM,
			YMM:        yMM,
			WidthMM:    rectWMM,
			HeightMM:   rectHMM,
			FontName:   rec.FontName,
			FontSize:   fontSize,
			Bold:       bold,
			Italic:     italic,
			Underline:  underline,
			Align:      alignToString(rec.Align),
		})
	}
	return out
}

func textStyleFromByte(style byte) (bold, italic, underline bool) {
	return style&0x01 != 0, style&0x02 != 0, style&0x04 != 0
}

func alignToString(align int16) string {
	switch align {
	case 1:
		return "center"
	case 2:
		return "right"
	default:
		return "left"
	}
}

var embeddedWMFBySize = map[int]string{
	476:   "cloro.wmf",
	632:   "secah.wmf",
	700:   "clorom.wmf",
	728:   "clorox.wmf",
	744:   "secas.wmf",
	856:   "secag.wmf",
	1226:  "secav.wmf",
	2816:  "secof.wmf",
	2828:  "secow.wmf",
	2904:  "secox.wmf",
	2928:  "seco-f.wmf",
	2990:  "ferro--.wmf",
	3166:  "tamborx.wmf",
	3262:  "seco-w.wmf",
	3316:  "lavx.wmf",
	3630:  "tambor-.wmf",
	3696:  "seco--w.wmf",
	4040:  "seco-p.wmf",
	4072:  "secop.wmf",
	4238:  "tambor--.wmf",
	4338:  "ferrox.wmf",
	5322:  "ferro-.wmf",
	6060:  "lav--40.wmf",
	6554:  "ferro---.wmf",
	6602:  "lav-40.wmf",
	6604:  "lav40.wmf",
	6868:  "lav70.wmf",
	7260:  "lavp--40.wmf",
	7304:  "lavp-40.wmf",
	7728:  "lavp40.wmf",
	7832:  "lav--30.wmf",
	8002:  "lav-50.wmf",
	8044:  "lav50.wmf",
	8272:  "lavp--30.wmf",
	8394:  "lav-60.wmf",
	8540:  "lav95.wmf",
	8558:  "lav-30.wmf",
	8560:  "lavp-30.wmf",
	8564:  "lav60.wmf",
	8632:  "lav30.wmf",
	8666:  "lav-95.wmf",
	9116:  "lavp30.wmf",
	9164:  "lavp-50.wmf",
	9640:  "lavp50.wmf",
	9748:  "lavp70.wmf",
	10570: "lavp-60.wmf",
	10740: "lavp60.wmf",
	12060: "lavp95.wmf",
	12778: "lavmao.wmf",
}

func extractEmbeddedWMFSymbols(data []byte) []model.WMFSymbol {
	return extractEmbeddedWMFSymbolsWithResolver(data, resolveEmbeddedWMFBySize)
}

func extractEmbeddedWMFSymbolsWithResolver(data []byte, resolveWMF embeddedWMFResolver) []model.WMFSymbol {
	if resolveWMF == nil {
		resolveWMF = resolveEmbeddedWMFBySize
	}
	var out []model.WMFSymbol
	for i := 0; i+64 < len(data); i++ {
		if !bytes.Equal(data[i:i+4], feMarker) {
			continue
		}
		flags := binary.LittleEndian.Uint32(data[i+8 : i+12])
		tag := binary.LittleEndian.Uint32(data[i+12 : i+16])
		if tag != 0 || flags != 0x80000008 {
			continue
		}
		aldusOff := i + 49
		if aldusOff+4 > len(data) {
			continue
		}
		if !bytes.Equal(data[aldusOff:aldusOff+4], []byte{0xd7, 0xcd, 0xc6, 0x9a}) {
			continue
		}
		std := aldusOff + 22
		if std+10 > len(data) {
			continue
		}
		words := int(binary.LittleEndian.Uint32(data[std+6 : std+10]))
		size := 22 + words*2
		if aldusOff+size > len(data) {
			continue
		}
		blob := data[aldusOff : aldusOff+size]
		name, ok := resolveWMF(blob, size)
		if !ok {
			name = ""
		}
		preOff := i - 83
		if preOff < 0 || preOff+20 > len(data) || !bytes.Equal(data[preOff:preOff+4], []byte{0xff, 0xff, 0xff, 0xff}) {
			continue
		}
		postX := binary.LittleEndian.Uint32(data[preOff+4 : preOff+8])
		postY := binary.LittleEndian.Uint32(data[preOff+8 : preOff+12])
		style := byte(0)
		var nextX, nextY uint32
		if end := aldusOff + size; end+21 <= len(data) {
			nextX = binary.LittleEndian.Uint32(data[end+12 : end+16])
			nextY = binary.LittleEndian.Uint32(data[end+16 : end+20])
			style = data[end+20]
		}
		headW := binary.LittleEndian.Uint32(data[i+16 : i+20])
		headH := binary.LittleEndian.Uint32(data[i+20 : i+24])
		if postX == 0 || postY == 0 || headW == 0 || headH == 0 {
			continue
		}
		out = append(out, model.WMFSymbol{
			FileOffset: i,
			FilePath:   name,
			Embedded:   append([]byte(nil), blob...),
			PreBlock:   append([]byte(nil), data[preOff:i]...),
			StyleByte:  style,
			NextX:      nextX,
			NextY:      nextY,
			XMM:        float64(postX) / 100.0,
			YMM:        float64(postY) / 100.0,
			WidthMM:    float64(headW) / 100.0,
			HeightMM:   float64(headH) / 100.0,
		})
	}
	return out
}

func resolveEmbeddedWMFBySize(_ []byte, size int) (string, bool) {
	name := embeddedWMFBySize[size]
	return name, name != ""
}

func wmfResolverForETQPath(path string) embeddedWMFResolver {
	idx := loadClipartWMFBodyIndex(path)
	return func(blob []byte, size int) (string, bool) {
		if len(blob) >= 22 && len(idx) > 0 {
			sum := sha256.Sum256(blob[22:])
			if name, ok := idx[sum]; ok {
				return name, true
			}
		}
		return resolveEmbeddedWMFBySize(blob, size)
	}
}

func loadClipartWMFBodyIndex(etqPath string) map[[32]byte]string {
	root := filepath.Dir(filepath.Dir(etqPath))
	dir := filepath.Join(root, "CLIPART", "Símbolos")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	idx := make(map[[32]byte]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".wmf") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil || len(data) < 22 {
			continue
		}
		idx[sha256.Sum256(data[22:])] = entry.Name()
	}
	return idx
}

func decodeTextPayload(data []byte) (string, float64, bool, bool) {
	data = bytes.TrimRight(data, "\x00")
	if len(data) == 0 {
		return "", 0, false, false
	}
	if bytes.HasPrefix(data, []byte("{\\rtf")) {
		rtf := decodeLatin1(data)
		text := strings.TrimSpace(decodeRTFPlainText(rtf))
		fontSize := 0.0
		if m := rtfFontSizeRe.FindStringSubmatch(rtf); len(m) == 2 {
			var halfPoints int
			_, _ = fmt.Sscanf(m[1], "%d", &halfPoints)
			if halfPoints > 0 {
				fontSize = float64(halfPoints) / 2.0
			}
		}
		return text, fontSize, rtfHasBold(rtf), text != ""
	}
	for _, b := range data {
		if b < 0x20 || b == 0x7f || b == 0xff {
			return "", 0, false, false
		}
	}
	return strings.TrimSpace(decodeLatin1(data)), 0, false, true
}

func decodeRTFPlainText(s string) string {
	var out strings.Builder
	depth := 0
	skipDepth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '{':
			depth++
			if skipDepth == 0 {
				j := i + 1
				if j < len(s) && s[j] == '\\' {
					j++
					start := j
					for j < len(s) && ((s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z')) {
						j++
					}
					word := s[start:j]
					if word == "fonttbl" || word == "colortbl" || word == "stylesheet" || word == "info" {
						skipDepth = depth
					}
				}
			}
			continue
		case '}':
			if skipDepth == depth {
				skipDepth = 0
			}
			if depth > 0 {
				depth--
			}
			continue
		case '\\':
			i++
			if i >= len(s) {
				break
			}
			if skipDepth != 0 {
				continue
			}
			if s[i] == '\\' || s[i] == '{' || s[i] == '}' {
				out.WriteByte(s[i])
				continue
			}
			if s[i] == '\'' && i+2 < len(s) {
				var v byte
				for j := 0; j < 2; j++ {
					v <<= 4
					ch := s[i+1+j]
					switch {
					case ch >= '0' && ch <= '9':
						v += ch - '0'
					case ch >= 'a' && ch <= 'f':
						v += ch - 'a' + 10
					case ch >= 'A' && ch <= 'F':
						v += ch - 'A' + 10
					}
				}
				out.WriteRune(rune(v))
				i += 2
				continue
			}
			start := i
			for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
				i++
			}
			word := s[start:i]
			for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '-') {
				i++
			}
			if word == "par" || word == "line" {
				out.WriteByte('\n')
			}
			if i < len(s) && s[i] != ' ' {
				i--
			}
		default:
			if skipDepth != 0 {
				continue
			}
			if c >= 0x20 || c == '\n' || c == '\r' || c == '\t' {
				out.WriteByte(c)
			}
		}
	}
	lines := strings.Fields(out.String())
	return strings.Join(lines, " ")
}

func rtfHasBold(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			continue
		}
		i++
		if i >= len(s) {
			break
		}
		if s[i] == '\\' || s[i] == '{' || s[i] == '}' {
			continue
		}
		if s[i] == '\'' {
			if i+2 < len(s) {
				i += 2
			}
			continue
		}
		start := i
		for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
			i++
		}
		word := s[start:i]
		paramStart := i
		for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '-') {
			i++
		}
		if word == "b" {
			if paramStart == i {
				return true
			}
			param := s[paramStart:i]
			if param != "0" && param != "-0" {
				return true
			}
		}
		if i < len(s) && s[i] != ' ' {
			i--
		}
	}
	return false
}

func isDocumentText(text string) bool {
	if len(strings.TrimSpace(text)) < 1 {
		return false
	}
	upper := strings.ToUpper(text)
	if upper == "ARIAL" || upper == "CONSOLAS" || upper == "MS SANS SERIF" || upper == "SERIF" || upper == "BVLFW" {
		return false
	}
	return strings.ContainsAny(upper, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789%:/.-")
}

func decodeLatin1(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		sb.WriteRune(rune(b))
	}
	return sb.String()
}

func findFirstPrintableAfter(data []byte, start int, prefixes []string) string {
	for _, prefix := range prefixes {
		idx := bytes.Index(data[start:], []byte(prefix))
		if idx < 0 {
			continue
		}
		idx += start
		end := idx
		for end < len(data) && data[end] >= 0x20 && data[end] < 0x7f {
			end++
		}
		return string(data[idx:end])
	}
	return ""
}

func extractNextTextElement(r *DFMReader) (*model.TextElement, error) {
	if r.Remaining() < 16 {
		return nil, io.ErrUnexpectedEOF
	}

	startPos := r.pos

	flags, _ := r.ReadBytes(3)
	_ = flags

	xRaw, _ := ReadSingleFromBytes(r)
	yRaw, _ := ReadSingleFromBytes(r)

	unknown, _ := r.ReadBytes(4)
	_ = unknown

	fontName, err := r.ReadString()
	if err != nil {
		r.pos = startPos
		return nil, err
	}
	fontName = strings.TrimRight(fontName, "\x00")

	fontData, _ := r.ReadBytes(12)

	fontSize := float32(8)
	if len(fontData) >= 4 {
		buf := bytes.NewReader(fontData[4:8])
		binary.Read(buf, binary.LittleEndian, &fontSize)
	}

	restLen, _ := r.ReadByte()
	if int(restLen) > 0 && restLen < 0xFE {
		_, _ = r.ReadBytes(int(restLen))
	}

	if fontName == "" && xRaw == 0 && yRaw == 0 {
		return nil, nil
	}

	text := ""

	return &model.TextElement{
		Text:     text,
		XMM:      float64(xRaw),
		YMM:      float64(yRaw),
		FontName: fontName,
		FontSize: float64(fontSize),
	}, nil
}

func ReadSingleFromBytes(r *DFMReader) (float32, error) {
	data, err := r.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	var v float32
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &v)
	return v, nil
}

func ReadDoubleFromBytes(r *DFMReader) (float64, error) {
	data, err := r.ReadBytes(8)
	if err != nil {
		return 0, err
	}
	var v float64
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &v)
	return v, nil
}

func ReadRawETQFields(data []byte) map[string]string {
	r := NewDFMReader(data)
	result := make(map[string]string)

	printer, err := r.ReadString()
	if err == nil {
		result["Printer"] = strings.TrimRight(printer, "\x00")
	}

	layoutType, err := r.ReadString()
	if err == nil {
		result["LayoutType"] = strings.TrimRight(layoutType, "\x00")
	}

	template, err := r.ReadString()
	if err == nil {
		result["Template"] = strings.TrimRight(template, "\x00")
	}

	textSearch(data, result)

	return result
}

func textSearch(data []byte, result map[string]string) {
	enc := binary.LittleEndian
	_ = enc

	for i := 0; i < len(data)-10; i++ {
		if isValidTextSeq(data, i) {
			end := i
			for end < len(data) && data[end] >= 0x20 && data[end] < 0x7F {
				end++
			}
			if end-i > 3 {
				text := string(data[i:end])
				key := fmt.Sprintf("Text_%d", i)
				result[key] = text
			}
		}
	}
}

func isValidTextSeq(data []byte, start int) bool {
	if start > 0 && data[start-1] < 0x20 {
		return false
	}
	count := 0
	for i := start; i < len(data) && data[i] >= 0x20 && data[i] < 0x7F; i++ {
		count++
	}
	return count >= 4
}
