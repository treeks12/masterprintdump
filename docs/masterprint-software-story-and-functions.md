# MasterPrint / CadMapa Software Story And Function Map

This report explains what Paulimaq MasterPrint/CadMapa appears to have been, how a shop operator likely used it, and what the visible UI functions and known reverse-engineered routines mean for a new implementation.

It complements `docs\masterprint-cadmapa-feature-inventory.md`. The inventory is a compatibility checklist. This document is the narrative and function map.

## Scope And Confidence

The goal is context, not a promise to clone every old behavior.

| Scope item | Status |
|---|---|
| Visible menus, toolbars, captions, hints, and event handlers | High confidence from decoded Delphi form resource |
| Main workflows | High confidence from UI resource, Paulimaq assets/catalogs, and ETQ samples |
| Text and embedded WMF care-symbol ETQ import | High confidence for common textile-label files |
| Object runtime families | Medium confidence from reverse function switches and samples |
| Barcode, image, OLE, database merge, counters | Real legacy features, but low import confidence until controlled samples exist |
| Full ETQ writer or exact CadMapa rendering clone | Unsafe / not proven |

Primary evidence:

| Evidence | Path |
|---|---|
| Decoded main UI form | `C:\Users\HB\Projects\paulimaq-reverse\ciabrafe\analysis\TLAYOUTDESKTOP_decoded.txt` |
| Decoded UI strings | `C:\Users\HB\Projects\paulimaq-reverse\ciabrafe\analysis\TLAYOUTDESKTOP_strings.txt` |
| CadMapa reverse source archive | `C:\Users\HB\Projects\paulimaq-reverse\cadmapa-decompiled` |
| ETQ reverse report | `docs\etq-format-report.md` |
| Legacy feature inventory | `docs\masterprint-cadmapa-feature-inventory.md` |
| New app plan | `docs\new-label-canvas-printer-plan.md` |

## Short Story Of The Software

MasterPrint was not only an LNT textile-label tool. It was a Windows desktop publishing shell for Paulimaq printable media.

The evidence points to a Delphi-style application named around `TLayoutDesktop`, with an embedded drawing/editor engine called CadMapa. It combined four things:

| Part | What it did |
|---|---|
| Paulimaq catalog browser | Lets the user choose paper/media category and model from Paulimaq `*.inf` catalogs |
| Label/page designer | Lets the user place text, symbols, shapes, barcodes, figures, and OLE objects on a design canvas |
| Production printer | Prints the design into repeated slots on sheets/forms/rolls with margins, columns, and spacing |
| Business workflow layer | Offers database merge, counters, templates, recent files, and clipart panels |

The old user experience was probably:

| Step | Operator action | Evidence |
|---|---|---|
| 1 | Create or open an `.ETQ` document | `Novo`, `Abrir`, save dialog `Documento|*.ETQ` |
| 2 | Choose a document/media layout | `Configurar Documento`; Paulimaq `layout.ini` and `*.inf` catalogs |
| 3 | Design one label | drawing toolbar, text toolbar, object menus, status coordinates |
| 4 | Use Paulimaq textile symbols or other cliparts/images | `Clips` panel, `Figura`, WMF assets in `CLIPART\Símbolos` |
| 5 | Optionally merge variable data or counters | `Banco de Dados`, `Incluir Campo...`, `Incluir Contador...` |
| 6 | Print onto physical Paulimaq stock | `Imprimir`, catalog margins/columns/spacing |
| 7 | Save `.ETQ` or template for reuse | `Salvar`, `Salvar Como...`, `Salvar Como Modelo...` |

For textile composition labels, MasterPrint acted like a small controlled version of CorelDRAW/Word/label-merge software. For the new product, the important lesson is not to copy the UI chrome. The important lesson is to preserve the production workflow: choose real Paulimaq stock, design in millimeters, use compliant care symbols, preview honestly, print accurately, and import old files where safe.

## Mental Model

The old app appears to have these layers:

