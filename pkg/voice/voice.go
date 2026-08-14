// Package voice describes the narration sidecar: the clips a synthesizer
// produced for a document's prose, and how a reader finds the one belonging to
// a step.
//
// Nothing here does I/O. `cinegram voice` writes the sidecar and pkg/loader
// reads it, and they agree only through the types and the key function in this
// file — which is what lets the key stay a pure function of the words and lets
// both halves be tested without a filesystem.
package voice

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// ManifestName is the file inside a sidecar directory that describes it.
const ManifestName = "manifest.json"

// Version is the sidecar format. A reader that finds a newer one declines to
// use it rather than guessing at fields it does not know.
const Version = 1

// Manifest is a sidecar directory's index.
//
// Clips are keyed by Key(text) rather than by step id, and that is the whole
// design: prose is what a clip is *of*, so renaming a scenario, reordering
// steps or retelling the same words in another scenario all reuse the recording
// they already had, and only rewriting the sentence costs a new one. It also
// makes staleness impossible to get wrong — a clip whose key is absent from the
// document is simply unused.
type Manifest struct {
	Version int `json:"version"`

	// Provider and Voice are recorded for the human reading the directory
	// later, and so `voice --force` can report what it is replacing. Neither is
	// matched against: changing voice mid-document is a mess the author can see
	// and fix, not something to fail a build over.
	Provider string `json:"provider,omitempty"`
	Voice    string `json:"voice,omitempty"`

	Clips map[string]Clip `json:"clips"`
}

// Clip is one synthesized line.
type Clip struct {
	File string `json:"file"`           // relative to the sidecar directory
	MIME string `json:"mime"`           // what the data URI has to declare
	Ms   int    `json:"ms,omitempty"`   // measured duration, 0 when unknown
	Text string `json:"text,omitempty"` // the opening of the line, for humans
}

// Loaded is a clip once a reader has inlined it: the bytes as a data URI,
// carrying the duration that was measured when the clip was made rather than
// one recovered by decoding it again.
type Loaded struct {
	Data string
	Ms   int
}

// Key identifies a clip by the words it says.
//
// Whitespace is collapsed first so that reflowing a `desc` across lines, or
// changing the indentation of the block it sits in, does not silently orphan a
// recording that says exactly the same sentence.
func Key(text string) string {
	sum := sha256.Sum256([]byte(strings.Join(strings.Fields(text), " ")))
	return hex.EncodeToString(sum[:])[:16]
}

// DirFor is the sidecar directory belonging to a source file: `x.dgm` keeps its
// narration in `x.voice/` alongside it.
//
// Beside the source rather than in a cache, because the recordings are part of
// the document in every way that matters — they are what it sounds like — and
// because a relative path is what makes a checked-out copy still narrate.
func DirFor(source string) string {
	ext := filepath.Ext(source)
	return strings.TrimSuffix(source, ext) + ".voice"
}

// MIMEFor maps a clip's extension onto the type a data URI must declare.
//
// Closed rather than guessed at, for the reason frameMIME is: a browser handed
// `data:;base64,…` plays nothing, and narration that silently fails to speak is
// worse than narration that refuses to build.
var MIMEFor = map[string]string{
	".wav":  "audio/wav",
	".m4a":  "audio/mp4",
	".mp4":  "audio/mp4",
	".aac":  "audio/aac",
	".mp3":  "audio/mpeg",
	".ogg":  "audio/ogg",
	".opus": "audio/ogg",
	".flac": "audio/flac",
	".aiff": "audio/aiff",
	".aif":  "audio/aiff",
}

// WAVMillis reads a RIFF/WAVE header and returns the clip's length.
//
// Only WAV is parsed, and that is deliberate: `voice` synthesizes to WAV first
// precisely so a duration can be had from the bytes with the standard library
// alone, and compresses afterwards carrying the number it already measured. A
// compressed clip therefore never needs decoding here.
//
// Chunks are walked rather than assumed to be in order — macOS `say` emits a
// `FLLR` padding chunk between `fmt ` and `data`, so reading at a fixed offset
// finds filler and reports nonsense.
func WAVMillis(data []byte) (int, bool) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return 0, false
	}

	byteRate := 0
	for i := 12; i+8 <= len(data); {
		id := string(data[i : i+4])
		size := int(binary.LittleEndian.Uint32(data[i+4 : i+8]))
		if size < 0 || i+8+size > len(data) {
			// A truncated final chunk still describes its length, and `data`
			// arriving short is exactly what a killed synthesizer leaves. Trust
			// the header's size for the maths and let the caller notice the file
			// is unplayable.
			size = len(data) - i - 8
		}
		body := data[i+8 : i+8+size]

		switch id {
		case "fmt ":
			if len(body) >= 16 {
				byteRate = int(binary.LittleEndian.Uint32(body[8:12]))
			}
		case "data":
			if byteRate > 0 {
				return size * 1000 / byteRate, true
			}
			return 0, false // `data` before `fmt `; nothing to divide by
		}

		i += 8 + size
		if size%2 == 1 {
			i++ // RIFF pads odd-sized chunks to an even boundary
		}
	}
	return 0, false
}
