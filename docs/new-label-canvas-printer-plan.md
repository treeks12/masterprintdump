# New Label Canvas / Printer Plan

This is the standalone handoff plan for a new program. It does not depend on the failed 1:1 MasterPrint/CadMapa clone context.

## Decision

Build a new native label canvas/editor/printer. Do not continue trying to clone CadMapa's hidden GDI engine.

Use the old project only as research: Paulimaq layout files, original WMF symbols, ETQ parser, corpus tests, and reverse notes.

## Product Goal

Create a reliable Windows program for Paulimaq-style labels across all supported paper/template categories. LNT-2 is the first proof target because it is the current production pain point, not because the product is limited to LNT-2.

Next-session non-negotiables:

| Rule | Why |
|---|---|
| Start with the engine, not UI polish | The first win is correct catalog/cell/print behavior, not visual chrome |
| Load all Paulimaq catalogs early | The product must not become LNT-2-only |
| Mark paper types production-ready one at a time | Loading a layout is not the same as calibrated real-paper output |
| Prove LNT-2 on real paper first | LNT-2 40 slots on A4 portrait is the first truth gate |
| Keep `pageOrientation` and `contentOrientation` separate | Paulimaq `Paisa` is not enough to model page vs per-cell rotation safely |
| Use original WMF symbols from day one | Textile care symbols are compliance-sensitive and must not be casually redrawn |
| Store calibration per printer and layout | Printer drift and stock geometry are layout-specific |
| Defer ETQ import until native save/print works | ETQ compatibility is useful, but not the product foundation |

Technology decision:

| Area | Choice | Reason |
|---|---|---|
| Application platform | C# / .NET Windows desktop | Native Windows APIs, mature printer access, maintainable desktop tooling |
| Editor UI | WPF | Strong native canvas, transforms, property panels, document UI, and DPI-aware desktop behavior |
| Production print renderer | GDI+ / Win32 print path | Precise printer HDC control, WMF playback, mm-to-device-unit transforms, calibration output |
| Preview/editor renderer | WPF renderer backed by the same document/layout model | Good interactive editing while preserving shared geometry rules |
| Existing Go code | Research/reference only | Parser and print discoveries are useful, but Go UI/printing is not the product foundation |

Do not build this as a web-based desktop app. Electron, Wails, browser canvas, and TypeScript-first desktop stacks are a poor fit for millimeter-accurate Windows printing, WMF rendering, and printer-driver calibration.

| Goal | Requirement |
|---|---|
| Paper compatibility | Load/support all Paulimaq paper catalogs (`*.inf`), including LNT sheets, tags, rolls, jewelry, shoes, cards, invites, CDs, plants, and bands |
| LNT paper printing | Correct A4 portrait slots, especially LNT-2 with 40 labels/page as the first physical gate |
| Label editing | Text blocks, composition fields, CNPJ/brand lines, ABNT care symbols, basic shapes |
| Production workflow | Fill one label, repeat across sheet, later merge product data |
| Symbol compliance | Use original Paulimaq WMF care symbols from `CLIPART\Símbolos` |
| File format | Use a new native format as truth; ETQ import is optional/preferable |
| Print correctness | Preview and printer must use the same layout/transform engine |

## Research Sources

| Source | Path / Evidence | Use |
|---|---|---|
| Paulimaq install | `C:\Program Files (x86)\paulimaq` | Layout catalogs, symbols, ETQ samples |
| Layout schema | `layout.ini` | Fixed-width INF fields |
| Paulimaq catalogs | `*.inf` files | All paper/template dimensions, margins, columns, roll/sheet categories |
| Composition layouts | `etiqueta.inf` | LNT model dimensions, margins, columns |
| Partial-page overrides | `pageovrr.ini` | Later support for partial sheets |
| Original symbols | `CLIPART\Símbolos\*.wmf` | ABNT/ISO care symbols |
| Corpus | `ARQUIVOS\*.ETQ` | Optional ETQ import tests |
| Current research repo | `C:\Users\HB\Projects\masterprint-native` | Parser/tests/reverse findings |
| ETQ report | `docs\etq-format-report.md` | Detailed read-only ETQ import structure, confidence levels, edge cases, and C# porting notes |
| Legacy feature inventory | `docs\masterprint-cadmapa-feature-inventory.md` | Old MasterPrint/CadMapa feature checklist for support and migration scoping |
| MasterPrint story/function map | `docs\masterprint-software-story-and-functions.md` | Narrative of what the old software was plus UI/reverse function explanations |
| Web research | LNT sellers and textile-label references | Confirms commercial LNT sizes and ABNT/ISO 3758 requirement |

