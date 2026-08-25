package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPalette adds the palette, resize and spritesheet tools.
func registerPalette(s *mcp.Server, r *Runner) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_palette",
		Description: "Read the whole palette of a sprite. Unlike get_sprite_info this is not " +
			"truncated. Does not modify the file.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, run[getPaletteArgs](r, luaGetPalette))

	mcp.AddTool(s, &mcp.Tool{
		Name: "set_palette",
		Description: "Replace the palette of a sprite, either with an explicit list of colors " +
			"or with one loaded from a palette file (.gpl, .pal, .aseprite or an image). " +
			inPlaceDoc,
	}, run[setPaletteArgs](r, luaSetPalette))

	mcp.AddTool(s, &mcp.Tool{
		Name: "save_palette",
		Description: "Write the palette of a sprite to its own file, in the format chosen by " +
			"the destination extension (.gpl, .pal, .png, ...). The sprite is not modified. " +
			"Fails if the destination exists unless overwrite is true.",
	}, run[savePaletteArgs](r, luaSavePalette))

	mcp.AddTool(s, &mcp.Tool{
		Name: "resize_sprite",
		Description: "Resize a sprite, either to an explicit size or by a scale factor. " +
			"Scaling up uses nearest neighbour, so pixel art stays crisp. " + inPlaceDoc,
	}, run[resizeSpriteArgs](r, luaResizeSprite))

	mcp.AddTool(s, &mcp.Tool{
		Name: "export_spritesheet",
		Description: "Export every frame into one sheet image, optionally with a JSON data " +
			"file describing the frame rectangles. The source sprite is not modified. " +
			"Fails if a destination file exists unless overwrite is true.",
	}, run[exportSpritesheetArgs](r, luaExportSpritesheet))
}

type getPaletteArgs struct {
	Path string `json:"path" jsonschema:"Absolute path of the sprite to read."`
}

type setPaletteArgs struct {
	Path     string   `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Colors   []string `json:"colors,omitempty" jsonschema:"Palette entries as #RRGGBB or #RRGGBBAA, in order. Ignored when fromFile is given."`
	FromFile string   `json:"fromFile,omitempty" jsonschema:"Absolute path of a palette file to load instead of listing colors."`
}

type savePaletteArgs struct {
	Path        string `json:"path" jsonschema:"Absolute path of the sprite to read the palette from."`
	Destination string `json:"destination" jsonschema:"Absolute path to write the palette to. The parent directory must already exist."`
	Overwrite   bool   `json:"overwrite,omitempty" jsonschema:"Replace the destination if it already exists. Defaults to false."`
}

type resizeSpriteArgs struct {
	Path   string   `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Width  *int     `json:"width,omitempty" jsonschema:"New width in pixels. Ignored when scale is given."`
	Height *int     `json:"height,omitempty" jsonschema:"New height in pixels. Ignored when scale is given."`
	Scale  *float64 `json:"scale,omitempty" jsonschema:"Multiply both dimensions by this factor, for example 2 to double the size."`
}

type exportSpritesheetArgs struct {
	Path        string `json:"path" jsonschema:"Absolute path of the sprite to export."`
	Destination string `json:"destination" jsonschema:"Absolute path of the sheet image to write. The parent directory must already exist."`
	DataFile    string `json:"dataFile,omitempty" jsonschema:"Absolute path of a JSON file describing the frames. Omit to skip it."`
	SheetType   string `json:"sheetType,omitempty" jsonschema:"Layout: packed (default), horizontal, vertical, rows or columns."`
	DataFormat  string `json:"dataFormat,omitempty" jsonschema:"JSON shape for dataFile: hash (default) or array."`
	Padding     *int   `json:"padding,omitempty" jsonschema:"Pixels of space between frames. Defaults to 0."`
	Trim        bool   `json:"trim,omitempty" jsonschema:"Trim transparent edges from each frame."`
	SplitLayers bool   `json:"splitLayers,omitempty" jsonschema:"Export each layer as its own set of frames."`
	SplitTags   bool   `json:"splitTags,omitempty" jsonschema:"Export each animation tag as its own set of frames."`
	Overwrite   bool   `json:"overwrite,omitempty" jsonschema:"Replace existing destination files. Defaults to false."`
}
