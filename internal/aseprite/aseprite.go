package aseprite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// sentinel marks the single line of stdout that carries a script's result.
// Aseprite writes unrelated diagnostics to stdout, so results need a marker.
const sentinel = "@@ASEMCP@@"

// prelude wraps a script body so that it always reports a structured result,
// even when the body raises. Its verbs are the params JSON, the sentinel and
// the body. luaHelpers is spliced in, so it must stay free of percent verbs.
const prelude = `local P = json.decode([==[%s]==])
local function emit(t) print(%q .. json.encode(t)) end
` + luaHelpers + `
local ok, res = pcall(function()
%s
end)
if ok then emit({ok = true, data = res}) else emit({ok = false, error = tostring(res)}) end
`

// candidates are searched in order when ASEPRITE_PATH is unset.
var candidates = []string{
	`C:\Program Files (x86)\Steam\steamapps\common\Aseprite\Aseprite.exe`,
	`C:\Program Files\Steam\steamapps\common\Aseprite\Aseprite.exe`,
	`C:\Program Files\Aseprite\Aseprite.exe`,
	`C:\Program Files (x86)\Aseprite\Aseprite.exe`,
}

// Runner executes Lua scripts through the Aseprite batch-mode CLI.
type Runner struct {
	ExePath string
	Timeout time.Duration
}

// NewRunner locates the Aseprite executable and prepares a runner.
func NewRunner() (*Runner, error) {
	exe, err := findAseprite()
	if err != nil {
		return nil, err
	}
	timeout := 60 * time.Second
	if v := os.Getenv("ASEPRITE_MCP_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ASEPRITE_MCP_TIMEOUT %q: %w", v, err)
		}
		timeout = d
	}
	return &Runner{ExePath: exe, Timeout: timeout}, nil
}

func findAseprite() (string, error) {
	if p := os.Getenv("ASEPRITE_PATH"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("ASEPRITE_PATH points at %q which is not readable: %w", p, err)
		}
		return p, nil
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if p, err := exec.LookPath("aseprite"); err == nil {
		return p, nil
	}
	return "", errors.New("could not locate Aseprite.exe; set the ASEPRITE_PATH environment variable")
}

// result is the envelope every generated script emits on the sentinel line.
type result struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

// Run renders body into a temporary script, executes it in batch mode and
// returns the JSON payload the script produced.
func (r *Runner) Run(ctx context.Context, body string, params any) (json.RawMessage, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encoding script params: %w", err)
	}
	if strings.Contains(string(raw), "]==]") {
		return nil, errors.New("script params contain a Lua long-bracket terminator")
	}

	dir, err := os.MkdirTemp("", "aseprite-mcp-")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	script := filepath.Join(dir, "script.lua")
	src := fmt.Sprintf(prelude, raw, sentinel, body)
	if err := os.WriteFile(script, []byte(src), 0o600); err != nil {
		return nil, fmt.Errorf("writing script: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.ExePath, "-b", "--script", script)
	out, runErr := cmd.CombinedOutput()
	text := string(out)

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("Aseprite timed out after %s; output so far:\n%s", r.Timeout, trim(text))
	}

	line, found := findSentinel(text)
	if !found {
		// No envelope means the script aborted before pcall could report, which
		// is what an uncaught Lua error or a startup failure looks like.
		if runErr != nil {
			return nil, fmt.Errorf("Aseprite failed (%v):\n%s", runErr, trim(text))
		}
		return nil, fmt.Errorf("Aseprite produced no result:\n%s", trim(text))
	}

	var res result
	if err := json.Unmarshal([]byte(line), &res); err != nil {
		return nil, fmt.Errorf("decoding script result %q: %w", line, err)
	}
	if !res.OK {
		return nil, errors.New(res.Error)
	}
	if len(res.Data) == 0 {
		return json.RawMessage("null"), nil
	}
	return res.Data, nil
}

// findSentinel returns the payload of the last sentinel line in out.
func findSentinel(out string) (string, bool) {
	var payload string
	var found bool
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if rest, ok := strings.CutPrefix(line, sentinel); ok {
			payload, found = rest, true
		}
	}
	return payload, found
}

// trim keeps diagnostic output short enough to stay readable in an error.
func trim(s string) string {
	s = strings.TrimSpace(s)
	const max = 2000
	if len(s) > max {
		return s[:max] + "\n... (truncated)"
	}
	return s
}
