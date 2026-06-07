package cmd

import (
	"context"

	"recall/internal/mcpserver"
)

// MCP runs the recall MCP server over stdio so LLM agents can read and write
// memory. version is reported to clients in the server implementation info. It
// blocks until the client disconnects.
func MCP(args []string, version string) error {
	e, err := openEngine()
	if err != nil {
		return err
	}
	defer e.Close()

	return mcpserver.Serve(context.Background(), e, version)
}
