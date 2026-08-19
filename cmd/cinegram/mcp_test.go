package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// mcpSession drives the real entry point — run(), with `mcp` as the command —
// over a scripted conversation, and returns the responses decoded.
//
// It asserts the invariant the transport rests on as it goes: every line the
// server wrote is one JSON message, and nothing at all reached stderr. A
// diagnostic printed the usual way would be a protocol violation, so the check
// belongs in every test rather than in one of them.
func mcpSession(t *testing.T, lines ...string) []map[string]any {
	t.Helper()

	var stdout, stderr bytes.Buffer
	in := strings.NewReader(strings.Join(lines, "\n") + "\n")
	if err := run([]string{"mcp"}, in, &stdout, &stderr); err != nil {
		t.Fatalf("run mcp: %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("the server wrote to stderr: %q", stderr.String())
	}

	var got []map[string]any
	for _, line := range strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("output line is not JSON: %q: %v", line, err)
		}
		got = append(got, m)
	}
	return got
}

// call builds a tools/call request. The arguments are written as JSON text
// rather than marshalled from a map so that the tests read as the wire does.
func call(id int, name, args string) string {
	return `{"jsonrpc":"2.0","id":` + itoa(id) + `,"method":"tools/call","params":{"name":"` + name + `","arguments":` + args + `}}`
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// result pulls the tool result out of a response, failing if the call came back
// as a protocol error instead.
func result(t *testing.T, resp map[string]any) (text string, isError bool) {
	t.Helper()
	if e, ok := resp["error"]; ok {
		t.Fatalf("protocol error where a result was expected: %v", e)
	}
	res, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in %v", resp)
	}
	var b strings.Builder
	for _, raw := range res["content"].([]any) {
		block := raw.(map[string]any)
		if block["type"] == "text" {
			b.WriteString(block["text"].(string))
		}
	}
	return b.String(), res["isError"] == true
}

// inlineArgs quotes a document as a `source` argument.
func inlineArgs(source string, extra ...string) string {
	encoded, err := json.Marshal(source)
	if err != nil {
		panic(err)
	}
	return `{"source":` + string(encoded) + strings.Join(extra, "") + `}`
}

// mcpCleanSource is the same little document lintFixSource is, with the
// misspelling corrected. The tools that produce output rather than diagnostics
// refuse a document with errors in it, which is the behaviour their subcommands
// have and the reason two fixtures are needed rather than one.
const mcpCleanSource = `flowchart LR
  client[Client]
  ing[Ingress]

  client --> ing

scenario "one hop"

  step go "the request arrives" {
    flow client -> ing
    highlight ing
  }
`

