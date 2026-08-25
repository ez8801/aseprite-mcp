package aseprite

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// defaultAnimations is the shape most character sheets start from.
const defaultAnimations = "idle:6,attack:8"

const defaultSize = "64x64"

// animationNotes carry what actually reads well for a given state, so the
// prompt says something concrete instead of "animate it".
var animationNotes = map[string]string{
	"idle": "A slow breath. Move the body mass one pixel up and back down over the " +
		"cycle, let a cloak hem or feather tips trail a frame behind, and blink or " +
		"flicker a light source once. Nothing should travel far.",
	"attack": "Three beats: wind up away from the target for the first quarter, snap " +
		"through it in one or two frames, then settle back. Push the anticipation " +
		"further than feels right and keep the strike frames short.",
	"cast": "Gather, release, recover. Build the effect over the first half with the " +
		"body leaning back, flash brightest on release, then let the glow fade while " +
		"the pose returns.",
	"walk": "Contact, down, passing, up, then the mirror of all four. Bob the body one " +
		"pixel and swing whatever hangs off it against the legs.",
	"hurt": "A hard snap backwards on frame one, held a beat, then a loose recovery. " +
		"A one-frame lighter-colored flash sells the impact.",
	"death": "Collapse in stages rather than smoothly: buckle, fall, settle. The last " +
		"frame should read clearly as a silhouette on the ground.",
}

// loopDirection picks the playback a state normally wants.
func loopDirection(name string) string {
	if name == "idle" {
		return "ping_pong"
	}
	return "forward"
}

// registerPrompts adds the reusable workflows this server offers.
func registerPrompts(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:  "animated_character",
		Title: "Animated character sprite",
		Description: "Walk through drawing a new character sprite and its animation " +
			"states with this server's tools, from palette and base pose to tagged " +
			"frames and an exported sheet.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "name",
				Description: "Character slug used for the filenames, for example owl_wizard.",
				Required:    true,
			},
			{
				Name: "description",
				Description: "What the character looks like. Describe the silhouette, " +
					"materials, colors and any prop it carries.",
				Required: true,
			},
			{
				Name:        "outputDir",
				Description: "Absolute directory to write the sprite and its exports into.",
			},
			{
				Name: "animations",
				Description: "Comma separated state:frames pairs, for example " +
					defaultAnimations + ". This is the default.",
			},
			{
				Name:        "size",
				Description: "Canvas size as WIDTHxHEIGHT. Defaults to " + defaultSize + ".",
			},
		},
	}, animatedCharacter)
}

