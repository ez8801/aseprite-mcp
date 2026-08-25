package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ez8801/aseprite-mcp/internal/aseprite"
)

// luaReadPixel reports one composited pixel as #RRGGBBAA. Aseprite packs a
// pixel as 0xAABBGGRR, so the channels are pulled out by arithmetic.
const luaReadPixel = `
local sprite = openSprite(P.path)
local flat = Image(sprite.width, sprite.height, ColorMode.RGB)
flat:drawSprite(sprite, whole(P.frame or 1))
local v = flat:getPixel(whole(P.x), whole(P.y))
return {
  color = string.format("#%02X%02X%02X%02X",
    v % 256, (v // 256) % 256, (v // 65536) % 256, (v // 16777216) % 256),
}
`

// luaReadCelBounds reports the bounds of one cel, which is how a stamp is
// checked for growing the cel further than it has to.
const luaReadCelBounds = `
local sprite = openSprite(P.path)
local layer = drawTarget(sprite, P.layer)
local cel = layer:cel(requireFrame(sprite, P.frame))
if cel == nil then error("no cel on that layer and frame", 0) end
return { x = cel.bounds.x, y = cel.bounds.y, width = cel.bounds.width, height = cel.bounds.height }
`

func runScript(t *testing.T, body string, params map[string]any, out any) {
	t.Helper()
	runner, err := aseprite.NewRunner()
	if err != nil {
		t.Fatalf("locating Aseprite: %v", err)
	}
	payload, err := runner.Run(context.Background(), body, params)
	if err != nil {
		t.Fatalf("running script: %v", err)
	}
	if err := json.Unmarshal(payload, out); err != nil {
		t.Fatalf("decoding result: %v", err)
	}
}

func readPixel(t *testing.T, path string, x, y int) string {
	t.Helper()
	var out struct {
		Color string `json:"color"`
	}
	runScript(t, luaReadPixel, map[string]any{"path": path, "x": x, "y": y}, &out)
	return out.Color
}

func TestStampSpritesComposites(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()

	// A red square with a transparent hole punched through the middle.
	badge := filepath.Join(dir, "badge.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": badge, "width": 4, "height": 4})
	call(t, ctx, s, "draw_shapes", map[string]any{
		"path": badge,
		"shapes": []any{map[string]any{
			"kind": "filled_rectangle", "from": map[string]any{"x": 0, "y": 0},
			"to": map[string]any{"x": 3, "y": 3}, "color": "#FF0000",
		}},
	})
	call(t, ctx, s, "clear_area", map[string]any{
		"path": badge, "rect": map[string]any{"x": 1, "y": 1, "width": 2, "height": 2},
	})

	board := filepath.Join(dir, "board.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": board, "width": 12, "height": 12})
	call(t, ctx, s, "draw_shapes", map[string]any{
		"path": board,
		"shapes": []any{map[string]any{
			"kind": "filled_rectangle", "from": map[string]any{"x": 0, "y": 0},
			"to": map[string]any{"x": 11, "y": 11}, "color": "#0000FF",
		}},
	})

	out := decode(t, call(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": board,
		"stamps": []any{
			map[string]any{"source": badge, "x": 2, "y": 2},
		},
	}))
	stamps, _ := out["stamps"].([]any)
	if len(stamps) != 1 {
		t.Fatalf("stamps = %v, want 1 entry", out["stamps"])
	}
	if first, _ := stamps[0].(map[string]any); first["clipped"] != false {
		t.Errorf("clipped = %v, want false", first["clipped"])
	}

	if got := readPixel(t, board, 2, 2); got != "#FF0000FF" {
		t.Errorf("corner of the stamp = %s, want #FF0000FF", got)
	}
	// The hole must let the destination show through rather than punching it out.
	if got := readPixel(t, board, 3, 3); got != "#0000FFFF" {
		t.Errorf("hole in the stamp = %s, want the blue background #0000FFFF", got)
	}
	if got := readPixel(t, board, 8, 8); got != "#0000FFFF" {
		t.Errorf("untouched area = %s, want #0000FFFF", got)
	}
}

// Stamps are applied in order, so a later one covers an earlier one.
func TestStampSpritesOrderAndBatch(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()

	red := filepath.Join(dir, "red.aseprite")
	green := filepath.Join(dir, "green.aseprite")
	for _, spec := range []struct{ path, color string }{{red, "#FF0000"}, {green, "#00FF00"}} {
		call(t, ctx, s, "create_sprite", map[string]any{"path": spec.path, "width": 4, "height": 4})
		call(t, ctx, s, "draw_shapes", map[string]any{
			"path": spec.path,
			"shapes": []any{map[string]any{
				"kind": "filled_rectangle", "from": map[string]any{"x": 0, "y": 0},
				"to": map[string]any{"x": 3, "y": 3}, "color": spec.color,
			}},
		})
	}

	scene := filepath.Join(dir, "scene.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": scene, "width": 16, "height": 8})

	out := decode(t, call(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": scene,
		"stamps": []any{
			map[string]any{"source": red, "x": 0, "y": 0},
			map[string]any{"source": green, "x": 2, "y": 0},
			map[string]any{"source": red, "x": 10, "y": 2},
		},
	}))
	if stamps, _ := out["stamps"].([]any); len(stamps) != 3 {
		t.Fatalf("stamps = %v, want 3 entries", out["stamps"])
	}

	if got := readPixel(t, scene, 0, 0); got != "#FF0000FF" {
		t.Errorf("(0,0) = %s, want the first stamp #FF0000FF", got)
	}
	if got := readPixel(t, scene, 3, 0); got != "#00FF00FF" {
		t.Errorf("(3,0) = %s, want the later stamp on top #00FF00FF", got)
	}
	if got := readPixel(t, scene, 11, 3); got != "#FF0000FF" {
		t.Errorf("(11,3) = %s, want the repeated source #FF0000FF", got)
	}

	// The cel should cover only what was stamped, not the whole sprite.
	var bounds struct {
		X, Y, Width, Height int
	}
	runScript(t, luaReadCelBounds, map[string]any{"path": scene, "frame": 1}, &bounds)
	if bounds.Width == 16 && bounds.Height == 8 {
		t.Errorf("cel grew to the whole sprite (%dx%d); it should cover only the stamps",
			bounds.Width, bounds.Height)
	}
	if bounds.X != 0 || bounds.Y != 0 || bounds.Width != 14 || bounds.Height != 6 {
		t.Errorf("cel bounds = %+v, want the union of the stamps (0,0,14,6)", bounds)
	}
}