const mcpInit = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`

// TestMCPHandshake covers the opening exchange a host performs before it will
// call anything: initialize, then the initialized notification, which must draw
// no response at all.
func TestMCPHandshake(t *testing.T) {
	got := mcpSession(t, mcpInit, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if len(got) != 1 {
		t.Fatalf("got %d responses, want 1: a notification is never answered", len(got))
	}

	res := got[0]["result"].(map[string]any)
	if res["protocolVersion"] != "2025-06-18" {
		t.Errorf("protocolVersion = %v", res["protocolVersion"])
	}
	info := res["serverInfo"].(map[string]any)
	if info["name"] != "cinegram" || info["version"] != version {
		t.Errorf("serverInfo = %v, want cinegram %s", info, version)
	}
}

// TestMCPToolsList pins the surface. frame and sheet are asserted here and
// nowhere else: calling them needs a headless browser, which a test may not
// have, so what a test can check is that they are offered and that their
// descriptions say what they need.
func TestMCPToolsList(t *testing.T) {
	got := mcpSession(t, mcpInit, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if len(got) != 2 {
		t.Fatalf("got %d responses, want 2", len(got))
	}

	tools := got[1]["result"].(map[string]any)["tools"].([]any)
	var names []string
	byName := map[string]map[string]any{}
	for _, raw := range tools {
		tool := raw.(map[string]any)
		name := tool["name"].(string)
		names = append(names, name)
		byName[name] = tool
	}
	if want := "lint narrate mermaid frame sheet"; strings.Join(names, " ") != want {
		t.Errorf("tools = %v, want %s", names, want)
	}

	for name, tool := range byName {
		if tool["title"] == "" || tool["description"] == "" {
			t.Errorf("%s is missing a title or description: %v", name, tool)
		}
		if _, ok := tool["inputSchema"].(map[string]any); !ok {
			t.Errorf("%s has inputSchema %v, want an object", name, tool["inputSchema"])
		}
	}
	for _, name := range []string{"frame", "sheet"} {
		desc := byName[name]["description"].(string)
		if !strings.Contains(desc, "CINEGRAM_CHROME") {
			t.Errorf("%s does not say it needs a browser: %q", name, desc)
		}
	}
}

// TestMCPSchemasAreJSON is the check no client can do for us: a schema that does
// not parse makes every call to that tool a guess.
func TestMCPSchemasAreJSON(t *testing.T) {
	for _, tool := range mcpServer().Tools {
		var schema struct {
			Type                 string                     `json:"type"`
			Properties           map[string]json.RawMessage `json:"properties"`
			AdditionalProperties *bool                      `json:"additionalProperties"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Errorf("%s: %v\n%s", tool.Name, err, tool.InputSchema)
			continue
		}
		if schema.Type != "object" {
			t.Errorf("%s: type = %q", tool.Name, schema.Type)
		}
		if schema.AdditionalProperties == nil || *schema.AdditionalProperties {
			t.Errorf("%s: additionalProperties = %v, want false", tool.Name, schema.AdditionalProperties)
		}
		for _, want := range []string{"path", "source", "as"} {
			if _, ok := schema.Properties[want]; !ok {
				t.Errorf("%s: schema has no %q", tool.Name, want)
			}
		}
	}
}

// TestMCPLintCarriesFixes is the whole reason a lint tool is worth offering: the
// array a model gets back is the one `lint --format=json` writes, machine
// applicable corrections included.
func TestMCPLintCarriesFixes(t *testing.T) {
	got := mcpSession(t, mcpInit, call(2, "lint", inlineArgs(lintFixSource)))
	text, isError := result(t, got[1])
	if isError {
		t.Fatalf("lint reported a failure for a document it could read: %s", text)
	}

	var diags []struct {
		Message string `json:"message"`
		Fix     *struct {
			Old string `json:"old"`
			New string `json:"new"`
		} `json:"fix"`
	}
	if err := json.Unmarshal([]byte(text), &diags); err != nil {
		t.Fatalf("lint did not return JSON: %v\n%s", err, text)
	}
	if len(diags) == 0 {
		t.Fatal("no diagnostics for a document with a misspelt node")
	}

	found := false
	for _, d := range diags {
		if d.Fix != nil && d.Fix.Old == "ingres" && d.Fix.New == "ing" {
			found = true
		}
	}
	if !found {
		t.Errorf("no fix of ingres -> ing in %s", text)
	}
}

// TestMCPLintOnAnUnreadableFileIsAnError draws the line the other way: a
// diagnostic is data, but a document that does not exist is a failed call, and
// the model has to be told which it got.
func TestMCPLintOnAnUnreadableFileIsAnError(t *testing.T) {
	got := mcpSession(t, mcpInit, call(2, "lint", `{"path":"/nonexistent/nothing.dgm"}`))
	text, isError := result(t, got[1])
	if !isError {
		t.Errorf("a missing file came back as a successful lint: %s", text)
	}
	if !strings.Contains(text, "nothing.dgm") {
		t.Errorf("the error does not name the file: %s", text)
	}
}

