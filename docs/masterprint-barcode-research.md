# MasterPrint / CadMapa Barcode Research

This report deepens the barcode section of the MasterPrint/CadMapa reverse notes. It is based on UI resource evidence and local decompiled CadMapa functions only.

Important boundary: there is currently no `.ETQ` file with a barcode object. That means the runtime/render path is identifiable, but barcode ETQ serialization bytes are not proven.

## Status Summary

| Area | Confidence | Evidence |
|---|---|---|
| Barcode was a real visible tool | High | `btnBarcode`, hint `Código de Barras` in `TLAYOUTDESKTOP_decoded.txt` |
| Barcode maps to CadMapa object type `7` | High | `FUN_004b5d8c` and `FUN_004ba3e0` case `7` |
| Barcode uses a separate `bc_*` barcode runtime | High | `bc_CreateBarCode`, `bc_HeightPercent`, `bc_Type`, `bc_Rotate`, `bc_Size`, `bc_Code`, `bc_Draw` |
| Barcode render pipeline | Medium-high | `FUN_004b3830` configures `bc_*` calls then draws to HDC |
| Barcode editable properties | Medium | `FUN_004c3a4c` applies code/type/rotation/height/size-like properties |
| Barcode symbology names | Low | Only `EAN14` string is visible; type-name table is not recovered |
| Barcode ETQ bytes/import | Unknown | No barcode `.ETQ` sample exists |
| Barcode ETQ writer | Unsafe | No serializer proof and no sample round-trip |

## UI Evidence

The main object toolbar contains:

| Toolbar item | Evidence | Meaning |
|---|---|---|
| `btnBarcode` | hint `Código de Barras` | User-facing barcode insertion mode |
| handler | `OnClick = WDSpeedButton2Click` | Same generic tool-selection handler as line/text/shape/image modes |

The decoded resource does not expose a barcode-specific properties dialog caption in the extracted `TLayoutDesktop` form. Barcode settings appear in decompiled property/apply functions instead.

## Runtime Object Type

Barcode is CadMapa runtime object type `7`.

Creation/load path evidence from `FUN_004b5d8c`:

```text
case 7:
  object + 0x189 = 7
  handler = FUN_004befdc(...PTR_PTR_004bb638...)
  object[99] = handler
  FUN_004bea88(handler, 0, param_5)
```

Generic handler switch evidence from `FUN_004ba3e0`:

```text
case 7:
  handler = FUN_004befdc(...PTR_PTR_004bb638...)
  object + 0x18c = handler
```

Capability-check evidence from `FUN_004b5870`:

```text
if selected object type == 7:
  only command category param_2 == 3 returns enabled
```

The meaning of command category `3` is not fully decoded, but this proves CadMapa treated barcode as a distinct object family with limited applicable object commands.

## Barcode Handler Constructor

`FUN_004befdc` constructs the type-7 barcode handler.

Observed initialization:

| Field | Initial value | Interpretation |
|---:|---:|---|
| `+0x28` | `100` | Likely barcode height percent, because it is sent to `bc_HeightPercent` through the helper state |
| `+0x2a` | `0` | Orientation/rotation option, because it is sent to `bc_Rotate` and set from four radio-style controls |
| `+0x38` | double `2.0` | Size/module-scale-like value, because later property code writes a numeric control here before `bc_Size` |

Constructor excerpt:

```text
FUN_004bb77c(base)
handler + 0x38 = double 2.0
handler + 0x28 = 100
handler + 0x2a = 0
```

The handler also has a helper/runtime pointer at `+0x2c`, created by `FUN_004bea88`.

## Barcode Runtime Helper

`FUN_004b3664` creates the low-level barcode runtime helper.

Observed behavior:

```text
FUN_00402f60(base)
helper[2] = FUN_00402f60(...PTR_DAT_004b356c...)
helper[1] = bc_CreateBarCode()
```

Meaning:

