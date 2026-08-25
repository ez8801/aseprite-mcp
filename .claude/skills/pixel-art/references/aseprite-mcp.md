# aseprite MCP — tool surface and sharp edges

Every tool call is one Aseprite process: it opens the file, runs a script, writes
the file, exits. Nothing persists between calls — no open document, no active
layer, no selection. Each call takes absolute paths and does its whole job.

The behaviors below were measured against Aseprite 1.3.18.2, not assumed. Trust
them over intuition about how a GUI would behave.

## Universal rules

- **Absolute paths only.** Relative paths are rejected before the script runs,
  because they would otherwise resolve against Aseprite's working directory.
- **Creating tools refuse to clobber.** `create_sprite`, `save_sprite_as`,
  `save_palette` and `export_spritesheet` fail if the destination exists unless
  you pass `overwrite: true`. Editing tools rewrite in place by design and have
  no such guard — there is no undo.
- **Frames and tags are 1-indexed. Pixel coordinates are 0-indexed** from the top
  left corner.
- **Group layers cannot hold pixels.** Any drawing tool aimed at a group is
  rejected. When you omit `layer`, the tools target the first image layer.
- **Failures come back as tool errors with a message, not as crashes.** Read the
  message; it usually names the exact problem (missing layer, frame out of range,
  file exists).
- `aseprite_health` reports the executable path and version. Use it first when a
  call fails for no obvious reason.

## There is no pixel-read tool

`get_sprite_info` returns size, color mode, layer list, frame count, tags and up
to 32 palette entries. `get_palette` returns the full palette. Neither returns
image data, and nothing else does either.

To see your own work:

```
save_sprite_as   source: work.aseprite   destination: scratch/look.png
resize_sprite    path: scratch/look.png  scale: 8        # nearest neighbour, stays crisp
Read             scratch/look.png                         # renders as an image
```

Resize the exported copy, never the working file — `resize_sprite` edits in
place. Delete or overwrite the scratch PNG on the next look rather than
accumulating files.

## Drawing

### `draw_pixels`
`path`, `pixels: [{x, y, color}]`, optional `layer`, `frame`.

Colors are `#RRGGBB`, `#RRGGBBAA`, or a palette index like `5` on an indexed sprite.

**Send every pixel of an edit in one call.** Each call reopens and rewrites the
whole file, and internally the tool issues one Aseprite draw per pixel — so
this is the expensive tool. Use it for detail and cleanup, not for filling area.

(Historical note on why it is one draw per pixel: passing several points to a
single Aseprite `useTool` call draws a *stroke connecting them*, not separate
dots. Adjacent pixels hide this, which is why the bug survived early testing.)

### `draw_shapes`
`path`, `shapes: [{kind, from: {x,y}, to: {x,y}, color, brushSize?}]`, optional `layer`, `frame`.

`kind` is one of `line`, `rectangle`, `filled_rectangle`, `ellipse`,
`filled_ellipse`, `contour`. Shapes draw in the order given. This is the cheap
way to lay in blocks and silhouettes.

### `fill_area`
`path`, `x`, `y`, `color`, optional `layer`, `frame`, `tolerance` (default 0),
`contiguous` (default true). The paint bucket. Good for enclosed regions once an
outline exists.

### `clear_area`
`path`, optional `rect: {x, y, width, height}`, `layer`, `frame`. Omit `rect` to
clear the whole cel.

### `stamp_sprites`
`destination`, `stamps: [{source, x, y, sourceFrame?, opacity?, blendMode?}]`,
optional `layer`, `frame`. Composites whole sprite files into another, back to
front, the way a scene is assembled from separate character and prop files.
Sources are untouched. Stamps falling partly outside are clipped, and the result
reports it.

## Color mode

**Author in `rgb` unless the user needs indexed output.** Drawing an RGB image
into an indexed sprite quantizes it against a palette in ways that surprise you —
a red pixel can land on a palette index that is not the palette's own red. The
server works around this by converting pixel-by-pixel through the destination
palette for indexed and grayscale targets, but RGB destinations are the
predictable path.

You can still apply a palette to an RGB sprite with `set_palette` so the palette
travels with the file; the discipline of only using colors from it is yours to
keep, since nothing locks it.

## Layers, frames, tags

- `add_layer`: `path`, `name`, optional `group` (make a group instead of an image
  layer), `parent` (nest inside an existing group), `opacity`, `visible`.
- `update_layer`: `path`, `name`, plus at least one of `newName`, `opacity`,
  `visible`, `blendMode`. Groups have no blend mode.
- `delete_layer`: deleting a group deletes everything inside it.
- `add_frames`: `path`, optional `count` (default 1), `after` (default: append),
  `empty`. **New frames copy the previous frame unless `empty: true`** — copy is
  usually what you want for animation, empty for a fresh pose.
- `delete_frames`: `path`, `frames: [n]`. A sprite must keep at least one frame.
- `set_frame_durations`: `path`, `durations: [{frame, durationMs}]`.
- `set_tag`: `path`, `name`, `from`, `to`, optional `aniDir` (`forward`,
  `reverse`, `ping_pong`, `ping_pong_reverse`), `repeats` (0 = forever). An
  existing tag with the same name is replaced.
- `delete_tag`: `path`, `name`.

## Palette

- `get_palette`: full palette, untruncated.
- `set_palette`: `path`, and either `colors: ["#RRGGBB", ...]` in order or
  `fromFile` pointing at a `.gpl`, `.pal`, `.aseprite` or image.
- `save_palette`: `path`, `destination` (format from the extension: `.gpl`,
  `.pal`, `.png`), `overwrite`. Useful for handing the user a reusable palette
  file alongside the sprite.

## Export

### `save_sprite_as`
`source`, `destination`, `overwrite`. Format follows the destination extension.

**Multi-frame gotcha:** exporting a multi-frame sprite to a single-image format
such as `.png` does not produce the filename you asked for — Aseprite writes one
numbered file per frame (`anim1.png`, `anim2.png`, ...). The result reports the
real filenames in `files` and sets `splitIntoSequence`, and the overwrite check
covers those generated names. Use `.gif` for one animated file, or
`export_spritesheet` for an engine asset.

### `export_spritesheet`
`path`, `destination`, optional `dataFile`, `sheetType` (`packed` default,
`horizontal`, `vertical`, `rows`, `columns`), `dataFormat` (`hash` default or
`array`), `padding` (default 0), `trim`, `splitLayers`, `splitTags`, `overwrite`.

Pass `padding: 2` when the sheet will be atlased or filtered, so neighboring
frames cannot bleed into each other. Pass `dataFile` when an engine needs frame
rectangles. Avoid `trim` if the consumer relies on a consistent frame box and
pivot — trimming makes every frame a different size.

### `resize_sprite`
`path`, and either `width`/`height` or `scale`. Scaling up uses nearest
neighbour so pixel art stays crisp. Edits in place, so use it on an exported
copy when the goal is just to look at something enlarged.

## Two crash-adjacent facts worth knowing

- The server never calls Aseprite's `Clear` command, because it crashes Aseprite
  with an access violation whenever the active cel is missing — a group layer is
  active, or the frame is empty. `clear_area` goes through image clearing and cel
  deletion instead. If you ever extend this server, do not reintroduce that command.
- Aseprite writes diagnostics to stdout and never to stderr, and a caught Lua
  error still exits 0. The server therefore tags results with a sentinel and
  scans stdout for it rather than trusting exit codes. This is why tool errors
  arrive as readable messages instead of transport failures.
