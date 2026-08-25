package main

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPromptsAreListed(t *testing.T) {
	ctx, s := connect(t)
	res, err := s.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Prompts) != 1 || res.Prompts[0].Name != "animated_character" {
		t.Fatalf("prompts/list = %+v, want just animated_character", res.Prompts)
	}

	required := map[string]bool{}
	for _, arg := range res.Prompts[0].Arguments {
		required[arg.Name] = arg.Required
	}
	for _, name := range []string{"name", "description", "outputDir", "animations", "size"} {
		if _, ok := required[name]; !ok {
			t.Errorf("argument %q missing from the prompt", name)
		}
	}
	if !required["name"] || !required["description"] {
		t.Errorf("name and description should be required, got %v", required)
	}
}

func promptText(t *testing.T, ctx context.Context, s *mcp.ClientSession, args map[string]string) string {
	t.Helper()
	res, err := s.GetPrompt(ctx, &mcp.GetPromptParams{Name: "animated_character", Arguments: args})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	var b strings.Builder
	for _, m := range res.Messages {
		if m.Role != "user" {
			t.Errorf("unexpected role %q", m.Role)
		}
		if tc, ok := m.Content.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// The frame ranges are what the caller cannot work out on their own, so they
// have to come out of the argument rather than being left generic.
func TestPromptNumbersTheFramesPerState(t *testing.T) {
	ctx, s := connect(t)

	text := promptText(t, ctx, s, map[string]string{
		"name":        "owl_wizard",
		"description": "a hooded barn owl carrying a lantern",
		"animations":  "idle:6,attack:8",
		"outputDir":   "C:/art/owl",
	})

	for _, want := range []string{
		"owl_wizard",
		"a hooded barn owl carrying a lantern",
		"C:/art/owl",
		"idle, frames 1 to 6",
		"attack, frames 7 to 14",
		`set_tag "idle" from 1 to 6, aniDir ping_pong`,
		`set_tag "attack" from 7 to 14, aniDir forward`,
		"64x64",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt is missing %q", want)
		}
	}
}

func TestPromptDefaultsAndOverrides(t *testing.T) {
	ctx, s := connect(t)

	base := promptText(t, ctx, s, map[string]string{
		"name": "sprite", "description": "a thing",
	})
	if !strings.Contains(base, "idle, frames 1 to 6") || !strings.Contains(base, "attack, frames 7 to 14") {
		t.Errorf("default animations did not come through:\n%s", base)
	}
	if !strings.Contains(base, "No output directory was given") {
		t.Errorf("a missing outputDir should be called out:\n%s", base)
	}

	custom := promptText(t, ctx, s, map[string]string{
		"name": "sprite", "description": "a thing",
		"animations": "walk:8,hurt:3", "size": "32x32",
	})
	if !strings.Contains(custom, "walk, frames 1 to 8") || !strings.Contains(custom, "hurt, frames 9 to 11") {
		t.Errorf("custom animations did not come through:\n%s", custom)
	}
	if !strings.Contains(custom, "32x32") || strings.Contains(custom, "64x64") {
		t.Errorf("size override did not come through:\n%s", custom)
	}
	if strings.Contains(custom, "ping_pong") {
		t.Errorf("only idle should loop ping_pong:\n%s", custom)
	}
}

func TestPromptRejectsBadArguments(t *testing.T) {
	ctx, s := connect(t)

	cases := []struct {
		name string
		args map[string]string
		want string
	}{
		{"missing description", map[string]string{"name": "x"}, "required"},
		{"missing name", map[string]string{"description": "x"}, "required"},
		{"malformed animations", map[string]string{
			"name": "x", "description": "y", "animations": "idle",
		}, "state:frames"},
		{"zero frames", map[string]string{
			"name": "x", "description": "y", "animations": "idle:0",
		}, "positive frame count"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.GetPrompt(ctx, &mcp.GetPromptParams{
				Name: "animated_character", Arguments: c.args,
			})
			if err == nil {
				t.Fatalf("expected an error for %v", c.args)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("expected %q in the error, got %v", c.want, err)
			}
		})
	}
}
