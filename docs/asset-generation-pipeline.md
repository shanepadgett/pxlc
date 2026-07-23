# Generated Pixel-Asset Pipeline

This document records the asset-generation process that produced the strongest
results during PXLC development. It is a handoff for an agent building the
same workflow in another game or packaging it as a Pi extension.

The process covers isolated sprites and sprite sheets. It does not cover terrain
composition, procedural level generation, entity placement, or Y-sorting.

## Summary

The working pipeline is:

1. Define the sheet layout, logical cell size, camera, palette direction, and
   cell contents before generation.
2. Generate an oversized raster sheet on a flat saturated magenta background.
3. Reject sheets with bad perspective, scale, separation, or pixel clustering.
4. Divide the accepted source into equal cells.
5. Reduce every cell independently with direct RGB box averaging.
6. Preserve the reduced colors and magenta background without quantization or
   cleanup.
7. Use a broad magenta chroma key when importing individual cells.
8. Inspect the native result and a nearest-neighbor enlargement.
9. Record the prompt, source, settings, and hashes before accepting the asset.

The useful shorthand is **chroma-keyed generate and crush**. Image generation
supplies design and rendering. Deterministic code imposes dimensions, pixel
density, repeatable output, and provenance.

## Determinism Boundary

Image generation is not deterministic in the strong build-system sense. A
provider may accept a seed, but model revisions, backend changes, and hidden
sampling behavior can still change the image.

Treat every generated source image as immutable input. Once that file exists,
the following stages can be deterministic:

- Sheet validation
- Per-cell reduction
- Preview enlargement
- Chroma-key classification
- Manifest creation, excluding volatile fields
- Hashing
- Export

Reproducing an accepted asset means rebuilding from the saved source image and
saved settings. It does not mean asking the image provider to generate the same
image again.

An extension should never overwrite an accepted source with a fresh generation.
Write each generation as a new candidate and identify it by a content hash or
monotonic candidate number.

## Asset Contract

Define the contract before writing the image prompt. At minimum it contains:

- Asset or sheet ID
- Number of columns and rows
- Logical cell width and height
- Ordered cell descriptions
- Camera angle and facing convention
- Approximate occupancy within each cell
- Chroma-key color family
- Whether the object may contain that color family
- Palette and lighting direction
- Shadow policy

The strongest experiment used these contracts:

| Category | Layout | Logical cell |
| --- | ---: | ---: |
| NPCs | 3 columns by 2 rows | 32 by 32 |
| General props | 4 columns by 3 rows | 32 by 32 |
| Ground details | Same accepted prop sheet | 16 by 16 |
| Trees | 4 columns by 2 rows | 64 by 64 |
| Wagon | 1 column by 1 row | 96 by 64 |

Different categories need different logical sizes. Forcing an elongated wagon
or a broad tree into a 32 by 32 square destroys useful shape information.

### Camera Contract

Use concrete camera language. The successful outdoor assets used a steep
three-quarter overhead view, approximately 65 to 70 degrees downward.

Repeat the camera clause in every category prompt. Do not assume a reference
image will enforce it. Reject assets when the camera becomes side-on, fully
top-down, or internally inconsistent across cells.

For directional characters, define both camera and facing. For example:

> The camera remains fixed in a steep three-quarter overhead view. The subject
> faces screen-left. Show the top of the head and shoulders while keeping the
> front and side planes readable.

### Cell Occupancy Contract

Ask for one centered object per implied cell with empty padding on every side.
Objects must not cross cell boundaries. Do not ask the image model to draw a
visible grid because grid lines contaminate cell edges during reduction.

Keep comparable objects at comparable scale. Trees can vary in crown shape and
height while maintaining similar root position and crown occupancy. Characters
should use the same foot line and apparent body scale.

### Shadow Contract

Ask for self-shadowing on the object and no ground plane or cast ground shadow.
Self-shadowing describes the form. Ground shadows belong to runtime rendering
because they depend on light direction, terrain, sprite transformation, and
support points.

Useful prompt language is:

> Preserve self-shadowing and material shading on the object. Do not draw a
> grounding oval, cast shadow, floor, terrain patch, or ambient shadow outside
> the object's silhouette.

Small dark pixels between wheels, branches, or legs can be legitimate object
pixels. A detached shadow shape on the chroma background is not.

## Image-Generation Prompting

### General Prompt Template

Replace bracketed fields and list every cell in row-major order.

```text
Create a [C]-column by [R]-row sprite sheet containing [COUNT] isolated
[ASSET CATEGORY] for a [GAME AND SETTING DESCRIPTION].

Use a fixed steep three-quarter overhead game camera, approximately [ANGLE]
degrees downward. Keep every object at a consistent apparent scale and use the
same light direction in every cell.

Render enlarged low-resolution pixel art with broad deliberate pixel clusters,
hard readable silhouettes, restrained material shading, and a limited muted
palette. Prefer connected color masses over fine illustrative marks. Avoid
gradients, painterly texture, noisy highlights, random isolated detail pixels,
and excessive internal outlines.

Place exactly one centered object in each implied equal-sized cell. Leave
generous empty padding around every object. No object may touch another object
or cross a cell boundary. Do not draw visible grid lines, dividers, labels,
captions, or borders.

Fill every empty pixel with a flat, uniform, saturated magenta chroma-key
background. The background must contain no texture, gradient, lighting,
vignette, scenery, ground plane, or decoration.

Preserve self-shadowing on each object. Do not draw cast ground shadows,
grounding ovals, glow, or ambient shadow outside object silhouettes.

Cell order, left to right and then top to bottom:
1. [CELL DESCRIPTION]
2. [CELL DESCRIPTION]
[CONTINUE THROUGH COUNT]
```

The prompt describes enlarged pixel art because image models are better at
medium-resolution form and texture than exact 16, 32, or 64 pixel grids. The
reducer creates the final logical resolution.

### Category Clauses

Add only the clauses relevant to the requested category.

#### Characters

- Specify facing for each cell.
- Keep feet or another ground contact visible.
- Keep head and hand identity marks large enough to survive reduction.
- Avoid loose straps or weapons crossing cell boundaries.
- Ask for a readable silhouette before costume detail.

#### Trees

- Specify the root or trunk contact near the lower center of the cell.
- Ask for irregular crowns made from a few connected masses.
- Request controlled overlap between branches within one tree.
- Prohibit bright yellow-green tips, airbrushed foliage, and random leaf noise.
- Keep the whole crown and trunk inside the cell.

#### Small Props

- Use simple silhouettes and two or three major material regions.
- Exaggerate identity features that must survive at 16 or 32 pixels.
- Avoid tiny labels, filigree, or repeated one-pixel decoration.
- Keep similar props at a consistent apparent scale.

#### Elongated Props and Vehicles

- Use a rectangular logical cell.
- Specify the object's screen-space diagonal explicitly.
- Require every support point, wheel, and hitch to remain visible.
- Leave extra padding at both ends.
- Prohibit a generic horizontal grounding oval.

### Why Magenta Works

The outdoor palette used green foliage, brown wood and leather, gray stone,
muted clothing, and warm skin. Saturated magenta was absent from legitimate
asset pixels, making it an effective segmentation color.

Image generators rarely produce exact `#ff00ff` across the entire background.
They introduce darker pinks and edge blends. The importer therefore needs a
color-family test rather than exact equality.

Do not use this key when legitimate assets contain purple, pink, or magenta.
Choose a key family absent from the project's art direction and record it in
the asset contract.

## Generation Review

Review the oversized source before reduction. Reduction cannot repair major
art-direction failures.

Reject or regenerate when any of these are present:

- Wrong or inconsistent camera angle
- Objects crossing implied cell boundaries
- Visible grid lines or labels
- Ground planes, cast shadows, or grounding ovals
- Inconsistent scale between comparable cells
- Important parts cropped by image bounds
- Background texture or a large background color drift
- Bright synthetic highlights outside the intended palette
- Fine painterly noise that will average into mud
- Weak silhouettes that rely on internal detail
- Duplicate cells when distinct assets were requested

