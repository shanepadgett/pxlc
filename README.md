# PXLC

PXLC compiles compact text-based pixel art into deterministic PNG images and
versioned, engine-independent metadata. Text remains the canonical asset
source, which gives coding agents a small and reviewable edit surface.

The current release implements still assets with named palettes, ordered
layers, literal grids, pixels, spans, rectangles, validation, runtime builds,
and integer-scaled previews.

## Build

PXLC requires Go 1.26.5.

```bash
go build -o pxlc ./cmd/pxlc
./pxlc --version
```

## Commands

```bash
mise run pxlc -- check art/player.pxl
mise run pxlc -- check art/

mise run pxlc -- build art/ --output build/art

mise run pxlc -- preview art/player.pxl \
  --scale 8 \
  --background transparent \
  --output build/previews/player.png
```

The task runs the current checkout without a separate install. A compiled
`pxlc` binary accepts the same arguments. Flags may appear before or after the
input path. `--quiet` suppresses success output, while `--verbose` reports each
checked input or written artifact.

See [the source and output specification](docs/language.md) for exact syntax,
raster behavior, limits, artifact naming, metadata, and exit statuses. The
[demonstration icon](examples/icon.pxl) is a complete working asset.

## Experimental Concept Crushing

`pxlc-crush` converts an isolated PNG or JPEG concept into ordinary PXLC grid
source. Generate the concept on a flat `#00ff00` background so dark outlines
remain distinct from transparency.

```bash
mise run pxlc:crush -- \
  --asset censer-child \
  --width 64 \
  --height 96 \
  concept.png \
  art/censer-child.pxl

mise run pxlc -- preview art/censer-child.pxl \
  --scale 6 \
  --background '#201d20' \
  --output build/censer-child.png
```

The command crops the occupied bounds, downsamples them, applies a fixed muted
horror palette, and removes small raster noise. The generated `.pxl` file is
the canonical source after import. Review and edit it like any other asset.
This command is experimental; its palette and cleanup rules may change.

For compact directional characters, use the `sprite` profile. It fits the
figure to a smaller footprint, reduces the palette, flattens dark values, and
derives a connected silhouette outline:

```bash
mise run pxlc:crush -- \
  --profile sprite \
  --asset bell-warden-front \
  --width 32 \
  --height 32 \
  front.png \
  art/bell-warden-front.pxl
```

For generated sheets that already depict enlarged pixel blocks, reduce each
cell directly without palette conversion or cleanup:

```bash
mise run pxlc:crush-sheet -- \
  --columns 4 \
  --rows 3 \
  --cell-size 32 \
  --preview build/items-preview.png \
  generated-items.png \
  build/items.png
```

This path divides the source into equal cells and area-averages each one to its
logical size. It retains the source background and colors. The optional preview
uses nearest-neighbor scaling only.

Rectangular assets can set `--cell-width` and `--cell-height` instead of the
square `--cell-size`, such as `--cell-width 64 --cell-height 96` for trees.

## Scope

PXLC is a non-interactive compiler. Animation, reusable stamps, imports, game
metadata, terrain validation, formatting, and editor integration remain on the
[roadmap](docs/plans/roadmap.md).
