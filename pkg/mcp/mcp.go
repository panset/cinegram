// Package mcp is a Model Context Protocol server, spoken over stdio.
//
// It implements the parts of specification revision 2025-06-18 that a tool
// provider needs — the handshake, ping, tools and resources — and nothing else.
// A host launches the process, writes JSON-RPC requests on its stdin and reads
// responses from its stdout; the package is deliberately ignorant of what the
// tools do, so the same server serves a compiler, a linter or a screenshotter.
//
// # Framing
//
// MCP's stdio transport is newline-delimited JSON, not the Content-Length
// framing that LSP uses, so there are no headers to write. Reading goes through
// a json.Decoder rather than a bufio.Scanner because a scanner caps a token at
// 64 KB by default and a tools/call argument — a whole source file, a base64
// image — is easily larger than that. Writing goes through a json.Encoder with
// HTML escaping off, since the payloads carry `<`, `>` and `&` (Mermaid arrows,
// diagram labels) that must survive as themselves.
//
// Nothing but protocol messages may ever reach the writer. A handler that
// wants to say something to a person says it inside its own Result — the
// handlers in cmd/cinegram hand their subcommands a bytes.Buffer for a stderr
// and fold what lands there into the result text — because a stray write to
// stdout desynchronises the stream and the session ends.
//
// # One request at a time
//
// Serve is a single goroutine: it reads a request, runs its handler to
// completion, writes the response, and only then reads again. MCP allows
// concurrent requests, and this server is within its rights to decline them —
// the client sees responses arrive in order, which is a subset of what it must
// already handle. It is a deliberate v1 choice: cinegram's slow handlers shell
// out to a headless browser, and two of those running at once would fight over
// the same temporary directories and ports for no gain, since the model is
// waiting for both anyway. Making the loop concurrent later changes no wire
// shape, only the scheduling of it.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ProtocolVersion is the specification revision this package implements, and
// the version the server answers with when the client asks for one it does not
// recognise.
const ProtocolVersion = "2025-06-18"

// knownVersions are the revisions the server will speak if asked. MCP's rule is
// that the server echoes the client's version when it supports it and offers
// its own when it does not; these three differ in ways that do not touch what
// this server implements, so echoing any of them is honest.
var knownVersions = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
}

// JSON-RPC error codes. They are reserved for protocol faults — a message that
// did not parse, a method that does not exist, parameters that cannot be acted
// on. A tool that ran and failed is not a protocol fault: it comes back as a
// Result with IsError set, where the model can read the reason and try again.
const (
	codeParse          = -32700
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternal       = -32603
)

// Content is one block of a tool result: a piece of text, or an image.
//
// The zero value is not a valid block — Type decides which of the other fields
// carry meaning — so construct one with Text or Image rather than by hand.
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
}

// Text is a text block.
func Text(s string) Content { return Content{Type: "text", Text: s} }

// Image is an image block: base64 payload and the MIME type that decodes it.
func Image(data, mime string) Content {
	return Content{Type: "image", Data: data, MIMEType: mime}
}

// MarshalJSON writes only the fields the block's type gives meaning to.
//
// The spec requires `text` on a text block and forbids assuming it elsewhere,
// so an empty caption must still serialise as `"text": ""` while an image block
// must not carry the key at all. A plain struct tag cannot express both.
func (c Content) MarshalJSON() ([]byte, error) {
	type wire struct {
		Type     string  `json:"type"`
		Text     *string `json:"text,omitempty"`
		Data     string  `json:"data,omitempty"`
		MIMEType string  `json:"mimeType,omitempty"`
	}
	w := wire{Type: c.Type, Data: c.Data, MIMEType: c.MIMEType}
	if c.Type == "text" {
		w.Text = &c.Text
	}
	return json.Marshal(w)
}

// Result is what a tool hands back: the blocks the model sees, and whether the
// call is to be read as a failure.
type Result struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError"`
}

// Errorf is the failure Result: the message as a text block, IsError set.
//
// It is not a Go error and does not travel as one. The distinction is the whole
// point of IsError — the model is meant to see this, reason about it and call
// again with something better, which it cannot do with a JSON-RPC error the
// host swallows.
func Errorf(format string, args ...any) Result {
	return Result{Content: []Content{Text(fmt.Sprintf(format, args...))}, IsError: true}
}

// Tool is one callable the server advertises.
//
// InputSchema is JSON Schema, carried raw because it is data the server relays
// rather than anything it interprets. Handler receives the client's arguments
// as raw JSON — an absent `arguments` member arrives as `{}`, so a handler
// never has to test for nil before unmarshalling.
type Tool struct {
	Name        string
	Title       string
	Description string
	InputSchema json.RawMessage
	Handler     func(args json.RawMessage) Result
}

// Resource is one document the server can be asked to read.
//
// Read is a function rather than a string so a resource can be produced when it
// is asked for; an error from it is a protocol fault, because a resource the
// server listed and then could not read is the server's problem, not something
// the model can act on.
type Resource struct {
	URI         string
	Name        string
	Title       string
	Description string
	MIMEType    string
	Read        func() (string, error)
}

// Server is the whole configuration: who this is, and what it offers.
type Server struct {
	Name      string
	Title     string
	Version   string
	Tools     []Tool
	Resources []Resource
}

// --- the wire ------------------------------------------------------------------

// request is an incoming JSON-RPC message.
//
// ID stays raw so that whatever the client used — a number, a string, a large
// integer Go would round — comes back byte-identical. Absent (or null) means a
// notification, which is never answered.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
}

