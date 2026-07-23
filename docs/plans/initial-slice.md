# PXLC: Initial Slice

## Project Brief

**PXLC** turns text-based pixel-art source into deterministic, game-ready images and metadata.

It is designed for coding agents, programmers, and small teams that want pixel assets to behave like source code: readable, reviewable, repeatable, and suitable for automated builds.

PXLC is a compiler first. Interactive painting, image generation, and engine integration remain separate concerns.

This document is the implementation target for the first usable slice: lock the basic semantics, then compile still images. Later composition, metadata, animation, and tool-hardening work lives in [roadmap.md](roadmap.md).

## Product Goals

PXLC should let a user or coding agent:

1. Define pixel art in a compact text format.
2. Reuse palettes, shapes, stamps, layers, and base frames as the project grows.
3. Create still images first, followed by frame-based animation.
4. Attach game metadata such as origins, sockets, hitboxes, and masks in later phases.
5. Validate the source before producing artifacts.
6. Generate deterministic PNG files and engine-independent metadata.
7. Produce enlarged previews and contact sheets for visual review.
8. Run the entire process non-interactively from a terminal or CI system.

The initial workflow should look like this:

```bash
pxlc check art/player.pxl
pxlc build art/player.pxl --output build/art
pxlc preview art/player.pxl --output build/previews/player.png
```

The agent edits source, runs PXLC, inspects the rendered result, and repeats. No manual PNG editing should be required.

## Core Principles

### Text is the canonical source

The `.pxl` files are the assets developers edit and review. PNG files and metadata are generated artifacts.

The source format must be:

- Human-readable.
- Comfortable for coding agents to produce.
- Friendly to line-based version-control diffs.
- Compact enough for sprites larger than tiny icons.
- Strict enough to produce useful diagnostics.
- Independent of any game engine or programming language.

### Pixel-native behavior

Every rendering operation works in integer pixel coordinates.

The compiler must avoid:

- Accidental antialiasing.
- Subpixel positioning.
- Renderer-dependent curve behavior.
- Implicit filtering.
- Platform-dependent geometry.

A given operation must have one documented raster result.

### Deterministic builds

Given the same source, configuration, PXLC version, and declared seeds, PXLC should produce byte-identical artifacts across repeated builds.

Generated files must not contain timestamps, machine paths, random ordering, or other incidental state.

### Engine-independent output

The initial standard output consists of PNG images and versioned metadata. A game should not need to embed PXLC or use its internal data structures at runtime.

### Agent-friendly operation

All important behavior must be available through the CLI. Failures should include source paths, lines, columns, error codes, and actionable explanations.

PXLC should eventually offer machine-readable diagnostics, but it must remain pleasant to use directly from a terminal.

## Initial Source Model

The first slice defines the foundation for the broader source model in the roadmap.

### Canvas

- Logical width and height.
- Transparent background or declared background color.
- Configurable size limits.
- Integer coordinates with a documented origin and axis direction.

Named origins or pivots are deferred with game metadata.

### Palettes

- Named colors.
- Short symbols for literal pixel grids.
- Transparent entries.
- Validation that rendered pixels belong to the declared palette.
- Optional maximum-color constraints.

The initial version should favor fully opaque colors plus complete transparency. Partial alpha can be considered after its rendering and compositing rules are clearly specified.

Shared palette imports are deferred with imports and reusable composition.

### Layers

- Named layers.
- Explicit drawing order.
- Normal deterministic compositing.

Visibility controls for preview and output, and named auxiliary layers or masks, are later work. Complex blend modes are outside the initial scope.

### Pixel geometry

Phase 1 covers:

- Individual pixels.
- Horizontal and vertical spans.
- Embedded ASCII pixel grids.
- Filled rectangles.

All operations need exact clipping and overlap rules. Lines, polygons, ellipses, flood fill, transforms, outlining, and stamping are in the roadmap.

## CLI Requirements

The first slice exposes a small command set.

### `pxlc check`

Parse and validate assets without writing normal build output.

```bash
pxlc check art/
pxlc check art/player.pxl
```

### `pxlc build`

Compile one asset or an asset tree.

```bash
pxlc build art/ --output build/art
```

Build should perform validation automatically and leave existing valid artifacts untouched if compilation fails.

### `pxlc preview`

Produce review artifacts without changing runtime output.

```bash
pxlc preview art/player.pxl \
  --scale 8 \
  --output build/previews/player.png
```

Initial previews may be static contact sheets. Animation playback can arrive later.

### Common CLI behavior

All commands should support:

