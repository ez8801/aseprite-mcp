package aseprite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// pathFields are the argument names that carry a filesystem path. Every one of
// them is checked and cleaned before a script sees it.
var pathFields = []string{"path", "source", "destination", "fromFile", "dataFile"}

// run builds a tool handler that hands the arguments to a Lua script.
func run[In any](r *Runner, body string) mcp.ToolHandlerFor[In, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args In) (*mcp.CallToolResult, any, error) {
		params, err := normalizePaths(args)
		if err != nil {
			return failure(err), nil, nil
		}
		return respond(r.Run(ctx, body, params))
	}
}

// normalizePaths turns the arguments into the table a script receives, with
// every path field validated.
func normalizePaths(args any) (map[string]any, error) {
	raw, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("encoding arguments: %w", err)
	}
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("decoding arguments: %w", err)
	}
	if err := cleanPaths(params); err != nil {
		return nil, err
	}
	return params, nil
}

// cleanPaths rewrites every path field in an argument tree. It walks nested
// values as well, so a path inside an array such as stamps[] is checked the
// same way as a top-level one.
func cleanPaths(node any) error {
	switch value := node.(type) {
	case map[string]any:
		for _, field := range pathFields {
			path, ok := value[field].(string)
			if !ok || path == "" {
				continue
			}
			clean, err := absolute(path, field)
			if err != nil {
				return err
			}
			value[field] = clean
		}
		for _, child := range value {
			if err := cleanPaths(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range value {
			if err := cleanPaths(child); err != nil {
				return err
			}
		}
	}
	return nil
}

// register wires every tool and prompt this server exposes onto s.
func Register(s *mcp.Server, r *Runner) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "aseprite_health",
		Description: "Report the Aseprite executable the server is driving and its version. " +
			"Use this first when a call fails, to confirm Aseprite is reachable.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, run[healthArgs](r, luaHealth))

	mcp.AddTool(s, &mcp.Tool{
		Name: "create_sprite",
		Description: "Create a new empty sprite and write it to disk. " +
			"The file format follows the extension of path (.aseprite, .ase, .png, ...). " +
			"Fails if the file already exists unless overwrite is true.",
	}, run[createSpriteArgs](r, luaCreateSprite))

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_sprite_info",
		Description: "Read a sprite file and report its size, color mode, layers, frames, " +
			"animation tags and palette. Lists at most the first 32 palette entries; " +
			"use get_palette for the whole palette. Does not modify the file.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, run[spriteInfoArgs](r, luaSpriteInfo))

	mcp.AddTool(s, &mcp.Tool{
		Name: "save_sprite_as",
		Description: "Save a copy of a sprite in another location or format, chosen by the " +
			"destination extension (.aseprite, .png, .gif, ...). The source file is left " +
			"untouched. Exporting a multi-frame sprite to a single-image format such as " +
			".png makes Aseprite write one numbered file per frame (anim1.png, anim2.png, " +
			"...) rather than the requested name; the result reports what was actually " +
			"written in files and sets splitIntoSequence. Use .gif for a single animated " +
			"file. Fails if any destination file exists unless overwrite is true.",
	}, run[saveSpriteAsArgs](r, luaSaveSpriteAs))

	registerEdit(s, r)
	registerPalette(s, r)
	registerPrompts(s)
}

type healthArgs struct{}

type createSpriteArgs struct {
	Path      string `json:"path" jsonschema:"Absolute path of the file to create. The parent directory must already exist."`
	Width     int    `json:"width" jsonschema:"Sprite width in pixels."`
	Height    int    `json:"height" jsonschema:"Sprite height in pixels."`
	ColorMode string `json:"colorMode,omitempty" jsonschema:"Color mode: rgb (default), gray or indexed."`
	Overwrite bool   `json:"overwrite,omitempty" jsonschema:"Replace the file if it already exists. Defaults to false."`
}

type spriteInfoArgs struct {
	Path string `json:"path" jsonschema:"Absolute path of the sprite file to inspect."`
}

type saveSpriteAsArgs struct {
	Source      string `json:"source" jsonschema:"Absolute path of the sprite to read."`
	Destination string `json:"destination" jsonschema:"Absolute path to write the copy to. The parent directory must already exist."`
	Overwrite   bool   `json:"overwrite,omitempty" jsonschema:"Replace the destination if it already exists. Defaults to false."`
}

// absolute rejects relative paths, which would otherwise resolve against the
// Aseprite process working directory rather than anything the caller expects.
func absolute(path, field string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be an absolute path, got %q", field, path)
	}
	return filepath.Clean(path), nil
}

// failure reports a problem the model can act on, rather than a transport error.
func failure(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}

// respond turns a script payload into a tool result, reporting script failures
// as tool errors rather than transport errors so the model can react to them.
func respond(payload json.RawMessage, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return failure(err), nil, nil
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, payload, "", "  "); err != nil {
		pretty.Write(payload)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: pretty.String()}},
	}, nil, nil
}
