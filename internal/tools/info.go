package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/version"
)

func registerCore(r *Registry, d *Deps) {
	add(r, groupCore, &mcp.Tool{
		Name:        "lounge_info",
		Title:       "Server information",
		Description: "Get the agent-lounge-mcp server version and its database schema version. Use it to check that the server and its database respond.",
		Annotations: readOnly(),
	}, d.loungeInfo)
}

// InfoInput defines arguments for the lounge_info tool.
type InfoInput struct{}

// InfoOutput is the result of the lounge_info tool.
type InfoOutput struct {
	Version       string `json:"version" jsonschema:"server version"`
	SchemaVersion int    `json:"schema_version" jsonschema:"schema version of the database file"`
}

func (d *Deps) loungeInfo(ctx context.Context, _ *mcp.CallToolRequest, _ InfoInput) (*mcp.CallToolResult, InfoOutput, error) {
	v, err := d.Store.SchemaVersion(ctx)
	if err != nil {
		return nil, InfoOutput{}, err
	}
	return nil, InfoOutput{Version: version.Version, SchemaVersion: v}, nil
}
