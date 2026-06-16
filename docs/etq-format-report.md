# Paulimaq `.ETQ` Reverse-Engineering Report

This report summarizes what is currently known about Paulimaq/MasterPrint `.ETQ` files from the archived `masterprint-native` research project. It is intended for a new implementation that wants read-only `.ETQ` import without inheriting the failed CadMapa clone path.

Status summary:

| Area | Confidence | Notes |
|---|---|---|
| Common textile LNT ETQ text records | High | Proven against 58 installed corpus files |
| Common textile LNT embedded WMF records | High | Proven against corpus and original clipart hashes |
| Header layout/template extraction | Medium | Works for common header strings, not a full container parser |
| Object order / chain | Medium | `nextX/nextY` behavior understood enough to avoid corruption, but duplicate keys force file-offset fallback |
| RTF text import | Medium | Plain display text and raw RTF preserved; full RichEdit layout is not reconstructed |
| ETQ writing | Low / unsafe | Only no-op copy and narrow same-length/position patch experiments exist |
| Barcode objects | Runtime known / ETQ bytes unknown | MasterPrint has a real type-7 barcode object, but no barcode `.ETQ` sample exists |
| Non-text/WMF objects | Unknown | Barcode/OLE/table/mapa-risco/custom objects are not decoded |

Important conclusion: treat `.ETQ` as an import/reference format. Do not make it the native source of truth. The new program should import supported objects into native JSON and keep unknown bytes only for diagnostics.

## Evidence Sources

| Source | Path |
|---|---|
| Parser | `C:\Users\HB\Projects\masterprint-native\internal\etq\parser.go` |
| Patcher experiment | `C:\Users\HB\Projects\masterprint-native\internal\etq\patcher.go` |
| Corpus baseline | `C:\Users\HB\Projects\masterprint-native\internal\etq\corpus_test.go` |
| Parser edge tests | `C:\Users\HB\Projects\masterprint-native\internal\etq\parser_test.go` |
| Clipart identity tests | `C:\Users\HB\Projects\masterprint-native\internal\etq\clipart_catalog_test.go` |
| Save/patch gates | `C:\Users\HB\Projects\masterprint-native\etq_save.go` and `etq_save_test.go` |
| Round-trip sidecar/MPN harness | `C:\Users\HB\Projects\masterprint-native\etq_roundtrip_test.go` |
| Research notes | `C:\Users\HB\Projects\masterprint-native\docs\research\grok-etq-samples.md` |
| Decompiled text notes | `C:\Users\HB\Projects\masterprint-native\docs\research\grok-cadmapa-text.md` |
| Decompiled WMF notes | `C:\Users\HB\Projects\masterprint-native\docs\research\grok-cadmapa-wmf.md` |

Known sample bundle for the new project:

```text
C:\Users\HB\Projects\masterprint-new-handoff\samples\Canelado algodão (Classic Wave Ramado) lunelli.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\samples\ADAR SOFA CANELADO.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\samples\FAVERO.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\samples\LASER 4 - VICENZA E LINO.ETQ
```

Controlled LNT baseline set:

```text
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt0.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt1.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt2.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt3.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt4.ETQ
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt0.jpg
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt1.jpg
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt2.jpg
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt3.jpg
C:\Users\HB\Projects\masterprint-new-handoff\test\lnt4.jpg
```

These files were created from the same visual textile label idea across LNT-0 through LNT-4. Positions are manual/approximate, but the text and symbols are intentionally comparable. The paired JPGs are the visual truth source from MasterPrint.

Current parser dump for the controlled set:

| File | Detected template | Text objects | WMF objects | Unknown objects | Notes |
|---|---|---:|---:|---:|---|
| `lnt0.ETQ` | `LNT-0` | 7 | 7 | 0 | Visual reference is clipped because the design is taller/wider than the short LNT-0 canvas |
| `lnt1.ETQ` | `LNT-1` | 7 | 7 | 0 | Visual reference is clipped at the bottom/right compared with larger LNTs |
| `lnt2.ETQ` | `LNT-2` | 6 | 7 | 0 | Good baseline for current production target |
| `lnt3.ETQ` | `LNT-3` | 6 | 7 | 0 | Same object set as LNT-2/LNT-4 by current parser |
| `lnt4.ETQ` | `LNT-4` | 6 | 7 | 0 | Same object set as LNT-2/LNT-3 by current parser |

Symbols found in all five controlled files:

| Symbol | Evidence |
|---|---|
| `clorox.wmf` | embedded body hash match |
| `tamborx.wmf` | embedded body hash match |
| `secox.wmf` | embedded body hash match |
| `secah.wmf` | embedded body hash match |
| `lavmao.wmf` | embedded body hash match |
| `ferro--.wmf` | embedded body hash match |
| `seco--w.wmf` | embedded body hash match |

Important observation from this controlled set: switching the selected LNT template changes the detected header/template and sometimes object offsets/counts, but many embedded objects keep identical geometry and hashes. This makes the set useful for testing template detection, symbol identity, clipping behavior, and import consistency across LNT variants.

## MasterPrint Page Configuration Reference Image

Reference file:

```text
C:\Users\HB\Projects\masterprint-new-handoff\test\exemplos do que o masterprint faz.jpg
```

This image is not byte-level `.ETQ` evidence. It is UI/catalog evidence from MasterPrint's `Configuração da Página` dialog. Use it to understand user-facing categories, model names, and the page setup fields operators see. Numeric truth must still come from `layout.ini`, `*.inf`, and real `.ETQ` samples.

Visible common page setup fields:

| UI field | Meaning for new program |
|---|---|
| `Tipo` | Paper/catalog category selector |
| `Modelo` | Layout/model within the selected category |
| `Núm. de Colunas` | Column count for sheet/form/roll previews |
| `Margem Esquerda` | Left margin in millimeters |
| `Margem Superior` | Top margin in millimeters |
| `Largura` | Physical label/slot width in millimeters |
| `Altura` | Physical label/slot height in millimeters |
| `Entre Colunas` | Horizontal spacing/gap between columns |
| `Entre Linhas` | Vertical spacing/gap or feed gap |
| Preview panel | Visual paper/category preview only; do not reverse numeric dimensions from pixels |
| `Medidas Originais` | Reset/restore original catalog dimensions |

