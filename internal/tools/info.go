package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/version"
)

func registerCore(r *Registry, d *Deps) {
	add(r, groupCore, &mcp.Tool{
		Name:        toolInfo,
		Title:       "Server information",
		Description: "Get the agent-lounge-mcp server version and its database schema version. Use it to check that the server and its database respond; it works before lounge_join.",
		Annotations: readOnly(),
	}, d.loungeInfo)
}

// InfoInput defines arguments for the lounge_info tool.
type InfoInput struct{}

// InfoOutput is the result of the lounge_info tool.
type InfoOutput struct {
	Version         string                         `json:"version" jsonschema:"server version"`
	SchemaVersion   int                            `json:"schema_version" jsonschema:"schema version of the database file"`
	MemoryRetention map[string]MemoryRetentionInfo `json:"memory_retention,omitempty" jsonschema:"memory retention settings by scope kind when they differ"`
	unreadField
}

// MemoryRetentionInfo describes one scope kind's retention settings.
type MemoryRetentionInfo struct {
	HalfLife  string  `json:"half_life" jsonschema:"half-life duration, zero disables fading"`
	Threshold float64 `json:"fade_threshold" jsonschema:"retention score below which memories fade"`
	Grace     string  `json:"grace" jsonschema:"deletion grace duration, zero keeps faded memories"`
}

func (d *Deps) loungeInfo(ctx context.Context, _ *mcp.CallToolRequest, _ InfoInput) (*mcp.CallToolResult, InfoOutput, error) {
	v, err := d.Store.SchemaVersion(ctx)
	if err != nil {
		return nil, InfoOutput{}, err
	}
	out := InfoOutput{Version: version.Version, SchemaVersion: v}
	policies := d.Cfg.RetentionPolicies()
	if _, uniform := policies.Uniform(); !uniform {
		out.MemoryRetention = make(map[string]MemoryRetentionInfo)
		for _, kind := range store.ScopeKinds() {
			p := policies.For(kind)
			out.MemoryRetention[kind] = MemoryRetentionInfo{
				HalfLife: config.FormatDuration(p.HalfLife), Threshold: p.Threshold, Grace: config.FormatDuration(p.Grace),
			}
		}
	}
	return nil, out, nil
}
