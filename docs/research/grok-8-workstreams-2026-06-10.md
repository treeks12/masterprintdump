# Grok Research: 8 Workstreams, 2026-06-10

Todas as sessoes foram somente leitura, sem screenshots, sem web e sem edicao de arquivos.

## Resumo Executivo

O que impede a copia 1:1 hoje nao e falta de UI parecida. Sao lacunas especificas de engenharia reversa e cobertura:

1. O original e um host COM do CadMapa; o clone nativo precisa replicar o engine, nao apenas o shell.
2. O writer `.ETQ` ainda nao e seguro porque o envelope `FE FF FF FF` nao foi localizado no decompile.
3. A UI `TLayoutDesktop` tem menus/dialogs/statusbar/toolbars ainda nao copiados por recurso.
4. Render texto/RTF/WMF e print usam caminhos GDI especificos, nao approximacoes.
5. Layout e print dependem de `.INF`, `layout.ini`, `pageovrr.ini`, rows, landscape e DEVMODE.
6. Sidecar atual e seguro para nao tocar ETQ, mas incompleto para round-trip 1:1.
7. A verificacao precisa virar corpus/harness, nao validacao manual.

## Workstream 1: EXE Shell E API CadMapa

Confirmado:

- MasterPrint shell guarda `ICadMapaApp` em `TfrmMasterPrint + 0x2D8`.
- CadMapa expoe `Load`, `Print`, `NewAssist`, `ExportToFile`, `ResizeWorkSpace`, eventos `BeforeSaveFile`, `BeforeOpenFile`, `BeforePrint`, `AfterPrint`, `CaptionChanged`.
- `TLayoutDesktop` e o designer real: toolbars, canvas, dialogs, print e ETQ I/O.

Implica:

- O clone nativo nao pode ser 1:1 apenas reproduzindo menus; precisa replicar o engine CadMapa.
- Hospedar o COM original seria outro caminho tecnico, nao o clone nativo atual.

## Workstream 2: ETQ Serializer

Confirmado:

- `FUN_004b74e8` salva container `WDDESIGNVCM`.
- `FUN_004b7dc8` carrega container.
- `FUN_004bf340` salva payload interno de texto.
- `FUN_004bf12c` carrega payload interno de texto.
- `FUN_00423f00` / `FUN_00424010` salvam WMF/Aldus.

Bloqueio:

- Nenhuma funcao ainda localizada escreve literalmente o envelope `FE FF FF FF`, tags, post-fields, `nextX/nextY`, style ou pre-block WMF `FE-83`.

Decisao:

- Nao escrever `.ETQ` ate writer faseado passar no-op byte-identico e round-trip no app original.

## Workstream 3: UI Resources

Confirmado:

- `TLayoutDesktop` tem client 800x540, top dock 102, bottom clips 65, menu bar, toolbars padrao/fonte/objetos/DB.
- 50 glyphs foram extraidos; assets principais estao presentes.
- Faltam menu completo, context menu, toolbar DB separada, `btnMapaRisc`, dialogs, statusbar completo, cursores e bitmaps auxiliares.

Decisao:

- Copiar estrutura do recurso Delphi e checklist de glyph/order; nao redesenhar.

## Workstream 4: Text Rendering

Confirmado:

- `FUN_004c0118` usa `DrawTextA` com flags `0x810`, `0x811`, `0x812`.
- Fonte vem de `RECT.bottom - RECT.top`.
- `FUN_004bf99c` ajusta largura de caractere.
- RTF usa caminho RichEdit/OLE; parser atual so extrai plain text.

Bloqueios:

- Style byte ETQ `4/5/6` ainda nao esta provado como bitmask GDI.
- RTF completo e char-width fit ainda faltam.

## Workstream 5: WMF E Clipart

Confirmado:

- ETQ guarda WMF embutido, nao nome de arquivo.
- Hash `sha256(blob[22:])` e o caminho primario correto para identidade.
- Canvas CadMapa usa `PlayEnhMetaFile` com retangulo externo.
- Print usa `EnumEnhMetaFile`, nao o caminho GDI simplificado atual.
- Drag/drop original usa coordenada do cursor e dimensoes intrinsecas, nao default 5x5/8x8 mm.

Decisao:

- Remover defaults inventados somente quando drop/intrinsic estiver implementado por evidencia.

## Workstream 6: INF/Layout/Page/Print

Confirmado:

- `.INF` usa nome fixo 35 caracteres + campos posicionais definidos por `layout.ini`.
- `NumCol` pode ser `NxM`.
- `pageovrr.ini` limita labels/page.
- CadMapa imprime em grade row-major com margens, espacos, rows, landscape e multipagina.

Gaps atuais:

- Parser Go ainda carrega subset de INF e nao usa `layout.ini`.
- Print atual e single-row/single-label parcial.

## Workstream 7: Persistence

Confirmado:

- `.ETQ` e `.ETM` usam `WDDESIGNVCM4`.
- Sidecar atual nao toca ETQ e por isso e seguro para original, mas nao preserva tudo para writer futuro.

Regras:

- Nunca modificar `.ETQ` ate writer provado.
- Sidecar load continua opt-in.
- Sidecar precisa evoluir com `fileOffset`, raw RTF, WMF blob/ref, style, chain, layout/printer.

## Workstream 8: Verification Harness

Gates recomendados:

- P0: corpus ETQ parser, 7 anchors e sweep 58 arquivos.
- P1: invariantes render `MulDiv`, font height, DrawText flags, WMF rect.
- P2: INF/print/layout tests.
- P3: UI glyph/order checklist.
- P4: corpus coverage e baselines.

Comandos padrao:

```powershell
go test ./...
$env:GOARCH="amd64"; go build -o build\masterprint-native.exe .
```

## Proxima Ordem De Trabalho

1. Criar corpus/harness primeiro, porque sem ele cada mudanca 1:1 vira opiniao.
2. Corrigir INF/layout/pageovrr e print, porque isso e formato original e afeta todos ETQs.
3. Completar UI por recurso, nao por desenho manual.
4. Corrigir WMF/text render por formulas CadMapa.
5. Evoluir sidecar para preservar dados crus.
6. So entao iniciar writer `.ETQ` faseado.
