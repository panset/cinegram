package loader

import (
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/voice"
)

// narrated is a document with one spoken step, plus the sidecar that speaks it.
// The clip bytes are nonsense on purpose: the loader inlines whatever it is given
// and never decodes audio.
func narrated(prose string) (files map[string]string, key string) {
	key = voice.Key(prose)
	return map[string]string{
		"talk.dgm": doc("scenario \"x\"\n  step one \"One\" {\n    desc: \"" + prose + "\"\n    flow a -> b { dur: 400ms }\n  }\n"),
		"talk.voice/manifest.json": `{"version":1,"provider":"say","clips":{"` + key +
			`":{"file":"` + key + `.m4a","mime":"audio/mp4","ms":4200}}}`,
		"talk.voice/" + key + ".m4a": "not really audio",
	}, key
}

// TestVoiceIsNotLoadedUnlessAsked is the load-bearing half of the option. A
// recorded document's clips outweigh everything else it carries, and compilation
// has to stay a pure function of the checked-in sources — otherwise the demo site
// regenerates differently depending on whether whoever ran it had recorded any
// narration locally.
func TestVoiceIsNotLoadedUnlessAsked(t *testing.T) {
	files, _ := narrated("The app hands you a note.")

	b, err := Load("talk.dgm", fakeFS(files))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(b.Units[0].VoiceData); got != 0 {
		t.Errorf("loaded %d clips without WithVoice, want none", got)
	}
}

func TestVoiceInlinesTheSidecarAsADataURI(t *testing.T) {
	files, key := narrated("The app hands you a note.")

	b, err := Load("talk.dgm", fakeFS(files), WithVoice())
	if err != nil {
		t.Fatal(err)
	}
	clip, ok := b.Units[0].VoiceData[key]
	if !ok {
		t.Fatalf("no clip under the prose's key; got %v", b.Units[0].VoiceData)
	}
	if !strings.HasPrefix(clip.Data, "data:audio/mp4;base64,") {
		t.Errorf("clip is not a self-contained data URI: %.40q", clip.Data)
	}
	// base64 cannot produce a '<', which is what makes it safe to embed in the
	// <script> element the timeline travels in.
	if strings.ContainsAny(clip.Data, "<>") {
		t.Errorf("the data URI is not script-safe: %.60q", clip.Data)
	}
	// The duration comes from the manifest rather than from decoding the clip.
	if clip.Ms != 4200 {
		t.Errorf("ms = %d, want the 4200 recorded in the manifest", clip.Ms)
	}
}

// TestMissingSidecarIsSilent: a document that was never recorded is the ordinary
// case, not a problem to report.
func TestMissingSidecarIsSilent(t *testing.T) {
	files := map[string]string{"quiet.dgm": doc("scenario \"x\"\n  step one \"One\" {\n    flow a -> b { dur: 400ms }\n  }\n")}

	b, err := Load("quiet.dgm", fakeFS(files), WithVoice())
	if err != nil {
		t.Fatal(err)
	}
	if b.Units[0].Bag.Len() != 0 {
		t.Errorf("a document with no narration was diagnosed: %s", b.Units[0].Bag)
	}
}

// TestHalfWrittenSidecarWarnsAndStillAnimates: narration is an addition to a
// document, so losing it must never cost the diagram.
func TestHalfWrittenSidecarWarnsAndStillAnimates(t *testing.T) {
	files, key := narrated("The app hands you a note.")
	delete(files, "talk.voice/"+key+".m4a") // manifest promises a clip that is gone

	b, err := Load("talk.dgm", fakeFS(files), WithVoice())
	if err != nil {
		t.Fatal(err)
	}
	if b.Units[0].Bag.HasErrors() {
		t.Errorf("a missing clip should warn, not fail: %s", b.Units[0].Bag)
	}
	if b.Units[0].Bag.Len() == 0 {
		t.Error("a missing clip passed without a word")
	}
	if len(b.Units[0].VoiceData) != 0 {
		t.Error("a clip that could not be read was inlined anyway")
	}
}

func TestUnreadableManifestWarns(t *testing.T) {
	files, _ := narrated("The app hands you a note.")
	files["talk.voice/manifest.json"] = "{this is not json"

	b, err := Load("talk.dgm", fakeFS(files), WithVoice())
	if err != nil {
		t.Fatal(err)
	}
	if b.Units[0].Bag.HasErrors() {
		t.Errorf("a corrupt manifest should warn, not fail: %s", b.Units[0].Bag)
	}
	if b.Units[0].Bag.Len() == 0 {
		t.Error("a corrupt manifest passed silently")
	}
}

// TestNewerSidecarIsAnError is the one narration failure worth stopping for:
// carrying on would mean guessing at fields this binary does not know.
func TestNewerSidecarIsAnError(t *testing.T) {
	files, key := narrated("The app hands you a note.")
	files["talk.voice/manifest.json"] = `{"version":99,"clips":{"` + key + `":{"file":"x.m4a","mime":"audio/mp4"}}}`

	b, err := Load("talk.dgm", fakeFS(files), WithVoice())
	if err != nil {
		t.Fatal(err)
	}
	if !b.Units[0].Bag.HasErrors() {
		t.Error("a sidecar from the future was accepted")
	}
	if !strings.Contains(b.Units[0].Bag.String(), "version") {
		t.Errorf("the error should name the version: %s", b.Units[0].Bag)
	}
}
