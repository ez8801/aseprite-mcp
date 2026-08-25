# Aseprite MCP

MCP server that drives Aseprite through its batch-mode CLI.

## Architecture

Every tool call is one Aseprite process. `Runner.Run` (aseprite.go) renders a
Lua body into a temp file, runs `Aseprite.exe -b --script <file>`, and reads
the result back off stdout.

- `aseprite.go` — locating Aseprite, the script envelope, running it.
- `scripts.go` / `scripts_edit.go` / `scripts_structure.go` /
  `scripts_palette.go` — the Lua bodies, plus the shared helpers in
  `luaHelpers`.
- `tools.go` / `tools_edit.go` / `tools_palette.go` — tool definitions and the
  argument structs the schema is inferred from.
- `prompts.go` — the `animated_character` MCP prompt, which templates a sprite
  and animation workflow from its arguments.

Handlers are uniform: `run[Args](runner, luaBody)` marshals the arguments to
JSON, validates every field named in `pathFields` at any depth, and hands the
rest to the script. Validation beyond paths belongs in Lua, where the sprite is open.

## Verified CLI behaviour

All of the following was measured against 1.3.18.2, not assumed. Changing any
of it needs a fresh probe rather than a guess.

- Aseprite writes diagnostics to stdout and never to stderr, so results are
  tagged with the `@@ASEMCP@@` sentinel and recovered by scanning lines.
  `ExportSpriteSheet` also dumps its data JSON to stdout.
- A caught Lua error still exits 0, and `Sprite{fromFile=<missing>}` returns
  nil rather than raising. Scripts validate explicitly and report through the
  `pcall` envelope in `prelude`; exit codes are not trusted. An *uncaught*
  error does exit non-zero, which is the no-sentinel path.
- **`app.command.Clear()` crashes Aseprite with an access violation whenever
  the active cel is missing** — a group layer is active, or the frame is
  empty. Clearing goes through `cel.image:clear(rect)` and
  `sprite:deleteCel()` instead. Do not reintroduce the command.
- `json.decode` returns **userdata proxies, not Lua tables**. Indexing, `#`,
  `ipairs` and arithmetic all work, but `type(v) == "table"` is false. Probe
  nested values by field.
- Every number from `json.decode` is a float, so `whole()` rounds anything
  used as a count, index or coordinate. Without it `99.0` leaks into messages.
- Drawing on a group layer silently does nothing, so `drawTarget` rejects
  groups up front.
- `app.useTool` creates a cel when the frame has none, and expands an existing
  cel while preserving its content. Cels are trimmed to their non-transparent
  bounds, so `cel.bounds` is not the sprite rect and image coordinates are
  relative to the cel.
- Exporting a multi-frame sprite to a single-image format does not produce the
  requested filename: Aseprite writes `<title><frame>.<ext>` per frame. The
  overwrite check covers those generated names too, and the result reports the
  real filenames.
- Parameters are embedded in the generated script as a JSON literal inside a
  Lua long bracket instead of passed with `--script-param`, which avoids
  Windows argument escaping and `=` splitting in paths.
- Passing several points to one `app.useTool` call draws a **stroke that
  connects them**, not separate dots, so `draw_pixels` calls the tool once per
  pixel. Adjacent pixels hide this, which is why it survived the first tests.
- **Drawing an rgb image into an indexed one quantizes against a default
  palette, not the destination sprite's**, whatever `app.sprite` is set to. A
  red pixel lands on index 27 instead of the palette's own red. Stamping into
  an indexed or grayscale sprite therefore converts pixel by pixel through the
  palette in `luaConverters`; only rgb destinations use Aseprite's blending.
- Opening a sprite makes it the active one, so anything that depends on the
  active sprite has to be re-pointed after a `Sprite{fromFile=...}`.
- A newly created sprite already owns a cel the size of the whole canvas, so
  growing a cel to the union of its old bounds and a new stamp would keep every
  cel full size. `Image:shrinkBounds()` trims it back to real content.
- `luaHelpers` is spliced into `prelude`, which is a format string, so it must
  stay free of percent verbs.

## Conventions

- Go sources are LF; everything else is CRLF (see .gitattributes). This
  deviates from the global CRLF rule because gofmt enforces LF.
- Tool handlers report script failures as `IsError` results, not Go errors, so
  the model can read and react to the message.
- Tools that create a file refuse to overwrite unless `overwrite` is set.
  Editing tools rewrite in place by design and have no such guard.

## Testing

`go test ./...` builds the server and drives it over a real stdio session.
Tests skip when Aseprite is not installed. Fixtures that need features the
tools do not expose are built by calling `Runner.Run` directly.
