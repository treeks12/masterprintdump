package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"masterprint-native/internal/etq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: etqdump <file-or-dir> [...]")
		os.Exit(2)
	}
	var files []string
	for _, arg := range os.Args[1:] {
		st, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", arg, err)
			os.Exit(1)
		}
		if !st.IsDir() {
			files = append(files, arg)
			continue
		}
		entries, err := os.ReadDir(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", arg, err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".etq") {
				files = append(files, filepath.Join(arg, entry.Name()))
			}
		}
	}
	sort.Strings(files)
	for _, path := range files {
		doc, err := etq.ParseETQ(path)
		if err != nil {
			fmt.Printf("FILE %s\nERROR %v\n\n", filepath.Base(path), err)
			continue
		}
		fmt.Printf("FILE %s\n", filepath.Base(path))
		fmt.Printf("  layoutType=%q template=%q printer=%q\n", doc.LayoutType, doc.TemplateName, doc.PrinterName)
		fmt.Printf("  texts=%d wmfs=%d unknown=%d\n", len(doc.TextElements), len(doc.WMFElements), len(doc.UnknownObjects))
		for _, t := range doc.TextElements {
			fmt.Printf("  TEXT off=%#x xy=%.2f,%.2f wh=%.2f,%.2f style=%d align=%s font=%q size=%.2f b=%v i=%v u=%v text=%q\n", t.FileOffset, t.XMM, t.YMM, t.WidthMM, t.HeightMM, t.StyleByte, t.Align, t.FontName, t.FontSize, t.Bold, t.Italic, t.Underline, t.Text)
		}
		for _, w := range doc.WMFElements {
			hash := ""
			if len(w.Embedded) >= 22 {
				sum := sha256.Sum256(w.Embedded[22:])
				hash = fmt.Sprintf("%x", sum[:8])
			}
			fmt.Printf("  WMF off=%#x xy=%.2f,%.2f wh=%.2f,%.2f style=%d file=%q bytes=%d bodySHA=%s\n", w.FileOffset, w.XMM, w.YMM, w.WidthMM, w.HeightMM, w.StyleByte, w.FilePath, len(w.Embedded), hash)
		}
		for _, u := range doc.UnknownObjects {
			fmt.Printf("  UNKNOWN off=%#x flags=%#x tag=%#x kind=%s\n", u.Offset, u.Flags, u.Tag, u.Kind)
		}
		fmt.Println()
	}
}