ABNT/Inmetro research notes:

| Topic | Finding |
|---|---|
| Care symbols | Textile care processes must be described by symbols and/or text following ABNT/NM ISO 3758 / ISO 3758 references |
| Treatments | Wash, bleach, dry, iron, professional cleaning symbols are the relevant groups |
| Practical consequence | Keep Paulimaq's original symbol set as the initial approved symbol library; do not redraw symbols casually |

## Correct LNT Paper Model

The physical paper is A4 portrait. The `Paisa=1` flag in Paulimaq layouts means the label content/design is rotated inside the physical slot. It must not be interpreted as turning the whole sheet to landscape.

LNT-2 is the first physical acceptance gate. The architecture must still load and print every Paulimaq catalog type.

## Full Paper Catalog Scope

The new program must treat Paulimaq `*.inf` files as the paper catalog source. `layout.ini` defines each fixed-width record schema, so the catalog loader should be generic and not hardcoded to only LNT.

Installed catalog examples from `C:\Program Files (x86)\paulimaq`:

| Catalog | Category / use |
|---|---|
| `etiqueta.inf` | Composition labels in sheets, including LNT-0/1/2/3/4 and SONTARA variants |
| `etiqueta_m.inf` | Composition labels in continuous/form layouts |
| `etiqueta_r.inf` | Composition labels in roll layouts |
| `tag.inf` | Tags in sheets/forms |
| `Tag2.inf`, `tag3.inf` | Fast label / Pauli-Tab style tags |
| `joia.inf` | Jewelry labels |
| `sapato.inf` | Shoe-box labels |
| `invite.inf` | Invitation/card formats |
| `fixbands.inf` | Wristband/band formats |
| `plantas.inf` | Plant labels |
| `photoA4.inf`, `ncd.inf`, `etred.inf`, `minicd.inf`, `pcd.inf` | Photo/CD/card related formats |

Reference image:

```text
C:\Users\HB\Projects\masterprint-new-handoff\test\exemplos do que o masterprint faz.jpg
```

This image is a visual collage of MasterPrint's page-configuration dialogs. It confirms the user-facing category breadth and typical page setup fields, but it must not override `layout.ini`/`*.inf` numeric parsing.

The size/margin/spacing fields in this dialog are editable in MasterPrint. Treat INF values as catalog defaults, not necessarily the final saved document values. The native document format should support optional per-document layout overrides for width, height, margins, columns, and spacing once ETQ storage for those overrides is identified.

Visible categories include:

| Category shown in MasterPrint UI | Notes |
|---|---|
| `Etiq. para Composições em Folhas` | LNT-0/1/2/3/4, SONTARA, Nylon ECNY sheet labels |
| `Etiq. para Composições em Formulários` | NT/TY/NY form labels with column variants |
| `Etiq. para Composições em Rolo` | TYB/NYR roll labels, feed spacing visible |
| `Caixa de Cartões - PRINT BOX` | box/card package layout |
| `Cartões de Visita - PRINT CARD` | vertical/horizontal/plus business card layouts |
| `Etiq. para Caixas de Calçados` | shoe-box labels such as `CS0210 / LJA 272` and `FACS` |
| Other dropdown categories | CD labels, fast labels, bands, invites, jewelry, Pauli-Tab |

The image reinforces the catalog rule: support all INF-defined layouts generically, then mark individual physical stocks production-ready only after calibration.

Product rule:

```text
MVP validates LNT-2 on real paper.
Catalog engine supports all INF-defined layouts from day one.
Each new physical paper type gets its own calibration gate before being marked production-ready.
```

Catalog support states:

| State | Meaning |
|---|---|
| Loaded | INF record parsed, visible in picker, metadata preserved |
| Previewable | Sheet/roll geometry can be drawn in preview without fake defaults |
| Calibrated | A numbered calibration page/segment was printed and adjusted for one printer |
| Production-ready | Real stock was printed acceptably for that layout category |

Do not build an LNT-only data model. A paper layout must be a generic object:

```json
{
  "id": "LNT-2",
  "category": "etiqueta",
  "page": "A4",
  "mediaKind": "sheet",
  "physicalWidthMM": 25.0,
  "physicalHeightMM": 55.5,
  "columns": 8,
  "rows": 5,
  "marginLeftMM": 5.0,
  "marginTopMM": 11.0,
  "gapXMM": 0.0,
  "gapYMM": 0.0,
  "orientation": "rotated-content"
}
```

For non-sheet papers, the same abstraction should allow roll/continuous layouts by replacing fixed rows/page with media length, feed direction, or driver page height.

### LNT Layouts

Exact values are from Paulimaq `etiqueta.inf` local audit plus real LNT-2 paper evidence.

| Model | Slot W x H mm | Columns | Rows on A4 portrait | Labels/page | Left mm | Top mm | H/V spacing | Content orientation |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| LNT-1 | 25.0 x 40.0 | 8 | 7 | 56 | 5.0 | 11.0 | 0 / 0 | Rotated/wide design |
| LNT-2 | 25.0 x 55.5 | 8 | 5 | 40 | 5.0 | 11.0 | 0 / 0 | Rotated/wide design |
| LNT-3 | 33.0 x 55.5 | 6 | 5 | 30 | 5.0 | 11.0 | 0 / 0 | Rotated/wide design |
| LNT-4 | 33.0 x 69.9 | 6 | 4 | 24 | 6.0 | 8.7 | 0 / 0 | Rotated/wide design |

LNT-2 physical slots:

```text
Page: A4 portrait, 210 x 297 mm
slotW = 25.0
slotH = 55.5
left = 5.0
top = 11.0
cols = 8
rows = 5

slotX = left + col * slotW
slotY = top  + row * slotH
```

For LNT-2, row 4 bottom is `11 + 5*55.5 = 288.5 mm`, which fits A4 portrait height 297 mm.

## Core Design Rule

Treat every label slot as an independent printable cell.

Do not reproduce CadMapa's hidden canvas state. The new engine must have a simple explicit model:

```text
Sheet
  Layout: LNT-2, A4 portrait, 8x5 cells
  Cells: 40 independent label slots
  Job: repeat one label into all cells, or fill cells from data records

LabelTemplate
  Design canvas: user-facing label design
  Objects: text, symbol, line, rect, barcode later
```

For LNT layouts, the design canvas can be shown as the wide readable label:

| Physical slot | Design canvas |
|---|---|
| LNT-2 25.0 x 55.5 mm | 55.5 x 25.0 mm |
| LNT-3 33.0 x 55.5 mm | 55.5 x 33.0 mm |
| LNT-4 33.0 x 69.9 mm | 69.9 x 33.0 mm |

Print transform for each cell must be centralized and reused by preview/export/print.

## Print Engine

### LNT Sheet Generation

```text
for row in 0..rows-1:
  for col in 0..cols-1:
    cell = Rect(
      left + col * (slotW + gapX),
      top  + row * (slotH + gapY),
      slotW,
      slotH,
    )
    drawLabelInCell(cell, labelTemplate)
```

### Cell Drawing

The engine must draw in this order:

1. Save graphics state.
2. Clip to the physical slot rectangle.
3. Translate to the slot origin.
4. Apply the layout's orientation transform.
5. Draw all label objects in design coordinates.
6. Restore graphics state.

For LNT rotated labels, support both clockwise and counter-clockwise rotation as a layout property and print a calibration page to confirm the default. The preview must use the same transform.

