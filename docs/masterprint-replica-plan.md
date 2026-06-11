# MasterPrint Native: Usability-First Replica Plan

This project is a native replacement for Paulimaq MasterPrint 3.0 / CadMapa. The goal is not a screenshot clone; the goal is a usable open/edit/save/print workflow that converges toward CadMapa parity using reverse evidence only.

## Non-Negotiable Rule

Every behavior must be backed by at least one of these sources:

| Source | Governs |
|---|---|
| Real `.ETQ` files | document structure, object order, text, RTF, WMF, unknown records |
| Real `.INF`, `layout.ini`, `pageovrr.ini` | layout, columns, margins, landscape, sheet caps |
| `cadmapa-decompiled` | render, print, object fields, serialization, GDI behavior |
| `ciabrafe/decompiled` | MasterPrint shell workflow and CadMapa integration |
| `TLAYOUTDESKTOP_decoded.txt` | menu, toolbar, popup, statusbar structure |
| Original assets | glyphs, clipart, WMF identity |

Forbidden: screenshot tuning, invented geometry, silent fallbacks, fake commands that pretend to work, default object sizes without reverse proof, full `.ETQ` writing before the envelope writer is proven.

## Current Repo Truth

| Area | Exists now | Still blocks real use |
|---|---|---|
| Open | ETQ parser, INF/layout catalogs, sidecar load, `.mpn` load | unsupported objects are not surfaced strongly enough |
| Edit | select, drag, resize, text props, shapes, symbols, undo | UI is still unclear about what edits are safe for ETQ vs native save |
| Save | sidecar, `.mpn`, gated W0 ETQ patcher | default save UX is still confusing; full ETQ writer is blocked |
| Canvas | WMF, RTF RichEdit preview, plain text fallback | not full CadMapa canvas parity; enough for operator feedback only |
| Print | sheet grid, page overrides, multipage, landscape, RTF, WMF, shapes, text fit | no real-printer acceptance gate yet; WMF `EnumEnhMetaFile` path deferred |
| Merge | toolbar chrome only | no WDPE/WDPR2 reader/nav/substitution in the app |
| Parity | uicheck, glyph manifest, corpus tests | many menu commands are still stubs and must not fake behavior |

Recent important completed work:

| Work | Status |
|---|---|
| ETQ parser: text, RTF raw, WMF embedded bytes, unknown objects | done for current corpus scope |
| Native `.mpn` and sidecar raw preservation | done, still needs UX clarity |
| Print: `dmCopies=1`, landscape, page override, multipage | done, needs hardware proof |
| Print: CadMapa text fit (`FUN_004bf99c`) and `DrawTextA` path | done for plain boxed text |
| RTF: RichEdit `EM_FORMATRANGE` for print and canvas | done as direct-HDC slice |
| WMF: CadMapa-style two-pass `SetWinMetaFileBits` | done; full print enum replay deferred |

## Execution Spine

Work only moves through these milestones. Do not loop on random small fixes outside the current milestone.

### P0: Operator Loop

Goal: a shop user can open a real ETQ, understand limitations, edit supported objects, save safely, reopen without loss, and print one physical job.

| Deliverable | Acceptance gate |
|---|---|
| Round-trip harness | `Canelado algodao (Classic Wave Ramado) lunelli.ETQ`, `ADAR SOFA CANELADO.ETQ`, and `FAVERO.ETQ`: edit -> save `.mpn`/sidecar -> reopen without losing `FileOffset`, `PayloadRaw`, `RTFRaw`, `WMFRaw`, style, position |
| Save UX clarity | Save outcome says `.mpn`, sidecar, or ETQ patch; ETQ patch remains opt-in and preflighted |
| Unknown-object UX | open shows unsupported object count/kinds/offsets; print/save warns instead of silently dropping |
| Dirty/title/status | title/status make dirty state and persistence target obvious |
| Print acceptance | LNT-2 physical sheet: landscape, 8x3 grid, copies, `pageovrr`, multipage verified on one printer |
| Critical UI honesty | unavailable menu/toolbar commands are disabled or explain the reverse blocker; no fake success |

P0 defer list:

| Defer | Reason |
|---|---|
| Full `TLayoutDesktop` polish | not required for operator loop |
| Merge/batch print | requires WDPE/field binding proof and basic print gate first |
| Full `.ETQ` writer | FE envelope writer and structural serializers still blocked |
| Barcode/OLE/table/mapaRisc | object parsers/serializers not proven |
| Full WMF `EnumEnhMetaFile` print replay | larger reverse task; current WMF path is safer than guessing |
| Custom paper/roll physical-offset wiring | reverse anchors exist, but printer-mode/golden proof is missing |

### P1: Production Workflow

Goal: data-driven production labels after P0 print is proven.

| Deliverable | Acceptance gate |
|---|---|
| WDPE/WDPR2 reader | parse real/synthetic fixed records with decompile-backed sizes and slots |
| Record navigation | first/prev/next/last changes preview data without saving ETQ |
| Direct merge binding | selected text object can bind to a proven WDPE field; preview substitutes one record |
| Batch print | N records x copies fill the existing sheet grid; no silent field drop |
| Production print smoke | 5-record merge job prints on same printer used by P0 |

P1 defer list:

