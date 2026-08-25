---
name: pixel-art
description: Draw pixel art character sprites and animations in Aseprite through the aseprite MCP server, following verified pixel art rules for palettes, shading, outlines, limb thickness, and frame timing. Use this whenever the user wants a sprite, dot character, 도트/픽셀 캐릭터, game sprite, idle/walk/attack animation, sprite sheet, character palette, or asks to draw, edit, shade, animate, or export anything pixel art — including vague asks like "make me a little knight" or "animate this guy walking". Also use it when reviewing or fixing existing pixel art, since the anti-pattern checks (pillow shading, cluster banding, noise, off-palette color) live here.
---

# Pixel Art with Aseprite

You are drawing real pixel art through a batch-mode Aseprite CLI. Every tool call
is a separate Aseprite process that opens the file, edits it, and writes it back.
That shapes everything below: batch your edits, and never assume state carries
between calls.

The rules here come from primary sources (Derek Yu, Slynyrd, Lospec) that were
then stress-tested — several widely repeated tutorial rules turned out to be
wrong or resolution-dependent, and this skill carries the corrected versions.
Where a number is a calibrated guess rather than a measured fact, it says so.
`references/character-rules.md` holds the reasoning; read it when a decision
feels underdetermined or when the user pushes back on a choice.

## The one thing that will bite you

**There is no tool that reads pixels back.** `get_sprite_info` reports size,
layers, frames, tags and palette — never image content. So you are drawing
blind unless you deliberately look:

1. `save_sprite_as` the sprite to a PNG in a scratch directory
2. optionally `resize_sprite` that PNG copy by `scale: 8` (nearest neighbour, stays crisp)
3. `Read` the PNG — it renders as an image you can actually see

Do this after the silhouette, after the shading pass, and before declaring done.
Resize the **copy**, never the working file — `resize_sprite` edits in place.
A sprite you never looked at is a sprite you cannot claim works.

## Workflow

Work in this order. It is not ceremony: each stage constrains the next, and
detail added before form is detail you will delete.

### 1. Pin the spec before drawing

Decide and state: canvas size, view (side / top-down / 3/4), palette, light
direction, what the sprite must read as at 1x. If the user did not say, pick
sensible defaults and name them — 32x32, side view, top-front light — rather
than asking a round of questions for a small sprite.

Canvas guidance: 16x16 reads at a glance and is fast; 32x32 is the comfortable
default for a character with a readable face; 48x48+ when the user wants
detail. Whatever you pick, every asset in the same set shares one pixel
density — one asset drawn 1:1 next to another drawn 2x and downscaled is the
"mixel" failure, and it looks broken in a way no amount of shading fixes.

### 2. Build the palette first, and hold the line

Write the palette out explicitly as hex before you draw, then apply it with
`set_palette` so it travels with the file. Nothing enforces a palette lock
through MCP, so the discipline is yours: **never introduce a hex value that is
not on the list you wrote.** If a color is missing, add it to the list
deliberately and say why.

Structure — ramps, not freehand picks. One ramp per material, 3–4 steps
(shadow / base / light, plus one extreme). Per-sprite color budget, driven by
cluster economics — a color that cannot own a cluster of ~4px does not read:

| Canvas | Max colors incl. outline |
|---|---|
| 16x16 | 6 |
| 32x32 | 10 |
| 64x64 | 16 |

Two rules that matter more than the counts:

- **Step lightness perceptually, not in HSB.** HSB Brightness is not perceived
  lightness — uniform B steps produce visibly uneven ramps (yellow at B=100 is
  far brighter than blue at B=100). Reason in OKLCH/Lab lightness when you
  choose ramp steps. This single habit fixes more color problems than any other
  rule here.
- **Hue-shift by the light, not by a formula.** Highlights shift toward the key
  light's hue; shadows shift toward the ambient/fill hue. In daylight that means
  warm highlights and cool shadows — the version every tutorial repeats. Under
  moonlight, underwater, or a torch behind the subject it inverts: cool
  highlights, warm shadows. Ask what is lighting the scene before shifting.
  Keep the shift modest; huge shifts read as a tutorial cliché rather than as form.

### 3. Silhouette

Block the shape in one flat color. Judge it alone: if it does not read as the
subject in silhouette, no shading will save it. Check that it is one connected
shape with no stray floating pixels.

At 16x16 and below the stages genuinely collapse — a single pixel is
simultaneously silhouette, shading and detail — so iterate rather than march
through stages. At 32px+ keep the stages separate.

**Limb thickness** — the rule most tutorials get wrong. A full 1px outline sits
on *both* sides of a limb's cross-section, so holding even 1px of fill needs
`width >= 2 * outline + 1`, i.e. 3px with a 1px outline. Below that you have two
legal choices, and picking one deliberately is what matters:

- **no-fill limb**: render the limb as a single dark stroke, outline color doing
  double duty as fill. This is exactly what Mega Man's legs and Pokémon
  overworld arms do. Fine for short limbs (~4px) or when the silhouette carries
  the read.
- **selective outline**: outline one side only, then `width >= 2` suffices.

Never widen a limb to 2px "because 1px is banned" — 1px limbs are standard at
16x16 in shipped games. The bad outcome is an unintentional limb, not a thin one.

### 4. Flats

Fill each material with its base color. Use the right tool for the job — this is
a real performance concern, since `draw_pixels` issues one Aseprite tool call
per pixel:

