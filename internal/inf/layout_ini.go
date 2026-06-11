package inf

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FieldDef struct {
	Name  string
	Size  int
	IsNum bool
}

type RecordDef struct {
	FileCodes []string
	Fields    []FieldDef
}

func ParseLayoutINI(path string) (map[string]RecordDef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]RecordDef)
	var currentSection string
	var currentDef RecordDef

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentSection != "" {
				result[currentSection] = currentDef
			}
			currentSection = line[1 : len(line)-1]
			currentDef = RecordDef{}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if key == "FileCode" {
			codes := strings.Split(val, ",")
			for i, c := range codes {
				codes[i] = strings.TrimSpace(c)
			}
			currentDef.FileCodes = codes
		} else {
			size, _ := strconv.Atoi(val)
			isNum := key == "Pagina" || key == "Matri" || key == "Paisa" ||
				key == "DobraCentral" || key == "InicioTira" || key == "AlturaTira" ||
				key == "NumCol" || key == "FileCode" ||
				strings.Contains(key, "Tam") ||
				strings.Contains(key, "Pos") ||
				strings.Contains(key, "Esp") ||
				strings.Contains(key, "Marg") ||
				strings.Contains(key, "Dobra") ||
				strings.Contains(key, "Inicio") ||
				strings.Contains(key, "Altura")
			currentDef.Fields = append(currentDef.Fields, FieldDef{
				Name:  key,
				Size:  size,
				IsNum: isNum,
			})
		}
	}

	if currentSection != "" {
		result[currentSection] = currentDef
	}

	return result, nil
}

func (r RecordDef) RecordSize() int {
	total := 0
	for _, f := range r.Fields {
		total += f.Size
	}
	return total
}

func (r RecordDef) FieldOffset(name string) (int, bool) {
	offset := 0
	for _, f := range r.Fields {
		if f.Name == name {
			return offset, true
		}
		offset += f.Size
	}
	return 0, false
}

func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func ParseINFWithLayout(infPath, layoutPath string) ([]string, error) {
	_ = layoutPath
	layouts, err := ParseFile(infPath)
	if err != nil {
		return nil, fmt.Errorf("parsing INF: %w", err)
	}

	names := make([]string, len(layouts))
	for i, l := range layouts {
		names[i] = l.Name
	}
	return names, nil
}
