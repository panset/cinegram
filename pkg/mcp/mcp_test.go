package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// testServer is the fixture every session runs against: two tools that cover
// the only two outcomes a handler has, and one resource.
func testServer() *Server {
	return &Server{
		Name:    "test",
		Title:   "Test Server",
		Version: "9.9.9",
		Tools: []Tool{
			{
				Name:        "echo",
				Title:       "Echo",
				Description: "Echo the text back.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"additionalProperties":false}`),
				Handler: func(args json.RawMessage) Result {
					var in struct {
						Text string `json:"text"`
					}
					if err := json.Unmarshal(args, &in); err != nil {
						return Errorf("bad arguments: %v", err)
					}
					return Result{Content: []Content{Text(in.Text)}}
				},
			},
			{
				Name:        "boom",
				Description: "Always fails.",
				Handler: func(json.RawMessage) Result {
					return Errorf("no such thing as %s", "boom")
				},
			},
		},
		Resources: []Resource{{
			URI:      "test://doc.md",
			Name:     "doc.md",
			MIMEType: "text/markdown",
			Read:     func() (string, error) { return "# Doc\n", nil },
		}},
	}
}

// run feeds the lines to a server and returns every response, decoded.
//
// It also enforces the rule the transport rests on: every line written to out
// is one JSON message. Nothing else may reach that writer — not a warning, not
// a progress note — because the client is parsing, not reading.
func run(t *testing.T, s *Server, lines []string) []map[string]any {
	t.Helper()

	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	var got []map[string]any
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("output line is not JSON: %q: %v", line, err)
		}
		if m["jsonrpc"] != "2.0" {
			t.Errorf("response %q is missing jsonrpc 2.0", line)
		}
		got = append(got, m)
	}
	return got
}

const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`

func TestSessions(t *testing.T) {
	tests := []struct {
		name  string
		in    []string
		check func(t *testing.T, got []map[string]any)
	}{
		{
			name: "handshake echoes a version we know",
			in:   []string{initialize},
			check: func(t *testing.T, got []map[string]any) {
				res := one(t, got)["result"].(map[string]any)
				if res["protocolVersion"] != "2025-06-18" {
					t.Errorf("protocolVersion = %v", res["protocolVersion"])
				}
				info := res["serverInfo"].(map[string]any)
				if info["name"] != "test" || info["version"] != "9.9.9" || info["title"] != "Test Server" {
					t.Errorf("serverInfo = %v", info)
				}
				caps := res["capabilities"].(map[string]any)
				if _, ok := caps["tools"]; !ok {
					t.Errorf("capabilities missing tools: %v", caps)
				}
				if _, ok := caps["resources"]; !ok {
					t.Errorf("capabilities missing resources: %v", caps)
				}
			},
		},
		{
			name: "handshake echoes an older version we know",
			in:   []string{`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`},
			check: func(t *testing.T, got []map[string]any) {
				res := one(t, got)["result"].(map[string]any)
				if res["protocolVersion"] != "2024-11-05" {
					t.Errorf("protocolVersion = %v, want the client's own", res["protocolVersion"])
				}
			},
		},
		{
			name: "handshake falls back on a version we do not know",
			in:   []string{`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`},
			check: func(t *testing.T, got []map[string]any) {
				res := one(t, got)["result"].(map[string]any)
				if res["protocolVersion"] != ProtocolVersion {
					t.Errorf("protocolVersion = %v, want %s", res["protocolVersion"], ProtocolVersion)
				}
			},
		},
		{
			name: "a notification is never answered",
			in: []string{
				initialize,
				`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
				`{"jsonrpc":"2.0","method":"notifications/somethingElse"}`,
			},
			check: func(t *testing.T, got []map[string]any) {
				if len(got) != 1 {
					t.Fatalf("got %d responses, want only the one for initialize", len(got))
				}
			},
		},
		{
			name: "ping answers with an empty result",
			in:   []string{`{"jsonrpc":"2.0","id":7,"method":"ping"}`},
			check: func(t *testing.T, got []map[string]any) {
				resp := one(t, got)
				if resp["id"] != float64(7) {
					t.Errorf("id = %v", resp["id"])
				}
				res, ok := resp["result"].(map[string]any)
				if !ok || len(res) != 0 {
					t.Errorf("result = %v, want {}", resp["result"])
				}
			},
		},
		{
			name: "tools/list describes every tool",
			in:   []string{`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`},
			check: func(t *testing.T, got []map[string]any) {
				tools := one(t, got)["result"].(map[string]any)["tools"].([]any)
				if len(tools) != 2 {
					t.Fatalf("got %d tools, want 2", len(tools))
				}
				first := tools[0].(map[string]any)
				if first["name"] != "echo" || first["title"] != "Echo" || first["description"] != "Echo the text back." {
					t.Errorf("first tool = %v", first)
				}
				if _, ok := first["inputSchema"].(map[string]any); !ok {
					t.Errorf("inputSchema = %v, want an object", first["inputSchema"])
				}
				// A tool that declared no schema still advertises one, because
				// inputSchema is required and clients read it unconditionally.
				second := tools[1].(map[string]any)
				if _, ok := second["inputSchema"].(map[string]any); !ok {
					t.Errorf("schemaless tool has inputSchema %v", second["inputSchema"])
				}
			},
		},
		{
			name: "tools/call returns the handler's content",
			in:   []string{`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"a <b> & c"}}}`},
			check: func(t *testing.T, got []map[string]any) {
				res := one(t, got)["result"].(map[string]any)
				if res["isError"] != false {
					t.Errorf("isError = %v", res["isError"])
				}
				block := res["content"].([]any)[0].(map[string]any)
				if block["type"] != "text" || block["text"] != "a <b> & c" {
					t.Errorf("content = %v, want the text unescaped", block)
				}
			},
		},
		{
			name: "a failing tool is a result, not a protocol error",
			in:   []string{`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"boom"}}`},
			check: func(t *testing.T, got []map[string]any) {
				resp := one(t, got)
				if _, ok := resp["error"]; ok {
					t.Fatalf("handler failure surfaced as a JSON-RPC error: %v", resp["error"])
				}
				res := resp["result"].(map[string]any)
				if res["isError"] != true {
					t.Errorf("isError = %v, want true", res["isError"])
				}
				block := res["content"].([]any)[0].(map[string]any)
				if !strings.Contains(block["text"].(string), "no such thing as boom") {
					t.Errorf("content = %v, want the reason the model can read", block)
				}
			},
		},
		{
			name: "an unknown tool is invalid params",
			in:   []string{`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nope"}}`},
			check: func(t *testing.T, got []map[string]any) {
				wantError(t, one(t, got), codeInvalidParams, "nope")
			},
		},
		{
			name: "resources round-trip",
			in: []string{
				`{"jsonrpc":"2.0","id":6,"method":"resources/list"}`,
				`{"jsonrpc":"2.0","id":7,"method":"resources/read","params":{"uri":"test://doc.md"}}`,
			},
			check: func(t *testing.T, got []map[string]any) {
				if len(got) != 2 {
					t.Fatalf("got %d responses, want 2", len(got))
				}
				listed := got[0]["result"].(map[string]any)["resources"].([]any)[0].(map[string]any)
				if listed["uri"] != "test://doc.md" || listed["mimeType"] != "text/markdown" || listed["name"] != "doc.md" {
					t.Errorf("listed resource = %v", listed)
				}
				read := got[1]["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)
				if read["uri"] != "test://doc.md" || read["mimeType"] != "text/markdown" || read["text"] != "# Doc\n" {
					t.Errorf("read resource = %v", read)
				}
			},
		},
		{
			name: "an unknown resource is invalid params",
			in:   []string{`{"jsonrpc":"2.0","id":8,"method":"resources/read","params":{"uri":"test://absent"}}`},
			check: func(t *testing.T, got []map[string]any) {
				wantError(t, one(t, got), codeInvalidParams, "test://absent")
			},
		},
		{
			name: "an unknown method is method not found",
			in:   []string{`{"jsonrpc":"2.0","id":9,"method":"prompts/list"}`},
			check: func(t *testing.T, got []map[string]any) {
				wantError(t, one(t, got), codeMethodNotFound, "prompts/list")
			},
		},
		{
			name: "a string id comes back a string",
			in:   []string{`{"jsonrpc":"2.0","id":"call-1","method":"ping"}`},
			check: func(t *testing.T, got []map[string]any) {
				if id := one(t, got)["id"]; id != "call-1" {
					t.Errorf("id = %#v, want the string %q back", id, "call-1")
				}
			},
		},
		{
			name: "a malformed message is a parse error",
			in:   []string{`{"jsonrpc":"2.0","id":1,`},
			check: func(t *testing.T, got []map[string]any) {
				resp := one(t, got)
				if resp["id"] != nil {
					t.Errorf("id = %v, want null: there is no id to answer with", resp["id"])
				}
				wantError(t, resp, codeParse, "")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, run(t, testServer(), tc.in))
		})
	}
}

