package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ez8801/aseprite-mcp/internal/aseprite"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newSprite creates a sprite through the server and returns its path.
func newSprite(t *testing.T, ctx context.Context, s *mcp.ClientSession, dir string, width, height int) string {
	t.Helper()
	path := filepath.Join(dir, "sprite.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{
		"path": path, "width": width, "height": height,
	})
	return path
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s was not written: %v", path, err)
	}
}

// Nested arguments reach Lua as userdata proxies rather than tables, so every
// tool that takes objects or arrays is exercised here in one workflow.
func TestEditWorkflow(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()
	path := newSprite(t, ctx, s, dir, 32, 32)

	call(t, ctx, s, "add_layer", map[string]any{"path": path, "name": "Character", "group": true})
	call(t, ctx, s, "add_layer", map[string]any{"path": path, "name": "Body", "parent": "Character"})

	call(t, ctx, s, "draw_shapes", map[string]any{
		"path": path, "layer": "Body",
		"shapes": []any{
			map[string]any{
				"kind":  "filled_rectangle",
				"from":  map[string]any{"x": 4, "y": 4},
				"to":    map[string]any{"x": 27, "y": 27},
				"color": "#E8B07A",
			},
			map[string]any{
				"kind": "line", "from": map[string]any{"x": 0, "y": 0},
				"to": map[string]any{"x": 31, "y": 31}, "color": "#000000", "brushSize": 2,
			},
		},
	})
	call(t, ctx, s, "draw_pixels", map[string]any{
		"path": path, "layer": "Body",
		"pixels": []any{
			map[string]any{"x": 10, "y": 10, "color": "#FF0000"},
			map[string]any{"x": 11, "y": 10, "color": "#FF0000"},
			map[string]any{"x": 12, "y": 10, "color": "#00FF00"},
		},
	})
	call(t, ctx, s, "fill_area", map[string]any{
		"path": path, "layer": "Body", "x": 0, "y": 31, "color": "#0000FF",
	})

	call(t, ctx, s, "add_frames", map[string]any{"path": path, "count": 3})
	call(t, ctx, s, "set_frame_durations", map[string]any{
		"path": path,
		"durations": []any{
			map[string]any{"frame": 1, "durationMs": 120},
			map[string]any{"frame": 4, "durationMs": 250},
		},
	})
	call(t, ctx, s, "set_tag", map[string]any{
		"path": path, "name": "walk", "from": 1, "to": 4, "aniDir": "ping_pong",
	})
	call(t, ctx, s, "update_layer", map[string]any{
		"path": path, "name": "Body", "blendMode": "multiply", "opacity": 200,
	})

	info := decode(t, call(t, ctx, s, "get_sprite_info", map[string]any{"path": path}))
	if info["frameCount"] != 4.0 {
		t.Errorf("frameCount = %v, want 4", info["frameCount"])
	}
	if info["tagCount"] != 1.0 {
		t.Errorf("tagCount = %v, want 1", info["tagCount"])
	}
	frames, _ := info["frames"].([]any)
	if len(frames) != 4 {
		t.Fatalf("frames = %v", info["frames"])
	}
	first, _ := frames[0].(map[string]any)
	if first["durationMs"] != 120.0 {
		t.Errorf("frame 1 durationMs = %v, want 120", first["durationMs"])
	}

	// Body lives inside the Character group, so it must be found by recursion.
	layers, _ := info["layers"].([]any)
	var body map[string]any
	for _, l := range layers {
		group, _ := l.(map[string]any)
		if group["isGroup"] != true {
			continue
		}
		children, _ := group["layers"].([]any)
		for _, c := range children {
			child, _ := c.(map[string]any)
			if child["name"] == "Body" {
				body = child
			}
		}
	}
	if body == nil {
		t.Fatalf("Body layer missing from %v", info["layers"])
	}
	if body["opacity"] != 200.0 {
		t.Errorf("Body opacity = %v, want 200", body["opacity"])
	}
	if body["celCount"] != 4.0 {
		t.Errorf("Body celCount = %v, want 4 (drawing should have created cels)", body["celCount"])
	}
}