- Stable nonzero exit codes for failure.
- Quiet and verbose operation.
- Human-readable diagnostics.
- Relative paths in reproducible output.
- No interactive prompts during automated builds.
- `pxlc --version`.
- Clear reporting of source-format and output-schema versions.

An eventual structured diagnostic format is roadmap work.

## Generated Artifacts

### Runtime images

PXLC should initially generate one PNG for a still asset. Packed atlases are deferred.

### Metadata

A versioned, engine-independent metadata file should contain, as those features become available:

- Asset identifier.
- Canvas dimensions.
- Output image paths.
- Palette information where requested.
- Source and compiler version information.
- Content hashes useful for caching and verification.

JSON is acceptable as generated interchange data. It should not be required for hand-authoring pixel geometry.

### Review output

Preview generation should support native 1× rendering, integer nearest-neighbor enlargement, transparent backgrounds, and configurable solid backgrounds. Frame labels, durations, animation strips, layer or mask contact sheets, and metadata overlays belong to later phases.

## Validation and Diagnostics

PXLC must reject invalid input through ordinary errors rather than crashes or partial output.

The initial slice validates:

- Invalid syntax.
- Duplicate layer, palette, or metadata names where applicable.
- Invalid canvas sizes.
- Out-of-bounds coordinates.
- Undeclared colors.
- Invalid or empty palettes.
- Excessive dimensions, operations, or allocation requests.
- Unsupported source-format versions.

Example:

```text
art/player.pxl:84:17 PXLC-E031
Color #7e5244 is outside palette "wasteland_16".
```

Warnings should be distinguishable from errors and individually suppressible once stable warning codes exist.

## Reliability and Testing

The compiler needs strong automated coverage because a subtle rasterization change can alter every asset in a game.

Required initial tests include:

- Parser and formatter round trips when formatting exists.
- Golden-image tests for every implemented drawing operation.
- Golden metadata tests.
- Determinism across repeated builds.
- Stable ordering independent of filesystem enumeration.
- Clipping and boundary behavior.
- Palette enforcement.
- Malformed and adversarial source files.
- Resource-limit enforcement.
- Failed builds preserving prior valid output.
- Cross-platform artifact comparison.

Each source-format or rasterization change should state whether it intentionally changes existing output.

## Initial Demonstration Asset

The first slice should prove itself with a small palette-constrained icon. The demonstration should show source diffs beside generated previews. It should be possible for an agent to alter a palette color without rewriting the whole asset.

## Explicit Initial Non-Goals

The first release should exclude:

- A brush-based pixel editor.
- A full graphical asset-authoring application.
- Natural-language prompt processing.
- Built-in image-generation services.
- Automatic conversion of arbitrary illustrations into good pixel art.
- SVG or general vector rendering.
- Arbitrary user scripts or executable asset source.
- Skeletal animation, inverse kinematics, or tweened subpixel animation.
- Packed-atlas optimization.
- Engine-specific plugins and runtime libraries.
- Tilemap editing and complete autotiling systems.
- Shaders and runtime post-processing.
- Cloud storage, accounts, or collaborative editing.
- Plugin marketplaces.
- PNG optimization as a primary concern.
- Support for every raster image format.
- Subjective claims that an asset is artistically good.

PXLC can validate mechanical rules. Palette checks cannot determine whether a silhouette reads well or an animation has personality.

## Delivery Plan: Phases 0–1

### Phase 0: Lock semantics

Define and test:

- Coordinate system.
- Palette and transparency behavior.
- Layer compositing.
- Clipping.
- Drawing-operation rasterization for the initial operation set.
- Output file naming.
- Metadata schema.
- Source-format versioning.

**Gate:** A short language specification answers each behavior without relying on a particular renderer’s defaults.

### Phase 1: Compile still images

Implement:

- Parser and diagnostics.
- Canvas and palettes.
- Layers.
- Literal grids, pixels, spans, and rectangles.
- Deterministic PNG output.
- `check`, `build`, and basic `preview`.
- Golden rendering tests.

**Gate:** A coding agent can create and revise a palette-constrained still sprite using only text source and CLI output.

## Initial Definition of Done

This first useful slice is complete when:

- `.pxl` source is documented and versioned.
- Still assets compile deterministically.
- Palettes and layers work.
- PNG images and versioned metadata are generated.
- Previews are readable at native and enlarged scales.
- Errors identify the source location and concrete repair.
- Batch and CI operation require no human interaction.
- Malformed input cannot cause unbounded allocation or partial replacement.
- The demonstration asset builds identically in a clean environment.
- A coding agent can create, inspect, revise, and validate an asset without manually editing generated images.