Recommended transform fields:

```json
{
  "pageOrientation": "portrait",
  "contentOrientation": "cw90",
  "physicalWidthMM": 25.0,
  "physicalHeightMM": 55.5,
  "designWidthMM": 55.5,
  "designHeightMM": 25.0,
  "offsetXMM": 0.0,
  "offsetYMM": 0.0,
  "scaleX": 1.0,
  "scaleY": 1.0
}
```

Orientation rules:

| Field | Meaning |
|---|---|
| `pageOrientation` | Printer/page setup: portrait or landscape A4/job orientation |
| `contentOrientation` | Transform inside each physical slot: none, cw90, ccw90, rotate180 |
| `Paisa` from INF | Input hint only; map it into explicit page/content orientation after calibration |

Never collapse these into one boolean. LNT-2 is A4 portrait physical paper with rotated content in each slot.

Printer calibration must be a first-class feature:

| Calibration | Purpose |
|---|---|
| Global X/Y offset | Correct printer hardware unprintable margin drift |
| Scale X/Y | Correct driver scaling if needed |
| Rotation direction | Confirm how the stock must be fed and read |
| Test page | Print cell numbers 1..40, border, axes, and ruler ticks |

Calibration storage key:

```text
printerName + layoutId + pageOrientation + contentOrientation
```

Do not use one global calibration for every paper type.

## Symbol Library

Use the original Paulimaq symbols from:

```text
C:\Program Files (x86)\paulimaq\CLIPART\Símbolos
```

Known families:

| Family | Examples |
|---|---|
| Washing | `lav-30.wmf`, `lav40.wmf`, `lavmao.wmf`, `lavp60.wmf` |
| Bleach | `cloro.wmf`, `clorox.wmf`, `clorom.wmf` |
| Drying | `seco-w.wmf`, `secah.wmf`, `secop.wmf`, `tamborx.wmf` |
| Ironing | `ferro-.wmf`, `ferro--.wmf`, `ferrox.wmf` |

Implementation rules:

| Rule | Reason |
|---|---|
| Copy WMFs into app assets | Avoid depending on Paulimaq install path at runtime |
| Preserve filenames as IDs | Operators recognize the legacy names |
| Store SHA-256 of `wmf[22:]` | Existing ETQ imports identify embedded symbols by WMF body hash |
| Render WMF to preview and print through the same code path | Prevent preview/print mismatch |
| Do not redraw ABNT symbols manually in MVP | Avoid legal/compliance and geometry drift |

## Native File Format

Use a new JSON document format first. Example:

```json
{
  "schemaVersion": 1,
  "documentKind": "label-template",
  "layoutId": "LNT-2",
  "page": "A4",
  "template": {
    "widthMM": 55.5,
    "heightMM": 25.0,
    "objects": [
      {
        "type": "text",
        "xMM": 2.0,
        "yMM": 1.5,
        "wMM": 30.0,
        "hMM": 3.0,
        "text": "72% ALGODAO",
        "font": "Arial Narrow",
        "sizePt": 8,
        "bold": true,
        "align": "center"
      },
      {
        "type": "symbol",
        "symbolId": "lav-30",
        "xMM": 2.0,
        "yMM": 19.0,
        "wMM": 4.5,
        "hMM": 4.5
      }
    ]
  }
}
```

Preferred extension: `.mpLabel` or `.mpt`.

## MVP Scope

### P0: Catalog Engine + Useful LNT-2 Printer

| Feature | Acceptance |
|---|---|
| Generic catalog loader | Load all Paulimaq `*.inf` catalogs through `layout.ini`; LNT-2 is default selection |
| LNT layout catalog | LNT-1/2/3/4 plus SONTARA variants loaded from `etiqueta.inf` |
| Label canvas | Wide 55.5 x 25.0 mm design for LNT-2 |
| Text objects | Add/edit/move/resize, font family, size, bold/italic/underline, align |
| Symbol objects | Insert/move/resize original WMF symbols |
| Repeat print | One template repeated into all 40 LNT-2 cells |
| Calibration print | Cell borders/numbers/axes for LNT-2 |
| Native save/load | Save and reopen without loss |
| Real printer gate | HP Laser 103 107 108 prints LNT-2 40 slots aligned enough for production |

