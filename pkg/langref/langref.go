// Package langref carries the .dgm language reference as a string.
//
// The canonical copy is skills/cinegram/references/language.md, because that
// file ships to users: the authoring skill installs it, and an agent reads it
// out of the skill folder before writing a line of .dgm. The copy here exists
// for the reader that has no skill folder — a model talking to `cinegram mcp`,
// which asks the server for the reference over the wire and gets it out of the
// binary.
//
// Two copies of a file is a promise to keep them equal, and the only promise
// worth making is one a build can check. //pkg/langref:langref_test is that
// check, and //pkg/langref:sync is its fix. The duplication itself is forced:
// go:embed cannot reach outside its own package directory, so the bytes have to
// live next to the Go code that inlines them.
package langref

import _ "embed"

//go:embed assets/language.md
var markdown string

// Markdown is the reference, exactly as the skill folder ships it.
func Markdown() string { return markdown }