The fields shown in this dialog are editable in MasterPrint. That matters: `layout.ini` and `*.inf` define catalog defaults, but a saved document may have user-edited page/layout values.

Import implication:

| Concept | Rule |
|---|---|
| Catalog layout | Default values loaded from `layout.ini` + `*.inf` |
| Document layout override | Values saved/edited in an `.ETQ`, if found or inferred |
| Reset/original dimensions | Equivalent to going back to catalog defaults |
| Native document | Store both `layoutId` and optional `layoutOverride` |

Do not assume `TemplateName = LNT-2` is enough to reconstruct the exact saved page setup. The importer should first resolve the catalog layout by ID, then compare/import any document-level dimensions, margins, columns, or spacing discovered in the ETQ/header/reverse data.

Visible `Tipo` categories in the dropdown include:

| Category shown | Import/catalog implication |
|---|---|
| `Etiq. para Composições em Folhas` | Sheet composition labels, including LNT and SONTARA variants |
| `Etiq. para Composições em Formulários` | Continuous/form composition labels with multiple column variants |
| `Etiq. para Composições em Rolo` | Roll composition labels with feed spacing |
| `TAG'S em Folhas e Formulários` | Tags in sheets/forms |
| `Etiq. para Jóias` | Jewelry labels |
| `Etiq. para Caixas de Calçados` | Shoe-box labels |
| `Cartões de Visita - PRINT CARD` | Business-card sheets and orientations |
| `Caixa de Cartões - PRINT BOX` | Card/box package layout |
| `Convites - PRINT INVITE` | Invitation/card layouts |
| `Pulseiras Bands` | Wristband/band layouts |
| `Etiq. Box para CD - PRINT CD FACE` | CD box/face layout |
| `Etiquetas para CD - CD Center` | CD center labels |
| `Etiquetas para CD - CD FAST LABEL 2` | CD fast-label category |
| `Etiquetas para CD - PRINT CD LABEL 2` | CD label category |
| `Etiquetas para CD - Mini CD` | Mini CD layout |
| `Etiquetas para CD - PRINT CD LABEL 3` | CD label category |
| `Print CD Cards` | CD card layout |
| `Photo Quality Álbum` | Photo album layout |
| `Etiq. para Plantas` | Plant labels |
| `Etiq. Ades. Fast Label (Redonda)` | Round adhesive fast labels |
| `Etiq. Ades. Fast Label (Padrão)` | Standard adhesive fast labels |
| `Pauli - Tab` | Pauli-Tab style labels |

Visible model examples:

| Category | Model examples visible in the image |
|---|---|
| Composition sheets | `LNT-0 (15,2x40,0mm)`, `LNT-1 (25,0x40,0mm)`, `LNT-2 (25,0x55,0mm)`, `LNT-3 (33,0x55,0mm)`, `LNT-4 (33,0x69,9mm)` |
| Composition sheets | `SONTARA LCS-1 (25,0x40,0mm)`, `SONTARA LCS-2 (25,0x55,0mm)`, `SONTARA LCS-3 (33,0x55,0mm)`, `SONTARA LCS-4 (33,0x69,9mm)` |
| Composition sheets | `Nylon ECNY (23x48mm)`, `Nylon ECNY (32x48mm)` |
| Composition forms | `NT/TY/NY-1 (25,4x44,5mm) 8 COL.`, `NT / TY-1 (25,4x44,5mm) 6 COL.`, `NT / TY-2 (25,4x50,8mm)`, `NT / TY-3 (33,0x50,8mm)`, `NT / TY-4 (33,0x76,2mm)` |
| Composition forms | `NY-2A (25,4x54,0mm)`, `NY-3A (33,0x54,0mm)`, `NY-4A (33,0x69,9mm)`, `NY-5A (38,1x63,5mm)` |
| Composition rolls | `TYB-2 (25,4x51,0mm)`, `TYB-3 (33,0x51,0mm)`, `TYB-4 (33,0x76,2mm)`, `TYB-5 (50,0x76,2mm)`, `TYB-6 (25,4x76,2mm)`, `TYB-7 (19,0x43,0mm)` |
| Composition rolls | `NYR-1 (33,0x55,0mm)`, `NYR-2 (33,0x76,0mm)`, `NYR-3 (25,0x51,0mm)` |
| Card box | `Caixa Padrão` |
| Business cards | `Cartões de Visita Print Card VER`, `Cartões de Visita Print Card HOR`, `PRINT CARD PLUS VER`, `PRINT CARD PLUS HOR` |
| Shoe-box labels | `CS0210 / LJA 272`, `FACS` |

Implications for `.ETQ` import:

| Observation | Import behavior |
|---|---|
| ETQ header may contain a user-facing category string | Preserve it as `layoutType` metadata and use it as a hint for catalog lookup |
| ETQ header may contain a model like `LNT-2` | Prefer exact model ID lookup in the parsed INF catalog |
| The same visual object set can be saved under different LNT models | Importer must not hardcode LNT-2 geometry for every ETQ |
| Some categories are sheets, some forms, some rolls, some special shapes | Native catalog model must expose media kind and category, not only width/height |
| UI preview is category-specific | Preview renderer should be driven by parsed layout metadata, not screenshot-matched drawings |
| Visible fields match INF concepts | Catalog loader should keep columns, margins, label size, and spacing as first-class fields |
| MasterPrint lets users edit size/spacing fields | Native documents need optional per-document layout overrides instead of only immutable catalog presets |

Recommended use in tests:

```text
1. Assert every visible category maps to at least one loaded INF catalog category or a documented unsupported category.
2. Assert LNT-0/1/2/3/4 model names from ETQ headers resolve to catalog layouts.
3. Assert form and roll examples remain loaded/previewable even before they are production-calibrated.
4. Add tests for a layout document with edited width/height/margins/spacing once ETQ storage for overrides is identified.
5. Do not create pixel-based tests from this JPG. Use it only for category/model coverage and UI vocabulary.
```

## File-Level Observations

The current parser does not decode a clean top-level Delphi/DFM object graph. It scans the byte stream for validated object envelopes and extracts useful records.

