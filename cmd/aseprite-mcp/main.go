// Command aseprite-mcp exposes Aseprite as an MCP server over stdio.
//
// Every tool call renders a Lua script and runs it through the Aseprite
// batch-mode CLI, so each call is independent: there is no document that stays
// open between calls.
package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ez8801/aseprite-mcp/internal/aseprite"
)

const version = "1.0.0"

func main() {
	// stdout carries the MCP protocol, so diagnostics must go to stderr.
	log.SetOutput(os.Stderr)
	log.SetFlags(0)
	log.SetPrefix("aseprite-mcp: ")

	runner, err := aseprite.NewRunner()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("using %s", runner.ExePath)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "aseprite",
		Title:   "Aseprite",
		Version: version,
	}, &mcp.ServerOptions{
		Instructions: "Drives Aseprite through its batch-mode CLI. Calls are stateless: there is " +
			"no open document, so every tool takes absolute file paths and reads or writes " +
			"them directly. Tools that write refuse to clobber an existing file unless " +
			"overwrite is set. Call aseprite_health to check that Aseprite is reachable.",
	})
	aseprite.Register(server, runner)

	// A closed stdin is how a client disconnects, so it is not a failure.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil && !errors.Is(err, io.EOF) {
		log.Fatal(err)
	}
}
