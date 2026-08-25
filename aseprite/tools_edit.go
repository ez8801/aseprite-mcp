package aseprite

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// inPlaceDoc marks the tools that rewrite the sprite they are given.
const inPlaceDoc = "The sprite file is modified in place."

// registerEdit adds the tools that change pixels, layers, frames and tags.
func registerEdit(s *mcp.Server, r *Runner) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "draw_pixels",
		Description: "Set individual pixels on one layer and frame. " + inPlaceDoc +
			" Send every pixel of an edit in one call: each call reopens and rewrites the file.",
	}, run[drawPixelsArgs](r, luaDrawPixels))

	mcp.AddTool(s, &mcp.Tool{
		Name: "draw_shapes",
		Description: "Draw lines, rectangles and ellipses on one layer and frame, in the order " +
			"given. " + inPlaceDoc,
	}, run[drawShapesArgs](r, luaDrawShapes))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "fill_area",
		Description: "Flood fill from a point, like the paint bucket tool. " + inPlaceDoc,
	}, run[fillAreaArgs](r, luaFillArea))

	mcp.AddTool(s, &mcp.Tool{
		Name: "clear_area",
		Description: "Erase a rectangle on one layer and frame, or the whole cel when no rect " +
			"is given. " + inPlaceDoc,
	}, run[clearAreaArgs](r, luaClearArea))

	mcp.AddTool(s, &mcp.Tool{
		Name: "stamp_sprites",
		Description: "Copy whole sprites into another sprite at given positions, the way a " +
			"scene is assembled from separate character and prop files. Each stamp is " +
			"composited with transparency, in the order given, so later stamps sit in " +
			"front. Source files are not modified; the destination is. Stamps that fall " +
			"partly outside the destination are clipped and reported.",
	}, run[stampSpritesArgs](r, luaStampSprites))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_layer",
		Description: "Add an image layer or a layer group. " + inPlaceDoc,
	}, run[addLayerArgs](r, luaAddLayer))

	mcp.AddTool(s, &mcp.Tool{
		Name: "update_layer",
		Description: "Rename a layer or change its opacity, visibility or blend mode. Pass at " +
			"least one property to change. " + inPlaceDoc,
	}, run[updateLayerArgs](r, luaUpdateLayer))

	mcp.AddTool(s, &mcp.Tool{
		Name: "delete_layer",
		Description: "Delete a layer. Deleting a group also deletes the layers inside it. " +
			inPlaceDoc,
	}, run[deleteLayerArgs](r, luaDeleteLayer))

	mcp.AddTool(s, &mcp.Tool{
		Name: "add_frames",
		Description: "Append or insert animation frames. New frames copy the previous frame " +
			"unless empty is true. " + inPlaceDoc,
	}, run[addFramesArgs](r, luaAddFrames))

	mcp.AddTool(s, &mcp.Tool{
		Name: "delete_frames",
		Description: "Delete animation frames by number. A sprite must keep at least one " +
			"frame. " + inPlaceDoc,
	}, run[deleteFramesArgs](r, luaDeleteFrames))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_frame_durations",
		Description: "Set how long individual frames are shown. " + inPlaceDoc,
	}, run[setFrameDurationsArgs](r, luaSetFrameDurations))

	mcp.AddTool(s, &mcp.Tool{
		Name: "set_tag",
		Description: "Create an animation tag, or replace an existing tag with the same name. " +
			inPlaceDoc,
	}, run[setTagArgs](r, luaSetTag))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_tag",
		Description: "Delete an animation tag by name. " + inPlaceDoc,
	}, run[deleteTagArgs](r, luaDeleteTag))
}

type point struct {
	X int `json:"x" jsonschema:"Pixel column, 0 at the left edge."`
	Y int `json:"y" jsonschema:"Pixel row, 0 at the top edge."`
}

type rect struct {
	X      int `json:"x" jsonschema:"Left edge in pixels."`
	Y      int `json:"y" jsonschema:"Top edge in pixels."`
	Width  int `json:"width" jsonschema:"Width in pixels."`
	Height int `json:"height" jsonschema:"Height in pixels."`
}

type pixel struct {
	X     int    `json:"x" jsonschema:"Pixel column, 0 at the left edge."`
	Y     int    `json:"y" jsonschema:"Pixel row, 0 at the top edge."`
	Color string `json:"color" jsonschema:"Color as #RRGGBB or #RRGGBBAA, or a palette index such as 5 on an indexed sprite."`
}

type drawPixelsArgs struct {
	Path   string  `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Pixels []pixel `json:"pixels" jsonschema:"Pixels to set. Aseprite ignores coordinates outside the sprite."`
	Layer  string  `json:"layer,omitempty" jsonschema:"Layer name. Defaults to the first image layer. A group cannot hold pixels and is rejected."`
	Frame  *int    `json:"frame,omitempty" jsonschema:"Frame number, starting at 1. Defaults to 1."`
}

type shape struct {
	Kind      string `json:"kind" jsonschema:"One of line, rectangle, filled_rectangle, ellipse, filled_ellipse, contour."`
	From      point  `json:"from" jsonschema:"Start point, or one corner of a rectangle or ellipse."`
	To        point  `json:"to" jsonschema:"End point, or the opposite corner."`
	Color     string `json:"color" jsonschema:"Color as #RRGGBB or #RRGGBBAA, or a palette index such as 5 on an indexed sprite."`
	BrushSize *int   `json:"brushSize,omitempty" jsonschema:"Stroke thickness in pixels. Defaults to 1."`
}

type drawShapesArgs struct {
	Path   string  `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Shapes []shape `json:"shapes" jsonschema:"Shapes to draw, in order."`
	Layer  string  `json:"layer,omitempty" jsonschema:"Layer name. Defaults to the first image layer. A group cannot hold pixels and is rejected."`
	Frame  *int    `json:"frame,omitempty" jsonschema:"Frame number, starting at 1. Defaults to 1."`
}