Useful header strings appear near the beginning of common files. The parser extracts layout metadata by scanning the first 512 bytes for printable Latin-1 strings and matching:

```text
([A-Z]{2,5}-\d+)\s*\((\d+(?:,\d+)?)x(\d+(?:,\d+)?)mm\)
```

For `Canelado algodão (Classic Wave Ramado) lunelli.ETQ`, this yields:

| Field | Value |
|---|---|
| `LayoutType` | `Etiq. para Composições em Folhas` |
| `TemplateName` | `LNT-2` |

The parser also tries to find a printer name by scanning for printable strings with prefixes such as `Epson`, `HP`, `Canon`, and `Zebra`. This is useful metadata only, not structural truth.

## Units

Most object geometry fields observed in common textile ETQs are stored as integer hundredths of a millimeter:

```text
raw = round(mm * 100)
mm = raw / 100.0
```

Examples:

| Raw | Millimeters |
|---:|---:|
| `2056` | `20.56 mm` |
| `353` | `3.53 mm` |
| `445` | `4.45 mm` |
| `3637` | `36.37 mm` |

Do not mix this with printer pixels or WPF device-independent units. The importer should convert ETQ raw values to millimeters immediately, then render using the new engine's layout transform.

## Common Object Envelope

Supported objects are found through a recurring marker:

```hex
FE FF FF FF
```

In little-endian `uint32`, this is `0xFFFFFFFE`.

The observed envelope has common fields at fixed offsets from the marker. The names below are descriptive, not official Paulimaq names.

| Offset from `FE` | Size | Text meaning | WMF meaning |
|---:|---:|---|---|
| `+0` | 4 | Marker `FE FF FF FF` | Marker `FE FF FF FF` |
| `+8` | 4 | `flags`, usually `0` | `flags`, `0x80000008` |
| `+12` | 4 | `tag`, usually `1`, sometimes `0` for text-like records | `tag`, `0` |
| `+16` | 4 | `x` in hundredths of mm | `width` in hundredths of mm |
| `+20` | 4 | `y` in hundredths of mm | `height` in hundredths of mm |
| `+32` | 2 | horizontal alignment for text | not used for WMF import |
| `+38` | 2 | text payload length | not used for WMF import |
| `+40` | variable | text or RTF payload | part of WMF envelope padding before Aldus blob |

The scanner must validate records; it must not treat every `FE FF FF FF` sequence as a real object. Some ETQs contain false-positive `FE` bytes inside embedded WMF blobs.

## Text Records

### Detection Rules

A text record is currently accepted when:

```text
data[offset:offset+4] == FE FF FF FF
flags = u32le(offset + 8) == 0
tag = u32le(offset + 12) is 1 or 0
rawX = u32le(offset + 16)
rawY = u32le(offset + 20)
textLength = u16le(offset + 38)
1 <= textLength <= 4096
payload is present at offset + 40
payload is followed by FF FF FF FF
payload decodes as plausible document text or RTF with non-empty plain text
```

Tag `1` is the normal text tag. Tag `0` can also be text-like in real files and must not be automatically ignored.

Observed tag `0` text-like examples:

| File | Offset | Text |
|---|---:|---|
| `ADAR SOFA CANELADO.ETQ` | `0x72c` | `HB GIRLS` |
| `RIBANA CANELADO NEON - 98% 2%.ETQ` | `0x3d0` | `ÚNICO` |
| `RIBANA CANELADO NEON - 98% 2%.ETQ` | `0x844` | `HB GIRLS` |
| `SUEDE ERRADO.ETQ` | `0x54b` | `ÚNICO` |
| `tuly.ETQ` | `0x26a` | `ÚNICO` |

### Text Payload

Payload starts at:

```text
payloadOffset = feOffset + 40
payloadLength = u16le(feOffset + 38)
```

The payload can be plain Latin-1 text or RTF.

Plain text rules used by the parser:

```text
trim trailing NUL bytes
reject bytes below 0x20, byte 0x7F, and byte 0xFF
decode as Latin-1
trim whitespace
```

RTF rules used by the parser:

```text
payload starts with "{\rtf"
decode bytes as Latin-1
extract plain text by skipping font/color/style groups
preserve raw RTF bytes separately
detect \fsN as half-point font size if needed
detect \b as bold hint
```

The RTF importer is intentionally not a full RTF renderer. It is a way to recover display text and preserve raw bytes for diagnostics or future import improvements.

### Text Terminator and Post Fields

After the text payload, a four-byte terminator appears:

```hex
FF FF FF FF
```

After that terminator, the parser reads post fields:

| Offset | Size | Meaning |
|---:|---:|---|
| `post + 0` | 4 | text rectangle height in hundredths of mm |
| `post + 4` | 4 | text rectangle width in hundredths of mm |
| `post + 8` | 4 | `nextX` chain coordinate |
| `post + 12` | 4 | `nextY` chain coordinate |
| `post + 16` | 1 | style byte |

Where:

```text
post = feOffset + 40 + payloadLength + 4
```

Example from `Canelado algodão (Classic Wave Ramado) lunelli.ETQ`, offset `0x15d3` / decimal `5587`:

| Field | Value |
|---|---:|
| Text | `72% ALGODÃO` |
| Raw X | `2056` = `20.56 mm` |
| Raw Y | `353` = `3.53 mm` |
| Rect height | `161` = `1.61 mm` |
| Rect width | `3637` = `36.37 mm` |
| Align | `0` = left |
| Style byte | `5` |
| Next X | `2238` |
| Next Y | `419` |

Important: text rectangle height comes before width in the post block. Do not name or use these fields backwards.

### Font Name

The current parser extracts font name heuristically from bytes before the `FE` marker:

```text
fontBlockStart = feOffset - 48
fontNameStart = fontBlockStart + 3
fontName = bytes until NUL or FE marker
fallback = Arial
```

This works for the current corpus but is not a fully proven structural field map. Treat the font name as useful imported metadata, not proof that the entire text object was decoded.

### Font Size

The most reliable render-related size for normal text records is the rectangle height, not the RTF `\fs` value.

Current mapping:

```text
fontSizePt = rectHeightMM * 72 / 25.4
```