### P1: Production Workflow

| Feature | Acceptance |
|---|---|
| Cell selection | Print from first cell N, skip used cells |
| Data records | CSV/JSON product list with fields like composition, CNPJ, brand, size |
| Merge preview | Show one record in the label before printing |
| Batch print | Fill 40 cells per page from records |
| Multiple LNT templates | LNT-1/3/4 with calibration pages |
| Other paper categories | Select and preview at least one tag, roll, jewelry, shoe, card, invite, CD, and band layout from installed catalogs |

### P1.5: Paper Compatibility Expansion

| Feature | Acceptance |
|---|---|
| Sheet layouts | Any INF layout with fixed columns/rows can print a numbered calibration page |
| Roll/continuous layouts | Composition roll/form layouts can preview and print one calibrated label/feed segment |
| Specialized shapes | Jewelry, CD, invite, plant, and band layouts expose their extra fields from `layout.ini` without data loss |
| Per-layout calibration | Store offsets/scale/rotation per printer + layout ID |

### P2: Compatibility

| Feature | Acceptance |
|---|---|
| ETQ read-only import | Import text and symbols from common LNT-2 ETQs into native format |
| Symbol hash matching | Embedded ETQ WMFs map to original symbol IDs |
| Legacy template starter | Convert `LNT-2.ETM`/ETQ samples to native templates if safe |

Photo/logo/raster/OLE objects inside `.ETQ` are not covered by the current parser proof. Reverse evidence points to separate OLE/DIB object paths, including `BDOC` OLE payloads and bitmap/DIB rendering, but no available corpus file contains a controlled photo/logo example. Treat these as unsupported in MVP until controlled `.ETQ` samples exist.

Required future samples for raster/OLE import:

```text
figure-bmp-lnt2.ETQ + original BMP + screenshot
ole-bmp-lnt2.ETQ + original BMP + screenshot
figure-wmf-lnt2.ETQ + original WMF + screenshot
photo-jpg-lnt2.ETQ + original JPG + screenshot, if MasterPrint accepts JPG
```

ETQ writing is explicitly not required.

## Technical Architecture

Recommended split:

| Layer | Responsibility |
|---|---|
| `catalog` | Paper layouts, LNT models, Paulimaq INF import |
| `document` | Native label template schema and validation |
| `symbols` | WMF library, IDs, hashes, thumbnails |
| `canvas` | Interactive design surface in design coordinates |
| `layout` | A4 sheet cells and page filling |
| `print` | Windows printer backend, calibration, preview/export |
| `import_etq` | Optional ETQ importer from research repo |

Use C#/.NET as the implementation language. Use WPF for the interactive editor and shell, but do not use "print the WPF visual" as the production print path. Production print must render from the document model into a GDI+/Win32 printer graphics context using explicit millimeter geometry.

Rendering architecture:

```text
Native document model
  -> catalog/layout/cell transform engine
    -> WPF preview/editor renderer
    -> GDI+/Win32 production print renderer
```

Rules:

| Rule | Reason |
|---|---|
| Keep layout math outside WPF controls | Preview and print must share geometry, not duplicated UI logic |
| Use WPF for interaction, not as the print authority | WPF visual printing can introduce DPI/scaling surprises |
| Use GDI+/Win32 for final output | Direct printer HDC control is better for calibration and WMF replay |
| Keep renderer adapters thin | All object positions and transforms must come from the shared model |
| Treat Go code as reference | Reuse algorithms/findings, not the old app architecture |

Stacks to avoid:

| Stack | Reason |
|---|---|
| Electron / TypeScript desktop | Browser print pipeline is imprecise; WMF and text metrics are awkward |
| Go + Wails | Same browser print/WMF issues plus bridge complexity |
| Go native desktop UI | Weak canvas/editor ecosystem for this type of app |
| Avalonia | Cross-platform tradeoffs are not useful here; Windows-native printing/WMF matter more |
| Qt | Powerful, but heavier and less natural for Windows WMF/print integration than .NET |

## Critical Tests

### Layout Tests

```text
LNT-2 cell count = 40
LNT-2 cell[0] = x=5.0 y=11.0 w=25.0 h=55.5
LNT-2 cell[7] = x=180.0 y=11.0
LNT-2 cell[8] = x=5.0 y=66.5
LNT-2 last cell = x=180.0 y=233.0 bottom=288.5
```

### Print Tests

```text
Print calibration page to PDF/XPS
Print calibration page to HP Laser 103 107 108
Verify all 40 borders land on physical cut/slot marks
Verify label content appears once per slot, not once per page
```

### Symbol Tests

```text
All 49 WMF symbols load
Each has stable ID and hash
Preview render and print render use same bounding box
```

### Import Tests

```text
Import Canelado/ADAR/FAVERO ETQ into native template
Compare object count and symbol IDs
Do not require exact CadMapa placement in MVP
```

## What Not To Do

| Do not | Reason |
|---|---|
| Do not clone CadMapa UI | It did not solve print correctness |
| Do not use `Paisa=1` as page landscape | LNT-2 physical sheet is A4 portrait with 40 slots |
| Do not make ETQ the native format | ETQ writer is still structurally unsafe |
| Do not redraw ABNT symbols | Use original WMF symbols first |
| Do not hardcode the engine to LNT-2 | LNT-2 is only the first physical proof; catalog/data model must support all Paulimaq paper types |
| Do not mark a paper type production-ready without calibration | Every physical stock needs its own real-printer gate |
| Do not hide calibration | Physical printers vary; calibration must be visible |

## First Week Implementation Plan

1. Create new repo `masterprint-label-studio` as a C#/.NET WPF application.
2. Create separate projects/namespaces for core model/catalog/layout logic and WPF UI.
3. Copy Paulimaq-derived layout catalog and original WMF symbols into app assets.
4. Implement a generic Paulimaq INF catalog loader using `layout.ini` schemas.
5. Add tests that load every installed `*.inf` catalog and expose all layout IDs/categories.
6. Implement LNT-2 as the first calibrated sheet layout using the generic catalog data.
7. Implement native document model and JSON save/load.
8. Implement a minimal WPF canvas with text and symbol objects in the selected layout's design coordinates.
9. Implement a GDI+/Win32 production print renderer using the same cell transform engine as preview.
10. Implement WPF preview from the same shared document/layout model.
11. Print LNT-2 calibration page to HP and adjust offsets.
12. Print a simple repeated template to all 40 LNT-2 slots.
13. Add calibration-page support for every sheet-type INF layout.
14. Only after this passes, add ETQ import.

Week-one priority order:

```text
engine > catalog > calibration > minimal canvas > native save/load > ETQ import
```

Do not spend week one on toolbar chrome, CadMapa UI parity, database screens, ETQ writing, Electron/Wails experiments, or browser-canvas prototypes.

## Reserved Handoff Bundle

The next session should start from the handoff folder created from this plan. Expected contents:

```text
masterprint-new-handoff/
  docs/new-label-canvas-printer-plan.md
  docs/etq-format-report.md
  paulimaq/layout.ini
  paulimaq/inf/*.inf
  paulimaq/etiqueta.inf
  paulimaq/pageovrr.ini
  paulimaq/symbols/*.wmf
  samples/*.ETQ
  test/lnt*.ETQ
  test/lnt*.jpg
  test/exemplos do que o masterprint faz.jpg
  research/masterprint-replica-plan.md
```

## Final Recommendation

Build the new engine around **generic Paulimaq paper catalogs + physical slots/feed segments + independent label cells + original WMF symbols + native JSON persistence**.

Treat CadMapa and ETQ as import/reference only. The product should be a reliable LNT label printer, not a perfect MasterPrint clone.
