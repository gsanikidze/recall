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

	return mcpserver.Serve(context.Background(), e, version, func(_ context.Context, path string) (mcpserver.ProjectOut, error) {
		out, err := useProject(path)
		if err != nil {
			return mcpserver.ProjectOut{}, err
		}
		return mcpserver.ProjectOut{ProjectPath: out.ProjectPath, VaultPath: out.VaultPath, DBPath: out.DBPath}, nil
	})
}