// TestMCPSourceIsExclusive covers the rule the schema states in prose and the
// handler enforces, in both directions.
func TestMCPSourceIsExclusive(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{"both", `{"path":"a.dgm","source":"flowchart LR"}`, "not both"},
		{"neither", `{}`, "give one of path"},
		{"as without source", `{"path":"a.dgm","as":"b.dgm"}`, "already has a name"},
		{"an argument no schema declares", `{"nonsense":true}`, "bad arguments"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mcpSession(t, mcpInit, call(2, "lint", tc.args))
			text, isError := result(t, got[1])
			if !isError {
				t.Fatalf("accepted %s: %s", tc.args, text)
			}
			if !strings.Contains(text, tc.want) {
				t.Errorf("message %q does not contain %q", text, tc.want)
			}
		})
	}
}

// TestMCPMermaidRoundTrip checks the tool that has to give back exactly what it
// was given: the diagram half, untouched, with the animation stripped.
func TestMCPMermaidRoundTrip(t *testing.T) {
	got := mcpSession(t, mcpInit, call(2, "mermaid", inlineArgs(mcpCleanSource)))
	text, isError := result(t, got[1])
	if isError {
		t.Fatalf("mermaid failed: %s", text)
	}
	for _, want := range []string{"flowchart LR", "client[Client]", "client --> ing"} {
		if !strings.Contains(text, want) {
			t.Errorf("the emitted Mermaid is missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "scenario") {
		t.Errorf("the animation blocks survived into the Mermaid:\n%s", text)
	}
}

// TestMCPNarrateFormats covers the one input that changes the shape of the
// answer rather than its content.
func TestMCPNarrateFormats(t *testing.T) {
	got := mcpSession(t, mcpInit,
		call(2, "narrate", inlineArgs(mcpCleanSource)),
		call(3, "narrate", inlineArgs(mcpCleanSource, `,"format":"json"`)),
		call(4, "narrate", inlineArgs(mcpCleanSource, `,"format":"sideways"`)))

	md, isError := result(t, got[1])
	if isError || !strings.HasPrefix(md, "#") {
		t.Errorf("narrate md = %q (isError %v)", md, isError)
	}

	asJSON, isError := result(t, got[2])
	if isError {
		t.Fatalf("narrate json failed: %s", asJSON)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(asJSON), &doc); err != nil {
		t.Errorf("narrate json did not parse: %v\n%s", err, asJSON)
	}

	bad, isError := result(t, got[3])
	if !isError || !strings.Contains(bad, "sideways") {
		t.Errorf("an unknown format was accepted: %q (isError %v)", bad, isError)
	}
}

// TestMCPLanguageResource covers the one thing the server offers that is not a
// command: the authoring reference, which a model reads before writing .dgm and
// which it cannot get any other way when the skill folder is not installed.
func TestMCPLanguageResource(t *testing.T) {
	const uri = "cinegram://reference/language.md"
	got := mcpSession(t, mcpInit,
		`{"jsonrpc":"2.0","id":2,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"`+uri+`"}}`)

	listed := got[1]["result"].(map[string]any)["resources"].([]any)
	if len(listed) != 1 {
		t.Fatalf("got %d resources, want 1", len(listed))
	}
	first := listed[0].(map[string]any)
	if first["uri"] != uri || first["mimeType"] != "text/markdown" {
		t.Errorf("listed resource = %v", first)
	}

	contents := got[2]["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)
	if contents["uri"] != uri || contents["mimeType"] != "text/markdown" {
		t.Errorf("read resource = %v", contents)
	}
	text := contents["text"].(string)
	if !strings.HasPrefix(text, "# The .dgm language") {
		t.Errorf("the reference starts %q", text[:min(40, len(text))])
	}
}

// TestMCPTakesNoArguments: the command is a server, and a word after it is
// almost certainly a mistake worth reporting rather than ignoring.
func TestMCPTakesNoArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"mcp", "--port", "8080"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("mcp accepted arguments it does not have")
	}
	if stdout.Len() != 0 {
		t.Errorf("wrote %q to stdout before failing", stdout.String())
	}
}