type fillAreaArgs struct {
	Path       string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	X          int    `json:"x" jsonschema:"Column to start filling from."`
	Y          int    `json:"y" jsonschema:"Row to start filling from."`
	Color      string `json:"color" jsonschema:"Color as #RRGGBB or #RRGGBBAA, or a palette index such as 5 on an indexed sprite."`
	Layer      string `json:"layer,omitempty" jsonschema:"Layer name. Defaults to the first image layer. A group cannot hold pixels and is rejected."`
	Frame      *int   `json:"frame,omitempty" jsonschema:"Frame number, starting at 1. Defaults to 1."`
	Tolerance  *int   `json:"tolerance,omitempty" jsonschema:"How far a color may differ and still be filled. Defaults to 0."`
	Contiguous *bool  `json:"contiguous,omitempty" jsonschema:"Fill only the connected region. Defaults to true."`
}

type clearAreaArgs struct {
	Path  string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Rect  *rect  `json:"rect,omitempty" jsonschema:"Region to erase. Omit to erase the whole cel."`
	Layer string `json:"layer,omitempty" jsonschema:"Layer name. Defaults to the first image layer. A group cannot hold pixels and is rejected."`
	Frame *int   `json:"frame,omitempty" jsonschema:"Frame number, starting at 1. Defaults to 1."`
}

type stamp struct {
	Source      string `json:"source" jsonschema:"Absolute path of the sprite to copy in."`
	X           int    `json:"x" jsonschema:"Column in the destination where the source's left edge lands."`
	Y           int    `json:"y" jsonschema:"Row in the destination where the source's top edge lands."`
	SourceFrame *int   `json:"sourceFrame,omitempty" jsonschema:"Which frame of the source to copy, starting at 1. Defaults to 1."`
	Opacity     *int   `json:"opacity,omitempty" jsonschema:"How opaque the stamp is, from 0 to 255. Defaults to 255."`
	BlendMode   string `json:"blendMode,omitempty" jsonschema:"Blend mode such as normal (default), multiply, screen, overlay or addition."`
}

type stampSpritesArgs struct {
	Destination string  `json:"destination" jsonschema:"Absolute path of the sprite to draw into. It is modified in place."`
	Stamps      []stamp `json:"stamps" jsonschema:"Sprites to copy in, back to front."`
	Layer       string  `json:"layer,omitempty" jsonschema:"Destination layer name. Defaults to the first image layer. A group cannot hold pixels and is rejected."`
	Frame       *int    `json:"frame,omitempty" jsonschema:"Destination frame number, starting at 1. Defaults to 1."`
}

type addLayerArgs struct {
	Path    string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Name    string `json:"name" jsonschema:"Name for the new layer."`
	Group   bool   `json:"group,omitempty" jsonschema:"Create a layer group instead of an image layer."`
	Parent  string `json:"parent,omitempty" jsonschema:"Name of an existing group to nest the new layer inside."`
	Opacity *int   `json:"opacity,omitempty" jsonschema:"Layer opacity from 0 to 255."`
	Visible *bool  `json:"visible,omitempty" jsonschema:"Whether the layer is visible."`
}

type updateLayerArgs struct {
	Path      string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Name      string `json:"name" jsonschema:"Name of the layer to change."`
	NewName   string `json:"newName,omitempty" jsonschema:"Rename the layer to this."`
	Opacity   *int   `json:"opacity,omitempty" jsonschema:"Layer opacity from 0 to 255."`
	Visible   *bool  `json:"visible,omitempty" jsonschema:"Whether the layer is visible."`
	BlendMode string `json:"blendMode,omitempty" jsonschema:"Blend mode such as normal, multiply, screen, overlay or addition. A group has no blend mode."`
}

type deleteLayerArgs struct {
	Path string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Name string `json:"name" jsonschema:"Name of the layer to delete."`
}

type addFramesArgs struct {
	Path  string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Count *int   `json:"count,omitempty" jsonschema:"How many frames to add. Defaults to 1."`
	After *int   `json:"after,omitempty" jsonschema:"Insert after this frame number. Defaults to appending at the end."`
	Empty bool   `json:"empty,omitempty" jsonschema:"Add blank frames instead of copies of the previous frame."`
}

type deleteFramesArgs struct {
	Path   string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Frames []int  `json:"frames" jsonschema:"Frame numbers to delete, starting at 1."`
}

type frameDuration struct {
	Frame      int `json:"frame" jsonschema:"Frame number, starting at 1."`
	DurationMs int `json:"durationMs" jsonschema:"How long the frame is shown, in milliseconds."`
}

type setFrameDurationsArgs struct {
	Path      string          `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Durations []frameDuration `json:"durations" jsonschema:"Frames to retime."`
}

type setTagArgs struct {
	Path    string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Name    string `json:"name" jsonschema:"Tag name. An existing tag with this name is replaced."`
	From    int    `json:"from" jsonschema:"First frame of the tag, starting at 1."`
	To      int    `json:"to" jsonschema:"Last frame of the tag, starting at 1."`
	AniDir  string `json:"aniDir,omitempty" jsonschema:"Playback direction: forward, reverse, ping_pong or ping_pong_reverse."`
	Repeats *int   `json:"repeats,omitempty" jsonschema:"How many times the tag loops. 0 means forever."`
}

type deleteTagArgs struct {
	Path string `json:"path" jsonschema:"Absolute path of the sprite to edit."`
	Name string `json:"name" jsonschema:"Name of the tag to delete."`
}