| Layer | Evidence | Meaning for new program |
|---|---|---|
| Shell/application UI | `TLayoutDesktop`, menu bars, toolbars, dialogs | Build a cleaner WPF shell; do not clone Delphi layout |
| Catalog/layout data | `layout.ini`, `*.inf`, `pageovrr.ini` | Parse generically and keep as production truth |
| Document format | `.ETQ` | Treat as legacy import/reference format |
| Canvas object model | runtime object type byte `+0x189`, object handlers | Rebuild simpler native model; do not inherit hidden state |
| Render engine | GDI, WMF/EMF, RichEdit/DrawText/TextOut, OLE draw | Use shared geometry engine plus WPF preview and GDI+/Win32 print |
| Print path | printer HDC, physical mm geometry | Production-critical; calibrate by printer/layout |
| Merge/workflow | database toolbar/menu, counters | Defer until layout/print foundation works |

## Main UI Areas

### Document Canvas

The main design surface is built around `ScrollBox1` and `Panel3`.

| Function | UI/resource evidence | Explanation |
|---|---|---|
| Scrollable canvas | `TScrollBoxEx ScrollBox1` | Allows viewing a page/design larger than the visible window |
| Drawing panel | `TPanelEx Panel3`, `Panel3Paint` | Main surface where labels/objects are drawn |
| Mouse editing | `Panel3MouseDown`, `Panel3MouseMove`, `Panel3MouseUp` | Selection, placement, drag, resize, object creation |
| Drag/drop | `Panel3DragOver`, `Panel3DragDrop` | Likely clipart/object insertion and possibly file drop |
| Ruler/header paint boxes | `PaintBox1`, `PaintBox2` | Horizontal/vertical ruler or guide areas |
| Keyboard handler | hidden `KeyHandler`, `KeyHandlerKeyDown`, `KeyHandlerKeyPress` | Central keyboard shortcuts and typed input routing |

New-product implication: implement a millimeter-aware WPF canvas with explicit selected-object coordinates. Do not reproduce the hidden CadMapa drawing surface.

### Status And Coordinates

The status bar includes labels:

| Label | Caption/hint | Meaning |
|---|---|---|
| `E:` | `Posição a esquerda` | Left/X position |
| `T:` | `Posição ao topo` | Top/Y position |
| `L:` | `Medida da Largura` | Width |
| `A:` | `Medida da altura` | Height |

This is important for operators. The new app should keep live numeric millimeter fields visible, because physical labels are small and precise.

### Clip Panel

The bottom dockable panel caption is `Clips`.

| Function | Evidence | Explanation |
|---|---|---|
| Clip category selector | `ComboBox2`, item `(Default)` | Asset category picker |
| Clip preview strip | `PaintBox3`, `TBXPageScroller1` | Scrollable clipart preview/list |
| Drag from clips | `PaintBox3StartDrag`, `PaintBox3MouseMove` | Likely drag-and-drop clip insertion |

For the new product, this maps naturally to a symbol library panel. Start with the original Paulimaq WMF textile-care symbols, not generic clipart.

## File Menu Functions

| Function / handler | UI caption | What it did | New-program stance |
|---|---|---|---|
| `NewClick` | `Novo`, toolbar `Nova Etiqueta` | Starts a new label/document | Core |
| `OpenClick` | `Abrir`, toolbar `Abrir Etiqueta` | Opens an existing `.ETQ` document | Native open core; ETQ read-only import later |
| `SaveClick` | `Salvar`, toolbar `Salvar Etiqueta` | Saves current document | Save native JSON; do not default to ETQ writer |
| `SaveAsClick` | `Salvar Como...` | Saves under a new name | Core |
| `SalvarComoModelo1Click` | `Salvar Como Modelo...` | Saves reusable template/model | Later production preset feature |
| `Reopen` submenu | `Reabrir` | Recent files | Nice-to-have |
| `ExportClick` | `Exportar...` | Exports document/output | Deferred; define formats after print works |
| `ConfigurarEtiqueta1Click` | `Configurar Documento` | Opens page/layout setup | Core; must use Paulimaq catalogs |
| `PrintClick` | `Imprimir` | Opens/starts print workflow | Core via GDI+/Win32 |
| `WDSpeedButton21Click` | toolbar `Imprimir` | Toolbar print action, maybe direct print or print dialog shortcut | Core but can map to same new print command |
| `Imprimirparafax1Click` | `Imprimir para fax`, hidden | Legacy fax print output | Ignore unless demanded |
| `Sair1Click` | `Sair`, toolbar `Saída` | Exits application | Normal shell function |