For a systematic failure such as the wrong camera, regenerate the whole sheet.
For an unusual large object, generate it in its own rectangular source rather
than forcing it into a general-purpose sheet.

Selection is part of the pipeline. Good results came from inspecting and
rejecting candidates, not accepting the first provider response.

## Accepted Sheet Reduction

The accepted implementation lives in:

- `internal/crush/sheet.go`
- `cmd/pxlc-crush-sheet/main.go`

Run it through:

```bash
mise run pxlc:crush-sheet -- [OPTIONS] INPUT.png OUTPUT.png
```

### Algorithm

`ReduceSheetCells` performs direct per-cell RGB box averaging.

Given a source cell with width `Cw` and height `Ch`, and an output cell with
width `W` and height `H`, destination pixel `(x, y)` maps to this integer source
rectangle:

```text
sx0 = floor(x       * Cw / W)
sx1 = floor((x + 1) * Cw / W)
sy0 = floor(y       * Ch / H)
sy1 = floor((y + 1) * Ch / H)
```

The output pixel is:

```text
R = arithmetic mean of source red values in the rectangle
G = arithmetic mean of source green values in the rectangle
B = arithmetic mean of source blue values in the rectangle
A = 255
```

Each sheet cell is reduced independently. This prevents color from one cell
bleeding into its neighbor. The implementation uses integer partitions and
does not calculate fractional source-pixel coverage at boundaries.

This is a conventional box or area-average reducer implemented directly in Go.
The implementation is custom; the mathematical method is standard.

### Deliberately Absent Operations

The accepted reducer does not perform:

- Palette quantization
- Dominant-color or majority voting
- Median filtering
- Dithering
- Edge detection
- Sharpening
- Alpha extraction
- Transparency cleanup
- Chroma spill correction
- Outline generation
- Morphological cleanup
- Lanczos or bicubic interpolation

Those operations were tried or considered during development. Dominant voting
discarded small identity features. Palette quantization changed useful source
colors. Foreground masks and cleanup damaged silhouettes. Direct area averaging
best preserved the generated artwork.

Do not silently add cleanup to this reduction path. Add a separately named
experimental reducer if another project needs to compare an alternative.

### Input Invariants

The current command validates that:

- Columns, rows, cell width, and cell height are positive.
- Each requested dimension is at most 4096.
- The complete output remains inside the configured pixel limit.
- Source width divides evenly by the number of columns.
- Source height divides evenly by the number of rows.
- Every source cell is at least as large as its logical output cell.

Crop or pad a generated source deterministically before invoking the reducer
when provider dimensions do not divide by the requested grid.

### Examples

Reduce a 4 by 3 prop sheet to 32 by 32 cells:

```bash
mise run pxlc:crush-sheet -- \
  --columns 4 \
  --rows 3 \
  --cell-size 32 \
  --preview build/props-preview.png \
  generated-props.png \
  build/props.png
```

Reduce the same accepted source into 16 by 16 ground details:

```bash
mise run pxlc:crush-sheet -- \
  --columns 4 \
  --rows 3 \
  --cell-size 16 \
  --preview build/details-preview.png \
  generated-props.png \
  build/details.png
```

Reduce a single wagon to a 96 by 64 cell:

```bash
mise run pxlc:crush-sheet -- \
  --columns 1 \
  --rows 1 \
  --cell-width 96 \
  --cell-height 64 \
  --preview build/wagon-preview.png \
  generated-wagon.png \
  build/wagon.png
```

The optional preview uses nearest-neighbor enlargement after reduction. Do not
use nearest-neighbor scaling for the reduction itself.

## Chroma-Key Import

`pxlc-crush-sheet` intentionally preserves the source background and emits an
opaque RGB sheet. Chroma-key removal happens when cells are imported.

The successful scene experiment classified magenta-family pixels with this
predicate:

```go
func isChroma(value color.NRGBA) bool {
 return int(value.R)+int(value.B) > 80 &&
  value.R > value.G && value.B > value.G
}
```

