//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"masterprint-native/internal/etq"
	"masterprint-native/internal/model"
)

const (
	mpnDocumentExt   = ".mpn"
	mpnDocumentKind  = "native"
	mpnSchemaVersion = 2
)

func isMPNDocument(path string) bool {
	return strings.EqualFold(filepath.Ext(path), mpnDocumentExt)
}

func ensureMPNExtension(path string) string {
	if strings.EqualFold(filepath.Ext(path), mpnDocumentExt) {
		return path
	}
	return path + mpnDocumentExt
}

func (a *App) hasETQSource() bool {
	return a.etqSourcePath != ""
}

func (a *App) buildSavedDocument(schemaVersion int, documentKind string) savedDocument {
	doc := savedDocument{SchemaVersion: schemaVersion, DocumentKind: documentKind, PrinterName: a.currentPrinter, LayoutType: a.currentLayoutType, TemplateName: a.currentTemplate}
	if a.currentLayout != nil {
		doc.LayoutName = a.currentLayout.Name
	}
	for _, obj := range a.unknownObjects {
		doc.UnknownObjects = append(doc.UnknownObjects, savedUnknownObject{Offset: obj.Offset, Flags: obj.Flags, Tag: obj.Tag, Kind: obj.Kind})
	}
	for _, el := range a.elements {
		if el.Type == "preview" {
			continue
		}
		doc.Elements = append(doc.Elements, savedElementFromLabel(el))
	}
	return doc
}

func savedElementFromLabel(el LabelElement) savedElement {
	return savedElement{Type: el.Type, FileOffset: el.FileOffset, FEFlags: el.FEFlags, FETag: el.FETag, PayloadRaw: el.PayloadRaw, RTFRaw: el.RTFRaw, WMFRaw: el.WMFRaw, WMFPreRaw: el.WMFPreRaw, StyleByte: el.StyleByte, NextX: el.NextX, NextY: el.NextY, XMM: el.XMM, YMM: el.YMM, WidthMM: el.WidthMM, HeightMM: el.HeightMM, Text: el.Text, FontName: el.FontName, FontSize: el.FontSize, Bold: el.Bold, Italic: el.Italic, Underline: el.Underline, ImagePath: el.ImagePath, SymbolName: el.SymbolName, Align: el.Align}
}

func (a *App) saveMPNDocument(path string) error {
	path = ensureMPNExtension(path)
	data, err := json.MarshalIndent(a.buildSavedDocument(mpnSchemaVersion, mpnDocumentKind), "", "  ")
	if err != nil {
		return fmt.Errorf("salvar documento: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("salvar documento: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("salvar documento: %w", err)
	}
	return nil
}

func (a *App) loadMPNDocument(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc savedDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("documento .mpn invalido: %w", err)
	}
	if doc.DocumentKind != mpnDocumentKind {
		return fmt.Errorf("documento .mpn invalido: falta documentKind %q", mpnDocumentKind)
	}
	layout, err := a.resolveLayoutFromSaved(doc)
	if err != nil {
		return err
	}
	a.currentLayout = layout
	a.currentDocPath = path
	a.currentPrinter = doc.PrinterName
	a.currentLayoutType = doc.LayoutType
	a.currentTemplate = doc.TemplateName
	a.etqSourcePath = ""
	a.etqBaseline = nil
	a.unknownObjects = nil
	a.applySavedDocument(doc)
	a.markDocumentSaved()
	layoutName := doc.LayoutName
	if a.currentLayout != nil && layoutName == "" {
		layoutName = a.currentLayout.Name
	}
	a.persistenceStatus = openStatusText(path, layoutName, sidecarLoadOutcome{}, len(a.unknownObjects))
	a.updateWindowTitle()
	a.restoreDocumentView()
	a.invalidateCanvas()
	a.updateStatus()
	a.notifyUnknownObjectsIfAny()
	return nil
}

func (a *App) resolveLayoutFromSaved(doc savedDocument) (*model.LayoutDefinition, error) {
	if doc.LayoutName == "" {
		return nil, fmt.Errorf("documento .mpn invalido: layoutName ausente")
	}
	category := layoutCategoryFromHeader(doc.LayoutType)
	if category == "" {
		category = doc.LayoutType
	}
	if category != "" {
		if layout := a.findLayout(category, doc.LayoutName); layout != nil {
			return layout, nil
		}
	}
	return nil, fmt.Errorf("layout %s/%s nao encontrado", category, doc.LayoutName)
}

func (a *App) applySavedElements(doc savedDocument) {
	a.elements = nil
	a.selectedIdx = -1
	a.resetHistory()
	for _, se := range doc.Elements {
		el := LabelElement{ID: nextElemID, Type: se.Type, FileOffset: se.FileOffset, FEFlags: se.FEFlags, FETag: se.FETag, PayloadRaw: se.PayloadRaw, RTFRaw: se.RTFRaw, WMFRaw: se.WMFRaw, WMFPreRaw: se.WMFPreRaw, StyleByte: se.StyleByte, NextX: se.NextX, NextY: se.NextY, XMM: se.XMM, YMM: se.YMM, WidthMM: se.WidthMM, HeightMM: se.HeightMM, Text: se.Text, FontName: se.FontName, FontSize: se.FontSize, Bold: se.Bold, Italic: se.Italic, Underline: se.Underline, ImagePath: se.ImagePath, SymbolName: se.SymbolName, Align: se.Align, Color: color.Black}
		nextElemID++
		a.elements = append(a.elements, el)
	}
}

func (a *App) applySavedDocument(doc savedDocument) {
	if doc.PrinterName != "" {
		a.currentPrinter = doc.PrinterName
	}
	if len(doc.UnknownObjects) > 0 {
		a.unknownObjects = nil
		for _, obj := range doc.UnknownObjects {
			a.unknownObjects = append(a.unknownObjects, etq.ETQUnknownObject{Offset: obj.Offset, Flags: obj.Flags, Tag: obj.Tag, Kind: obj.Kind})
		}
	}
	a.applySavedElements(doc)
}