Story interpretation: MasterPrint was centered on `.ETQ` documents. The old file workflow assumed `.ETQ` was the save format. The new app should deliberately break that assumption for safety: native JSON is source of truth, ETQ is import/reference.

## Edit Menu Functions

| Function / handler | UI caption | What it did | New-program stance |
|---|---|---|---|
| `EditClick` | `Editar` menu open | Updates enabled/disabled state for edit commands | Implement naturally |
| `CutClick` | `Recortar`, toolbar `Cortar` | Removes selected objects and puts them on clipboard | Core editor usability |
| `CopyClick` | `Copiar`, toolbar `Copiar` | Copies selected objects | Core editor usability |
| `PasteClick` | `Colar`, toolbar `Colar` | Pastes clipboard objects | Core editor usability |
| `DeleteClick` | `Apagar` | Deletes selected objects | Core |
| `Procurar1` | `Procurar`, hidden | Search/find command | Ignore initially |

For old users, these are expected Windows editor functions. They are not the hard part. The hard part is preserving physical geometry and avoiding ETQ corruption.

## Object Menu Functions

| Function / handler | UI caption | What it did | New-program stance |
|---|---|---|---|
| `Objecto1Click` | `Objeto` menu open | Updates object command availability | Implement naturally |
| `Cor1Click` | `Borda` | Opens/sets border stroke color/style | Later shape support |
| `Preenchimento1Click` | `Preenchimento` | Opens/sets fill color/style | Later shape support |
| `Fonte1Click` | `Fonte` | Opens font settings for selected text | Core for text |
| `WDSpeedButton18Click` | `Enviar Para Trás` | Sends selected object backward in z-order | Useful after multi-object editing |
| `WDSpeedButton22Click` | `Trazer Para Frente` | Brings selected object forward in z-order | Useful after multi-object editing |
| `WDSpeedButton15Click` | `Agrupar` | Groups selected objects | Deferred |
| `WDSpeedButton16Click` | `Desagrupar` | Ungroups selected group | Deferred |
| `Alinhar1Click` / `WDSpeedButton3Click` | `Alinhar` / toolbar `Alinhamento` | Aligns selected objects or opens alignment options | Later editor feature |
| `Escalonar1Click` | `Escalonar` | Scales selected object(s) | Deferred; risky around physical dimensions |
| `PropertiesClick` | `Propriedades` | Opens object properties | Core concept via property panel |

The context popup menu duplicates these object commands. This means old users likely used right-click properties, border/fill/font, z-order, group, align, scale.

## Font Toolbar Functions

| Function / handler | UI caption/hint | What it did | New-program stance |
|---|---|---|---|
| `ComboBox3Click` | `Fontes` | Selects font family | Core |
| `ComboBox3DrawItem` | font combo owner-draw | Draws font list entries | WPF can handle differently |
| `ComboBox3Exit` | font combo exit | Commits/cancels font value | Implement naturally |
| `ComboBox3KeyDown` | font combo key handling | Keyboard font selection | Implement naturally |
| `ComboBox5Click` | `Tamanho da Fonte` | Selects font size | Core |
| `ComboBox5Exit` | size combo exit | Commits/cancels font size | Implement naturally |
| `ComboBox5KeyDown` | size combo key handling | Keyboard size entry | Implement naturally |
| `PopupPrintersClick` | `Negrito`, `Itálico`, `Sublinhado`, `Cortado` | Despite the misleading name, this handler is wired to bold/italic/underline/strikeout toolbar items | Core for bold/italic; underline/strike later |
| `WDSpeedButton30Click` | left/center/right alignment | Sets horizontal text alignment | Core for textile labels |
| `ToolbarButton971Click` | `Marcadores` | Toggles bullets/markers | Deferred |
| `ColorCombo3Change` | `Cor do Texto` | Sets text color | Core, default black |