- `draw_shapes` with `filled_rectangle` / `filled_ellipse` / `contour` for blocks
- `fill_area` for enclosed regions
- `draw_pixels` for detail and cleanup — but send **every** pixel of an edit in
  one call, because each call reopens and rewrites the whole file

### 5. Shading

One light source, stated in step 1. Think in volume — the sprite is a clay form,
not a line drawing. Apply the shadow step to bottom and back-facing surfaces,
the light step where the key light lands.

Three failure modes to actively avoid, all of which look plausible while you are
drawing and obviously wrong afterwards:

- **Pillow shading** — shading inward from the outline in even rings, so the
  form has no direction. The most reliable defect in the whole discipline: if
  your shadow follows the silhouette at uniform distance, it is wrong.
- **Cluster banding** — two color regions whose boundary runs parallel and
  adjacent along its whole length, which reads as blur. Break the cluster shapes
  so the boundaries diverge. (Note: a deliberate 1px dark line along the inside
  bottom edge for ground contact is technically this, and is correct — the
  defect is the *unintentional* parallel hug.)
- **Noise** — scattered isolated pixels. Group pixels into clusters that own an
  area. Deliberate texture on dirt or grass is the exception, not the rule.

Squint at the result (or view it downscaled): a few large light and dark
clusters should survive. If only noise survives, the shading failed.

### 6. Outline

For a sprite that will sit on unknown or moving backgrounds — which is the
normal case for a game sprite — default to a **closed 1px full outline in a
dark hue-shifted version of the sprite's own dominant color**: very low
lightness (OKLCH L roughly 0.12–0.22), low chroma, hue borrowed from the sprite.
Pure black is the safe fallback; it never fails for readability, it only costs
cohesion.

This is a deliberate correction to tutorial fashion. Selective outlining looks
more sophisticated in isolation and is genuinely a valid style, but it is
miscalibrated for sprites whose background is unknown, which is why shipped
games (Shovel Knight, Nuclear Throne, classic Capcom/SNK) use dark outlines.
Use sel-out when the user asks for it or when you control the background.

Interior partitions — muscle, cloth folds — use that material's shadow step,
never the outline color, so the outline keeps one unambiguous job.

Whichever you choose, be consistent. Half-outlined sprites read as an unmade
decision, and outline closure is something you can check by eye on the exported PNG.

**Do not anti-alias the outer edge.** Interior AA at corners is fine and useful.
The outer silhouette should stay hard: this pipeline uses indexed/palette-locked
output where semi-transparent edge pixels break palette swaps, and at 4x–6x
display scale a ring of soft pixels around a hard interior reads as halo rather
than smoothing, and shimmers against a scrolling background.

### 7. Verify before moving on

Export and look (see "the one thing" above). Then check what is actually
checkable:

- silhouette reads at 1x
- no off-palette colors crept in
- outline closed and uniform
- no pillow shading, no parallel-hugging boundaries, no isolated pixels
- limb rule satisfied per limb
- if the sprite has an intended background, the outline clearly separates from it

## Animation

Add frames with `add_frames` — new frames copy the previous one unless you pass
`empty: true`, which is usually what you want for a copy-then-modify workflow.
Tag ranges with `set_tag`, and set per-frame timing with `set_frame_durations`.

**Timing beats frame count.** A 4-frame walk with well-chosen durations beats an
8-frame walk with uniform ones. Use variable durations: slow anticipation, fast
action, slow recovery. Shipped reference points (useful budgets, not gospel):
walk 4–8 frames, run 3–6, attack 3–6 total with 1–2 anticipation, one contact
frame, and 2 recovery.

At low resolution, 1–3px of movement is plenty. A 1px squash before a jump and a
1px stretch at the apex removes robotic stiffness and genuinely reads — at 4x–6x
display scale that is 4–6 screen pixels, held for two frames.

Use the contact frame for a smear: a stretched or blurred pose that exists for
one frame to sell speed. It does not need to look correct frozen.

Export with `export_spritesheet`. Pass `padding: 2` when the sheet will be
atlased, and `dataFile` when the engine needs frame rectangles. Note that
`save_sprite_as` to `.png` on a multi-frame sprite writes one numbered file per
frame rather than the name you asked for — use `.gif` for a single animated
file, or the spritesheet export for engine assets.

## Tool notes that will save you a failed call

Read `references/aseprite-mcp.md` before a session that involves indexed color,
stamping sprites together, or layer groups — those have sharp edges. The short version:

- Every path must be **absolute**. Relative paths are rejected.
- Creating tools refuse to overwrite unless `overwrite: true`. Editing tools
  rewrite in place with no such guard.
- Author in `rgb` color mode unless the user needs indexed output. Drawing into
  an indexed sprite quantizes through a palette in ways that surprise you.
- Group layers cannot hold pixels. Drawing to one is rejected or silently does nothing.
- Frames and tag ranges are 1-indexed. Coordinates are 0-indexed from the top left.
- If a call fails oddly, `aseprite_health` confirms Aseprite is reachable and reports its version.

## When the user asks for something these rules forbid

Say so in one sentence, then do what they asked. These are defaults derived from
what survives contact with real games, not laws — a user who wants pillow
shading for a soft dreamlike look, or 60 colors on a 32x32 sprite, has made a
style choice and it is theirs to make.
