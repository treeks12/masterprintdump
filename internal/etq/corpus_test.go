package etq

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type corpusExpectation struct {
	text     int
	wmf      int
	unknown  int
	template string
}

var corpusBaseline = map[string]corpusExpectation{
	"96% 4% THYRTY PLUS LISO.ETQ":                          {text: 6, wmf: 7, template: "LNT-2"},
	"ADAR SOFA CANELADO.ETQ":                               {text: 5, wmf: 6, template: "LNT-2"},
	"ALGODÃO.ETQ":                                          {text: 5, wmf: 7, template: "LNT-2"},
	"BRAND VISCOSE PLANO MACONHA.ETQ":                      {text: 7, wmf: 5, template: ""},
	"Brilhoso Shine golden eccentric.ETQ":                  {text: 9, wmf: 6, template: "LNT-2"},
	"CANELADO STRECH CHLOE ADAR (artec).ETQ":               {text: 7, wmf: 6, template: "LNT-2"},
	"CARACOL TULE.ETQ":                                     {text: 10, wmf: 7, template: "LNT-2"},
	"CIRRÊ.ETQ":                                            {text: 7, wmf: 7, template: "LNT-2"},
	"Canelado Ribana Linen.ETQ":                            {text: 8, wmf: 7, template: "LNT-2"},
	"Canelado algodão (Classic Wave Ramado) lunelli.ETQ":   {text: 7, wmf: 6, template: "LNT-2"},
	"DENSE 2 SUECO.ETQ":                                    {text: 7, wmf: 7, template: "LNT-2"},
	"FAVERO.ETQ":                                           {text: 7, wmf: 7, template: "LNT-2"},
	"LASER 4 - VICENZA E LINO.ETQ":                         {text: 10, wmf: 7, template: "LNT-4"},
	"LINHO DIAGONAL ELASTANO RAYON.ETQ":                    {text: 8, wmf: 6, template: "LNT-2"},
	"LINHO DIAGONAL.ETQ":                                   {text: 9, wmf: 5, template: "LNT-2"},
	"LINHO JOHNNY.ETQ":                                     {text: 6, wmf: 7, template: "LNT-2"},
	"LINHO LISTRADO TRENDY.ETQ":                            {text: 8, wmf: 7, template: "LNT-2"},
	"LINHO MONET BRAND (BARRADO ESTAMPA CRESCENTE).ETQ":    {text: 7, wmf: 5, template: "LNT-2"},
	"LINO LUNELLI (LINO CRAFT).ETQ":                        {text: 9, wmf: 6, template: "LNT-2"},
	"LINO STRIPE OFF-OURO.ETQ":                             {text: 8, wmf: 6, template: "LNT-2"},
	"Lore sueco (crepe).ETQ":                               {text: 7, wmf: 7, template: "LNT-2"},
	"Lã da CENTRAL.ETQ":                                    {text: 6, wmf: 5, template: "LNT-2"},
	"MALHA LAISE (tecido adar FURADO).ETQ":                 {text: 7, wmf: 5, template: "LNT-2"},
	"MALHA LINEN - linho nanetti.ETQ":                      {text: 8, wmf: 7, template: "LNT-2"},
	"MALHA TRICOT NANETE.ETQ":                              {text: 7, wmf: 7, template: "LNT-2"},
	"MOLETINHO KRAFT BIG LINE.ETQ":                         {text: 8, wmf: 6, template: "LNT-2"},
	"MOLETINHO NANETE.ETQ":                                 {text: 7, wmf: 6, template: "LNT-2"},
	"MOLETINHO SUECO.ETQ":                                  {text: 7, wmf: 7, template: "LNT-2"},
	"MOLETINHO.ETQ":                                        {text: 7, wmf: 7, template: "LNT-2"},
	"PETIT POA.ETQ":                                        {text: 7, wmf: 7, template: ""},
	"POLIAMIDA FARBE.ETQ":                                  {text: 7, wmf: 7, template: "LNT-2"},
	"RIBANA CANELADO NEON - 98% 2%.ETQ":                    {text: 5, wmf: 5, template: "LNT-2"},
	"SLIM.ETQ":                                             {text: 7, wmf: 7, template: "LNT-2"},
	"STRETTO TWIST RAMADO (viscolycra grossa lunelli).ETQ": {text: 7, wmf: 7, template: "LNT-2"},
	"STRIPE GOLDEN RAMADO.ETQ":                             {text: 7, wmf: 7, template: ""},
	"SUECO TURQUIA (POMBO CADEIRA).ETQ":                    {text: 8, wmf: 7, template: ""},
	"SUEDE ERRADO.ETQ":                                     {text: 5, wmf: 5, template: "LNT-2"},
	"SUEDE MAKRO CENTRAL.ETQ":                              {text: 7, wmf: 5, template: "LNT-2"},
	"Viscolycra UNICA.ETQ":                                 {text: 7, wmf: 7, template: "LNT-2"},
	"Viscolycra com Tamanho.ETQ":                           {text: 7, wmf: 7, template: ""},
	"bengaline possivelmente errado.ETQ":                   {text: 8, wmf: 5, template: "LNT-2"},
	"brand modal bosque goya.ETQ":                          {text: 8, wmf: 5, template: "LNT-2"},
	"canelado sueco.ETQ":                                   {text: 7, wmf: 7, template: "LNT-2"},
	"devorê e forro podrinha.ETQ":                          {text: 10, wmf: 7, template: "LNT-4"},
	"favero e podrinha.ETQ":                                {text: 10, wmf: 7, template: "LNT-2"},
	"hering - ALGODÃO SUECO (braided cotton).ETQ":          {text: 8, wmf: 6, template: "LNT-2"},
	"jeans.ETQ":                                            {text: 6, wmf: 5, template: "LNT-2"},
	"linen listrado nanetti.ETQ":                           {text: 9, wmf: 7, template: "LNT-2"},
	"linen nanetti listrado.ETQ":                           {text: 8, wmf: 7, template: "LNT-2"},
	"linho bordado diagonal.ETQ":                           {text: 9, wmf: 5, template: "LNT-2"},
	"lino trendy stripe mescla.ETQ":                        {text: 8, wmf: 7, template: "LNT-2"},
	"malha tricot everest makro.ETQ":                       {text: 5, wmf: 6, template: "LNT-2"},
	"tecido furado poliester ADAR.ETQ":                     {text: 7, wmf: 5, template: ""},
	"tuly e podrinha.ETQ":                                  {text: 10, wmf: 7, template: "LNT-2"},
	"tuly.ETQ":                                             {text: 5, wmf: 5, template: "LNT-2"},
	"vicenza com tamanho.ETQ":                              {text: 6, wmf: 7, template: ""},
	"vicenza unica.ETQ":                                    {text: 6, wmf: 7, template: "LNT-2"},
	"viscolycra sueco (lã).ETQ":                            {text: 7, wmf: 7, template: "LNT-2"},
}