Reverse text notes connect this UI to CadMapa text rendering:

| Reverse function | Role |
|---|---|
| `FUN_004bf12c` | Loads text object from ETQ stream |
| `FUN_004bf340` | Saves text object to ETQ stream |
| `FUN_004bf99c` | Adjusts average character width so text fits a rectangle |
| `FUN_004c036c` | Prepares font/rect/style and orchestrates text rendering |
| `FUN_004c0118` | Outputs text through metafile, `TextOutA`, or `DrawTextA` path |

Known CadMapa text fields from reverse notes:

| Field / behavior | Meaning |
|---|---|
| Geometry fields | Text objects carry X, Y, width, height in document units |
| Font height | Derived from rectangle height in the analyzed path |
| Style byte | Bits map to bold, italic, underline, strikeout |
| Alignment field | `0` left, `1` center, `2` right |
| `DrawTextA` path | Used for boxed/multiline style output |
| `TextOutA` path | Used for simpler point text output |

New-product implication: implement text as explicit native objects with font family, size/height, style, alignment, and rectangle. Do not attempt to exactly clone RichEdit/GDI quirks before printing is correct.

## Standard Toolbar Functions

| Function / handler | Toolbar hint | What it did |
|---|---|---|
| `NewClick` | `Nova Etiqueta` | Same as File > New |
| `OpenClick` | `Abrir Etiqueta` | Same as File > Open |
| `SaveClick` | `Salvar Etiqueta` | Same as File > Save |
| `WDSpeedButton21Click` | `Imprimir` | Print shortcut |
| `CutClick` | `Cortar` | Same as Edit > Cut |
| `CopyClick` | `Copiar` | Same as Edit > Copy |
| `PasteClick` | `Colar` | Same as Edit > Paste |
| `WDSpeedButton3Click` | `Alinhamento` | Alignment shortcut |
| `WDSpeedButton15Click` | `Agrupar` | Group shortcut |
| `WDSpeedButton16Click` | `Desagrupar` | Ungroup shortcut |
| `WDSpeedButton22Click` | `Trazer para Frente` | Bring forward shortcut |
| `WDSpeedButton18Click` | `Enviar para Trás` | Send backward shortcut |
| `WDSpeedButton33Click` | `Zoom de Página Inteira` | Whole-page zoom |
| `WDSpeedButton33Click` | `Zoom de Largura da Página` | Page-width zoom; same handler as whole-page item |
| `WDSpeedButton8Click` | `Zoom de 100%` | 100% zoom |
| `ComboBox1Click` | zoom combo | Applies selected zoom percentage/mode |
| `WDSpeedButton23Click` | `Ajuda` | Help shortcut |
| `Sair1Click` | `Saída` | Exit shortcut |

Zoom combo values were `25%`, `50%`, `75%`, `100%`, `150%`, `200%`, `300%`, `400%`, `Página Inteira`, and `Largura da Página`.

New-product implication: zoom must be purely visual. It must never alter physical label geometry.

## Drawing Tool Functions

Most drawing tool buttons are wired to the same handler, `WDSpeedButton2Click`. That strongly suggests the handler reads which button sent the event and sets the current creation tool.

