package main

import (
	"os"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/voice"
)

// TestLinesOfIsDistinctByWords is what makes an audience ladder affordable to
// record: three tellings of eight steps are 24 steps but only as many clips as
// there are different sentences, and a step two rungs phrase identically is one
// recording rather than two.
func TestLinesOfIsDistinctByWords(t *testing.T) {
	tl := &ir.Timeline{Views: []ir.View{{
		Scenarios: []ir.Scenario{
			{Steps: []ir.Step{
				{ID: "a", Desc: "The app hands you a note."},
				{ID: "b", Desc: "Nothing is said here."},
				{ID: "c", Desc: ""},
			}},
			{Steps: []ir.Step{
				// Same sentence reflowed: one clip, not two.
				{ID: "a", Desc: "The app hands you   a note."},
				{ID: "b", Desc: "Something else entirely."},
			}},
		},
	}}}

	got := linesOf(tl)
	if len(got) != 3 {
		t.Fatalf("lines = %q, want 3 distinct", got)
	}
	if got[0] != "The app hands you a note." {
		t.Errorf("first line = %q, want the prose in reading order", got[0])
	}
	for _, line := range got {
		if line == "" {
			t.Error("a step with no prose produced a line to record")
		}
	}
}

func TestSplitDataURI(t *testing.T) {
	mime, payload, ok := splitDataURI("data:audio/mp4;base64,AAAB")
	if !ok || mime != "audio/mp4" || payload != "AAAB" {
		t.Errorf("got (%q, %q, %v), want audio/mp4 / AAAB / true", mime, payload, ok)
	}
	for _, bad := range []string{"", "audio/mp4;base64,AAAB", "data:audio/mp4;base64", "http://x/y.mp3"} {
		if _, _, ok := splitDataURI(bad); ok {
			t.Errorf("%q was accepted as a data URI", bad)
		}
	}
}

// TestMixArgsDelaysEachLineToItsStep pins the choice of adelay over concat.
// Concatenating would butt the clips together and invent its own timing; each
// line has to land on the step that explains it.
func TestMixArgsDelaysEachLineToItsStep(t *testing.T) {
	mix := mixArgs([]cue{
		{path: "/tmp/a.m4a", start: 0, ms: 4000},
		{path: "/tmp/b.m4a", start: 4000, ms: 3000},
	}, "mp4")

	inputs := strings.Join(mix.inputs, " ")
	if inputs != "-i /tmp/a.m4a -i /tmp/b.m4a" {
		t.Errorf("inputs = %q, want one -i per clip in order", inputs)
	}

	joined := strings.Join(mix.mix, " ")
	// Input 0 is the frame sequence, so the clips are 1 and 2.
	for _, want := range []string{"[1:a]adelay=0:all=1[n0]", "[2:a]adelay=4000:all=1[n1]", "amix=inputs=2:normalize=0"} {
		if !strings.Contains(joined, want) {
			t.Errorf("filter is missing %q:\n%s", want, joined)
		}
	}
	// normalize=0 or an eight-step walkthrough comes out eight times too quiet.
	if strings.Contains(joined, "normalize=1") {
		t.Error("amix is normalizing, which quietens every extra line")
	}
	if !strings.Contains(joined, "-map 0:v") || !strings.Contains(joined, "-map [narration]") {
		t.Errorf("both streams must be mapped explicitly:\n%s", joined)
	}
	if !strings.Contains(joined, "-shortest") {
		t.Error("the video has to be the master; -shortest is missing")
	}
}

// TestMixArgsPicksACodecTheContainerPlays: aac in a webm plays in nothing.
func TestMixArgsPicksACodecTheContainerPlays(t *testing.T) {
	for format, want := range map[string]string{"mp4": "aac", "webm": "libopus"} {
		joined := strings.Join(mixArgs([]cue{{path: "/tmp/a.m4a"}}, format).mix, " ")
		if !strings.Contains(joined, "-c:a "+want) {
			t.Errorf("%s used %q, want -c:a %s", format, joined, want)
		}
	}
}

func TestMixArgsIsEmptyWithoutCues(t *testing.T) {
	mix := mixArgs(nil, "mp4")
	if len(mix.inputs) != 0 || len(mix.mix) != 0 {
		t.Errorf("mix = %+v, want nothing added when there is no narration", mix)
	}
}

// TestNarrationOverrun is the warning that saves an author from a video whose
// voice stops mid-sentence. The two clocks are set independently: a step is as
// long as its animation needs, a line as long as it takes to say.
func TestNarrationOverrun(t *testing.T) {
	cues := []cue{{start: 0, ms: 9000}, {start: 700, ms: 11000}}
	// The later line ends at 11700 against an animation 1400 long.
	if got := narrationOverrun(cues, 1400); got != 10300 {
		t.Errorf("overrun = %d, want 10300", got)
	}
	// With `pace: voice` every step is at least as long as its line.
	fitted := []cue{{start: 0, ms: 9000}, {start: 9000, ms: 11000}}
	if got := narrationOverrun(fitted, 20000); got != 0 {
		t.Errorf("overrun = %d, want 0 when the steps were paced to fit", got)
	}
	if got := narrationOverrun(nil, 1000); got != 0 {
		t.Errorf("overrun = %d, want 0 with no narration", got)
	}
}