func TestCorpusParseAllETQ(t *testing.T) {
	arquivos := filepath.Join(`C:\Program Files (x86)\paulimaq`, "ARQUIVOS")
	entries, err := os.ReadDir(arquivos)
	if err != nil {
		t.Skipf("ETQ corpus not installed at %s: %v", arquivos, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".ETQ") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	if len(files) != len(corpusBaseline) {
		t.Fatalf("corpus file count: got %d want %d; update corpusBaseline after reviewing installed corpus", len(files), len(corpusBaseline))
	}
	for _, name := range files {
		if _, ok := corpusBaseline[name]; !ok {
			t.Fatalf("unexpected ETQ %q; add baseline after reviewing parser output", name)
		}
	}

	for _, name := range files {
		name := name
		t.Run(name, func(t *testing.T) {
			want := corpusBaseline[name]
			doc, err := ParseETQ(filepath.Join(arquivos, name))
			if err != nil {
				t.Fatalf("ParseETQ: %v", err)
			}
			if got := len(doc.TextElements); got != want.text {
				t.Fatalf("text count: got %d want %d", got, want.text)
			}
			if got := len(doc.WMFElements); got != want.wmf {
				t.Fatalf("wmf count: got %d want %d", got, want.wmf)
			}
			if got := len(doc.UnknownObjects); got != want.unknown {
				t.Fatalf("unknown count: got %d want %d", got, want.unknown)
			}
			if doc.TemplateName != want.template {
				t.Fatalf("template: got %q want %q", doc.TemplateName, want.template)
			}
			assertCorpusParseInvariants(t, doc)
		})
	}
}

func assertCorpusParseInvariants(t *testing.T, doc *ETQFile) {
	t.Helper()
	seenTextOffsets := map[int]bool{}
	for _, txt := range doc.TextElements {
		if txt.Text == "" {
			t.Fatalf("empty displayed text at offset %#x", txt.FileOffset)
		}
		if txt.XMM <= 0 || txt.YMM <= 0 {
			t.Fatalf("non-positive text position: %#v", txt)
		}
		if strings.Contains(txt.Text, "fonttbl") || strings.Contains(txt.Text, "Arial Narrow;") {
			t.Fatalf("RTF metadata leaked into text: %q", txt.Text)
		}
		if seenTextOffsets[txt.FileOffset] {
			t.Fatalf("duplicate text FileOffset %#x", txt.FileOffset)
		}
		seenTextOffsets[txt.FileOffset] = true
	}

	seenWMFOffsets := map[int]bool{}
	for _, sym := range doc.WMFElements {
		if sym.FilePath == "" {
			t.Fatalf("empty WMF path at offset %#x", sym.FileOffset)
		}
		if sym.XMM <= 0 || sym.YMM <= 0 || sym.WidthMM <= 0 || sym.HeightMM <= 0 {
			t.Fatalf("invalid WMF rect: %#v", sym)
		}
		if seenWMFOffsets[sym.FileOffset] {
			t.Fatalf("duplicate WMF FileOffset %#x", sym.FileOffset)
		}
		seenWMFOffsets[sym.FileOffset] = true
	}
}