// capabilities declares only what this server has. listChanged stays false
// because the tool and resource lists are compiled in: they cannot change while
// the process runs, so promising notifications would be a promise never kept.
type capabilities struct {
	Tools     listChanged `json:"tools"`
	Resources listChanged `json:"resources"`
}

type listChanged struct {
	ListChanged bool `json:"listChanged"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// toolInfo is a Tool as it crosses the wire: everything but the handler.
type toolInfo struct {
	Name        string          `json:"name"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type resourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type readParams struct {
	URI string `json:"uri"`
}

type resourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

// --- the loop -------------------------------------------------------------------

// Serve reads requests from in and writes responses to out until in ends.
//
// A message that does not parse ends the session after one -32700 reply: a
// json.Decoder that has hit a syntax error is no longer positioned at a message
// boundary, so anything it read afterwards would be guesswork, and guessing at
// framing is how a stream turns into nonsense rather than an error. The host
// sees the reason and restarts the process, which costs it nothing.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)

	for {
		var req request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return enc.Encode(response{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: codeParse, Message: err.Error()},
			})
		}
		resp, answer := s.handle(req)
		if !answer {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
}

// handle answers one request. The second return is false for a notification,
// which by JSON-RPC's rule gets no reply at all — not even an error one.
func (s *Server) handle(req request) (response, bool) {
	if len(req.ID) == 0 || string(req.ID) == "null" {
		// Every notification this server understands — `initialized` is the
		// only one a client sends — is an announcement rather than a
		// state change, so there is nothing to do but let it pass. An
		// unknown notification is dropped for the same reason: a
		// notification cannot be answered, so it cannot be refused either.
		return response{}, false
	}

	resp := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		var p initializeParams
		if err := unmarshalParams(req.Params, &p); err != nil {
			return fail(resp, codeInvalidParams, err.Error()), true
		}
		version := ProtocolVersion
		if knownVersions[p.ProtocolVersion] {
			version = p.ProtocolVersion
		}
		resp.Result = initializeResult{
			ProtocolVersion: version,
			Capabilities:    capabilities{},
			ServerInfo:      serverInfo{Name: s.Name, Title: s.Title, Version: s.Version},
		}

	case "ping":
		// An empty object, which is what the spec asks for: the answer is that
		// there was an answer.
		resp.Result = struct{}{}

	case "tools/list":
		infos := make([]toolInfo, 0, len(s.Tools))
		for _, t := range s.Tools {
			infos = append(infos, toolInfo{
				Name: t.Name, Title: t.Title, Description: t.Description,
				InputSchema: schemaOrEmpty(t.InputSchema),
			})
		}
		resp.Result = map[string]any{"tools": infos}

	case "tools/call":
		var p callParams
		if err := unmarshalParams(req.Params, &p); err != nil {
			return fail(resp, codeInvalidParams, err.Error()), true
		}
		tool := s.tool(p.Name)
		if tool == nil {
			return fail(resp, codeInvalidParams, fmt.Sprintf("unknown tool %q", p.Name)), true
		}
		args := p.Arguments
		if len(args) == 0 {
			args = json.RawMessage("{}")
		}
		resp.Result = tool.Handler(args)

	case "resources/list":
		infos := make([]resourceInfo, 0, len(s.Resources))
		for _, r := range s.Resources {
			infos = append(infos, resourceInfo{
				URI: r.URI, Name: r.Name, Title: r.Title,
				Description: r.Description, MIMEType: r.MIMEType,
			})
		}
		resp.Result = map[string]any{"resources": infos}

	case "resources/read":
		var p readParams
		if err := unmarshalParams(req.Params, &p); err != nil {
			return fail(resp, codeInvalidParams, err.Error()), true
		}
		res := s.resource(p.URI)
		if res == nil {
			return fail(resp, codeInvalidParams, fmt.Sprintf("unknown resource %q", p.URI)), true
		}
		text, err := res.Read()
		if err != nil {
			// The server listed this and then could not produce it: its own
			// fault, and nothing the caller can phrase differently.
			return fail(resp, codeInternal, err.Error()), true
		}
		resp.Result = map[string]any{"contents": []resourceContents{
			{URI: res.URI, MIMEType: res.MIMEType, Text: text},
		}}

	default:
		return fail(resp, codeMethodNotFound, fmt.Sprintf("unknown method %q", req.Method)), true
	}
	return resp, true
}

func fail(resp response, code int, msg string) response {
	resp.Result = nil
	resp.Error = &rpcError{Code: code, Message: msg}
	return resp
}

// unmarshalParams tolerates absent params, which is what a request with nothing
// to say sends, and rejects params that are present but wrong.
func unmarshalParams(raw json.RawMessage, into any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, into)
}

// schemaOrEmpty keeps inputSchema a required, always-valid member: a tool that
// declared none takes an object with no properties, which is what "no
// arguments" means in JSON Schema.
func schemaOrEmpty(schema json.RawMessage) json.RawMessage {
	if len(schema) == 0 {
		return json.RawMessage(`{"type":"object"}`)
	}
	return schema
}

func (s *Server) tool(name string) *Tool {
	for i := range s.Tools {
		if s.Tools[i].Name == name {
			return &s.Tools[i]
		}
	}
	return nil
}

func (s *Server) resource(uri string) *Resource {
	for i := range s.Resources {
		if s.Resources[i].URI == uri {
			return &s.Resources[i]
		}
	}
	return nil
}