func animatedCharacter(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	name := strings.TrimSpace(args["name"])
	description := strings.TrimSpace(args["description"])
	if name == "" || description == "" {
		return nil, fmt.Errorf("both name and description are required")
	}
	size := valueOr(args["size"], defaultSize)
	states, err := parseAnimations(valueOr(args["animations"], defaultAnimations))
	if err != nil {
		return nil, err
	}

	dir := strings.TrimSpace(args["outputDir"])
	where := fmt.Sprintf("Write everything into %s.", dir)
	if dir == "" {
		where = "No output directory was given. Pick one inside the current project, " +
			"say art/" + name + ", and tell me the absolute path you settled on before " +
			"you start writing files."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Draw %s as an animated pixel-art sprite using the Aseprite tools on this server.\n\n", name)
	fmt.Fprintf(&b, "Look: %s\n\n", description)
	fmt.Fprintf(&b, "Canvas: %s, rgb color mode.\n%s\nThe sprite file is %s.aseprite.\n\n", size, where, name)

	b.WriteString("Work in this order.\n\n")

	b.WriteString("1. Plan before drawing anything.\n" +
		"   Pull a palette of 8 to 16 colors out of the description: a dark outline, two " +
		"or three steps per material, and one accent that only the focal point gets. " +
		"Say what the silhouette is in words, then say which parts move in each " +
		"animation. Parts that move independently want their own layer.\n\n")

	fmt.Fprintf(&b, "2. Build the base pose on frame 1.\n"+
		"   create_sprite at %s, then add_layer for each moving part. Block the big "+
		"masses with draw_shapes: a stack of one-pixel-tall filled_rectangle rows is how "+
		"you fill an arbitrary silhouette, and filled_ellipse handles round forms. Put "+
		"the outline down first as a silhouette one pixel larger, then fill inside it. "+
		"Use draw_pixels for eyes, highlights and other single pixels. Send a whole "+
		"pass in one call: every call reopens and rewrites the file.\n\n", size)

	b.WriteString("3. Look at the base pose before animating it.\n" +
		"   Copy the sprite with save_sprite_as, resize_sprite the copy by 6 or 8, " +
		"save_sprite_as that to .png, then actually read the image and judge it. Faces, " +
		"eyes and silhouettes rarely land on the first pass. Fix what reads wrong now, " +
		"while there is only one frame to fix, and show me the preview.\n\n")

	b.WriteString("4. Save the base pose aside.\n" +
		"   The editing tools rewrite in place and there is no undo, so keep a copy as " +
		name + "_base.aseprite before you start changing frames.\n\n")

	b.WriteString("5. Add the frames, one animation at a time.\n" +
		"   add_frames with empty left off copies the previous frame, so each new frame " +
		"starts from the last pose and you only draw the difference. Keep the frame " +
		"numbers straight: they are shared across the whole timeline, and the tags are " +
		"what separate the states.\n\n")

	first := 1
	for _, state := range states {
		last := first + state.frames - 1
		fmt.Fprintf(&b, "   %s, frames %d to %d\n", state.name, first, last)
		if note, ok := animationNotes[state.name]; ok {
			fmt.Fprintf(&b, "   %s\n", note)
		}
		b.WriteString("\n")
		first = last + 1
	}

	b.WriteString("6. Tag and time the states.\n")
	first = 1
	for _, state := range states {
		last := first + state.frames - 1
		fmt.Fprintf(&b, "   set_tag %q from %d to %d, aniDir %s.\n",
			state.name, first, last, loopDirection(state.name))
		first = last + 1
	}
	b.WriteString("   Then set_frame_durations. Hold the extremes longer than the " +
		"in-betweens; an idle breath sits around 150 to 200 ms a frame, an attack snap " +
		"around 60 to 80 ms with a longer hold on the wind up and the recovery.\n\n")

	b.WriteString("7. Check the motion, not just the drawing.\n" +
		"   Export a .gif with save_sprite_as and look at it. Flipping between frames is " +
		"the only way to catch a pop or a dead frame. Iterate until it reads.\n\n")

	b.WriteString("8. Export for use.\n" +
		"   export_spritesheet with sheetType horizontal and a dataFile so an engine can " +
		"read the frame rectangles, plus a .gif per state if you want previews.\n\n")

	b.WriteString("Things this server will hold you to:\n" +
		"- Every path argument must be absolute.\n" +
		"- Coordinates are 0-based and y grows downward.\n" +
		"- A group layer holds no pixels. Draw on image layers.\n" +
		"- Editing tools rewrite the file in place, with no undo.\n" +
		"- Saving a multi-frame sprite to .png writes one numbered file per frame " +
		"instead of the name you asked for. Use .gif or export_spritesheet.\n" +
		"- Assemble scenes from finished sprites with stamp_sprites rather than " +
		"redrawing them at an offset.\n")

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Sprite and animation workflow for %s", name),
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: b.String()}},
		},
	}, nil
}

func valueOr(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

type animationState struct {
	name   string
	frames int
}

// parseAnimations reads the "idle:6,attack:8" argument shape.
func parseAnimations(spec string) ([]animationState, error) {
	var states []animationState
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		stateName, count, ok := strings.Cut(part, ":")
		stateName = strings.TrimSpace(stateName)
		if !ok || stateName == "" {
			return nil, fmt.Errorf("animations entry %q should look like state:frames", part)
		}
		frames, err := strconv.Atoi(strings.TrimSpace(count))
		if err != nil || frames < 1 {
			return nil, fmt.Errorf("animations entry %q needs a positive frame count", part)
		}
		states = append(states, animationState{name: strings.ToLower(stateName), frames: frames})
	}
	if len(states) == 0 {
		return nil, fmt.Errorf("animations must list at least one state:frames pair")
	}
	return states, nil
}