This follows decompiled CadMapa text behavior: font height is derived from the object `RECT` height. In `ADAR SOFA CANELADO.ETQ`, a CNPJ RTF record has an RTF `\fs18`, but the parser test expects font size from ETQ rect height `5.74 mm`, not from `\fs18`.

### Style Byte

Observed style bytes include `4`, `5`, and `6`.

The parser decodes style as a bitmask:

| Bit | Meaning |
|---:|---|
| `0x01` | Bold |
| `0x02` | Italic |
| `0x04` | Underline |
| `0x08` | Strikeout, supported by decompile notes but not fully modeled in old Go structs |

Examples:

| Style byte | Decoded |
|---:|---|
| `4` | underline |
| `5` | bold + underline |
| `6` | italic + underline |

RTF bold can also force imported `Bold = true` even if the style byte alone does not.

### Alignment

Observed horizontal alignment field at `FE + 32`:

| Value | Meaning | CadMapa `DrawTextA` flags from reverse notes |
|---:|---|---|
| `0` | left | `0x810` |
| `1` | center | `0x811` |
| `2` | right | `0x812` |

No solid evidence of vertical centering was found in the analyzed CadMapa path.

### Text Edge Cases

The parser must handle these cases:

| Case | Evidence | Required importer behavior |
|---|---|---|
| Empty hidden text node | `SLIM.ETQ` offset `0x15d1` has `textLength=0` | Keep it out of visible objects; it can still participate in the chain |
| False-positive FE inside WMF | `ADAR SOFA CANELADO.ETQ` offset `0x713b` resembles text with payload `M` | Must be rejected; do not import as a text object |
| Truncated EOF text fragment | `ALGODÃO.ETQ` offset `0x5686` has readable bytes but no complete terminator/post block | Skip it; do not import partial object |
| Duplicate content | `CARACOL TULE.ETQ` has two `4% ELASTANO` records | Do not deduplicate by text content |
| Tag `0` text-like object | Several files | Promote to text when payload validation passes |

## WMF Symbol Records

### Detection Rules

A WMF record is currently accepted when:

```text
data[offset:offset+4] == FE FF FF FF
flags = u32le(offset + 8) == 0x80000008
tag = u32le(offset + 12) == 0
widthRaw = u32le(offset + 16)
heightRaw = u32le(offset + 20)
Aldus header starts at offset + 49
Aldus signature at offset + 49 is D7 CD C6 9A
there is a valid pre-block 83 bytes before FE
pre-block starts with FF FF FF FF
pre-block contains non-zero x/y
```

### WMF Geometry

Unlike text records, WMF position and size are split across two places.

Observed mapping:

| Field | Location | Meaning |
|---|---|---|
| `x` | `feOffset - 83 + 4` | object left in hundredths of mm |
| `y` | `feOffset - 83 + 8` | object top in hundredths of mm |
| `width` | `feOffset + 16` | object width in hundredths of mm |
| `height` | `feOffset + 20` | object height in hundredths of mm |

The 83-byte pre-block is stream/container metadata around the object; it is not part of the WMF payload itself.

Example from `Canelado algodão (Classic Wave Ramado) lunelli.ETQ`, `clorox.wmf` at offset `0x0111` / decimal `273`:

| Field | Value |
|---|---:|
| FE offset | `0x0111` |
| Pre-block offset | `0x00be` (`0x0111 - 83`) |
| Flags | `0x80000008` |
| Tag | `0` |
| Width raw | `445` = `4.45 mm` |
| Height raw | `445` = `4.45 mm` |
| X raw | `529` = `5.29 mm` |
| Y raw | `2696` = `26.96 mm` |
| Style byte | `6` |
| Next X/Y | `445`, `445` |

Example from `FAVERO.ETQ`, `secox.wmf` at offset `0x114f`:

| Field | Value |
|---|---:|
| Width raw | `496` = `4.96 mm` |
| Height raw | `639` = `6.39 mm` |
| X raw | `137` = `1.37 mm` |
| Y raw | `2988` = `29.88 mm` |

Important: WMF objects are not always square. Never assume `width == height`.

### Embedded Aldus WMF Payload

The ETQ stores embedded WMF bytes, not just filenames.

Observed payload start:

```text
aldusOffset = feOffset + 49
```

Expected Aldus placeable WMF signature:

```hex
D7 CD C6 9A
```

Payload size is computed from the standard WMF header after the 22-byte Aldus placeable header:

```text
std = aldusOffset + 22
words = u32le(std + 6)
wmfSizeBytes = 22 + words * 2
wmfBlob = data[aldusOffset : aldusOffset + wmfSizeBytes]
```

This blob should be preserved if the importer wants future export/debugging.

### WMF Identity

The correct symbol identity strategy is:

```text
sha256(wmfBlob[22:])
```

Rationale:

| Method | Status |
|---|---|
| Full blob hash | Can differ if Aldus placeable header differs |
| Body hash `blob[22:]` | Proven unique across 49 installed symbols |
| Blob size | Works for the installed 49-symbol catalog but should be fallback only |
| Filename in ETQ | Not stored as primary truth |

Clipart tests proved:

| Fact | Result |
|---|---|
| Installed WMF symbol count | `49` |
| Unique `sha256(wmf[22:])` count | `49` |
| `embeddedWMFBySize` entries | `49` |
| `cloro.wmf` hash prefix | `7d1d692fcee32f5c` |
| `secah.wmf` hash prefix | `9230d05c03b130c9` |
| `clorox.wmf` hash prefix | `7705481c7c0e30aa` |

The new importer should build a symbol catalog from `assets/paulimaq/symbols/*.wmf`:

```text
symbolId = lowercase filename without extension or original filename
bodyHash = sha256(fileBytes[22:])
sizeBytes = len(fileBytes)
```

Then, when importing ETQ:

```text
if sha256(blob[22:]) exists in catalog:
  assign that symbolId
else if len(blob) matches exactly one catalog symbol:
  assign fallback symbolId and mark confidence=fallback-size
else:
  preserve embedded blob and mark symbolId unknown
```

### Known Size-to-Filename Map

The old parser contains a size fallback map for all 49 installed symbols:

| Size bytes | File |
|---:|---|
| 476 | `cloro.wmf` |
| 632 | `secah.wmf` |
| 700 | `clorom.wmf` |
| 728 | `clorox.wmf` |
| 744 | `secas.wmf` |
| 856 | `secag.wmf` |
| 1226 | `secav.wmf` |
| 2816 | `secof.wmf` |
| 2828 | `secow.wmf` |
| 2904 | `secox.wmf` |
| 2928 | `seco-f.wmf` |
| 2990 | `ferro--.wmf` |
| 3166 | `tamborx.wmf` |
| 3262 | `seco-w.wmf` |
| 3316 | `lavx.wmf` |
| 3630 | `tambor-.wmf` |
| 3696 | `seco--w.wmf` |
| 4040 | `seco-p.wmf` |
| 4072 | `secop.wmf` |
| 4238 | `tambor--.wmf` |
| 4338 | `ferrox.wmf` |
| 5322 | `ferro-.wmf` |
| 6060 | `lav--40.wmf` |
| 6554 | `ferro---.wmf` |
| 6602 | `lav-40.wmf` |
| 6604 | `lav40.wmf` |
| 6868 | `lav70.wmf` |
| 7260 | `lavp--40.wmf` |
| 7304 | `lavp-40.wmf` |
| 7728 | `lavp40.wmf` |
| 7832 | `lav--30.wmf` |
| 8002 | `lav-50.wmf` |
| 8044 | `lav50.wmf` |
| 8272 | `lavp--30.wmf` |
| 8394 | `lav-60.wmf` |
| 8540 | `lav95.wmf` |
| 8558 | `lav-30.wmf` |
| 8560 | `lavp-30.wmf` |
| 8564 | `lav60.wmf` |
| 8632 | `lav30.wmf` |
| 8666 | `lav-95.wmf` |
| 9116 | `lavp30.wmf` |
| 9164 | `lavp-50.wmf` |
| 9640 | `lavp50.wmf` |
| 9748 | `lavp70.wmf` |
| 10570 | `lavp-60.wmf` |
| 10740 | `lavp60.wmf` |
| 12060 | `lavp95.wmf` |
| 12778 | `lavmao.wmf` |

Use this table only as a fallback. The body hash catalog is better.

### WMF Post Fields and Chain

After the embedded WMF blob, the old parser reads a post block:

| Offset from `wmfEnd` | Size | Meaning |
|---:|---:|---|
| `+0` | 4 | terminator `FF FF FF FF` |
| `+12` | 4 | `nextX` chain coordinate |
| `+16` | 4 | `nextY` chain coordinate |
| `+20` | 1 | style byte |

Some WMF records have self-referential or duplicate dimension keys, so chain traversal is not always safe.

## Object Chain and Ordering

ETQ records contain `nextX/nextY` fields that appear to link to another object's coordinate key.

For text objects:

```text
key = (x, y) from FE + 16/+20
next = (nextX, nextY) from text post block +8/+12
```

For WMF objects:

```text
key = (width, height) from FE + 16/+20
next = (nextX, nextY) from WMF post block +12/+16
```

This is odd but matches observed files: WMF chain keys use width/height, not x/y. That creates duplicates because multiple care symbols can be the same size.

Known duplicate/ambiguous cases:

| File | Case | Consequence |
|---|---|---|
| `Canelado algodão...lunelli.ETQ` | Two WMFs with key `445,445` at offsets `0x111` and `0x46d` | Cannot rely only on chain |
| `CARACOL TULE.ETQ` | Duplicate text key `2169,426` at offsets `0x5391` and `0x652b` | Cannot rely only on chain |
| `ADAR SOFA CANELADO.ETQ` | False-positive FE inside WMF blob | Must validate payload ranges |

Recommended importer ordering:

```text
1. Extract all validated display objects with FileOffset.
2. Sort/display by FileOffset by default.
3. Optionally compute chain graph for diagnostics.
4. Use chain order only if every key is unique, no cycles exist, and every display object is reachable.
```

Do not deduplicate objects by text, symbol name, geometry, or chain key.

## Barcode Objects

Current ETQ status: not implemented and not proven by samples.

MasterPrint/CadMapa has a real barcode object path, documented in `docs\masterprint-barcode-research.md`. The visible toolbar has `btnBarcode` / `Código de Barras`, and reverse evidence maps it to runtime object type `7` with a `bc_*` drawing path.

Known runtime evidence:

| Function | Evidence |
|---|---|
| `FUN_004b5d8c` | Creates object type `7` and calls `FUN_004bea88` |
| `FUN_004ba3e0` | Generic handler switch case `7` constructs barcode handler `FUN_004befdc` |
| `FUN_004befdc` | Initializes barcode handler defaults |
| `FUN_004bea88` | Applies barcode code string, type index, orientation, and helper state |
| `FUN_004b3830` | Calls `bc_HeightPercent`, `bc_Type`, `bc_Rotate`, `bc_Size`, `bc_Code`, and `bc_Draw` |

Known limitation:

```text
There is currently no .ETQ file with a barcode object.
```

Therefore, do not implement barcode ETQ import by guessing. Without a controlled sample, the following remain unknown:

| Unknown | Why it matters |
|---|---|
| Outer ETQ object envelope | Needed to detect barcode records safely |
| Serialized barcode code string | Needed to import value text |
| Serialized barcode type/symbology | Needed to render the same barcode |
| Serialized rotation/height/size values | Needed to match placement and output |
| Human-readable text setting | Needed to match visual output |
| Check-digit behavior | Needed to avoid wrong barcode values |

Safe importer behavior: if future scanner logic suspects a barcode object, preserve it as unsupported diagnostics until at least one controlled barcode `.ETQ` with screenshot/property notes exists.

## Photo, Logo, Raster, and OLE Objects

Current status: not implemented and not proven by samples.

The existing parser supports text records and embedded WMF care-symbol records. It does not support raster photos, logos, generic `Figura` objects, or embedded OLE objects.

Corpus scan result:

```text
Current handoff ETQs and installed ARQUIVOS ETQs were byte-scanned for:
- BDOC
- OLE compound-file signature D0 CF 11 E0 A1 B1 1A E1
- BMP file marker BM
- JPEG marker FF D8 FF
- PNG signature 89 50 4E 47 0D 0A 1A 0A

No matches were found in the available corpus.
```

