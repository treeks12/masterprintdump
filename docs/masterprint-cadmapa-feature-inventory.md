# MasterPrint / CadMapa Legacy Feature Inventory

This is a support-oriented inventory of the old MasterPrint/CadMapa editor. It is not a commitment to clone every behavior in the new label program.

The purpose is to preserve what old users may ask for, identify which legacy features matter for textile-label production, and separate safe MVP scope from risky or unsupported CadMapa internals.

## Sources

| Evidence | Path / Function |
|---|---|
| Decoded main editor form | `C:\Users\HB\Projects\paulimaq-reverse\ciabrafe\analysis\TLAYOUTDESKTOP_decoded.txt` |
| Decoded editor string table | `C:\Users\HB\Projects\paulimaq-reverse\ciabrafe\analysis\TLAYOUTDESKTOP_strings.txt` |
| CadMapa object handler switch | `FUN_004ba3e0` |
| Object-type capability checks | `FUN_004b5870` |
| OLE payload write path | `FUN_0049bdac` |
| OLE payload read path | `FUN_0049b7f8` |
| OLE draw path | `FUN_0049ba34` |
| Bitmap/DIB-style figure handler | `FUN_004bd71c` |
| OLE-backed figure/image wrapper | `FUN_004c066c` |
| Barcode runtime report | `docs\masterprint-barcode-research.md` |
| ETQ object parser/import research | `docs\etq-format-report.md` |
| New product plan | `docs\new-label-canvas-printer-plan.md` |

## High-Level Conclusion

MasterPrint/CadMapa is a general desktop publishing editor wrapped around Paulimaq label/catalog workflows. It is broader than the textile `.ETQ` subset currently decoded.

Legacy support should be handled in layers:

| Layer | New-program position |
|---|---|
| Paulimaq catalog/layout selection | Core requirement |
| Textile label text and care symbols | Core requirement |
| Correct physical printing/calibration | Core requirement |
| Common ETQ read-only import for text and embedded WMF care symbols | Important compatibility feature after native print works |
| Shapes and basic object formatting | Useful editor feature, but after layout/print truth |
| Barcode, database merge, counters, clipart browser, image/OLE import | Deferred until controlled samples and real user need exist |
| Full byte-compatible ETQ writer | Unsafe; do not promise |
| Full CadMapa UI/behavior clone | Rejected |

## Main Editor Surface

The decoded form is `TLayoutDesktop`. It has a scrollable drawing surface, horizontal/vertical ruler-style paint boxes, top docked menu/toolbars, a left/right docking area, a bottom `Clips` panel, and a status area showing selected object position and size.

Visible canvas/status evidence:

| UI element | Evidence | Meaning |
|---|---|---|
| Main canvas | `ScrollBox1`, `Panel3`, `Panel3Paint`, mouse/drag handlers | Interactive document design surface |
| Rulers or guides | `PaintBox1`, `PaintBox2` | Horizontal/vertical auxiliary drawing areas |
| Clip panel | dockable panel caption `Clips`, `ComboBox2`, `PaintBox3` | Clipart/asset browsing or insertion UI |
| Status dimensions | labels `E:`, `T:`, `L:`, `A:` | Selected object left, top, width, height display |
| Border/fill color controls | `ColorCombo1`, `ColorCombo2` hidden until relevant | Shape/object style controls |
| Text color control | `ColorCombo3`, hint `Cor do Texto` | Text color formatting |

Support implication: old users may expect object handles, drag/drop, rulers/guides, zoom, status coordinates, object property dialogs, and clipart insertion. The new app should prioritize a simpler explicit canvas but keep millimeter coordinates visible.

## File And Document Features

The `Arquivo` menu exposes:

| Feature | Evidence | New-program position |
|---|---|---|
| New document | `Novo`, toolbar hint `Nova Etiqueta` | Core |
| Open document | `Abrir`, `Abrir Etiqueta` | Native JSON core; ETQ import later |
| Save document | `Salvar`, `Salvar Etiqueta` | Native JSON core |
| Save as | `Salvar Como...` | Core |
| Save as model/template | `Salvar Como Modelo...` | Later, useful for production presets |
| Reopen recent files | `Reabrir` | Nice-to-have |
| Export | `Exportar...` | Deferred; define formats later |
| Configure document/page | `Configurar Documento` | Core, but driven by parsed Paulimaq catalogs |
| Print | `Imprimir`, toolbar `Imprimir` | Core via GDI+/Win32 print path |
| Print to fax | `Imprimir para fax`, hidden | Ignore unless a real user asks |
| Exit | `Sair`, toolbar `Saída` | Normal app shell feature |

