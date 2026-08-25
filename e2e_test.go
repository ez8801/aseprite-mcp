package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverBin is the freshly built server under test.
var serverBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "aseprite-mcp-test-")
	if err != nil {
		panic(err)
	}
	serverBin = filepath.Join(dir, "server")
	if runtime.GOOS == "windows" {
		serverBin += ".exe"
	}
	build := exec.Command("go", "build", "-o", serverBin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		panic("building server: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// connect starts the server and returns a connected client session. Tests are
// skipped when Aseprite is not installed, since the server exits at startup.
func connect(t *testing.T) (context.Context, *mcp.ClientSession) {
	t.Helper()
	if _, err := findAseprite(); err != nil {
		t.Skip("Aseprite not installed: ", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(serverBin)}, nil)
	if err != nil {
		t.Fatalf("connecting to server: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return ctx, session
}

// call invokes a tool and returns its text payload, failing on tool errors.
func call(t *testing.T, ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: transport error: %v", name, err)
	}
	text := textOf(t, res)
	if res.IsError {
		t.Fatalf("%s: tool error: %s", name, text)
	}
	return text
}

// callErr invokes a tool that is expected to fail and returns its message.
func callErr(t *testing.T, ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err.Error()
	}
	text := textOf(t, res)
	if !res.IsError {
		t.Fatalf("%s: expected an error, got: %s", name, text)
	}
	return text
}

func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func decode(t *testing.T, payload string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		t.Fatalf("decoding %q: %v", payload, err)
	}
	return m
}

func TestToolsAreListed(t *testing.T) {
	ctx, session := connect(t)
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	want := []string{
		"aseprite_health", "create_sprite", "get_sprite_info", "save_sprite_as",
		"draw_pixels", "draw_shapes", "fill_area", "clear_area",
		"stamp_sprites",
		"add_layer", "update_layer", "delete_layer",
		"add_frames", "delete_frames", "set_frame_durations",
		"set_tag", "delete_tag",
		"get_palette", "set_palette", "save_palette",
		"resize_sprite", "export_spritesheet",
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
	if len(res.Tools) != len(want) {
		t.Errorf("tools/list has %d tools, want %d", len(res.Tools), len(want))
	}
}

func TestHealthReportsVersion(t *testing.T) {
	ctx, session := connect(t)
	info := decode(t, call(t, ctx, session, "aseprite_health", map[string]any{}))
	if v, _ := info["version"].(string); v == "" {
		t.Errorf("expected a version, got %v", info["version"])
	}
}

func TestCreateInspectAndConvert(t *testing.T) {
	ctx, session := connect(t)
	dir := t.TempDir()
	sprite := filepath.Join(dir, "hero.aseprite")

	created := decode(t, call(t, ctx, session, "create_sprite", map[string]any{
		"path": sprite, "width": 16, "height": 24, "colorMode": "indexed",
	}))
	if created["width"] != 16.0 || created["height"] != 24.0 {
		t.Errorf("unexpected size in %v", created)
	}
	if _, err := os.Stat(sprite); err != nil {
		t.Fatalf("sprite was not written: %v", err)
	}

	info := decode(t, call(t, ctx, session, "get_sprite_info", map[string]any{"path": sprite}))
	if info["colorMode"] != "indexed" {
		t.Errorf("colorMode = %v, want indexed", info["colorMode"])
	}
	if info["frameCount"] != 1.0 || info["layerCount"] != 1.0 {
		t.Errorf("unexpected frame/layer count in %v", info)
	}

	png := filepath.Join(dir, "hero.png")
	call(t, ctx, session, "save_sprite_as", map[string]any{"source": sprite, "destination": png})
	if _, err := os.Stat(png); err != nil {
		t.Fatalf("png was not written: %v", err)
	}
}

func TestCreateRefusesToClobber(t *testing.T) {
	ctx, session := connect(t)
	sprite := filepath.Join(t.TempDir(), "dup.aseprite")
	args := map[string]any{"path": sprite, "width": 8, "height": 8}

	call(t, ctx, session, "create_sprite", args)
	if msg := callErr(t, ctx, session, "create_sprite", args); !strings.Contains(msg, "already exists") {
		t.Errorf("expected an already-exists error, got %q", msg)
	}

	withOverwrite := map[string]any{"path": sprite, "width": 8, "height": 8, "overwrite": true}
	call(t, ctx, session, "create_sprite", withOverwrite)
}

func TestErrorsAreReported(t *testing.T) {
	ctx, session := connect(t)
	missing := filepath.Join(t.TempDir(), "nope.aseprite")

	if msg := callErr(t, ctx, session, "get_sprite_info", map[string]any{"path": missing}); !strings.Contains(msg, "not found") {
		t.Errorf("expected a not-found error, got %q", msg)
	}
	if msg := callErr(t, ctx, session, "get_sprite_info", map[string]any{"path": "relative/x.aseprite"}); !strings.Contains(msg, "absolute") {
		t.Errorf("expected an absolute-path error, got %q", msg)
	}
	if msg := callErr(t, ctx, session, "create_sprite", map[string]any{
		"path": filepath.Join(t.TempDir(), "x.aseprite"), "width": 8, "height": 8, "colorMode": "cmyk",
	}); !strings.Contains(msg, "colorMode") {
		t.Errorf("expected a colorMode error, got %q", msg)
	}
}

// luaMakeAnimated builds a multi-frame sprite. The server exposes no tool for
// this yet, so the fixture is written through the runner directly.
const luaMakeAnimated = `
local sprite = Sprite(8, 8, ColorMode.RGB)
for _ = 2, P.frames do sprite:newFrame() end
sprite:saveAs(P.path)
return { frames = #sprite.frames }
`

func makeAnimated(t *testing.T, path string, frames int) {
	t.Helper()
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("locating Aseprite: %v", err)
	}
	if _, err := runner.Run(context.Background(), luaMakeAnimated, map[string]any{
		"path": path, "frames": frames,
	}); err != nil {
		t.Fatalf("building fixture: %v", err)
	}
}

// Aseprite cannot fit an animation into a single-image format, so it ignores
// the requested name and writes one numbered file per frame instead.
func TestMultiFrameExportSplitsIntoSequence(t *testing.T) {
	ctx, session := connect(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "anim.aseprite")
	makeAnimated(t, source, 3)

	destination := filepath.Join(dir, "anim.png")
	out := decode(t, call(t, ctx, session, "save_sprite_as", map[string]any{
		"source": source, "destination": destination,
	}))
	if out["splitIntoSequence"] != true {
		t.Errorf("splitIntoSequence = %v, want true", out["splitIntoSequence"])
	}
	files, _ := out["files"].([]any)
	if len(files) != 3 {
		t.Fatalf("files = %v, want 3 entries", out["files"])
	}
	for i, f := range files {
		name, _ := f.(string)
		want := filepath.Join(dir, fmt.Sprintf("anim%d.png", i+1))
		if name != want {
			t.Errorf("files[%d] = %q, want %q", i, name, want)
		}
		if _, err := os.Stat(want); err != nil {
			t.Errorf("%s was not written: %v", want, err)
		}
	}

	// The sequence names must count as existing files for the overwrite guard,
	// or a second export would silently replace them.
	msg := callErr(t, ctx, session, "save_sprite_as", map[string]any{
		"source": source, "destination": destination,
	})
	if !strings.Contains(msg, "anim1.png") || !strings.Contains(msg, "already exists") {
		t.Errorf("expected the guard to name anim1.png, got %q", msg)
	}
	call(t, ctx, session, "save_sprite_as", map[string]any{
		"source": source, "destination": destination, "overwrite": true,
	})
}

// A GIF holds every frame, so the requested name is what gets written.
func TestAnimationExportsToSingleGIF(t *testing.T) {
	ctx, session := connect(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "anim.aseprite")
	makeAnimated(t, source, 3)

	destination := filepath.Join(dir, "anim.gif")
	out := decode(t, call(t, ctx, session, "save_sprite_as", map[string]any{
		"source": source, "destination": destination,
	}))
	if out["splitIntoSequence"] != false {
		t.Errorf("splitIntoSequence = %v, want false", out["splitIntoSequence"])
	}
	if _, err := os.Stat(destination); err != nil {
		t.Errorf("gif was not written: %v", err)
	}
}
