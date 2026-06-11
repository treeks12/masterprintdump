//go:build windows

package main

import (
	"strings"
	"testing"

	"masterprint-native/internal/etq"
	"masterprint-native/internal/model"
)

func TestDocumentWindowTitle(t *testing.T) {
	base := "Paulimaq MasterPrint 3.0- C:\\doc.ETQ"
	if got := documentWindowTitle(base, false); got != base {
		t.Fatalf("clean title=%q want %q", got, base)
	}
	if got := documentWindowTitle(base, true); got != base+"*" {
		t.Fatalf("dirty title=%q", got)
	}
	if got := documentWindowTitle("", true); got != "" {
		t.Fatalf("empty title=%q", got)
	}
}

func TestPersistenceTargetLabel(t *testing.T) {
	cases := map[string]string{
		"":             "documento",
		`C:\a\doc.ETQ`: ".ETQ",
		`C:\a\doc.mpn`: ".mpn",
		`C:\a\doc.MPN`: ".mpn",
	}
	for path, want := range cases {
		if got := persistenceTargetLabel(path); got != want {
			t.Fatalf("persistenceTargetLabel(%q)=%q want %q", path, got, want)
		}
	}
}

func TestOpenStatusText(t *testing.T) {
	got := openStatusText(`C:\a\LNT-2.ETQ`, "LNT-2", sidecarLoadOutcome{Applied: true}, 0)
	want := "Aberto: .ETQ (LNT-2); auxiliar aplicado"
	if got != want {
		t.Fatalf("open ETQ applied=%q want %q", got, want)
	}
	got = openStatusText(`C:\a\LNT-2.ETQ`, "LNT-2", sidecarLoadOutcome{IgnoredReason: `layout "X" diferente de "LNT-2"`}, 2)
	want = `Aberto: .ETQ (LNT-2); auxiliar ignorado: layout "X" diferente de "LNT-2"; 2 objetos desconhecidos`
	if got != want {
		t.Fatalf("open ETQ ignored=%q want %q", got, want)
	}
	got = openStatusText(`C:\a\doc.mpn`, "LNT-2", sidecarLoadOutcome{}, 0)
	want = "Aberto: .mpn (LNT-2)"
	if got != want {
		t.Fatalf("open mpn=%q want %q", got, want)
	}
}

func TestEvaluateSidecarLoad(t *testing.T) {
	valid := sidecarCandidateData{
		Path: `C:\a\doc.ETQ.masterprint-native.json`,
		JSON: []byte(`{"layoutName":"LNT-2","elements":[]}`),
	}
	out := evaluateSidecarLoad([]sidecarCandidateData{valid}, "LNT-2", false, false)
	if !out.Applied || out.AppliedPath != valid.Path || out.IgnoredReason != "" {
		t.Fatalf("applied=%#v", out)
	}
	mismatch := sidecarCandidateData{
		Path: `C:\a\doc.ETQ.masterprint-native.json`,
		JSON: []byte(`{"layoutName":"ADAR","elements":[]}`),
	}
	out = evaluateSidecarLoad([]sidecarCandidateData{mismatch}, "LNT-2", false, false)
	if out.Applied || out.IgnoredReason != `layout "ADAR" diferente de "LNT-2"` {
		t.Fatalf("layout mismatch=%#v", out)
	}
	legacy := sidecarCandidateData{
		Path: `C:\a\doc.ETQ.masterprint-native.json`,
		JSON: []byte(`{"elements":[]}`),
	}
	out = evaluateSidecarLoad([]sidecarCandidateData{legacy}, "LNT-2", false, false)
	if out.Applied || out.IgnoredReason != "legado sem layoutName" {
		t.Fatalf("legacy=%#v", out)
	}
	out = evaluateSidecarLoad(nil, "LNT-2", false, false)
	if out.Applied || out.IgnoredReason != "" {
		t.Fatalf("empty=%#v", out)
	}
}