// luaReadRow reports which pixels of one row are opaque, as a string of # and
// dots. The server exposes no pixel reader, so the runner is used directly.
const luaReadRow = `
local sprite = openSprite(P.path)
local flat = Image(sprite.width, sprite.height, sprite.colorMode)
flat:drawSprite(sprite, 1)
local row = ""
for x = 0, sprite.width - 1 do
  row = row .. ((flat:getPixel(x, whole(P.y)) ~= 0) and "#" or ".")
end
return { row = row }
`

func readRow(t *testing.T, path string, y int) string {
	t.Helper()
	runner, err := aseprite.NewRunner()
	if err != nil {
		t.Fatalf("locating Aseprite: %v", err)
	}
	payload, err := runner.Run(context.Background(), luaReadRow, map[string]any{"path": path, "y": y})
	if err != nil {
		t.Fatalf("reading row %d: %v", y, err)
	}
	var out struct {
		Row string `json:"row"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("decoding row: %v", err)
	}
	return out.Row
}

// Handing several points to one useTool call makes Aseprite draw a stroke that
// connects them, so draw_pixels must place each pixel on its own.
func TestDrawPixelsPlacesIsolatedPixels(t *testing.T) {
	ctx, s := connect(t)
	path := newSprite(t, ctx, s, t.TempDir(), 12, 4)

	call(t, ctx, s, "draw_pixels", map[string]any{
		"path": path,
		"pixels": []any{
			map[string]any{"x": 1, "y": 1, "color": "#FF0000"},
			map[string]any{"x": 10, "y": 1, "color": "#FF0000"},
		},
	})

	if got, want := readRow(t, path, 1), ".#........#."; got != want {
		t.Errorf("row 1 = %q, want %q (same-colored pixels must not be joined)", got, want)
	}
}

// A group holds no pixels, and Aseprite silently ignores drawing on one.
func TestDrawingRejectsGroupLayer(t *testing.T) {
	ctx, s := connect(t)
	path := newSprite(t, ctx, s, t.TempDir(), 16, 16)
	call(t, ctx, s, "add_layer", map[string]any{"path": path, "name": "Group", "group": true})

	msg := callErr(t, ctx, s, "draw_pixels", map[string]any{
		"path": path, "layer": "Group",
		"pixels": []any{map[string]any{"x": 1, "y": 1, "color": "#FF0000"}},
	})
	if !strings.Contains(msg, "group") {
		t.Errorf("expected a group error, got %q", msg)
	}
}

func TestClearArea(t *testing.T) {
	ctx, s := connect(t)
	path := newSprite(t, ctx, s, t.TempDir(), 16, 16)
	call(t, ctx, s, "draw_shapes", map[string]any{
		"path": path,
		"shapes": []any{map[string]any{
			"kind": "filled_rectangle", "from": map[string]any{"x": 0, "y": 0},
			"to": map[string]any{"x": 15, "y": 15}, "color": "#FF0000",
		}},
	})

	out := decode(t, call(t, ctx, s, "clear_area", map[string]any{
		"path": path, "rect": map[string]any{"x": 0, "y": 0, "width": 4, "height": 4},
	}))
	if out["cleared"] != "rect" {
		t.Errorf("cleared = %v, want rect", out["cleared"])
	}

	out = decode(t, call(t, ctx, s, "clear_area", map[string]any{"path": path}))
	if out["cleared"] != "cel" {
		t.Errorf("cleared = %v, want cel", out["cleared"])
	}
}

// A tag cannot be moved in place, so setting an existing name replaces it.
func TestSetTagReplacesByName(t *testing.T) {
	ctx, s := connect(t)
	path := newSprite(t, ctx, s, t.TempDir(), 8, 8)
	call(t, ctx, s, "add_frames", map[string]any{"path": path, "count": 3})

	out := decode(t, call(t, ctx, s, "set_tag", map[string]any{
		"path": path, "name": "loop", "from": 1, "to": 4,
	}))
	if out["replaced"] != false {
		t.Errorf("first set_tag replaced = %v, want false", out["replaced"])
	}
	out = decode(t, call(t, ctx, s, "set_tag", map[string]any{
		"path": path, "name": "loop", "from": 2, "to": 3,
	}))
	if out["replaced"] != true {
		t.Errorf("second set_tag replaced = %v, want true", out["replaced"])
	}

	info := decode(t, call(t, ctx, s, "get_sprite_info", map[string]any{"path": path}))
	if info["tagCount"] != 1.0 {
		t.Errorf("tagCount = %v, want 1", info["tagCount"])
	}
}

func TestPaletteRoundTrip(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()
	path := newSprite(t, ctx, s, dir, 8, 8)
	colors := []any{"#00000000", "#112233FF", "#445566FF", "#778899FF"}

	call(t, ctx, s, "set_palette", map[string]any{"path": path, "colors": colors})
	out := decode(t, call(t, ctx, s, "get_palette", map[string]any{"path": path}))
	if out["paletteSize"] != 4.0 {
		t.Fatalf("paletteSize = %v, want 4", out["paletteSize"])
	}
	got, _ := out["colors"].([]any)
	if len(got) != 4 || got[1] != "#112233FF" {
		t.Errorf("colors = %v", out["colors"])
	}

	gpl := filepath.Join(dir, "palette.gpl")
	call(t, ctx, s, "save_palette", map[string]any{"path": path, "destination": gpl})

	other := filepath.Join(dir, "other.aseprite")
	call(t, ctx, s, "create_sprite", map[string]any{"path": other, "width": 8, "height": 8})
	call(t, ctx, s, "set_palette", map[string]any{"path": other, "fromFile": gpl})
	out = decode(t, call(t, ctx, s, "get_palette", map[string]any{"path": other}))
	if out["paletteSize"] != 4.0 {
		t.Errorf("loaded paletteSize = %v, want 4", out["paletteSize"])
	}
}

func TestResizeAndSpritesheet(t *testing.T) {
	ctx, s := connect(t)
	dir := t.TempDir()
	path := newSprite(t, ctx, s, dir, 16, 16)
	call(t, ctx, s, "add_frames", map[string]any{"path": path, "count": 2})

	out := decode(t, call(t, ctx, s, "resize_sprite", map[string]any{"path": path, "scale": 2}))
	if out["width"] != 32.0 || out["height"] != 32.0 {
		t.Errorf("resized to %v x %v, want 32 x 32", out["width"], out["height"])
	}

	sheet := filepath.Join(dir, "sheet.png")
	data := filepath.Join(dir, "sheet.json")
	call(t, ctx, s, "export_spritesheet", map[string]any{
		"path": path, "destination": sheet, "dataFile": data,
		"sheetType": "horizontal", "padding": 1,
	})
	mustExist(t, sheet)
	mustExist(t, data)

	// An existing sheet must not be replaced silently.
	if msg := callErr(t, ctx, s, "export_spritesheet", map[string]any{
		"path": path, "destination": sheet,
	}); !strings.Contains(msg, "already exists") {
		t.Errorf("expected an already-exists error, got %q", msg)
	}
}

func TestStructureGuards(t *testing.T) {
	ctx, s := connect(t)
	path := newSprite(t, ctx, s, t.TempDir(), 8, 8)

	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"delete every frame", "delete_frames",
			map[string]any{"path": path, "frames": []any{1}}, "at least one"},
		{"tag range backwards", "set_tag",
			map[string]any{"path": path, "name": "x", "from": 1, "to": 1, "aniDir": "sideways"}, "aniDir"},
		{"unknown layer", "update_layer",
			map[string]any{"path": path, "name": "ghost", "newName": "x"}, "layer not found"},
		{"empty update", "update_layer",
			map[string]any{"path": path, "name": "Layer 1"}, "nothing to update"},
		{"palette without input", "set_palette",
			map[string]any{"path": path}, "colors or fromFile"},
		{"zero scale", "resize_sprite",
			map[string]any{"path": path, "scale": 0}, "positive"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if msg := callErr(t, ctx, s, c.tool, c.args); !strings.Contains(msg, c.want) {
				t.Errorf("expected %q in the error, got %q", c.want, msg)
			}
		})
	}
}