| Tool | UI hint | Likely created object | New-program stance |
|---|---|---|---|
| Pointer | `Apontador` | Selection/edit mode | Core |
| Zoom | `Zoom`, handler `WDSpeedButton19Click` | Zoom interaction mode | Core/nice-to-have |
| Line | `Linha` | Line shape | Later |
| Rounded rectangle | `Retângulo Ovalado` | Rounded rectangle shape | Later |
| Rectangle/square | `Quadrado` | Rectangle shape | Later |
| Oval | `Oval` | Ellipse shape | Later |
| Simple text | `Texto Simples` | Plain text object | Core |
| Text box | `Caixa de Texto` | Boxed/multiline text | Core/later depending on first implementation |
| Artistic text | `Texto Artístico` | Special text/metafile mode | Deferred |
| Barcode | `Código de Barras` | Barcode object | Deferred, sample needed |
| Image | `Figura` via `btnImage` | Bitmap/figure object | Deferred, sample needed |
| Mapa Risco | `Figura` via `btnMapaRisc` | Domain-specific figure/object | Deferred, sample needed |
| OLE | `Ole` via `btnOle` | Embedded/linked OLE object | Unsupported initially |
| File-managed image | `Figura` via `btnFileMan` | External/file-managed figure path | Deferred |
| WordArt | `Ole`, hidden | WordArt OLE object | Ignore initially |
| Table | `Ole`, hidden | Table OLE object | Ignore initially |

This toolbar is the best evidence that CadMapa was a mini desktop-publishing engine, not a simple fixed label form.

## Database And Merge Functions

MasterPrint had a real database/merge area.

| Function / handler | UI caption/hint | What it did | New-program stance |
|---|---|---|---|
| `Dados1Click` | `Banco de Dados` menu open | Updates database menu state | Deferred |
| `WDSpeedButton6Click` | `Ativar/Desativar Mescla` | Toggles merge mode | Deferred |
| `WDSpeedButton7Click` | merge source dropdown `(nenhum)` | Selects active data source or merge field set | Deferred |
| `ToolbarButton972Click` | `Primeiro Registro` | Moves to first data record | Deferred |
| `WDSpeedButton4Click` | `Registro Anterior` | Moves to previous record | Deferred |
| `WDSpeedButton5Click` | `Próximo Registro` | Moves to next record | Deferred |
| `ToolbarButton973Click` | `Último Registro` | Moves to last record | Deferred |
| `IncluirCampo1Click` | `Incluir Campo...` | Inserts variable database field into document | Deferred |
| `Gerenciar1` submenu | `Gerenciar Dados` | Data-source management | Deferred |
| `TBXItem3Click` | `Incluir Contador...` | Inserts counter/serial field | Later if production needs numbering |
| `CountersClick` | `Gerenciar Contadores` | Manages counters | Later |

Story interpretation: MasterPrint could act like a label mail-merge tool. It probably allowed one design to be filled from records and printed in batches. For the new product, leave document-model room for variables, but do not implement this before physical layout/print is proven.

## Options And Help Functions

| Function / handler | UI caption | What it did | New-program stance |
|---|---|---|---|
| `Preferncias1Click` | `Preferências` | Opens app preferences | Later |
| `BarrasdeFerramenta1Click` | `Barras de Ferramentas` | Updates toolbar submenu | Optional |
| `Padro1Click` | `Padrão` | Toggles standard toolbar | Optional |
| `Fonte3Click` | `Fonte` | Toggles font toolbar | Optional |
| `BancodeDados2Click` | `Banco de Dados`, hidden | Toggles database toolbar | Ignore initially |
| `FerramentasdeDesenho1Click` | `Ferramentas de Desenho` | Toggles drawing toolbar | Optional |
| `TBXItem1Click` | `Cliparts` | Toggles clipart panel | Later |
| `ProcurarAjudaSobre1Click` | `Procurar Ajuda Sobre` | Help search | Optional |
| `ContentsClick` | `Conteúdo` | Opens help contents | Optional |
| `AboutClick` | `Sobre` | About dialog | Normal app shell |

The database toolbar toggle is marked hidden in the decoded resource. That suggests some capabilities may have been disabled by edition, configuration, or runtime state.

## Hidden And Special Functions

| Function | Evidence | Meaning |
|---|---|---|
| Fax printing | `Imprimir para fax`, hidden | Legacy output path; not relevant now |
| Search | `Procurar`, hidden | Search was present but hidden |
| WordArt | `btnWordArt`, hidden | OLE-style WordArt path existed but hidden |
| Table | `btnTable`, hidden | OLE-style table path existed but hidden |
| Database toolbar menu item | `Banco de Dados`, hidden under toolbar toggles | Merge toolbar may be conditionally exposed |