func TestStampSpritesClipsAndRejects(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()

	tile := filepath.Join(dir, "tile.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": tile, "width": 6, "height": 6})
	call(t, ctx, s, "draw_shapes", map[string]any{
		"path": tile,
		"shapes": []any{map[string]any{
			"kind": "filled_rectangle", "from": map[string]any{"x": 0, "y": 0},
			"to": map[string]any{"x": 5, "y": 5}, "color": "#FFAA00",
		}},
	})

	board := filepath.Join(dir, "board.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": board, "width": 10, "height": 10})

	// Hanging over the left edge is fine, and reported.
	out := decode(t, call(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": board,
		"stamps":      []any{map[string]any{"source": tile, "x": -3, "y": 0}},
	}))
	stamps, _ := out["stamps"].([]any)
	first, _ := stamps[0].(map[string]any)
	if first["clipped"] != true {
		t.Errorf("clipped = %v, want true for a stamp hanging off the edge", first["clipped"])
	}
	if got := readPixel(t, board, 0, 0); got != "#FFAA00FF" {
		t.Errorf("(0,0) = %s, want the clipped stamp #FFAA00FF", got)
	}

	if msg := callErr(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": board,
		"stamps":      []any{map[string]any{"source": tile, "x": 40, "y": 40}},
	}); !strings.Contains(msg, "outside") {
		t.Errorf("expected an outside-the-destination error, got %q", msg)
	}

	// A path nested inside stamps[] has to be checked like a top-level one.
	if msg := callErr(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": board,
		"stamps":      []any{map[string]any{"source": "tile.aseprite", "x": 0, "y": 0}},
	}); !strings.Contains(msg, "absolute") {
		t.Errorf("expected a relative-path rejection, got %q", msg)
	}

	if msg := callErr(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": board,
		"stamps":      []any{map[string]any{"source": tile, "x": 0, "y": 0, "sourceFrame": 9}},
	}); !strings.Contains(msg, "sourceFrame") {
		t.Errorf("expected a sourceFrame range error, got %q", msg)
	}
}

// Drawing a sprite straight into an indexed image picks the wrong palette
// entries, so the stamp goes through an rgb render first.
func TestStampSpritesIntoIndexedDestination(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()

	mark := filepath.Join(dir, "mark.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": mark, "width": 4, "height": 4})
	call(t, ctx, s, "draw_shapes", map[string]any{
		"path": mark,
		"shapes": []any{map[string]any{
			"kind": "filled_rectangle", "from": map[string]any{"x": 0, "y": 0},
			"to": map[string]any{"x": 3, "y": 3}, "color": "#FF0000",
		}},
	})

	board := filepath.Join(dir, "indexed.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{
		"path": board, "width": 8, "height": 8, "colorMode": "indexed",
	})
	call(t, ctx, s, "set_palette", map[string]any{
		"path": board, "colors": []any{"#00000000", "#0000FFFF", "#FF0000FF", "#00FF00FF"},
	})

	call(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": board,
		"stamps":      []any{map[string]any{"source": mark, "x": 2, "y": 2}},
	})

	if got := readPixel(t, board, 3, 3); got != "#FF0000FF" {
		t.Errorf("stamped pixel = %s, want the palette's red #FF0000FF", got)
	}
}

// A source in a different color mode has to arrive with its real colors.
func TestStampSpritesFromIndexedSource(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()

	mark := filepath.Join(dir, "mark.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{
		"path": mark, "width": 4, "height": 4, "colorMode": "indexed",
	})
	call(t, ctx, s, "set_palette", map[string]any{
		"path": mark, "colors": []any{"#00000000", "#00FF00FF"},
	})
	call(t, ctx, s, "draw_shapes", map[string]any{
		"path": mark,
		"shapes": []any{map[string]any{
			"kind": "filled_rectangle", "from": map[string]any{"x": 0, "y": 0},
			"to": map[string]any{"x": 3, "y": 3}, "color": "1",
		}},
	})

	board := filepath.Join(dir, "board.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": board, "width": 8, "height": 8})
	call(t, ctx, s, "stamp_sprites", map[string]any{
		"destination": board,
		"stamps":      []any{map[string]any{"source": mark, "x": 2, "y": 2}},
	})

	if got := readPixel(t, board, 3, 3); got != "#00FF00FF" {
		t.Errorf("stamped pixel = %s, want the source palette's green #00FF00FF", got)
	}
}