func TestSidecarStatusClause(t *testing.T) {
	if got := sidecarStatusClause(sidecarLoadOutcome{Applied: true}); got != "auxiliar aplicado" {
		t.Fatalf("applied=%q", got)
	}
	if got := sidecarStatusClause(sidecarLoadOutcome{IgnoredReason: "JSON invalido (x.json)"}); got != "auxiliar ignorado: JSON invalido (x.json)" {
		t.Fatalf("ignored=%q", got)
	}
	if got := sidecarStatusClause(sidecarLoadOutcome{}); got != "sem documento auxiliar" {
		t.Fatalf("missing=%q", got)
	}
}

func TestUnknownObjectsStatusClause(t *testing.T) {
	if got := unknownObjectsStatusClause(0); got != "" {
		t.Fatalf("zero=%q", got)
	}
	if got := unknownObjectsStatusClause(1); got != "1 objeto desconhecido" {
		t.Fatalf("one=%q", got)
	}
	if got := unknownObjectsStatusClause(2); got != "2 objetos desconhecidos" {
		t.Fatalf("many=%q", got)
	}
}

func TestUnknownObjectsWarningText(t *testing.T) {
	got := unknownObjectsWarningText(nil)
	if got != "" {
		t.Fatalf("empty=%q", got)
	}
	objects := []etq.ETQUnknownObject{
		{Offset: 0x72c, Kind: "text-like"},
		{Offset: 0x800, Kind: ""},
	}
	got = unknownObjectsWarningText(objects)
	if !strings.Contains(got, "2 objeto(s) nao suportado(s)") {
		t.Fatalf("count missing: %q", got)
	}
	if !strings.Contains(got, "offset 0x72C  text-like") {
		t.Fatalf("first detail missing: %q", got)
	}
	if !strings.Contains(got, "offset 0x800  desconhecido") {
		t.Fatalf("blank kind missing: %q", got)
	}
	if !strings.Contains(got, "Salvar no .ETQ permanece bloqueado enquanto existirem objetos desconhecidos") {
		t.Fatalf("save note missing: %q", got)
	}
}

func TestIsStubMenuLabel(t *testing.T) {
	stubs := []string{"Salvar Como Modelo...", "Borda", "&Alinhar"}
	for _, label := range stubs {
		if !isStubMenuLabel(label) {
			t.Fatalf("%q should be stub", label)
		}
	}
	if isStubMenuLabel("Salvar") || isStubMenuLabel("Propriedades") {
		t.Fatal("implemented commands must not be stubs")
	}
}

func TestIsStubToolbarCommand(t *testing.T) {
	stubs := []string{"group", "mapaRisc", "navNext", "help"}
	for _, id := range stubs {
		if !isStubToolbarCommand(id) {
			t.Fatalf("%q should be stub", id)
		}
	}
	if isStubToolbarCommand("save") || isStubToolbarCommand("image") {
		t.Fatal("implemented commands must not be stubs")
	}
}

func TestAppWindowTitleBaseWithUnknownObjects(t *testing.T) {
	a := NewApp()
	a.currentDocPath = `C:\docs\test.ETQ`
	a.unknownObjects = []etq.ETQUnknownObject{{Offset: 0x100, Kind: "text-like"}}
	want := `Paulimaq MasterPrint 3.0- C:\docs\test.ETQ [1 objeto desconhecido]`
	if got := a.windowTitleBase(); got != want {
		t.Fatalf("title=%q want %q", got, want)
	}
}

func TestAppWindowTitleBaseAndDirty(t *testing.T) {
	a := NewApp()
	a.currentLayout = &model.LayoutDefinition{Name: "LNT-2", WidthMM: 25, HeightMM: 55}
	a.currentLayoutType = "Etiq. para Composições em Folhas"
	want := "MasterPrint 3.0 - LNT-2 (Etiq. para Composições em Folhas) 25.0x55.0mm"
	if got := a.windowTitleBase(); got != want {
		t.Fatalf("layout base=%q want %q", got, want)
	}
	a.currentDocPath = `C:\docs\test.ETQ`
	want = `Paulimaq MasterPrint 3.0- C:\docs\test.ETQ`
	if got := a.windowTitleBase(); got != want {
		t.Fatalf("path base=%q want %q", got, want)
	}
	if got := documentWindowTitle(a.windowTitleBase(), true); got != want+"*" {
		t.Fatalf("dirty path title=%q", got)
	}
}
