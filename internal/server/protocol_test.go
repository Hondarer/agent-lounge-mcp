package server

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// rpcResponse is a JSON-RPC response, as the tests read it.
type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"error"`
}

// rpc posts one JSON-RPC request to the MCP endpoint of base with the given headers and returns
// the HTTP response and the JSON-RPC response it carries (as JSON or in an event stream).
func rpc(t *testing.T, base, body string, header map[string]string) (*http.Response, rpcResponse) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, base+MCPPath, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	var out rpcResponse
	data := string(b)
	if strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		data = ""
		for _, line := range strings.Split(string(b), "\n") {
			if rest, ok := strings.CutPrefix(line, "data:"); ok {
				data = rest
			}
		}
	}
	kind := res.Header.Get("Content-Type")
	if (strings.HasPrefix(kind, "application/json") || strings.HasPrefix(kind, "text/event-stream")) && strings.TrimSpace(data) != "" {
		if err := json.Unmarshal([]byte(data), &out); err != nil {
			t.Fatalf("response %d %q: %v", res.StatusCode, b, err)
		}
	}
	return res, out
}

// legacyVersions are the protocol versions the server serves, newest first.
var legacyVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// The server does not serve protocol 2026-07-28, but tells such a client which versions it
// serves, so that the client connects again with initialize (see Handler).
func TestProtocol20260728(t *testing.T) {
	base := serveHTTP(t)
	const meta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{},` +
		`"io.modelcontextprotocol/clientInfo":{"name":"new-cli","version":"1"}}`
	t.Run("server/discover lists the versions served", func(t *testing.T) {
		h := map[string]string{"Mcp-Protocol-Version": "2026-07-28", "Mcp-Method": "server/discover"}
		res, out := rpc(t, base, `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{`+meta+`}}`, h)
		if res.StatusCode != http.StatusOK || out.Error != nil {
			t.Fatalf("status %d, error %+v", res.StatusCode, out.Error)
		}
		var r struct {
			SupportedVersions []string `json:"supportedVersions"`
		}
		if err := json.Unmarshal(out.Result, &r); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(r.SupportedVersions, legacyVersions) {
			t.Errorf("supported versions %v, want %v", r.SupportedVersions, legacyVersions)
		}
	})

	t.Run("other requests fail with -32022", func(t *testing.T) {
		h := map[string]string{"Mcp-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/list"}
		_, out := rpc(t, base, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{`+meta+`}}`, h)
		if out.Error == nil || out.Error.Code != -32022 {
			t.Fatalf("error %+v, want -32022", out.Error)
		}
		var d struct {
			Supported []string `json:"supported"`
		}
		if err := json.Unmarshal(out.Error.Data, &d); err != nil {
			t.Fatalf("data %s: %v", out.Error.Data, err)
		}
		if !slices.Equal(d.Supported, legacyVersions) {
			t.Errorf("supported versions %v, want %v", d.Supported, legacyVersions)
		}
	})

	t.Run("initialize with 2025-11-25 succeeds", func(t *testing.T) {
		res, out := rpc(t, base, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25",`+
			`"capabilities":{},"clientInfo":{"name":"new-cli","version":"1"}}}`, nil)
		if res.StatusCode != http.StatusOK || out.Error != nil {
			t.Fatalf("status %d, error %+v", res.StatusCode, out.Error)
		}
		var r struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(out.Result, &r); err != nil {
			t.Fatal(err)
		}
		if r.ProtocolVersion != "2025-11-25" || res.Header.Get("Mcp-Session-Id") == "" {
			t.Errorf("protocol %q, session %q", r.ProtocolVersion, res.Header.Get("Mcp-Session-Id"))
		}
	})
}