The save dialog filter is `Documento|*.ETQ`, confirming `.ETQ` as the legacy document extension. Current reverse work only supports safe read-only import for common text/WMF textile documents. Native documents must not depend on `.ETQ` as the source of truth.

## Editing And Object Operations

The `Editar` and `Objeto` menus plus the standard toolbar expose normal desktop publishing operations:

| Feature | Evidence | New-program position |
|---|---|---|
| Cut/copy/paste | `Recortar`, `Copiar`, `Colar`; toolbar `Cortar`, `Copiar`, `Colar` | Core editor usability |
| Delete | `Apagar` | Core |
| Search | `Procurar`, hidden | Ignore initially |
| Border style/color | `Borda` | Later for shapes; not needed for first text/symbol print gate |
| Fill style/color | `Preenchimento` | Later for shapes |
| Font properties | `Fonte` | Core for text objects |
| Send backward | `Enviar Para Trás` | Useful once multiple object types exist |
| Bring forward | `Trazer Para Frente` | Useful once multiple object types exist |
| Group | `Agrupar` | Deferred |
| Ungroup | `Desagrupar` | Deferred |
| Align | `Alinhar` | Useful editor feature after MVP |
| Scale | `Escalonar` | Deferred; risk of confusing physical dimensions |
| Properties | `Propriedades` | Core as property panel/dialog concept |

Keyboard shortcut numeric values appear in the decoded resource, but the new app should use normal Windows shortcuts instead of trying to preserve Delphi shortcut IDs exactly.

## Font And Text Formatting

The `Fonte` toolbar exposes rich text formatting controls:

| Feature | Evidence | New-program position |
|---|---|---|
| Font family picker | `ComboBox3`, hint `Fontes` | Core for text |
| Font size picker | `ComboBox5`, hint `Tamanho da Fonte` | Core for text |
| Bold | `Negrito` | Core |
| Italic | `Itálico` | Core |
| Underline | `Sublinhado` | Core or early later |
| Strikeout | `Cortado` | Later |
| Align left | `Alinhamento a Esquerda` | Core for multiline text |
| Align center | `Alinhamento ao Centro` | Core for textile labels |
| Align right | `Alinhamento a Direita` | Core for multiline text |
| Bullets/markers | `Marcadores` | Deferred |
| Text color | `Cor do Texto` | Core, default black |

ETQ import can currently recover plain text and some RTF hints, but it does not reproduce full RichEdit layout. New native text layout should be explicit in millimeters and should not depend on CadMapa's hidden RichEdit/GDI behavior.

## Drawing And Object Creation Tools

The `Manipulação de Objetos` toolbar is the clearest user-facing list of object types:

| Tool | Evidence | New-program position |
|---|---|---|
| Pointer/select | `Apontador` | Core |
| Zoom tool | `Zoom` | Core preview usability |
| Line | `Linha` | Later, simple shape |
| Rounded rectangle | `Retângulo Ovalado` | Later, simple shape |
| Rectangle/square | `Quadrado` | Later, simple shape |
| Oval/ellipse | `Oval` | Later, simple shape |
| Simple text | `Texto Simples` | Core |
| Text box | `Caixa de Texto` | Core/later depending on multiline behavior |
| Artistic text | `Texto Artístico` | Deferred |
| Barcode | `Código de Barras`; type `7`; `bc_*` runtime | Real legacy object; native support later, ETQ import blocked by missing barcode sample |
| Figure/image | `Figura`, `btnImage` | Deferred; controlled samples needed |
| Mapa risco figure | `Figura`, `btnMapaRisc` | Deferred; domain-specific and not decoded |
| OLE object | `Ole`, `btnOle` | Unsupported initially |
| File-managed figure | `Figura`, `btnFileMan` | Deferred; not decoded |
| WordArt | `Ole`, `btnWordArt`, hidden | Ignore initially |
| Table | `Ole`, `btnTable`, hidden | Ignore initially |

Reverse evidence confirms the object model has a runtime type byte at object offset `+0x189`. `FUN_004ba3e0` switches on that type and allocates different handlers. Known mapping confidence:

| Runtime type | Reverse evidence | Likely feature family | Import confidence |
|---:|---|---|---|
| `0`, `1`, `2`, `3` | Shared handler via `FUN_004bb77c` | Basic vector/text-like objects | Low without more samples |
| `4`, `5`, `11` | Handler via `FUN_004bf068` | Text/rich/artistic text family likely | Medium for visible text only through current scanner |
| `6` | Handler via `FUN_004bd71c` | Bitmap/DIB-style figure path | Low; controlled samples needed |
| `7` | Handler via `FUN_004befdc`; configured by `FUN_004bea88`; rendered through `bc_*` calls | Barcode | Medium-high for runtime behavior; ETQ bytes still unknown |
| `8` | Handler via `FUN_004bb77c` using different vtable pointer | Alternate object type | Low |
| `9` | Handler via `FUN_004c066c`, OLE wrapper initialized | OLE-backed figure/image | Low; unsupported initially |
| `10` | Handler via `FUN_004bb77c` using another vtable pointer | File-managed/special figure candidate | Low |

The current ETQ scanner intentionally bypasses this full object graph and only validates common text and embedded Aldus WMF records. That is the right safety posture for a new implementation.

### Barcode Object Evidence

Barcode is no longer just a toolbar assumption. The decoded UI has `btnBarcode` with hint `Código de Barras`, and the decompiled runtime maps barcode to object type `7`.

Known barcode function chain:

| Function | Role |
|---|---|
| `FUN_004b5d8c` | Creates/loads object type `7`, assigns handler from `FUN_004befdc`, and calls `FUN_004bea88` |
| `FUN_004ba3e0` | Generic object-handler switch; case `7` constructs the barcode handler |
| `FUN_004befdc` | Initializes barcode handler defaults, including likely height percent `100`, orientation `0`, and size/module scale `2.0` |
| `FUN_004bea88` | Applies barcode code string, type index, orientation, and helper state |
| `FUN_004b3664` | Creates low-level barcode helper and calls `bc_CreateBarCode` |
| `FUN_004b3830` | Render path: calls `bc_HeightPercent`, `bc_Type`, `bc_Rotate`, `bc_Size`, `bc_Code`, then `bc_Draw` on an HDC |
| `FUN_004c3a4c` | Applies barcode properties from the properties UI to the selected object |

Current limitation: there is no `.ETQ` sample containing a barcode. Therefore, old barcode import/export bytes are not known. Barcode should remain a documented legacy feature and a future native object, not an MVP ETQ-import promise.

## Zoom And View Features

The standard toolbar exposes:

| Feature | Evidence | New-program position |
|---|---|---|
| Page width zoom | `Zoom de Largura da Página` | Core preview usability |
| Whole page zoom | `Zoom de Página Inteira` | Core preview usability |
| 100% zoom | `Zoom de 100%` | Core preview usability |
| Fixed zoom list | `25%`, `50%`, `75%`, `100%`, `150%`, `200%`, `300%`, `400%`, `Página Inteira`, `Largura da Página` | Nice-to-have |

New-program note: zoom is only a UI transform. Physical layout and print calibration must stay in millimeters and printer device units.

## Database Merge And Counters

The decoded UI includes a `Banco de Dados` toolbar and menu:

| Feature | Evidence | New-program position |
|---|---|---|
| Enable/disable merge | `Ativar/Desativar Mescla` | Deferred |
| Current merge source | `Mesclar:`, `(nenhum)` | Deferred |
| Record navigation | `Primeiro Registro`, `Registro Anterior`, `Próximo Registro`, `Último Registro` | Deferred |
| Insert field | `Incluir Campo...` | Deferred |
| Manage data | `Gerenciar Dados` | Deferred |
| Insert counter | `Incluir Contador...` | Later if production numbering is needed |
| Manage counters | `Gerenciar Contadores` | Later if counters are implemented |

This is a real legacy feature area, but it is not needed for first LNT-2 physical print correctness. The new native document model should leave room for future variable fields and repeat/merge jobs without implementing database import immediately.

## Toolbars, Preferences, Help

The `Opções` and `Ajuda ?` menus expose:

| Feature | Evidence | New-program position |
|---|---|---|
| Preferences | `Preferências` | Later |
| Toolbar visibility | `Barras de Ferramentas` | Optional; WPF app can use fixed panels initially |
| Standard toolbar toggle | `Padrão` | Optional |
| Font toolbar toggle | `Fonte` | Optional |
| Database toolbar toggle | `Banco de Dados`, hidden | Ignore initially |
| Drawing toolbar toggle | `Ferramentas de Desenho` | Optional |
| Clipart panel toggle | `Cliparts` | Later when asset browser exists |
| Help search/content/about | `Procurar Ajuda Sobre`, `Conteúdo`, `Sobre` | Normal app shell feature, not compatibility-critical |

## Image, Clipart, Bitmap, And OLE Risk

Image and OLE support is the highest-risk compatibility area because the UI exposes several similar insertion modes and the ETQ structure is not decoded enough.

Reverse evidence:

| Area | Evidence | Meaning |
|---|---|---|
| OLE payload writer | `FUN_0049bdac` writes a 12-byte header and payload bytes | The header commonly starts with `BDOC` (`0x434F4442`) when not in special graphical mode |
| OLE payload reader | `FUN_0049b7f8` reads the 12-byte header and rejects non-`BDOC` unless special mode is active | Import needs true OLE/object samples, not guessed image signatures |
| OLE rendering | `FUN_0049ba34` calls `OleDraw(object + 0x204, aspect +0x208, hdc, rect)` | Rendering depends on Windows OLE object state and draw aspect |
| OLE creation | `FUN_0049ac24` references `OleCreate`, `OleCreateFromFile`, `OleCreateLinkToFile`, `OleCreateFromData`, `OleCreateLinkFromData` | Multiple insert/link modes may exist |
| Bitmap/DIB path | functions around `FUN_004bd71c`, `GetDIBits`, `StretchDIBits`, `BM` load/write | Some figure mode likely stores/renders DIB/BMP-style data |
| Current corpus scan | `cmd/etqdump` signature scan | Current controlled samples do not contain `BDOC`, OLE compound, BMP, JPEG, or PNG signatures |

Support implication: if an old user says a legacy label has a logo/photo/image, request the original `.ETQ`, original image file if available, and a screenshot/export from MasterPrint. Do not promise import until controlled samples are decoded.

## Legacy Feature Priority For The New Program

Recommended product scope by phase:

| Phase | Include |
|---|---|
| MVP foundation | Generic Paulimaq INF catalog loader, native JSON document, LNT-2 calibration, WPF canvas, GDI+/Win32 print path |
| MVP editor objects | Text objects, original WMF textile-care symbols, object coordinates/sizes, repeat-one-label-across-sheet printing |
| First compatibility pass | Read-only ETQ import for validated text records and embedded WMF symbols by hash |
| Editor usability pass | Copy/paste/delete, font formatting, alignment, zoom, object property panel, simple z-order |
| Shape pass | Line, rectangle, rounded rectangle, oval, border/fill controls |
| Production workflow pass | Templates, partial sheets, counters, data/merge model |
| Compatibility research pass | Barcode, image/logo/photo, OLE, file-managed figures, Mapa Risco, artistic text |

Do not move image/OLE/barcode into a promised phase until there are controlled ETQ samples and print/export references.

## User-Support Talking Points

Use these statements consistently:

| User asks | Accurate answer |
|---|---|
| “Will it open my old `.ETQ` files?” | “The goal is read-only import for common textile labels with text and Paulimaq care symbols. Some old objects such as logos, OLE, barcodes, and database fields may need sample-specific support.” |
| “Will it save `.ETQ`?” | “No, not as the main format. The new app should save a safer native document format. Byte-compatible ETQ writing is not proven safe.” |
| “Can it print LNT-2 correctly?” | “That is the first physical calibration target.” |
| “Will it support other Paulimaq papers?” | “The catalog loader must load all Paulimaq INF catalogs, but each physical stock becomes production-ready only after calibration.” |
| “Will it have the same UI?” | “No. It will preserve the important label workflow and compatibility where safe, not clone the old CadMapa UI.” |
| “Can it import logos/photos?” | “Only after controlled samples are decoded. The old program used bitmap/OLE paths that are not yet safe to guess.” |
| “Can it do database merge?” | “It is a known legacy feature, but it is deferred behind correct layout, native documents, and printing.” |

## Open Research Items

| Item | Needed evidence |
|---|---|
| Barcode import | Controlled `.ETQ` with barcode type/value/settings plus MasterPrint screenshot/print |
| Image/logo import | Controlled `.ETQ`, source image file, and screenshot/export |
| OLE import | Controlled `.ETQ` with embedded and linked OLE variants if possible |
| Mapa Risco | Controlled `.ETQ` showing the object and user-facing workflow |
| Artistic text | Controlled `.ETQ` with simple and styled examples |
| Database merge | ETQ plus data source/project files, screenshot of merge fields, and generated output |
| Counters | ETQ with counter object/settings and generated multi-label output |
| Full saved page overrides | ETQ where page setup dimensions were manually edited away from catalog defaults |

## Implementation Guardrails

1. Keep this report as a legacy-support inventory, not a clone checklist.
2. Keep ETQ import read-only until the native print workflow is already useful.
3. Preserve unsupported ETQ diagnostics instead of silently dropping unknown objects.
4. Use original Paulimaq WMF care symbols for compliance-sensitive textile symbols.
5. Keep layout, content orientation, preview, and print transforms centralized in the new engine.
6. Require controlled samples before implementing barcode, image, OLE, database, or custom object import.
