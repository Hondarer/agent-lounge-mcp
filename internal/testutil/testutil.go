// Package testutil holds helpers shared by tests across packages. It must not import packages
// whose internal tests use it.
package testutil

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Connect serves srv over in-memory transports and returns a connected client session that is
// closed when the test ends.
func Connect(t *testing.T, srv *mcp.Server, copts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, copts).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cs.Close()
		ss.Wait()
	})
	return cs
}

// ConnectHTTP connects to the Streamable HTTP endpoint and returns a client session that is
// closed when the test ends.
func ConnectHTTP(t *testing.T, endpoint string) *mcp.ClientSession {
	t.Helper()
	return ConnectHTTPWith(t, endpoint, nil)
}

// ConnectHTTPWith is ConnectHTTP with client options.
func ConnectHTTPWith(t *testing.T, endpoint string, copts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, copts).
		Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// Text returns the first text content of a tool result.
func Text(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

// Call calls a tool, fails the test on an error result, and decodes the structured output.
func Call[T any](t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) T {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("%s returned error: %v", name, Text(res))
	}
	var out T
	b, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// CallErr calls a tool, checks that it fails and returns the error text.
func CallErr(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatalf("%s %v should fail", name, args)
	}
	return Text(res)
}

// ToolSet returns the session's tool names as a set.
func ToolSet(t *testing.T, cs *mcp.ClientSession) map[string]bool {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, tool := range res.Tools {
		set[tool.Name] = true
	}
	return set
}

// WaitFor polls cond every 5ms until it holds, failing the test with msg after 5 seconds.
func WaitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
