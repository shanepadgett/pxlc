# PXLC Source and Output Specification, Version 1

This document defines source format 1 and metadata schema 1. Behavior described
here is part of the compiler contract.

## Source Files

A lowercase `.pxl` file contains one still asset. Source uses printable ASCII
with LF or CRLF line endings. Spaces, tabs, and newlines separate tokens. `//`
starts a comment outside a quoted string and continues through the end of the
line.

Names match `[A-Za-z_][A-Za-z0-9_-]*`. Integers are written in decimal. Braces
delimit palette, layer, and grid bodies. Quoted strings do not support escapes.

Every file starts with its source-format version:

```pxl
pxlc 1
```

Declarations may refer to palettes and colors declared later in the file. One
asset declaration, one canvas declaration, one background declaration, at
least one nonempty palette, and at least one layer are required.

## Complete Example

```pxl
pxlc 1

asset badge
canvas 8 8

palette ui max 4 {
  transparent clear "."
  color outline "K" #172038
  color fill "F" #3f6f8f
  color shine "S" #f4f4dc
}

background ui clear

layer body using ui {
  grid 1 1 {
    "..KK.."
    ".KFFK."
    "KFFFFK"
    "KFSFFK"
    ".KFFK."
    "..KK.."
  }
}
```

## Canvas and Coordinates

```pxl
asset badge
canvas 8 8
```

The asset name is its engine-independent identifier. Canvas width and height
are positive integer pixel counts.

The origin `(0, 0)` is the top-left pixel. X increases to the right and Y
increases downward. Coordinates identify pixels, not pixel boundaries.

Phase 1 rejects every operation whose complete extent is outside the canvas.
It does not silently clip valid source. The rasterizer checks the same bounds
before writing, so invalid input cannot become an out-of-range memory access.

## Palettes

```pxl
palette ui max 4 {
  transparent clear "."
  color outline "K" #172038
  color fill "F" #3f6f8f
}
```

A palette has a unique name and one or more entries. Each entry has a unique
color name and a unique one-byte printable ASCII grid symbol.

Opaque colors use `#RRGGBB`. Transparent entries have RGBA value
`#00000000`. Partial alpha is invalid in source format 1. Different entries may
have the same RGBA value.

`max N` is optional. When present, the palette may contain at most `N` entries,
including transparent entries. The standalone compiler also enforces its
documented safety ceiling.

The background selects a named color from a named palette:

```pxl
background ui clear
```

## Layers and Compositing

```pxl
layer details using ui {
  pixel 2 2 shine
}
```

Layer names are unique. Layers are drawn and composited in declaration order;
later layers appear above earlier layers. Each layer starts completely
transparent and uses one declared palette.

Operations execute in source order. Drawing a transparent color clears an
earlier pixel in the same layer. A transparent layer pixel leaves lower layers
unchanged during compositing. An opaque layer pixel replaces the lower pixel.
These binary-alpha rules avoid renderer-dependent blending.

## Drawing Operations

All sizes and lengths must be greater than zero.

```pxl
pixel X Y COLOR
hspan X Y LENGTH COLOR
vspan X Y LENGTH COLOR
rect X Y WIDTH HEIGHT COLOR
```

- `pixel` writes one pixel.
- `hspan` writes `LENGTH` pixels from left to right, including `(X, Y)`.
- `vspan` writes `LENGTH` pixels from top to bottom, including `(X, Y)`.
- `rect` fills `WIDTH` by `HEIGHT` pixels with top-left pixel `(X, Y)`.

A grid infers its width and height from its rows:

```pxl
grid X Y {
  ".KK."
  "KFFK"
}
```

The grid must contain at least one nonempty row, every row must have the same
byte width, and every symbol must belong to the layer palette. Each grid cell
is a write, including transparent cells.

## Validation Limits

Standalone source format 1 uses these fixed ceilings:

- 8 MiB per source file.
- 1,000,000 tokens per source file.
- 4,096 pixels per canvas dimension.
- 16,777,216 pixels per canvas.
- 256 palettes, 256 entries per palette, and 256 layers per asset.
- 1,000,000 declared drawing operations per asset.
- 67,108,864 total pixel writes per asset.
- 67,108,864 pixels in one generated preview.
- 10,000 source files in one CLI input tree.
- 100 semantic diagnostics per source file before a limit diagnostic.

Counts and dimensions are checked before allocation.

## Runtime Artifacts

For input root `art` containing source path `actors/player.pxl`, a tree build
writes:

```text
<output>/actors/player.png
<output>/actors/player.pxlc.json
```

A single-file build writes the two files directly under the output directory.
PNG encoding uses fixed compression settings and contains no timestamps, source
paths, or machine metadata.

Metadata schema 1 has this shape:

```json
{
  "schema": "pxlc.asset",
  "version": 1,
  "asset": "player",
  "canvas": { "width": 16, "height": 16 },
  "image": "player.png",
  "palettes": [
    {
      "name": "main",
      "colors": [
        { "name": "clear", "symbol": ".", "rgba": "#00000000" }
      ]
    }
  ],
  "source": {
    "path": "actors/player.pxl",
    "formatVersion": 1,
    "sha256": "<lowercase hex SHA-256>"
  },
  "compiler": { "version": "<PXLC version>" },
  "content": { "imageSHA256": "<lowercase hex SHA-256>" }
}
```

Arrays preserve source declaration order. Object field order is stable in the
generated bytes. Paths use forward slashes. Metadata source paths are relative
to the CLI input root; a single-file build uses the source basename. Ambient
working-directory paths never enter generated output.

Build output is staged while the sorted input tree is compiled. Source failures
discard all staged files and leave existing artifacts untouched. Staged files
replace their corresponding outputs in stable path order. Each replacement is
atomic where the host filesystem provides atomic same-filesystem rename
replacement; a multi-file build is not a filesystem transaction.

## Previews

Preview scaling uses integer nearest-neighbor replication. Scale 1 is native
resolution. The background may remain transparent or use one opaque
`#RRGGBB` color. Preview settings do not change runtime output.

## Diagnostics and Exit Status

Source diagnostics have a stable one-line representation:

```text
art/player.pxl:18:9 PXLC-E023 drawing extent (15, 4) 2x1 is outside the 16x16 canvas
```

Exit statuses are:

- `0`: success.
- `1`: invalid PXLC source.
- `2`: invalid command usage.
- `3`: filesystem, rendering, or encoding failure.