| Helper field | Meaning |
|---|---|
| `helper[1]` | Low-level barcode handle returned by `bc_CreateBarCode` |
| `helper[2]` | Additional barcode state object used by render/setup code |

The `bc_*` functions decompile as indirect jump stubs/wrappers, not as normal C logic. The installed Paulimaq folder does not contain a separate visible barcode DLL/OCX, so this may be statically linked library code or an import/thunk pattern the decompiler could not recover.

## Barcode Property Setup

`FUN_004bea88(handler, typeIndex, codeString)` updates the barcode handler and helper state.

Observed fields:

| Handler field | Source | Sent to helper/runtime | Interpretation |
|---:|---|---|---|
| `+0x2c` | new `FUN_004b3664` helper | helper pointer | Low-level barcode runtime state |
| `+0x28` | existing short, default `100` | `helperState + 4` | Height percent-like value |
| `+0x2a` | existing short, default `0` | `helperState + 0x0c` | Rotation/orientation option |
| `+0x30` | `typeIndex` argument | maps through `DAT_0052c908[typeIndex]` | Barcode type/symbology index |
| `+0x34` | code string, or default `DAT_004beb28` if null | copied to `helperState + 0x14` | Barcode data/code text |

Special logic:

```text
if helper was newly created:
  helperState + 8 = 0x5a
else:
  helperState + 8 = DAT_0052c908[typeIndex]
```

There is a visible string comparison against `EAN14`:

```text
FUN_004041bc(codeString, "EAN14")
```

The decompiled output does not show the comparison result being used. Treat this as evidence that `EAN14` is a barcode-related literal, not proof that every default barcode is EAN-14.

## Property Dialog Apply Path

`FUN_004c3a4c` applies object properties back to the selected object. In the `case 7` branch it updates barcode fields.

Observed property-to-handler mapping:

| UI/property storage | Handler field | Interpretation |
|---:|---:|---|
| `param + 0x3a0` | `+0x34` via `FUN_004397d0` then `FUN_004bea88` | Barcode code/data text |
| `param + 0x3a4` | `+0x38` as double | Size/module-scale-like numeric value |
| `param + 0x3a8` | `+0x28` as short | Height percent-like numeric value |
| `param + 0x3ac` | `+0x2a = 0` if checked | Rotation/orientation option 0 |
| `param + 0x3b0` | `+0x2a = 1` if checked | Rotation/orientation option 1 |
| `param + 0x3b4` | `+0x2a = 2` if checked | Rotation/orientation option 2 |
| `param + 0x3b8` | `+0x2a = 3` if checked | Rotation/orientation option 3 |
| `param + 0x3bc` | `typeIndex` argument to `FUN_004bea88` | Barcode type/symbology selector |

The captions for those controls were not recovered from `TLayoutDesktop_decoded.txt`, so do not name them as official labels yet.

## Render Pipeline

`FUN_004b3830` is the clearest barcode render path.

Practical sequence:

```text
helperState = *(handler + 0x2c)->state

bc_HeightPercent(...)
bc_Type(...)
bc_Rotate(...)
bc_Size(...)

if helperState.typeCode == 0x5a:
  transformedCode = FUN_004b37d8(originalCode)
  bc_Code(transformedCode)
else:
  bc_Code(originalCode)

hdc = FUN_004207f4(renderContext)
bc_Draw(hdc, ...)
```

The decompiler loses the real arguments to `bc_*`, but the caller prepares fields immediately before each call. That is enough to identify the pipeline order.

### Special Type `0x5a`

`FUN_004b3830` treats helper type code `0x5a` specially.

For this type, it copies the code string from `helperState + 0x14`, transforms it with `FUN_004b37d8`, then passes the transformed result to `bc_Code`.

`FUN_004b37d8` calls `FUN_004b36e8`, which parses characters from the input string from right to left and converts digit characters to numbers. The exact final algorithm is not cleanly recovered, but it looks like a numeric rearrangement/check-digit helper for a special barcode symbology.