For new-product planning, hidden features should not be prioritized unless a real customer file depends on them.

## Object Runtime Function Map

CadMapa uses an object type byte at runtime offset `+0x189`. `FUN_004ba3e0` stores the type and allocates a handler object.

| Type byte | Handler evidence | Likely family | Confidence |
|---:|---|---|---|
| `0` | `FUN_004bb77c` | Basic object, possibly line/shape base | Low |
| `1` | `FUN_004bb77c` | Basic object, possibly line/shape base | Low |
| `2` | `FUN_004bb77c` | Basic object, possibly line/shape base | Low |
| `3` | `FUN_004bb77c` | Basic object, possibly line/shape base | Low |
| `4` | `FUN_004bf068` | Text-family object | Medium |
| `5` | `FUN_004bf068` | Text-family object | Medium |
| `6` | `FUN_004bd71c` | Bitmap/DIB-style figure | Medium-low |
| `7` | `FUN_004befdc` | Barcode or specialized object candidate | Low |
| `8` | `FUN_004bb77c` with different pointer | Alternate object family | Low |
| `9` | `FUN_004c066c` | OLE-backed figure/image wrapper | Medium |
| `10` | `FUN_004bb77c` with another pointer | File-managed/special figure candidate | Low |
| `11` | `FUN_004bf068` | Text-family object | Medium |

`FUN_004b5870` checks the selected object's type and decides whether command categories are available. That explains why menu items such as border, fill, font, alignment, and properties enable/disable depending on the selected object.

Important: this type map is not enough to write ETQ files. It only helps explain what the old engine could contain.

## Text Rendering Function Map

The text reverse work shows a practical CadMapa text pipeline:

| Reverse function | Practical explanation |
|---|---|
| `FUN_004bf12c` | Reads text-object fields from the ETQ stream |
| `FUN_004bf340` | Writes text-object fields back to the ETQ stream |
| `FUN_004bf99c` | Computes/fits average character width to make text fit its box |
| `FUN_004c036c` | Builds GDI font parameters from object fields and calls output logic |
| `FUN_004c0118` | Chooses final output route: metafile, `TextOutA`, or `DrawTextA` |

Observed behavior relevant to the new app:

| Behavior | Meaning |
|---|---|
| Text object geometry is rectangular | New text objects should have explicit X/Y/W/H in mm |
| Font height can be tied to object rectangle height | The old app may scale text by resizing the object |
| Alignment is a field, not only a toolbar state | Import should preserve left/center/right when known |
| RTF exists in ETQ payloads | Preserve raw RTF for diagnostics, but import plain text safely |

## WMF / Symbol Function Map

The WMF reverse work shows how Paulimaq care symbols were drawn.

| Reverse function | Practical explanation |
|---|---|
| `FUN_00423530` | Renders WMF/EMF through `PlayEnhMetaFile(hdc, hmf, rect)` |
| `FUN_00423740` | Gets intrinsic metafile width |
| `FUN_00423728` | Gets intrinsic metafile height |
| `FUN_00423d68` | Sets intrinsic metafile width |
| `FUN_00423d2c` | Sets intrinsic metafile height |
| `FUN_004238c4` | Loads metafile payload |
| `FUN_00423f00` | Saves metafile payload |
| `FUN_00423a00` | Loads Aldus WMF |
| `FUN_00423930` | Loads EMF |
| `FUN_00424010` | Saves Aldus WMF |
| `FUN_00423f84` | Saves EMF |

Important lessons:

| Lesson | New-program rule |
|---|---|
| Embedded WMF blob identity is not the filename | Identify old symbols by hash of embedded WMF body |
| Placement rectangle is external to Aldus bounds | Draw symbol in decoded object rect, not at WMF internal bounds |
| Original Paulimaq WMFs are compliance-sensitive | Use the installed/bundled WMFs directly |

## Bitmap, Figure, And OLE Function Map

The old UI exposes several image-like tools. Reverse evidence shows this area is complex.

