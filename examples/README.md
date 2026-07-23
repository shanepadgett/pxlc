# Example Asset

Build and enlarge the demonstration icon:

```bash
mise run pxlc -- check examples/icon.pxl
mise run pxlc -- build examples/icon.pxl --output build/art
mise run pxlc -- preview examples/icon.pxl \
  --scale 8 \
  --output build/previews/icon.png
```

Change the `metal` hex value in `icon.pxl`, rebuild, and compare the source diff
with the preview. Geometry stays untouched because operations refer to the
named palette color and grids use its `M` symbol.