This means the current files do not contain a known photo/logo/OLE example. Reverse evidence identifies likely code paths, but there is no byte-level ETQ sample yet to validate object envelope, placement, and payload extraction.

### UI Evidence

`TLAYOUTDESKTOP_decoded.txt` shows separate toolbar object modes:

| Toolbar item | Hint | Meaning |
|---|---|---|
| `btnImage` | `Figura` | Figure/image mode |
| `btnMapaRisc` | `Figura` | Another figure/map-risk mode |
| `btnFileMan` | `Figura` | File-managed figure mode |
| `btnOle` | `Ole` | Generic OLE object mode |
| `btnWordArt` | `Ole` | Hidden WordArt/OLE mode |
| `btnTable` | `Ole` | Hidden table/OLE mode |

This confirms that MasterPrint/CadMapa distinguishes care-symbol WMF insertion from more general image/OLE objects.

### Reverse Evidence: Object Type Map

CadMapa stores an object type byte at runtime offset `+0x189` in the high-level graphical object. Function `FUN_004ba3e0` constructs object handlers by type.

Relevant cases:

| Type | Constructor path | Current interpretation |
|---:|---|---|
| `6` | `FUN_004bd71c` | Bitmap/DIB-style figure path |
| `9` | `FUN_004c066c` | OLE-backed figure/image wrapper |
| `10` | `FUN_004bb77c(...PTR_PTR_004bb254...)` | Another figure/file-managed path, not decoded |
| `4`, `5`, `0xb` | `FUN_004bf068` | Text/artistic text style paths, not photo/logo |

Creation/loading also reaches type `9` in `FUN_004b5d8c`:

```text
case 9:
  objectType = 9
  handler = FUN_004c066c(...)
```

The save/apply path reaches type `9` in `FUN_004c3a4c`:

```text
case 9:
  FUN_004c0748(piVar6[99], *(param_1 + 0x4ec), ...)
```

`FUN_004c0748` then calls:

```text
FUN_0049bdac(param_2, stream)   // serialize OLE/figure payload into stream
FUN_0049b7f8(target, stream)    // reload/attach payload
```

### Reverse Evidence: OLE Payload Serialization

Function `FUN_0049bdac` is the clearest save-side evidence for OLE-backed objects.

Observed behavior:

```text
if object dirty:
  OleSave(IPersistStorage, IStorage, TRUE)
  SaveCompleted(...)

if normal mode:
  hglobal = GetHGlobalFromILockBytes(object + 0x1fc)
else special mode:
  copy current IStorage into new ILockBytes
  hglobal = GetHGlobalFromILockBytes(new ILockBytes)

write 12-byte header
write hglobal bytes
```

Normal 12-byte header written by `FUN_0049bdac`:

| Offset | Size | Meaning |
|---:|---:|---|
| `+0` | 4 | signature `BDOC`, bytes `42 44 4F 43`, from little-endian `0x434F4442` |
| `+4` | 4 | OLE draw aspect, copied from runtime field `+0x208` |
| `+8` | 4 | payload byte size, `GlobalSize(hglobal)` |
| `+12` | variable | OLE compound-storage bytes from `HGLOBAL` |

The corresponding load-side function `FUN_0049b7f8` reads 12 bytes and rejects the record if the first dword is not `BDOC`, unless a special graphical mode flag at runtime offset `+0x23e` is set.

Important uncertainty: this describes the inner OLE payload stream, not the complete surrounding ETQ object envelope. A real sample is needed to locate the outer `FE` record, geometry fields, object type byte, and payload length in actual `.ETQ` bytes.

### Reverse Evidence: OLE Object Creation and Rendering

Function `FUN_0049ac24` creates OLE objects using standard Windows OLE APIs:

| Case | API |
|---:|---|
| `0` | `OleCreate` |
| `1` | `OleCreateFromFile` |
| `2` | `OleCreateLinkToFile` |
| `3` | `OleCreateFromData` |
| `4` | `OleCreateLinkFromData` |

This means imported logos/photos may not be stored as simple PNG/JPEG/BMP bytes. They may be embedded in an OLE compound storage wrapper created by Windows, depending on how MasterPrint inserted the file.

Function `FUN_0049ba34` renders the OLE object with:

```text
OleDraw(object + 0x204, aspect at object + 0x208, hdc, rect)
```

It also uses runtime field `+0x23f` to decide how to fit/center/stretch/icon-render the object inside the rectangle. Exact semantics need a real sample and UI testing.

### Reverse Evidence: Bitmap/DIB Rendering Path

There is also a non-OLE bitmap/DIB path:

| Function | Evidence |
|---|---|
| `FUN_004bd71c` | constructs type `6` object and allocates bitmap-related state |
| `FUN_004bde2c` | render path that branches on internal image mode and can call `FUN_004bd3c0` |
| `FUN_004bd3c0` | renders bitmap data via `GetDIBits` and `StretchDIBits` |
| `FUN_004261c4` | loads BMP stream with file marker `BM` (`0x4D42`) |
| `FUN_0048ba50` | writes BMP file header `BM` and bitmap bits |

This path may be used for some `Figura` objects, but no available `.ETQ` sample proves how it appears in the ETQ container.

### What a Photo/Logo ETQ Sample Should Contain

To solve this safely, create controlled samples in original MasterPrint/CadMapa.

Minimum sample set:

```text
photo-bmp-embedded-lnt2.ETQ
photo-jpg-embedded-lnt2.ETQ
logo-png-or-bmp-embedded-lnt2.ETQ
ole-from-file-lnt2.ETQ
ole-linked-file-lnt2.ETQ
```

If the original program only accepts `BMP`/`WMF`, then create at least:

```text
figure-bmp-lnt2.ETQ
ole-bmp-lnt2.ETQ
figure-wmf-lnt2.ETQ
```

For each sample, include:

```text
original inserted image file
ETQ file
MasterPrint screenshot/JPG reference
notes saying which toolbar/action was used: Figura, Ole, FileMan, MapaRisc, etc.
whether the file was embedded or linked
approximate position and size
```

Recommended simple image content:

```text
100x60 px BMP with a red square, blue circle, and black text LOGO
another JPG/photo if accepted by MasterPrint
```