Do not label `0x5a` as EAN-14, Interleaved 2-of-5, or any other symbology until a controlled sample confirms it.

## What Is Known Versus Unknown

Known:

| Fact | Evidence |
|---|---|
| Barcode insertion exists | `btnBarcode`, `Código de Barras` |
| Runtime object type is `7` | `FUN_004b5d8c`, `FUN_004ba3e0` |
| Barcode has code/data text | `FUN_004bea88` copies string into helper state and `FUN_004b3830` passes it to `bc_Code` |
| Barcode has type/symbology selector | `FUN_004c3a4c` reads selector at `+0x3bc`, `FUN_004bea88` maps type index through `DAT_0052c908` |
| Barcode has rotation/orientation option | four controls map to `+0x2a = 0..3`, then `bc_Rotate` is called |
| Barcode has height percent-like value | default `100`, copied to helper state before `bc_HeightPercent` |
| Barcode has size/module-scale-like value | default double `2.0`, property apply writes `+0x38`, render calls `bc_Size` |
| Rendering goes to an HDC | `FUN_004207f4(renderContext)` before `bc_Draw` |

Unknown:

| Unknown | Why |
|---|---|
| Exact ETQ record bytes for barcode | No barcode `.ETQ` exists |
| Barcode save/load serializer | Not isolated in current evidence and no sample to validate |
| Symbology names behind `DAT_0052c908` | Data table values/names not recovered |
| Meaning of type code `0x5a` | Special transform exists, but label is unproven |
| Exact meaning of `+0x38` | Likely size/module scale, but no UI caption/sample |
| Exact meaning of `+0x28` | Likely height percent, but no UI caption/sample |
| Human-readable text display under barcode | Not proven by current decompile snippets |
| Check-digit behavior | Special transform exists but algorithm is not confidently decoded |

## New Program Guidance

Barcode support should not block the LNT-2/text/symbol MVP. It should be a later feature with its own sample gate.

Recommended staged approach:

| Stage | Action |
|---|---|
| 1 | Keep barcode as a documented legacy feature, not MVP scope |
| 2 | In the native model, reserve a future `BarcodeObject` with `code`, `symbology`, `rotation`, `heightPercent`, and `moduleScale` fields |
| 3 | Do not import barcode ETQ objects until a controlled barcode `.ETQ` exists |
| 4 | When implementing native barcode creation, use a maintained .NET barcode library or custom renderer, not CadMapa `bc_*` |
| 5 | Match old MasterPrint behavior only after comparing controlled samples and prints |

## Controlled Sample Needed

Create barcode samples in original MasterPrint/CadMapa before implementing import.

Minimum sample set:

```text
barcode-default-lnt2.ETQ
barcode-default-lnt2.jpg or screenshot
notes: exact value typed, selected barcode type, rotation, size/height settings
```

Better sample matrix:

```text
barcode-type0-123456789012-lnt2.ETQ
barcode-type1-123456789012-lnt2.ETQ
barcode-type2-123456789012-lnt2.ETQ
barcode-type3-123456789012-lnt2.ETQ
barcode-ean14-if-visible-lnt2.ETQ
barcode-rot0-lnt2.ETQ
barcode-rot1-lnt2.ETQ
barcode-rot2-lnt2.ETQ
barcode-rot3-lnt2.ETQ
```

For each file, capture:

```text
ETQ file
MasterPrint screenshot/export
typed barcode value
selected barcode type label
all visible barcode property values
paper layout/model
whether human-readable text is shown below the bars
```

## Report Wording For Users

Use this support wording:

```text
MasterPrint had a real barcode object, and we have identified the old runtime/render path. However, there is no barcode ETQ sample yet, so old barcode-file import is not proven. Native barcode creation can be added later, but legacy barcode import needs controlled samples from the original app.
```
