English | [한국어](README.ko.md)

[![CI](https://github.com/ez8801/aseprite-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/ez8801/aseprite-mcp/actions/workflows/ci.yml)

# Aseprite MCP

An [MCP](https://modelcontextprotocol.io) server that lets an AI assistant
create, edit and export [Aseprite](https://www.aseprite.org/) sprites.

It drives the Aseprite batch-mode CLI, so it needs a local Aseprite
installation (tested against 1.3.18.2) but no plugin and no running GUI.

## Tools

### Files and documents

| Tool | Purpose |
| --- | --- |
| `aseprite_health` | Report which Aseprite executable is in use, and its version. |
| `create_sprite` | Create an empty sprite and write it to disk. |
| `get_sprite_info` | Report size, color mode, layers, frames, tags and palette. |
| `save_sprite_as` | Save a copy in another location or format. |

### Drawing

| Tool | Purpose |
| --- | --- |
| `draw_pixels` | Set individual pixels, each with its own color. |
| `draw_shapes` | Draw lines, rectangles and ellipses, filled or outlined. |
| `fill_area` | Flood fill from a point. |
| `clear_area` | Erase a rectangle, or a whole cel. |
| `stamp_sprites` | Copy whole sprites into another sprite, to assemble a scene. |

### Layers, frames and tags

| Tool | Purpose |
| --- | --- |
| `add_layer` | Add an image layer or a layer group. |
| `update_layer` | Rename, or change opacity, visibility or blend mode. |
| `delete_layer` | Delete a layer, and a group's contents with it. |
| `add_frames` | Append or insert frames, copied or blank. |
| `delete_frames` | Delete frames by number. |
| `set_frame_durations` | Retime individual frames. |
| `set_tag` | Create or replace an animation tag. |
| `delete_tag` | Delete an animation tag. |

### Palette and export

| Tool | Purpose |
| --- | --- |
| `get_palette` | Read the whole palette, untruncated. |
| `set_palette` | Replace the palette from a color list or a palette file. |
| `save_palette` | Write the palette to its own file (`.gpl`, `.pal`, ...). |
| `resize_sprite` | Resize to a size or by a scale factor. |
| `export_spritesheet` | Export all frames to one sheet, with an optional JSON data file. |

## Prompt

The server also exposes one MCP prompt, `animated_character`, which lays out the
workflow for drawing a character and its animation states with these tools:
palette and silhouette, base pose, a look-at-it checkpoint, frames per state,
tags and timing, then export. It numbers the frame ranges for you from the
states you ask for.

In Claude Code it appears as a slash command:

```
/aseprite:animated_character
```

| Argument | Meaning |
| --- | --- |
| `name` | Required. Character slug used for the filenames. |
| `description` | Required. What the character looks like. |
| `outputDir` | Absolute directory to write into. |
| `animations` | Comma separated `state:frames` pairs. Defaults to `idle:6,attack:8`. |
| `size` | Canvas size as `WIDTHxHEIGHT`. Defaults to `64x64`. |

## How calls behave

Calls are stateless: there is no document that stays open between them, so
every tool takes absolute paths and reads or writes files directly.

Editing tools rewrite the sprite they are given, in place. Each call opens and
saves the file once, so send a whole edit in one call rather than one call per
pixel.

Tools that create a new file refuse to replace an existing one unless
`overwrite` is set to `true`. Editing tools have no such guard, since changing
the file is the point.

Colors are `#RRGGBB` or `#RRGGBBAA`. On an indexed sprite a bare number such as
`5` is used as a palette index instead.

A scene is assembled with `stamp_sprites`: keep each character and prop in its
own file, then copy them into a background at the positions you want. Stamps
are composited with transparency in the order given, so later ones sit in
front, and the source files are left untouched.

Frames are numbered from 1. A layer name is matched inside groups too, and the
first match in stacking order wins. Drawing tools default to the first image
layer; a group holds no pixels and is rejected.

## Install

Download `aseprite-mcp.exe` from the
[latest release](https://github.com/ez8801/aseprite-mcp/releases/latest), or
build it yourself.

Release binaries are built by GitHub Actions and carry a build attestation, so
you can check that a download came from this repository before running it:

```
gh attestation verify aseprite-mcp.exe --repo ez8801/aseprite-mcp
```

The release also ships `checksums.txt` for a plain hash comparison. The binary
is not Authenticode signed, so Windows SmartScreen will still warn on first
run.

## Build

Go 1.26 or newer:

```
go build -o aseprite-mcp.exe ./cmd/aseprite-mcp
```

The repository is laid out as a single command over one internal package:

```
cmd/aseprite-mcp/     the server binary, and the end-to-end tests
internal/aseprite/    the runner, the Lua scripts, the tools and the prompt
```

## Configure

Register the binary with your MCP client. For Claude Code:

```
claude mcp add aseprite -- C:\path\to\aseprite-mcp.exe
```

Or add it to `.mcp.json`:

```json
{
  "mcpServers": {
    "aseprite": {
      "command": "C:/path/to/aseprite-mcp.exe"
    }
  }
}
```

### Environment

| Variable | Meaning |
| --- | --- |
| `ASEPRITE_PATH` | Path to `Aseprite.exe`. Only needed when auto-detection fails. |
| `ASEPRITE_MCP_TIMEOUT` | Per-call timeout as a Go duration, e.g. `30s`. Defaults to `60s`. |

Auto-detection looks in the usual Steam and standalone install locations, then
falls back to `aseprite` on `PATH`.

## Notes

- A single-image format such as `.png` cannot hold an animation. Exporting a
  multi-frame sprite to one makes Aseprite write a numbered file per frame
  (`anim1.png`, `anim2.png`, ...) instead of the name you asked for. The result
  lists the real filenames in `files` and sets `splitIntoSequence`. Use `.gif`
  to get a single animated file, or `export_spritesheet` for one sheet.
- `get_sprite_info` lists at most the first 32 palette entries; `paletteSize`
  always reports the real count and `paletteTruncated` says whether the list
  was cut. Use `get_palette` for the whole palette.

## Test

```
go test ./...
```

The suite builds the server and drives it over a real stdio MCP session. It
skips when Aseprite is not installed.