Keep the image visually distinctive. A distinctive image makes it easier to identify raw bitmap/JPEG/compound-storage bytes inside the ETQ.

### Import Strategy Until Samples Exist

Do not implement photo/logo import as if it is known.

Safe MVP behavior:

| Situation | Behavior |
|---|---|
| Parser sees known text/WMF only | Import normally |
| Parser detects `BDOC`/OLE marker later | Preserve as unsupported `ole-object` with offset and raw bytes until geometry is proven |
| Parser detects BMP/DIB marker later | Preserve as unsupported `bitmap-object` with offset and raw bytes until object envelope is proven |
| User imports ETQ with missing photo/logo | Warn clearly that raster/OLE objects are not imported |

Do not attempt to extract images by scanning for random `BM`, JPEG, or PNG signatures and placing them on the canvas. Without the outer object geometry and mode fields, that would create false confidence and wrong output.

## Corpus Baseline

The old parser has a baseline for 58 installed `.ETQ` files under:

```text
C:\Program Files (x86)\paulimaq\ARQUIVOS
```

The baseline asserts text counts, WMF counts, unknown count, and template name. In the archived parser, all listed files have `unknown=0` because the parser only counts validated text-like unknowns after excluding known text and WMF records.

Representative entries:

| File | Text | WMF | Template |
|---|---:|---:|---|
| `Canelado algodão (Classic Wave Ramado) lunelli.ETQ` | 7 | 6 | `LNT-2` |
| `ADAR SOFA CANELADO.ETQ` | 5 | 6 | `LNT-2` |
| `FAVERO.ETQ` | 7 | 7 | `LNT-2` |
| `LASER 4 - VICENZA E LINO.ETQ` | 10 | 7 | `LNT-4` |
| `devorê e forro podrinha.ETQ` | 10 | 7 | `LNT-4` |

Template extraction is blank for some files in the corpus, even though text/WMF extraction works. Do not fail import just because template name is missing.

## Known Good Parser Algorithm

Use two independent scanners: one for text, one for WMF. Do not try to parse a single object stream unless the full container is later reversed.

High-level algorithm:

```text
parseETQ(data, symbolCatalog):
  metadata = scanHeaderStrings(data)
  textRecords = []
  wmfRecords = []

  for offset in 0..len(data)-minimum:
    if data[offset:offset+4] != FE FF FF FF:
      continue

    if looksLikeValidText(data, offset):
      textRecords.append(parseText(data, offset))
      continue

    if looksLikeValidWMF(data, offset):
      wmfRecords.append(parseWMF(data, offset, symbolCatalog))
      continue

  sort textRecords and wmfRecords by FileOffset or merge all objects by FileOffset
  unknowns = scanPotentialUnknownTextLikeFE(data, knownOffsets)
  return importedDocument
```

Text validator checklist:

```text
offset + 40 <= len(data)
flags == 0
tag == 1 or tag == 0
1 <= payloadLength <= 4096
offset + 40 + payloadLength + 4 <= len(data)
payload terminator == FF FF FF FF
payload decodes to plausible plain text or RTF plain text
reject obvious font names / metadata-only strings
```

WMF validator checklist:

```text
offset + 64 <= len(data)
flags == 0x80000008
tag == 0
aldusOffset = offset + 49
data[aldusOffset:aldusOffset+4] == D7 CD C6 9A
wmfEnd computed from standard WMF words is <= len(data)
preOffset = offset - 83
preOffset >= 0
data[preOffset:preOffset+4] == FF FF FF FF
x, y, width, height are non-zero
```

## Recommended Native Import Model

For the new program, import `.ETQ` objects into native objects like this:

```json
{
  "source": {
    "format": "ETQ",
    "path": "...",
    "templateName": "LNT-2",
    "layoutType": "Etiq. para Composições em Folhas"
  },
  "layoutId": "LNT-2",
  "template": {
    "widthMM": 55.5,
    "heightMM": 25.0,
    "objects": []
  },
  "importDiagnostics": []
}
```

Text object mapping:

| Native field | ETQ source |
|---|---|
| `type` | `text` |
| `xMM` | `rawX / 100.0` |
| `yMM` | `rawY / 100.0` |
| `wMM` | `rectWidth / 100.0` |
| `hMM` | `rectHeight / 100.0` |
| `text` | decoded plain text or RTF plain text |
| `fontFamily` | pre-FE font name heuristic, fallback `Arial` |
| `fontSizePt` | `rectHeightMM * 72 / 25.4` |
| `bold` | style bit `0x01` or RTF bold |
| `italic` | style bit `0x02` |
| `underline` | style bit `0x04` |
| `align` | `0=left`, `1=center`, `2=right` |
| `source.fileOffset` | FE offset |
| `source.payloadRawBase64` | original payload |
| `source.rtfRawBase64` | raw RTF when present |

WMF object mapping:

| Native field | ETQ source |
|---|---|
| `type` | `symbol` |
| `xMM` | `preBlockX / 100.0` |
| `yMM` | `preBlockY / 100.0` |
| `wMM` | `headWidth / 100.0` |
| `hMM` | `headHeight / 100.0` |
| `symbolId` | body-hash match to bundled WMF catalog |
| `source.fileOffset` | FE offset |
| `source.wmfBodySha256` | `sha256(blob[22:])` |
| `source.embeddedWmfBase64` | original WMF blob if desired |
| `source.preBlockBase64` | original 83-byte pre-block if desired |

The new program should convert imported LNT ETQ coordinates into its explicit design canvas. Be careful with orientation: old ETQ coordinates are design coordinates, and LNT physical slots are rotated on the page. Do not interpret ETQ import coordinates as physical A4 sheet coordinates.

## Writing and Patching Status

Full `.ETQ` writing is not solved.

The archived code has only these safe-ish operations:

| Operation | Status |
|---|---|
| No-op save/copy | Byte-identical copy of original file |
| Sidecar save | Saves parsed/imported state without modifying ETQ |
| MPN native save | Saves app-native model without modifying ETQ |
| Patch text position | Experimentally updates raw X/Y and relinks predecessor when unambiguous |
| Patch same-length plain Latin-1 text | Experimentally updates payload only when byte length is unchanged |
| Patch WMF rect | Experimentally updates width/height and pre-block x/y when unambiguous |