// TestServeStopsAtEOF is the loop's termination rule: a closed stdin ends the
// session cleanly, which is how a host shuts the server down.
func TestServeStopsAtEOF(t *testing.T) {
	var out bytes.Buffer
	if err := (&Server{Name: "test"}).Serve(strings.NewReader(""), &out); err != nil {
		t.Fatalf("Serve on an empty stream: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("wrote %q to a client that said nothing", out.String())
	}
}

// TestServeReportsAWriteFailure covers the other end: a client that goes away
// mid-session is an error to the caller, not a silent spin.
func TestServeReportsAWriteFailure(t *testing.T) {
	err := testServer().Serve(strings.NewReader(initialize+"\n"), brokenWriter{})
	if !errors.Is(err, errBrokenPipe) {
		t.Errorf("Serve = %v, want the write failure", err)
	}
}

var errBrokenPipe = errors.New("broken pipe")

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errBrokenPipe }

// TestImageContentOmitsText guards the one place a struct tag could not say
// what the spec needs: a text block keeps an empty caption, an image block
// carries no text member at all.
func TestImageContentOmitsText(t *testing.T) {
	encoded, err := json.Marshal(Image("QUJD", "image/png"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(encoded, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["text"]; ok {
		t.Errorf("image block carries text: %s", encoded)
	}
	if m["type"] != "image" || m["data"] != "QUJD" || m["mimeType"] != "image/png" {
		t.Errorf("image block = %s", encoded)
	}

	encoded, err = json.Marshal(Text(""))
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"type":"text","text":""}` {
		t.Errorf("empty text block = %s", encoded)
	}
}

// one asserts a single response and returns it.
func one(t *testing.T, got []map[string]any) map[string]any {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("got %d responses, want 1: %v", len(got), got)
	}
	return got[0]
}

func wantError(t *testing.T, resp map[string]any, code int, contains string) {
	t.Helper()
	e, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("response has no error: %v", resp)
	}
	if e["code"] != float64(code) {
		t.Errorf("code = %v, want %d", e["code"], code)
	}
	if _, ok := resp["result"]; ok {
		t.Errorf("an error response also carries a result: %v", resp)
	}
	if contains != "" && !strings.Contains(e["message"].(string), contains) {
		t.Errorf("message = %v, want it to name %q", e["message"], contains)
	}
}