| Reverse function / API | Practical explanation |
|---|---|
| `FUN_004bd71c` | Allocates/initializes a bitmap/DIB-style figure handler |
| `FUN_004c066c` | Allocates an OLE-backed figure/image wrapper |
| `FUN_0049ac24` | Uses `OleCreate`, `OleCreateFromFile`, `OleCreateLinkToFile`, `OleCreateFromData`, `OleCreateLinkFromData` |
| `FUN_0049bdac` | Writes OLE payload header and storage bytes; normal header appears as `BDOC` |
| `FUN_0049b7f8` | Reads OLE payload header and rejects non-`BDOC` unless special graphical mode is active |
| `FUN_0049ba34` | Draws embedded object through `OleDraw` into a destination rectangle |
| DIB/BMP paths | Reverse notes include `GetDIBits`, `StretchDIBits`, and `BM` load/write paths |

Support rule: old files with logos/photos/images/OLE need controlled samples. Ask for the `.ETQ`, original image file if available, and a MasterPrint screenshot/export. Do not guess import from a `BM`, JPEG, PNG, or OLE signature alone.

## ETQ Document Function Map

Common textile ETQ files currently decode through validated scanning rather than a full Delphi object-graph reader.

| ETQ area | Current understanding |
|---|---|
| Text record marker | `FE FF FF FF` with `flags=0`, tag usually `1`, sometimes `0` |
| WMF record marker | `FE FF FF FF` with WMF payload nearby and Aldus signature `D7 CD C6 9A` |
| Geometry units | Usually hundredths of millimeters |
| Text payload | Plain Latin-1 or RTF-like payload at `FE+40` with length at `FE+38` |
| WMF placement | X/Y in pre-block before marker, width/height near marker |
| Object chain | `nextX/nextY` behavior observed, but duplicates require file-offset ordering fallback |
| Unknown objects | Must be preserved as diagnostics, not silently dropped |

New-product ETQ policy:

| Policy | Reason |
|---|---|
| Import only validated objects | Avoid false positives and data loss |
| Preserve unsupported-object diagnostics | User must know when an old file is partial |
| Do not save ETQ by default | Writer is not proven safe |
| Convert to native document after import | New model should be explicit and maintainable |

## Page Setup And Catalog Functions

The `Configurar Documento` flow is central. MasterPrint's page setup dialog, observed in the handoff reference image, exposes:

| Field | Meaning |
|---|---|
| `Tipo` | Paper/category selector |
| `Modelo` | Model/layout selector |
| `Núm. de Colunas` | Column count |
| `Margem Esquerda` | Left margin |
| `Margem Superior` | Top margin |
| `Largura` | Label/slot width |
| `Altura` | Label/slot height |
| `Entre Colunas` | Horizontal spacing |
| `Entre Linhas` | Vertical spacing/feed gap |
| `Medidas Originais` | Restore catalog defaults |

Story interpretation: Paulimaq catalogs supplied defaults, but the operator could edit dimensions. That means the new native document should store `layoutId` and optional `layoutOverride` instead of assuming model name fully defines the document.

## Print Function Map

The old print feature was the business reason the program existed. Its visible functions are simple, but the internal behavior is production-critical.

| Function | Explanation |
|---|---|
| `PrintClick` | File-menu print command |
| `WDSpeedButton21Click` | Toolbar print shortcut |
| `Imprimirparafax1Click` | Hidden fax print route |
| Catalog/page setup | Determines physical cells and margins |
| CadMapa render functions | Draw text, symbols, shapes, images into printer HDC |

New-product rules:

| Rule | Reason |
|---|---|
| Use GDI+/Win32 production print path | Need accurate printer HDC, WMF playback, and mm transforms |
| Do not print WPF visuals as final production path | Preview rendering is not enough for physical stock |
| Keep page orientation and content orientation separate | `Paisa=1` is not simply sheet landscape |
| Calibrate per printer/layout/orientation | Real printers drift and stocks differ |
| First physical gate is LNT-2 | Current production pain point |

## Lifecycle And Utility Handlers

These functions are not user-facing features, but they explain the old application's shell behavior.

