package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"masterprint-native/internal/etq"
)

func main() {
	dir := `C:\Program Files (x86)\paulimaq\ARQUIVOS`
	entries, _ := os.ReadDir(dir)
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".ETQ") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		path := filepath.Join(dir, name)
		doc, err := etq.ParseETQ(path)
		if err != nil {
			fmt.Printf("ERR %s %v\n", name, err)
			continue
		}
		fmt.Printf("%s text=%d wmf=%d tmpl=%q\n", name, len(doc.TextElements), len(doc.WMFElements), doc.TemplateName)
	}
}
