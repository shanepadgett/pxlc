# PXLC: Roadmap

This file holds work intentionally outside the initial still-image slice in [initial-slice.md](initial-slice.md). It preserves the intended direction without expanding the current implementation goal.

## Deferred Source Model

### Pixel geometry and transforms

The remaining initial operation set should cover:

- Pixel-perfect lines.
- Filled polygons.
- Filled circles or ellipses.
- Flood fill within a bounded layer.
- Translation.
- Horizontal and vertical flipping.
- Mirroring around an integer axis.
- Palette-color replacement.
- Outlining a mask or opaque region.
- Stamping one reusable shape into another asset or frame.

All operations need exact clipping and overlap rules.

### Reusable parts

Assets should support reusable declarations for items such as:

- Heads and faces.
- Hands and feet.
- Weapons.
- Wheels.
- Clothing pieces.
- Mechanical parts.
- Common UI symbols.
- Terrain details.

Reuse must remain declarative. The first version should not embed a general-purpose scripting language.

### Frames and animation

An animated asset should support:

- Named frames.
- Explicit frame duration.
- Named animation tags.
- Looping and one-shot animation metadata.
- Frame inheritance.
- Per-frame changes layered over a stable base frame.
- Stable frame ordering.
- Fixed-cell sprite-sheet export.

Frame inheritance is a major requirement. It lets an animation preserve the same body and palette while describing only the pixels or parts that move.

### Game metadata

PXLC should support engine-independent named metadata:

- Origin or pivot.
- Attachment points and sockets.
- Named rectangles.
- Named binary masks.
- Frame-specific metadata.
- Asset-wide metadata.

PXLC should not assign gameplay meaning to names such as `hitbox`, `muzzle`, or `emissive`. It records the data; the consuming game interprets it.

## Deferred CLI and Artifacts

### `pxlc fmt`

Format source into a canonical layout.

```bash
pxlc fmt art/
pxlc fmt --check art/
```

Canonical formatting reduces meaningless agent-generated diffs.

### Runtime images

Add:

- A fixed-cell PNG sheet for animated assets.
- Optional individual frame PNGs where requested.
- Separate PNG files for named binary masks when needed.

Packed atlases stay deferred. Fixed cells make frame coordinates stable and prevent one changed sprite from moving unrelated assets.

### Metadata

Extend versioned, engine-independent metadata with:

- Frame rectangles and durations.
- Animation tags and loop behavior.
- Origins and pivots.
- Named points, rectangles, and masks.

### Review output

Add frame labels and durations, animation strips, layer or mask contact sheets, and metadata overlays where requested.

## Deferred Validation and Configuration

Add validation for:

- Duplicate asset, frame, and metadata names.
- Missing imports and references.
- Import, stamp, or inheritance cycles.
- Invalid frame durations.
- Empty animations.
- Metadata outside its associated canvas.
- Mismatched mask dimensions.
- Invalid transforms.
- Invalid paths or attempts to escape the project root.

Example:

```text
art/player.pxl:117:5 PXLC-E064
Socket "muzzle" at (33, 18) is outside the 32×40 canvas.
```

PXLC should support a project-level configuration file for shared policy:

- Source roots.
- Output roots.
- Import search paths.
- Default preview scale and background.
- Maximum canvas dimensions.
- Maximum colors.
- Binary-alpha policy.
- Naming rules.
- Output metadata version.
- Whether generated files are checked for freshness.

Single-file use should remain possible without a project file.

## Later Demonstration Assets and Tests

Add demonstrations for:

1. A roughly character-sized sprite with idle and walk animations.
2. A mechanical object with several layers and an animated part.
3. An asset with an origin, attachment point, collision rectangle, and binary mask.
4. A sprite using shared stamps and frame inheritance.

The demonstration should show source diffs beside generated previews. It should be possible for an agent to alter a pose or attachment point without rewriting the whole asset.

Add frame-inheritance and stamp-composition tests, and cycle-detection tests.

## Delivery Plan: Phases 2–4

### Phase 2: Composition and metadata

Implement:

- Import and reference-resolution semantics.
- Remaining initial primitives.
- Transforms and mirroring.
- Reusable stamps.
- Imports.
- Origins, points, rectangles, and masks.
- Project configuration.
- Resource limits and malformed-input tests.

**Gate:** A composed mechanical asset can be built from reusable parts with valid game metadata.

### Phase 3: Animation

Implement:

- Frame ordering and inheritance semantics.
- Frames and durations.
- Frame inheritance.
- Animation tags.
- Fixed-cell sprite sheets.
- Frame metadata.
- Animation contact sheets.

**Gate:** The demonstration character can walk without duplicating its entire base construction in every frame.

### Phase 4: Tool hardening

Implement:

- Canonical formatter.
- Batch builds.
- Machine-readable diagnostics.
- Atomic artifact replacement.
- Artifact hashes and freshness checks.
- Cross-platform release packaging.
- Complete command documentation.
- Example project and CI integration.

**Gate:** A clean environment can install PXLC, build the example project, reproduce its artifacts, and receive useful diagnostics for deliberate failures.

## Near-Future Features

Add these after the initial workflow survives use in a real game:

- Watch mode with incremental rebuilds.
- Read-only animation preview application.
- Hot-reloading previews.
- Animation playback speed controls.
- Layer, mask, socket, and hitbox overlays.
- A/B comparison of asset variants.
- Palette ramps and validated palette variants.
- Indexed PNG output.
- Tile and seamless-edge validation.
- Basic tileset and autotiling declarations.
- Deterministically seeded texture operations.
- Atlas packing as a separate build stage.
- Binary metadata output.
- Syntax highlighting and editor integration.
- Language Server Protocol diagnostics.
- Importers for limited, palette-safe PNG workflows.
- Export adapters maintained separately from the compiler core.

Any importer should preserve the text source as canonical or clearly mark the imported raster as an external source.

## Far-Future Wishlist

These are experiments rather than promises:

- A source-aware visual debugger showing which operation produced each pixel.
- Clicking a rendered pixel to jump to its source declaration.
- Source-level animation onion skinning.
- Visual frame-difference and jitter analysis.
- Automatic detection of stray pixels and weak color clusters.
- Silhouette readability reports.
- Animation motion-path overlays.
- Constrained sprite-family generation from reusable body parts.
- Palette reduction suggestions with reviewable source patches.
- Symmetry and proportion constraints.
- Nine-slice UI asset support.
- Normal, height, material, and richer auxiliary maps.
- Shared libraries of palettes, stamps, and asset templates.
- A browser-based playground.
- A stable library API for embedding the compiler.
- An agent protocol or MCP wrapper over the CLI.
- Optional integration with established pixel editors.
- Review annotations that refer to frames and pixel coordinates.
- Reproducible asset packages with declared licenses and provenance.
- Safe extension points for custom exporters and validators.

## Full Project Definition of Done

PXLC’s first full release is complete when:

- `.pxl` source is documented and versioned.
- Still and animated assets compile deterministically.
- Palettes, layers, reusable stamps, and frame inheritance work.
- PNG sheets and versioned metadata are generated.
- Origins, points, rectangles, and binary masks survive compilation.
- Previews are readable at native and enlarged scales.
- Errors identify the source location and concrete repair.
- Batch and CI operation require no human interaction.
- Malformed input cannot cause unbounded allocation or partial replacement.
- The demonstration assets build identically in a clean environment.
- A coding agent can create, inspect, revise, and validate an asset without manually editing generated images.