Pixels matching the predicate become fully transparent. Other pixels become
fully opaque. The broad predicate catches dark magenta and averaged boundary
colors that are no longer exact `#ff00ff`.

This is a binary key. It can remove legitimate purple pixels and may slightly
change a silhouette when the subject contains pink edge colors. Record that
constraint in the manifest and prompt. The accepted workflow does not apply an
additional transparency-cleanup pass.

For another key family, define a named and versioned predicate. Do not change a
predicate in place after assets have been accepted because rebuilding the same
source would then produce different silhouettes.

## Native and Preview Inspection

Inspect at least two forms after every reduction:

1. Native logical resolution, which reveals whether the sprite reads at game
   scale.
2. Integer nearest-neighbor enlargement, usually 4 to 8 times, which reveals
   pixel clusters, fringes, holes, and isolated noise.

Never judge only the enlarged preview. A sprite can look attractive at 8 times
scale and become unreadable at its native size.

Check:

- Silhouette readability
- Camera consistency
- Ground-contact location
- Separation from the chroma background
- Connected color clusters
- Survival of identity features
- Absence of baked cast shadows
- No contamination from adjacent cells
- Consistent scale across related assets

Regenerate the source when form or camera is wrong. Change the output cell size
when the object lacks enough pixels. Hand-edit only when a small, well-defined
pixel correction is cheaper and clearer than regeneration.

## The Other Crusher

The repository also contains the earlier concept-to-PXLC path:

- `internal/crush/crush.go`
- `cmd/pxlc-crush/main.go`

`pxlc-crush` crops an isolated concept, fits it to a logical canvas, maps colors
to a fixed palette, removes small raster noise, and writes literal PXLC source.
Its compact sprite profile also uses foreground voting and derives a silhouette
outline.

That tool remains useful when literal editable PXLC source and a prescribed
palette are the goal. The generated sheets that worked best for the forest,
NPCs, props, and wagon used `pxlc-crush-sheet` with direct area averaging.
Keep these workflows distinct in an extension and in user-facing names.

## Provenance Manifest

Store a manifest beside every candidate. A practical schema contains:

```json
{
  "schemaVersion": 1,
  "id": "pnw-wagon",
  "status": "candidate",
  "prompt": "Full submitted image prompt",
  "generator": {
    "provider": "provider-name",
    "model": "model-name",
    "requestedWidth": 1536,
    "requestedHeight": 1024,
    "seed": null
  },
  "sheet": {
    "columns": 1,
    "rows": 1,
    "cellWidth": 96,
    "cellHeight": 64,
    "cells": [
      {
        "id": "wagon",
        "description": "Three-quarter overhead abandoned wooden wagon"
      }
    ]
  },
  "chroma": {
    "name": "magenta-dominant-v1"
  },
  "reducer": {
    "name": "rgb-box-average",
    "version": 1
  },
  "files": {
    "source": {
      "path": "source.png",
      "sha256": "..."
    },
    "reduced": {
      "path": "reduced.png",
      "sha256": "..."
    },
    "preview": {
      "path": "preview.png",
      "sha256": "..."
    }
  }
}
```

Also record reference-image hashes when references affect generation. A URL is
insufficient because remote content can change.

Timestamps and provider response IDs may be useful provenance, but they should
not affect deterministic asset hashes or output paths. Keep volatile metadata
outside the reproducible build description.

Suggested candidate layout:

```text
art-source/
  pnw-wagon/
    candidates/
      001/
        manifest.json
        source.png
        reduced.png
        preview.png
    accepted.json
art/
  pnw-wagon.png
```

Acceptance should point to or copy from an immutable candidate. Do not mutate a
candidate from `candidate` to `accepted` while also replacing its files.

## Pi Extension Design Brief

The extension should standardize the process without hiding review decisions.
Use Pi custom tools for deterministic operations and a bundled skill for the
prompting and inspection policy.

Relevant Pi documentation:

- `docs/extensions.md` in the installed Pi package
- `docs/packages.md` in the installed Pi package

### Package Shape

A distributable package can use this structure:

```text
pixel-asset-pipeline/
  package.json
  extensions/
    pixel-assets/
      index.ts
      manifest.ts
      paths.ts
  skills/
    pixel-assets/
      SKILL.md
```

Declare both resources in `package.json`:

```json
{
  "name": "pixel-asset-pipeline",
  "keywords": ["pi-package"],
  "peerDependencies": {
    "@earendil-works/pi-ai": "*",
    "@earendil-works/pi-coding-agent": "*",
    "typebox": "*"
  },
  "pi": {
    "extensions": ["./extensions"],
    "skills": ["./skills"]
  }
}
```

The skill should contain the asset contract, prompt template, review checklist,
and regeneration rules. The extension should contain path validation, command
execution, hashing, manifests, and atomic writes.

### Recommended Tools

Keep the public tools narrow.

#### `pixel_asset_generate`

Responsibilities:

- Validate an asset contract.
- Construct or accept the final image prompt.
- Call a configured image provider when generation is part of the package.
- Save the exact submitted prompt and reference hashes.
- Write a new immutable candidate source.
- Never claim the provider output is deterministic.

Provider integration should remain behind a small adapter. Do not bind the
reduction and manifest code to one image API. A useful first version may omit
generation and accept a source path produced by an existing image tool.

#### `pixel_asset_reduce`

Responsibilities:

- Resolve paths against `ctx.cwd`.
- Validate trusted project configuration.
- Validate source dimensions and sheet settings.
- Invoke `pxlc-crush-sheet` with structured arguments through `pi.exec()`.
- Create the reduced sheet and nearest-neighbor preview.
- Hash source and outputs.
- Write the candidate manifest atomically.
- Return concise paths, dimensions, and hashes.

Use the `AbortSignal` supplied to the tool when invoking `pi.exec()`. Throw on
failure so Pi records an error tool result.

#### `pixel_asset_inspect`

Responsibilities:

- Return the source, native reduction, and preview paths.
- Report dimensions, hashes, sheet layout, and cell order.
- Report validation warnings without modifying files.
- Make the images easy for the agent or user to open and compare.

Avoid automatic aesthetic scoring as an acceptance gate. Camera, silhouette,
and cluster quality still need visual judgment.

#### `pixel_asset_accept`

Responsibilities:

- Require an explicit candidate ID.
- Verify recorded hashes before copying or linking outputs.
- Refuse acceptance when source or reduced files changed.
- Write the accepted asset and pointer manifest atomically.
- Never generate or reduce as a hidden side effect.

### Tool Implementation Rules

Use `pi.registerTool()` with TypeBox schemas. Use `StringEnum` from
`@earendil-works/pi-ai` for string enums when needed.

Asset tools mutate project files, and Pi may execute sibling tools in parallel.
Resolve the real output path and wrap the complete read-modify-write window in
`withFileMutationQueue()`. Queueing only the final write still permits lost
updates.

Read project-local configuration only after checking `ctx.isProjectTrusted()`.
Use `CONFIG_DIR_NAME` rather than hard-coding `.pi` when locating Pi-specific
configuration.

Return small textual results and structured `details`. Large prompts or image
metadata should live in the manifest rather than flooding model context. Tool
results should identify truncation if they can exceed Pi's output limits.

Use staged files followed by atomic rename for manifests and PNG outputs. The
current Go commands already use staged writes at their CLI boundary.

Keep project asset manifests canonical. Pi session entries can improve tool
rendering or reconstruct ephemeral UI state, but a game asset must not depend
on one conversation branch.

### Reducer Integration

Code outside this Go module cannot import `internal/crush`. The extension has
three reasonable integration choices:

1. Invoke a pinned `pxlc-crush-sheet` executable.
2. Invoke this checkout through its `mise run pxlc:crush-sheet` task.
3. Port the small reducer into the extension and verify byte behavior against
   this implementation before adopting it.

Prefer the executable for a reusable package. Record its version in the
manifest. Pass arguments as an array to `pi.exec()` rather than constructing a
shell command string.

If the reducer is ported, preserve these semantics exactly:

- Split cells before resizing.
- Use integer source rectangles.
- Average eight-bit RGB values arithmetically.
- Ignore source alpha and emit alpha 255.
- Keep the chroma background.
- Use nearest-neighbor enlargement only for previews.
- Produce cells in row-major order.

### Configuration

A project-local configuration might define:

```json
{
  "schemaVersion": 1,
  "sourceRoot": "art-source",
  "outputRoot": "art",
  "previewScale": 6,
  "reducerCommand": "pxlc-crush-sheet",
  "defaultChroma": "magenta-dominant-v1",
  "contracts": {
    "npc": {
      "cellWidth": 32,
      "cellHeight": 32
    },
    "tree": {
      "cellWidth": 64,
      "cellHeight": 64
    },
    "wagon": {
      "cellWidth": 96,
      "cellHeight": 64
    }
  }
}
```

Keep initial configuration small. Add provider settings, alternative reducers,
or palette rules only when actual projects require them.

### Extension Workflow

The intended agent loop is:

1. Read the project configuration and existing asset contracts.
2. Create or validate one asset contract.
3. Generate a new source candidate or import an existing generated source.
4. Inspect the oversized sheet.
5. Reduce it with the recorded settings.
6. Inspect native and enlarged outputs.
7. Regenerate or revise the contract when the result fails review.
8. Accept one immutable candidate explicitly.
9. Rebuild accepted assets from saved sources when validating reproducibility.

Do not combine generation, reduction, aesthetic acceptance, and final overwrite
into one opaque tool call. The review boundary produced much of the quality in
this experiment.

## Failure Modes

### Muddy Reduced Pixels

The generated source contains gradients, fine texture, or detail much smaller
than one output pixel. Regenerate with broader clusters and less detail. A more
complicated reducer usually hides the symptom while damaging other assets.

### Magenta Fringe or Missing Edge Pixels

The provider blended subject colors heavily into the background, or the subject
contains pink and purple. Regenerate with harder edges or choose a different
key family. Do not add unrecorded cleanup to accepted assets.

### Cell Bleeding

An object crossed an implied boundary, a visible divider entered the image, or
the whole sheet was resized without splitting cells first. Fix the source or
split cells before reduction.

### Inconsistent Perspective

The prompt relied on style references without repeating the camera contract.
Regenerate with the angle and visible planes stated explicitly.

### Asset Loses Identity at Native Size

Increase the logical cell size, simplify the design, or exaggerate one or two
identity features. Additional generated micro-detail will not survive crushing.

### Rebuild Changes an Accepted Asset

Check the source hash, reducer version, dimensions, and chroma-key version. A
change in any of those inputs should create a new candidate rather than silently
replacing accepted output.

## Porting Checklist

Before calling another implementation compatible with this pipeline, verify:

- [ ] Generated source images are retained immutably.
- [ ] Prompts and reference hashes are recorded.
- [ ] Sheet layout and cell order are explicit.
- [ ] Source dimensions divide evenly by the sheet grid.
- [ ] Every cell is reduced independently.
- [ ] Reduction uses direct RGB box averaging.
- [ ] Reduction retains the opaque chroma background.
- [ ] Chroma removal is a separate, named, versioned import step.
- [ ] Native and nearest-neighbor previews are both inspected.
- [ ] Accepted output records source and result hashes.
- [ ] Regeneration creates a new candidate.
- [ ] Ground shadows remain outside visible sprite pixels.
- [ ] The Pi extension queues file mutations and honors cancellation.
- [ ] Acceptance requires an explicit tool call or user action.

## Current Reference Files

Read these files before reimplementing the behavior:

- `internal/crush/sheet.go`: accepted direct sheet reducer
- `cmd/pxlc-crush-sheet/main.go`: command validation and atomic PNG writes
- `internal/crush/crush.go`: separate palette-based concept crusher
- `cmd/pxlc-crush/main.go`: concept-crusher command
- `README.md`: current command examples and scope
- `mise.toml`: repeatable command tasks

The accepted visual result depended on a plain reducer, a strict generation
contract, and repeated inspection. Preserve those three properties before
adding provider abstractions, UI, palette systems, or automatic cleanup.