| Handler | Meaning |
|---|---|
| `FormCreate` | Initializes form, toolbars, document/editor state |
| `FormDestroy` | Releases resources |
| `FormCloseQuery` | Checks whether closing is allowed, likely unsaved changes |
| `FormResize` | Reflows docks/canvas/status areas |
| `Panel1Resize` | Reflows clipart panel |
| `Toolbar971Recreated` | Font toolbar recreation hook |
| `Toolbar974Recreated` | Standard toolbar recreation hook |
| `TipTimerTimer` | Tooltip/status timing behavior |
| `Timer1Timer` | Short-interval deferred UI/task processing |
| `ColorCombo1Change` | Border color change |
| `ColorCombo2Change` | Fill color change |
| `ColorCombo3Change` | Text color change |

## What MasterPrint Was In One Sentence

MasterPrint was a Paulimaq media-aware desktop publishing and print-production application: it let users pick Paulimaq stock, design label artwork with text/symbols/shapes/images/barcodes/OLE, optionally bind data/counters, and print repeated output onto physical sheets/forms/rolls.

## What The New Program Should Preserve

| Preserve | Why |
|---|---|
| Paulimaq catalog breadth | Old software covered many paper/media families |
| Millimeter-based design and status | Operators need physical precision |
| Original WMF care symbols | Textile compliance and visual continuity |
| Page setup vocabulary | Old users understand `Tipo`, `Modelo`, margins, columns, dimensions |
| Correct print behavior | This is the core production value |
| Safe ETQ import for common textile labels | Reduces migration pain |
| Honest unsupported-object reporting | Prevents hidden data loss |

## What The New Program Should Not Preserve Blindly

| Do not preserve blindly | Reason |
|---|---|
| Exact Delphi toolbar layout | Not the value; modern WPF can be simpler |
| Hidden OLE/WordArt/table behavior | Low confidence and likely obsolete |
| Full ETQ save semantics | Unsafe without byte-compatible writer proof |
| CadMapa's hidden render transforms | Caused the failed clone path |
| Screenshot-matched geometry | Physical printing must come from catalog, samples, and calibration |
| LNT-only assumptions | MasterPrint covered many Paulimaq catalog categories |

## Practical Compatibility Roadmap

| Order | Function area | Why this order |
|---:|---|---|
| 1 | Catalog/page setup/load all INF files | Defines physical truth |
| 2 | Native document save/load | Safe source of truth |
| 3 | LNT-2 canvas and print calibration | First real production gate |
| 4 | Text objects and original WMF symbols | Core textile label content |
| 5 | Read-only ETQ import for text/WMF | Highest-value legacy migration |
| 6 | Object operations: copy, paste, delete, z-order, properties | Makes editor usable |
| 7 | Simple shapes | Matches common DTP expectations |
| 8 | Templates and repeat/partial-sheet workflows | Production convenience |
| 9 | Counters/data merge | Real legacy feature, but after print truth |
| 10 | Barcode/image/OLE import research | Requires controlled samples |

## Remaining Unknowns

| Unknown | Evidence needed |
|---|---|
| Exact barcode object fields | ETQ with barcode plus screenshot/print/reference value |
| Image/logo storage variants | ETQ, original image, screenshot/export |
| OLE embedded versus linked behavior | Controlled ETQ samples for each mode |
| Mapa Risco object meaning | User workflow and controlled sample |
| Artistic text semantics | Simple controlled samples |
| Database merge file formats and bindings | ETQ plus data files and output examples |
| Saved page override bytes in ETQ | ETQ with manually edited page setup values |
| Full object-type mapping | Multiple controlled ETQs, one object type at a time |

## Final Product Interpretation

The new application should be a modern Paulimaq label studio, not a CadMapa clone. MasterPrint's history tells us which jobs users cared about: selecting real stock, building printable labels, reusing files/templates, using textile symbols, and printing reliably. The old UI also tells us which advanced features may appear in customer files: barcodes, figures, OLE, merge fields, counters, groups, and shapes.

The safest path is to build a clean native model and bring legacy features forward only when backed by real samples and reverse evidence.
