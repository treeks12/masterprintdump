//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tailscale/walk"
	"masterprint-native/internal/etq"
)

const (
	savePromptTitle       = "Salvar Arquivo"
	savePromptModifiedMsg = "O documento foi alterado. Deseja salvar as alteracoes?"
	dirtyTitleSuffix      = "*"
)

type sidecarLoadOutcome struct {
	Applied       bool
	AppliedPath   string
	IgnoredReason string
}

type sidecarCandidateData struct {
	Path string
	JSON []byte
}

func (a *App) isDocumentDirty() bool {
	return len(a.undoStack) > 0
}

func (a *App) markDocumentSaved() {
	a.etqBaseline = snapshotETQElements(a.elements)
	a.resetHistory()
	a.updateWindowTitle()
}

func (a *App) confirmDiscardChanges() bool {
	if a.currentLayout == nil || !a.isDocumentDirty() {
		return true
	}
	switch walk.MsgBox(a.mainWindow, savePromptTitle, savePromptModifiedMsg, walk.MsgBoxYesNoCancel|walk.MsgBoxIconQuestion) {
	case walk.DlgCmdYes:
		return a.saveWithGate()
	case walk.DlgCmdNo:
		return true
	default:
		return false
	}
}

func (a *App) saveWithGate() bool {
	if a.currentLayout == nil {
		return false
	}
	if a.currentDocPath == "" {
		a.onSaveAs()
		return a.currentDocPath != "" && !a.isDocumentDirty()
	}
	a.onSave()
	return !a.isDocumentDirty()
}

func documentWindowTitle(base string, dirty bool) string {
	if !dirty || base == "" {
		return base
	}
	return base + dirtyTitleSuffix
}

func (a *App) windowTitleBase() string {
	var base string
	if a.currentDocPath != "" {
		base = fmt.Sprintf("Paulimaq MasterPrint 3.0- %s", a.currentDocPath)
	} else if a.currentLayout != nil {
		base = fmt.Sprintf("MasterPrint 3.0 - %s (%s) %.1fx%.1fmm", a.currentLayout.Name, a.currentLayoutType, a.currentLayout.WidthMM, a.currentLayout.HeightMM)
	} else {
		base = "Paulimaq MasterPrint 3.0"
	}
	if clause := unknownObjectsStatusClause(len(a.unknownObjects)); clause != "" {
		base += " [" + clause + "]"
	}
	return base
}

func (a *App) updateWindowTitle() {
	if a.mainWindow == nil {
		return
	}
	a.mainWindow.SetTitle(documentWindowTitle(a.windowTitleBase(), a.isDocumentDirty()))
}

func (a *App) setPersistenceStatus(text string) {
	a.persistenceStatus = text
	a.updateStatus()
}

func persistenceTargetLabel(path string) string {
	if path == "" {
		return "documento"
	}
	if isMPNDocument(path) {
		return ".mpn"
	}
	return ".ETQ"
}

func openStatusText(sourcePath, layoutName string, sidecar sidecarLoadOutcome, unknownCount int) string {
	target := persistenceTargetLabel(sourcePath)
	msg := fmt.Sprintf("Aberto: %s", target)
	if layoutName != "" {
		msg = fmt.Sprintf("Aberto: %s (%s)", target, layoutName)
	}
	if !isMPNDocument(sourcePath) {
		msg += "; " + sidecarStatusClause(sidecar)
	}
	if clause := unknownObjectsStatusClause(unknownCount); clause != "" {
		msg += "; " + clause
	}
	return msg
}

func unknownObjectsStatusClause(count int) string {
	if count <= 0 {
		return ""
	}
	if count == 1 {
		return "1 objeto desconhecido"
	}
	return fmt.Sprintf("%d objetos desconhecidos", count)
}

func unknownObjectsWarningText(objects []etq.ETQUnknownObject) string {
	if len(objects) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "O arquivo contem %d objeto(s) nao suportado(s) pelo editor.\n\n", len(objects))
	limit := len(objects)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		obj := objects[i]
		kind := obj.Kind
		if kind == "" {
			kind = "desconhecido"
		}
		fmt.Fprintf(&b, "  offset 0x%X  %s\n", obj.Offset, kind)
	}
	if len(objects) > limit {
		fmt.Fprintf(&b, "  ... e mais %d\n", len(objects)-limit)
	}
	b.WriteString("\nEsses objetos nao sao editados nem impressos por este editor. Salvar no .ETQ permanece bloqueado enquanto existirem objetos desconhecidos.")
	return b.String()
}

func (a *App) notifyUnknownObjectsIfAny() {
	if a.mainWindow == nil || len(a.unknownObjects) == 0 {
		return
	}
	walk.MsgBox(a.mainWindow, "Objetos desconhecidos", unknownObjectsWarningText(a.unknownObjects), walk.MsgBoxIconExclamation)
}

func isStubMenuLabel(label string) bool {
	switch label {
	case "Salvar Como Modelo...", "Reabrir", "Exportar...", "Configurar Documento",
		"Borda", "Preenchimento", "Fonte", "A&grupar", "Desagr&upar", "&Alinhar", "Escalonar":
		return true
	default:
		return false
	}
}

func isStubToolbarCommand(id string) bool {
	switch id {
	case "alignDialog", "group", "ungroup", "mapaRisc", "ole", "fileMan", "field",
		"db", "dbopen", "dbtable", "bullets", "stop", "help", "mergeToggle",
		"navFirst", "navPrev", "navNext", "navLast":
		return true
	default:
		return false
	}
}

func stubFeatureMessage(feature string) string {
	return feature + ": funcao nao implementada ou nao comprovada."
}

func sidecarStatusClause(sidecar sidecarLoadOutcome) string {
	if sidecar.Applied {
		return "auxiliar aplicado"
	}
	if sidecar.IgnoredReason != "" {
		return "auxiliar ignorado: " + sidecar.IgnoredReason
	}
	return "sem documento auxiliar"
}

func evaluateSidecarLoad(candidates []sidecarCandidateData, expectedLayout string, allowLegacy, allowMismatch bool) sidecarLoadOutcome {
	if len(candidates) == 0 {
		return sidecarLoadOutcome{}
	}
	var lastIgnore string
	for _, candidate := range candidates {
		var doc savedDocument
		if err := json.Unmarshal(candidate.JSON, &doc); err != nil {
			lastIgnore = fmt.Sprintf("JSON invalido (%s)", filepath.Base(candidate.Path))
			continue
		}
		if doc.LayoutName == "" && !allowLegacy {
			lastIgnore = "legado sem layoutName"
			continue
		}
		if doc.LayoutName != "" && expectedLayout != "" && !strings.EqualFold(doc.LayoutName, expectedLayout) && !allowMismatch {
			lastIgnore = fmt.Sprintf("layout %q diferente de %q", doc.LayoutName, expectedLayout)
			continue
		}
		return sidecarLoadOutcome{Applied: true, AppliedPath: candidate.Path}
	}
	return sidecarLoadOutcome{IgnoredReason: lastIgnore}
}