The patcher refuses:

| Refusal | Reason |
|---|---|
| Unknown objects | Cannot guarantee no corruption |
| Added/deleted/type-changed objects | Structural serializer not proven |
| Variable-length text | Would require moving later bytes and rebuilding chains/containers |
| RTF text patch | Rich text payload semantics not fully proven |
| Font/style/alignment changes | Serializer fields are not fully proven |
| Image symbol swap | WMF serializer and object replacement not proven |
| Ambiguous chain relink | Duplicate keys make predecessor update unsafe |

The decompile research identified functions that save inner text and WMF payloads, but not the full envelope writer:

| Function | Known role |
|---|---|
| `FUN_004b74e8` | saves `WDDESIGNVCM` container |
| `FUN_004b7dc8` | loads container |
| `FUN_004bf340` | saves inner text payload |
| `FUN_004bf12c` | loads inner text payload |
| `FUN_00423f00` / `FUN_00424010` | saves WMF/Aldus payload |

Blocking unknown:

```text
No audited function was found that fully explains writing the FE FF FF FF envelope,
tags, post-fields, nextX/nextY, style byte, or WMF pre-block.
```

Therefore, ETQ writing should remain out of scope for the new MVP. If it ever returns, require:

```text
1. no-op writer byte-identical for corpus subset
2. original MasterPrint opens output files
3. one object type at a time
4. structural changes only after variable-length serializer is proven
```

## Main Failure Modes to Avoid

| Mistake | Why it breaks |
|---|---|
| Treat every `FE FF FF FF` as an object | False positives occur inside WMF blobs |
| Deduplicate by text or symbol name | Real files can contain duplicate visible objects |
| Assume all text has tag `1` | Some real text-like records use tag `0` |
| Import `tln=0` records as visible text | Hidden chain nodes exist |
| Import truncated EOF fragments | Some files contain readable but incomplete trailing bytes |
| Use WMF internal bounds as placement rect | Placement rect is external ETQ metadata |
| Assume WMF filename is stored | ETQ embeds bytes; filename must be resolved by hash |
| Use blob size as primary identity | It worked for installed symbols but hash is stronger |
| Assume square care symbols | `FAVERO.ETQ` proves non-square WMF placement |
| Use chain order blindly | Duplicate keys and false positives make it unsafe |
| Treat ETQ coordinates as sheet coordinates | They are label/design coordinates for the template |

## C# Implementation Notes

The new WPF/.NET program can implement the importer without porting the old Go app architecture.

Recommended C# structures:

```csharp
public sealed record EtqImportResult(
    EtqMetadata Metadata,
    IReadOnlyList<EtqImportedObject> Objects,
    IReadOnlyList<EtqDiagnostic> Diagnostics);

public sealed record EtqTextObject(
    int FileOffset,
    double XMM,
    double YMM,
    double WidthMM,
    double HeightMM,
    string Text,
    string FontFamily,
    double FontSizePt,
    bool Bold,
    bool Italic,
    bool Underline,
    string Align,
    byte[] PayloadRaw,
    byte[]? RtfRaw) : EtqImportedObject;

public sealed record EtqSymbolObject(
    int FileOffset,
    double XMM,
    double YMM,
    double WidthMM,
    double HeightMM,
    string? SymbolId,
    string WmfBodySha256,
    byte[] EmbeddedWmf,
    byte[] PreBlock) : EtqImportedObject;
```

Use little-endian helpers:

```csharp
static uint U32(ReadOnlySpan<byte> data, int offset) => BinaryPrimitives.ReadUInt32LittleEndian(data.Slice(offset, 4));
static ushort U16(ReadOnlySpan<byte> data, int offset) => BinaryPrimitives.ReadUInt16LittleEndian(data.Slice(offset, 2));
static double RawToMM(uint raw) => raw / 100.0;
```

Use Windows-1252 or Latin-1 carefully. The old parser maps each byte directly to the same Unicode code point, which works for observed Portuguese text bytes like `Ã`, `Ç`, and `Ú` when stored as Latin-1. In .NET, `Encoding.Latin1` is available in modern .NET. If future files contain Windows-1252 smart punctuation, consider trying `Encoding.GetEncoding(1252)` behind a controlled fallback.

Recommended tests to port first:

| Test | Purpose |
|---|---|
| Parse Lunelli text at `0x15d3` | Locks text fields, style, payload, next coords |
| Parse Lunelli `clorox.wmf` at `0x0111` | Locks WMF pre-block and embedded blob |
| Parse FAVERO non-square WMF at `0x114f` | Prevents square-symbol assumption |
| Parse ADAR RTF text | Prevents RTF metadata leaking into display text |
| Reject ADAR false-positive at `0x713b` | Prevents fake text from WMF blob |
| Promote tag `0` text-like records | Prevents missing brand/size text |
| Skip SLIM hidden `tln=0` node | Prevents empty visible objects |
| Skip ALGODÃO truncated EOF record | Prevents importing partial garbage |
| No dedupe by content | Preserves duplicate visible objects |
| All 49 WMFs hash uniquely | Locks symbol catalog identity |

## Recommended Import UX

When importing `.ETQ`, show a factual import summary:

```text
Imported ETQ: Canelado algodão (...).ETQ
Detected layout: LNT-2
Text objects: 7
Symbol objects: 6
Unknown/unsupported objects: 0
RTF text objects imported as plain text: N
Symbol identity: 6 hash matches, 0 size fallbacks, 0 unknown embedded WMFs
Saved as native .mpLabel; original ETQ was not modified.
```

If unsupported or suspicious records exist, keep importing supported objects but warn:

```text
Some ETQ records were not imported. The native file is usable, but it may not be a complete visual copy.
Original ETQ bytes were not modified.
```

## Practical Recommendation

For the new label program:

```text
Implement ETQ import as a read-only converter for common textile LNT templates.
Use native JSON after import.
Never write ETQ in the MVP.
Build object import confidence around FileOffset, validated envelopes, raw payload preservation, and symbol body hashes.
```

This gives the new program a realistic chance to read useful old labels while avoiding the dangerous part: pretending `.ETQ` is fully understood.