| Defer | Reason |
|---|---|
| WDPE writer / people-product database UI | not needed for reading production data first |
| ETQ merge table parse/write | ETQ binding table is not fully extracted |
| RTF merge | text substitution into RTF payload needs writer proof |
| Full original DB toolbar behavior | must follow WDPE and binding proof |

### P2: Function-By-Function CadMapa Parity

Goal: after the usable workflow is stable, replace partial native behavior with audited CadMapa functions one by one.

Rule: one parity change maps to one CadMapa function or one `TLayoutDesktop` resource row and adds a gate test.

| Priority | Truth anchor | Go target | Gate |
|---|---|---|---|
| P2.1 | `FUN_00464414` | audit existing `printlayout.Cells`, page iteration, matrix branch, guide lines | grid/page tests plus hardware print check |
| P2.2 | `FUN_004c0118`, `FUN_004c036c`, `FUN_004bf99c` | audit remaining text gaps after current print text-fit subset | real ETQ offset tests, GDI invariants |
| P2.3 | `FUN_00423530`, `FUN_0046273c` | WMF canvas/print | memory-DC and printer WMF tests |
| P2.4 | text/WMF save functions | current opt-in ETQ patcher and next proven fixed-field patches | byte-stable ETQ patch tests, original-app open |
| P2.5 | `TLayoutDesktop` decoded resource | menus/toolbars/status/dialogs | `internal/uicheck` coverage |
| P2.6 | remaining object functions | barcode/OLE/table/mapaRisc | parse/render/persist or explicit unknown gate |

P2 defer list:

| Defer | Reason |
|---|---|
| screenshot tuning | forbidden |
| COM `ICadMapaApp` hosting | different product path unless explicitly requested |
| broad UI cosmetics | only after function/resource gates |

### P3: ETQ Writer And 1:1 Completion

Goal: original-format persistence and final parity after function audits.

| Deliverable | Acceptance gate |
|---|---|
| FE envelope writer identified | decompile anchor for `FE FF FF FF` envelope, pre/post blocks, payload dispatch |
| ETQ no-op writer | byte-identical output for corpus subset |
| Text writer | same-length and then variable-length text with original app open test |
| WMF writer | rect and blob path proven by original app open test |
| Object writer expansion | only per object after parser/render/serializer proof |

## Next Implementation Tasks

These are the next tasks in order. Do not skip to P1/P2 until the P0 gates are green.

| Order | Task | Why |
|---:|---|---|
| 1 | Add round-trip harness for lunelli, ADAR, and FAVERO ETQs | proves open/edit/save/reopen instead of isolated parser tests |
| 2 | Improve save/open UX: status, dirty title, persistence target, sidecar applied/ignored reason | removes operator confusion before adding features |
| 3 | Surface unsupported objects and ETQ save limits in UI | prevents silent loss and fake confidence |
| 4 | Run/document real-printer smoke for LNT-2 and ADAR | print is the main production lane; unit tests are not enough |
| 5 | Disable or explain menu/toolbar stubs on non-critical commands | avoids fake UX loops while preserving `TLayoutDesktop` structure |
| 6 | Implement WDPE reader + merge preview only after P0 print gate | first production step, still read-only and reversible |

## UX Sprint A: Operator Honesty

Scope for the next UX-focused implementation sprint:

| Item | Implementation direction | Evidence gate |
|---|---|---|
| Open summary | status/dialog: source type, layout, sidecar applied, unknown count | ETQ load tests and manual open |
| Dirty title | append `*` while dirty, clear on save | existing dirty-state tests |
| Save result | status: `.mpn`, sidecar, ETQ patched/refused reason | save tests |
| Unsupported objects | one warning with offsets/kinds; no placeholder render | corpus unknown tests |
| Stub commands | disable/explain commands without reverse-backed behavior | `uicheck` still preserves captions/order |
| Print dialog facts | show layout, columns, labels/sheet, landscape, total labels | printlayout tests |

Do not use this sprint for visual polish, screenshots, toolbar spacing, or unproven dialogs.

## Verification Gates

| Gate | Command / evidence | Required when |
|---|---|---|
| Unit/corpus | `& "tools\verify.ps1" -RequireCorpus` | every code change |
| Build | amd64 build from `tools\verify.ps1` | every code change |
| Corpus ETQ | installed 58 ETQ parse baseline | parser/save/render changes |
| Real printer | documented LNT-2/ADAR physical output | before P1 merge/batch |
| Original app open | original MasterPrint opens output ETQ | before any default ETQ writer |

## Delegation Policy

Use Grok aggressively, but only for bounded research/review tasks:

| Use Grok for | Expected output |
|---|---|
| decompile function audits | exact function, offsets, branch behavior, uncertainty |
| implementation review | blockers only, no style bikeshedding |
| roadmap review | milestone order and gates |
| corpus/format questions | evidence table, no guesses |

Integration remains manual in this repo. Grok proposals are not truth unless tied to reverse/data evidence.

## Stop Conditions

Stop and ask or defer when:

| Condition | Action |
|---|---|
| behavior requires screenshot tuning | reject |
| reverse evidence conflicts | document blocker and choose fail-safe behavior |
| edit could corrupt ETQ bytes | require sidecar/MPN path |
| physical print behavior is unknown | create printer gate, do not invent formula |
| command has no reverse-backed behavior | disable/explain, do not fake |