func TestHasNarration(t *testing.T) {
	silent := &ir.Timeline{Views: []ir.View{{Scenarios: []ir.Scenario{{Steps: []ir.Step{{Desc: "words"}}}}}}}
	if hasNarration(silent) {
		t.Error("prose alone is not narration; only a clip is")
	}
	spoken := &ir.Timeline{Views: []ir.View{{Scenarios: []ir.Scenario{{Steps: []ir.Step{
		{Desc: "words"}, {Desc: "more", Audio: "data:audio/mp4;base64,AA"},
	}}}}}}
	if !hasNarration(spoken) {
		t.Error("a step with a clip was not found")
	}
}

// TestTTSCommandRequiresAnOutputPlaceholder: without {out} the synthesizer has
// nowhere to write, and the failure would otherwise be an empty directory and a
// confusing "wrote no file".
func TestTTSCommandRequiresAnOutputPlaceholder(t *testing.T) {
	t.Setenv("CINEGRAM_TTS_COMMAND", "say --file-format=WAVE")
	if _, _, err := ttsCommand(""); err == nil {
		t.Fatal("a command with no {out} was accepted")
	} else if !strings.Contains(err.Error(), "{out}") {
		t.Errorf("the error should name {out}: %v", err)
	}
}

func TestTTSCommandPrefersTheEnvironment(t *testing.T) {
	t.Setenv("CINEGRAM_TTS_COMMAND", "mytts --out {out}")
	cmd, provider, err := ttsCommand("Samantha")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "mytts --out {out}" {
		t.Errorf("command = %q, want the environment's verbatim", cmd)
	}
	if provider != "command" {
		t.Errorf("provider = %q, want command", provider)
	}
	// --voice is a convenience for the built-in default; a configured command
	// owns its own arguments and must not have flags spliced into it.
	if strings.Contains(cmd, "Samantha") {
		t.Error("--voice was injected into a user-configured command")
	}
}

func TestVoiceOptsOnlyAsksWhenTold(t *testing.T) {
	if len(voiceOpts(false)) != 0 {
		t.Error("narration was loaded without --with-voice")
	}
	if len(voiceOpts(true)) != 1 {
		t.Error("--with-voice did not ask the loader for narration")
	}
}

// TestPruneDropsClipsTheProseNoLongerAsksFor keeps a rewritten sentence from
// leaving its recording behind: the key is a hash of the words, so an edit
// orphans a clip, and a directory that only grew would fill with drafts.
func TestPruneDropsClipsTheProseNoLongerAsksFor(t *testing.T) {
	dir := t.TempDir()

	live := "The words that are still there."
	dead := "The words that were rewritten."
	liveKey, deadKey := voice.Key(live), voice.Key(dead)

	for _, key := range []string{liveKey, deadKey} {
		if err := os.WriteFile(dir+"/"+key+".m4a", []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := voice.Manifest{Version: voice.Version, Clips: map[string]voice.Clip{
		liveKey: {File: liveKey + ".m4a"},
		deadKey: {File: deadKey + ".m4a"},
	}}

	if got := prune(dir, m, []string{live}); got != 1 {
		t.Errorf("dropped %d, want 1", got)
	}
	if _, ok := m.Clips[liveKey]; !ok {
		t.Error("the clip still being said was dropped")
	}
	if _, ok := m.Clips[deadKey]; ok {
		t.Error("the orphaned clip is still in the manifest")
	}
	if _, err := os.Stat(dir + "/" + deadKey + ".m4a"); !os.IsNotExist(err) {
		t.Error("the orphaned clip is still on disk")
	}
	if _, err := os.Stat(dir + "/" + liveKey + ".m4a"); err != nil {
		t.Errorf("the live clip was deleted: %v", err)
	}
}

func TestExcerptCutsOnAWordBoundary(t *testing.T) {
	got := excerpt("The application has no credentials of its own to check.", 20)
	if strings.Contains(got, "credenti…") {
		t.Errorf("excerpt = %q, want a cut between words", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("excerpt = %q, want an ellipsis", got)
	}
	if short := excerpt("Short.", 20); short != "Short." {
		t.Errorf("excerpt = %q, want it left alone", short)
	}
	if flat := excerpt("wrapped\n  prose", 40); flat != "wrapped prose" {
		t.Errorf("excerpt = %q, want whitespace collapsed", flat)
	}
}
