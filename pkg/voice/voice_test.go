package voice

import (
	"encoding/binary"
	"testing"
)

// TestKeyIgnoresWhitespace is what lets a `desc` be reflowed without silently
// orphaning the recording of the identical sentence.
func TestKeyIgnoresWhitespace(t *testing.T) {
	a := Key("The app hands you a note.")
	b := Key("  The app hands\n  you a note.  ")
	if a != b {
		t.Errorf("keys differ across whitespace: %s vs %s", a, b)
	}
	if Key("The app hands you a note!") == a {
		t.Error("different words produced the same key")
	}
	if len(a) != 16 {
		t.Errorf("key length = %d, want 16", len(a))
	}
}

func TestDirForSitsBesideTheSource(t *testing.T) {
	for src, want := range map[string]string{
		"oauth.dgm":              "oauth.voice",
		"examples/oauth.dgm":     "examples/oauth.voice",
		"/tmp/a.b/oauth.dgm":     "/tmp/a.b/oauth.voice",
		"noextension":            "noextension.voice",
		"examples/oauth.old.dgm": "examples/oauth.old.voice",
	} {
		if got := DirFor(src); got != want {
			t.Errorf("DirFor(%q) = %q, want %q", src, got, want)
		}
	}
}

// wav builds a RIFF/WAVE file with the chunks given, in the order given.
func wav(t *testing.T, chunks ...[]byte) []byte {
	t.Helper()
	var body []byte
	for _, c := range chunks {
		body = append(body, c...)
	}
	out := append([]byte("RIFF"), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:8], uint32(4+len(body)))
	out = append(out, []byte("WAVE")...)
	return append(out, body...)
}

func chunk(id string, body []byte) []byte {
	out := append([]byte(id), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(body)))
	out = append(out, body...)
	if len(body)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

// fmtChunk is a 16-byte PCM format chunk carrying byteRate, the only field
// WAVMillis reads.
func fmtChunk(byteRate uint32) []byte {
	body := make([]byte, 16)
	binary.LittleEndian.PutUint16(body[0:2], 1) // PCM
	binary.LittleEndian.PutUint16(body[2:4], 1) // mono
	binary.LittleEndian.PutUint32(body[4:8], 22050)
	binary.LittleEndian.PutUint32(body[8:12], byteRate)
	binary.LittleEndian.PutUint16(body[12:14], 2)
	binary.LittleEndian.PutUint16(body[14:16], 16)
	return body
}

func TestWAVMillis(t *testing.T) {
	// 44100 bytes/sec, one second of data.
	got, ok := WAVMillis(wav(t, chunk("fmt ", fmtChunk(44100)), chunk("data", make([]byte, 44100))))
	if !ok || got != 1000 {
		t.Errorf("duration = %d (ok=%v), want 1000", got, ok)
	}
}

// TestWAVMillisWalksPastFiller is the case that made walking chunks necessary:
// macOS `say` writes a FLLR padding chunk between `fmt ` and `data`, so reading
// `data` at a fixed offset measures the filler instead.
func TestWAVMillisWalksPastFiller(t *testing.T) {
	got, ok := WAVMillis(wav(t,
		chunk("fmt ", fmtChunk(44100)),
		chunk("FLLR", make([]byte, 4044)),
		chunk("data", make([]byte, 22050)),
	))
	if !ok || got != 500 {
		t.Errorf("duration = %d (ok=%v), want 500", got, ok)
	}
}

// TestWAVMillisSurvivesATruncatedTail keeps a killed synthesizer from taking the
// whole run down: the header still says how long the clip was meant to be.
func TestWAVMillisSurvivesATruncatedTail(t *testing.T) {
	full := wav(t, chunk("fmt ", fmtChunk(44100)), chunk("data", make([]byte, 44100)))
	if _, ok := WAVMillis(full[:len(full)-20000]); !ok {
		t.Error("a truncated clip should still report a duration from its header")
	}
}

func TestWAVMillisRejectsWhatIsNotAWav(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":         nil,
		"short":         []byte("RIFF"),
		"not wave":      append([]byte("RIFF"), append(make([]byte, 4), []byte("AVI ")...)...),
		"no fmt":        wav(t, chunk("data", make([]byte, 100))),
		"fmt too small": wav(t, chunk("fmt ", make([]byte, 8)), chunk("data", make([]byte, 100))),
	} {
		if ms, ok := WAVMillis(data); ok {
			t.Errorf("%s: reported %dms, want no duration", name, ms)
		}
	}
}
