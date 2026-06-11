package inf

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"masterprint-native/internal/model"
)

type Header struct {
	Unknown1 byte
	Unknown2 byte
	Code1    byte
	Code2    byte
}

func ParseFile(path string) ([]model.LayoutDefinition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	br := bufio.NewReader(f)

	hdr := make([]byte, 4)
	if _, err := io.ReadFull(br, hdr); err != nil {
		return nil, fmt.Errorf("lendo header: %w", err)
	}
	typeCode := infTypeCode(hdr)

	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("lendo descricao: %w", err)
	}
	_ = strings.TrimSpace(line)
	recDef, hasRecDef := recordDefForINF(path, typeCode)

	var layouts []model.LayoutDefinition
	for {
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if err == io.EOF {
				break
			}
			continue
		}

		var l model.LayoutDefinition
		var parseErr error
		if hasRecDef {
			l, parseErr = parseLayoutLineWithRecordDef(line, typeCode, recDef)
		} else {
			l, parseErr = parseLayoutLine(line)
			l.TypeCode = typeCode
		}
		if parseErr != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		layouts = append(layouts, l)

		if err == io.EOF {
			break
		}
	}

	return layouts, nil
}

func infTypeCode(hdr []byte) string {
	if len(hdr) < 2 {
		return ""
	}
	return normalizeFileCode(string(hdr[:2]))
}

func recordDefForINF(infPath, typeCode string) (RecordDef, bool) {
	layoutPath := filepath.Join(filepath.Dir(infPath), "layout.ini")
	recs, err := ParseLayoutINI(layoutPath)
	if err != nil {
		return RecordDef{}, false
	}
	if typeCode == "" {
		typeCode = "0"
	}
	for _, rec := range recs {
		for _, code := range rec.FileCodes {
			if normalizeFileCode(code) == normalizeFileCode(typeCode) {
				return rec, true
			}
		}
	}
	return RecordDef{}, false
}

func normalizeFileCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.TrimLeft(code, "0")
	if code == "" {
		return "0"
	}
	return code
}

func parseLayoutLineWithRecordDef(line, typeCode string, rec RecordDef) (model.LayoutDefinition, error) {
	values := make(map[string]string)
	off := 0
	for _, field := range rec.Fields {
		end := off + field.Size
		var raw string
		if off >= len(line) {
			raw = ""
		} else if end > len(line) {
			raw = line[off:]
		} else {
			raw = line[off:end]
		}
		values[field.Name] = strings.TrimSpace(raw)
		off = end
	}
	name := values["Nome"]
	if name == "" {
		return model.LayoutDefinition{}, fmt.Errorf("nome vazio em %q", line)
	}
	if idx := strings.Index(name, "("); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}
	numCol, copiesPerColumn := parseNumCol(values["NumCol"])
	widthMM := parseFloat(values["TamHoriz"])
	if widthMM == 0 {
		widthMM = parseFloat(values["TamHoriz1"])
	}
	heightMM := parseFloat(values["TamVert"])
	if heightMM == 0 {
		heightMM = parseFloat(values["TamVert1"])
	}
	return model.LayoutDefinition{
		TypeCode:        normalizeFileCode(typeCode),
		Name:            name,
		NumCol:          numCol,
		CopiesPerColumn: copiesPerColumn,
		WidthMM:         widthMM,
		HeightMM:        heightMM,
		MarginLeft:      parseFloat(values["MargEsq"]),
		MarginTop:       parseFloat(values["MargSup"]),
		SpacingCol:      parseFloat(values["EspEntrCol"]),
		SpacingRow:      parseFloat(values["EspEntrLin"]),
		Landscape:       parseInt(values["Paisa"]),
		Fields:          values,
	}, nil
}

func parseLayoutLine(line string) (model.LayoutDefinition, error) {
	const nameFieldWidth = 35

	namePart := line
	dataPart := ""
	if len(line) > nameFieldWidth {
		namePart = line[:nameFieldWidth]
		dataPart = line[nameFieldWidth:]
	}

	name := strings.TrimSpace(namePart)
	if idx := strings.Index(name, "("); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}

	data := strings.Fields(dataPart)
	if len(data) < 9 {
		return model.LayoutDefinition{}, fmt.Errorf("campos insuficientes: need 9, got %d in %q", len(data), line)
	}

	numCol, copiesPerColumn := parseNumCol(data[0])
	widthMM := parseFloat(data[1])
	heightMM := parseFloat(data[2])
	marginLeft := parseFloat(data[3])
	marginTop := parseFloat(data[4])
	spacingCol := parseFloat(data[5])
	spacingRow := parseFloat(data[6])

	var flip, rotation, landscape int
	if len(data) > 7 {
		flip = parseInt(data[7])
	}
	if len(data) > 8 {
		rotation = parseInt(data[8])
	}
	if len(data) > 9 {
		landscape = parseInt(data[9])
	}

	return model.LayoutDefinition{
		Name:            name,
		NumCol:          numCol,
		CopiesPerColumn: copiesPerColumn,
		WidthMM:         widthMM,
		HeightMM:        heightMM,
		MarginLeft:      marginLeft,
		MarginTop:       marginTop,
		SpacingCol:      spacingCol,
		SpacingRow:      spacingRow,
		Flip:            flip,
		Rotation:        rotation,
		Landscape:       landscape,
	}, nil
}

func parseNumCol(s string) (cols, copiesPerColumn int) {
	s = strings.TrimSpace(strings.ToLower(s))
	if parts := strings.Split(s, "x"); len(parts) == 2 {
		return parseInt(parts[0]), parseInt(parts[1])
	}
	return parseInt(s), 0
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.Atoi(s)
	return v
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.ParseFloat(s, 64)
	return math.Round(v*100) / 100
}

func ParsePageOverride(path string) (map[string]model.PrintPage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]model.PrintPage)
	var currentSection string
	var currentPage model.PrintPage

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentSection != "" {
				result[currentSection] = currentPage
			}
			currentSection = line[1 : len(line)-1]
			currentPage = model.PrintPage{Overrides: make(map[int]int)}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if currentSection != "" {
			idx := parseInt(key)
			currentPage.Overrides[idx] = parseInt(val)
			if currentPage.LabelsPerPage < parseInt(val) {
				currentPage.LabelsPerPage = parseInt(val)
			}
		}
	}

	if currentSection != "" {
		result[currentSection] = currentPage
	}

	return result, nil
}
