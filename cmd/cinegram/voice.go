package main

// Recording narration.
//
// `voice` reads a document's prose and leaves a sidecar directory of clips
// beside it, which the loader then inlines into the timeline. Synthesis lives
// here in the command rather than in a pkg for the same reason `record` does:
// it shells out, and the packages stay pure.
//
// **No synthesizer is built in and none is depended on.** A TTS command is named
// by the environment, exactly as Chrome and ffmpeg are, so the choice of engine
// — macOS `say`, Piper, Kokoro, a curl to a hosted API — is the user's and costs
// this repository no dependency and no API key handling. On macOS there is a
// working default, so `cinegram voice x.dgm` does something useful with nothing
// installed and nothing configured.

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/ir"
	"github.com/tejaspanse/cinegram/pkg/loader"
	"github.com/tejaspanse/cinegram/pkg/voice"
)

// darwinSay is the default synthesizer on macOS.
//
// WAVE at 22 kHz because the duration then comes out of the header with the
// standard library alone — see voice.WAVMillis — and because compressing a WAV
// afterwards is easy while measuring a compressed clip is not. `say` reads the
// line from stdin when given no text of its own, which keeps the prose off the
// command line and out of reach of the shell.
const darwinSay = `say --file-format=WAVE --data-format=LEI16@22050 -o {out}`

// synthTimeout is per line. A synthesizer that has not produced a sentence in
// half a minute is wedged, and a wedged one would otherwise hang the whole run.
const synthTimeout = 30 * time.Second

func cmdVoice(args []string, stdout, stderr io.Writer) error {
	var (
		voiceName string
		force     bool
		keepWAV   bool
	)
	input, output, err := parseArgsWith("voice", args, func(fs *flag.FlagSet) {
		fs.StringVar(&voiceName, "voice", "", "voice to ask the synthesizer for")
		fs.BoolVar(&force, "force", false, "re-record lines that are already recorded")
		fs.BoolVar(&keepWAV, "keep-wav", false, "skip the AAC compression step")
	})
	if err != nil {
		return err
	}

	command, provider, err := ttsCommand(voiceName)
	if err != nil {
		return err
	}

	bundle, err := loader.Load(input, os.ReadFile)
	if err != nil {
		return err
	}
	timeline := compile.CompileBundle(bundle)
	if err := reportAll(bundle.Bags(), stderr); err != nil {
		return err
	}

	dir := output
	if dir == "" {
		dir = voice.DirFor(input)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// The manifest is read rather than rebuilt so that clips recorded on an
	// earlier run survive: re-recording every line on every edit would make a
	// hosted synthesizer expensive and a local one slow, for no gain.
	m := readManifest(dir)
	m.Version = voice.Version
	m.Provider = provider
	if voiceName != "" {
		m.Voice = voiceName
	}

	lines := linesOf(timeline)
	if len(lines) == 0 {
		return fmt.Errorf("%s has no `desc` prose to record", input)
	}

	// Compression is worth having on by default — narration is most of a page's
	// weight otherwise — but it must never be the reason the command fails, so a
	// machine with no ffmpeg simply keeps the WAV.
	encoder := ""
	if !keepWAV {
		if path, err := findFFmpeg(); err == nil {
			encoder = path
		} else {
			fmt.Fprintln(stderr, "cinegram: no ffmpeg, keeping uncompressed WAV clips")
		}
	}

	recorded, reused := 0, 0
	for _, line := range lines {
		key := voice.Key(line)
		if _, have := m.Clips[key]; have && !force {
			reused++
			continue
		}

		clip, err := synthesize(command, dir, key, line, encoder)
		if err != nil {
			return fmt.Errorf("recording %q: %w", excerpt(line, 40), err)
		}
		if m.Clips == nil {
			m.Clips = map[string]voice.Clip{}
		}
		m.Clips[key] = clip
		recorded++
		fmt.Fprintf(stderr, "cinegram: recorded %s (%.1fs) %s\n",
			clip.File, float64(clip.Ms)/1000, excerpt(line, 48))
	}

	// Clips whose prose is gone are dropped from the index and deleted. The key
	// is a hash of the words, so a rewritten sentence orphans its recording, and
	// a directory that only ever grew would quietly fill with the drafts of every
	// edit.
	dropped := prune(dir, m, lines)

	if err := writeManifest(dir, m); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s: %d recorded, %d reused, %d dropped\n", dir, recorded, reused, dropped)
	return nil
}

// ttsCommand resolves the synthesizer, mirroring findChrome and findFFmpeg: an
// explicit override wins, and being explicit about a broken one is an error
// rather than a silent fall back to a default the user did not ask for.
//
// The returned command is a shell line with `{out}` standing for the file to
// write. A shell is used on purpose — it lets a user configure a pipeline
// without this code growing an argument parser — and the prose never reaches
// it: every line is written to the process's stdin instead.
func ttsCommand(voiceName string) (command, provider string, err error) {
	if custom := os.Getenv("CINEGRAM_TTS_COMMAND"); custom != "" {
		if !strings.Contains(custom, "{out}") {
			return "", "", fmt.Errorf(
				"CINEGRAM_TTS_COMMAND must contain {out}, the file to write:\n" +
					`  export CINEGRAM_TTS_COMMAND='say --file-format=WAVE -o {out}'`)
		}
		return custom, "command", nil
	}
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("say"); err == nil {
			cmd := darwinSay
			if voiceName != "" {
				cmd += " -v " + shellQuote(voiceName)
			}
			return cmd, "say", nil
		}
	}
	return "", "", fmt.Errorf(
		"no text-to-speech command. Set CINEGRAM_TTS_COMMAND to one that writes\n" +
			"a WAV to {out} and reads its line from stdin, for example:\n" +
			`  export CINEGRAM_TTS_COMMAND='say --file-format=WAVE -o {out}'          # macOS` + "\n" +
			`  export CINEGRAM_TTS_COMMAND='piper --model en_US.onnx --output_file {out}'` + "\n" +
			"Any engine works; nothing about it is built in")
}

// voiceOpts turns the --with-voice flag into loader options, so the several
// commands that offer it all mean exactly the same thing by it.
func voiceOpts(with bool) []loader.Option {
	if !with {
		return nil
	}
	return []loader.Option{loader.WithVoice()}
}

// hasNarration reports whether any step ended up with a clip, which is how the
// commands tell "--with-voice and there was some" from "--with-voice and the
// sidecar is missing or empty". Asking for narration and silently getting none
// is the one failure worth a warning: nothing else about the page looks wrong.
func hasNarration(t *ir.Timeline) bool {
	for _, v := range t.Views {
		for _, sc := range v.Scenarios {
			for _, st := range sc.Steps {
				if st.Audio != "" {
					return true
				}
			}
		}
	}
	return false
}

// linesOf is every distinct piece of prose in the timeline, in the order a
// reader meets it.
//
// Distinct by text rather than by step, because two scenarios retelling the same
// sentence — or two steps that happen to say the same thing — are one recording.
// That falls straight out of keying clips by their words.
func linesOf(t *ir.Timeline) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range t.Views {
		for _, sc := range v.Scenarios {
			for _, st := range sc.Steps {
				text := strings.TrimSpace(st.Desc)
				if text == "" || seen[voice.Key(text)] {
					continue
				}
				seen[voice.Key(text)] = true
				out = append(out, text)
			}
		}
	}
	return out
}

// synthesize records one line, and compresses it when an encoder was found.
func synthesize(command, dir, key, text, encoder string) (voice.Clip, error) {
	wavPath := filepath.Join(dir, key+".wav")

	cmd := exec.Command("sh", "-c", strings.ReplaceAll(command, "{out}", shellQuote(wavPath)))
	cmd.Stdin = strings.NewReader(text)
	var errBuf strings.Builder
	cmd.Stderr = &errBuf

	if err := cmd.Start(); err != nil {
		return voice.Clip{}, fmt.Errorf("cannot run the synthesizer: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return voice.Clip{}, fmt.Errorf("%w\n%s", err, tail(errBuf.String()))
		}
	case <-time.After(synthTimeout):
		_ = cmd.Process.Kill()
		return voice.Clip{}, fmt.Errorf("the synthesizer did not finish within %s", synthTimeout)
	}

	data, err := os.ReadFile(wavPath)
	if err != nil {
		return voice.Clip{}, fmt.Errorf("the synthesizer wrote no file: %w", err)
	}

	// The duration is measured here, once, from the WAV — and then carried in the
	// manifest. That is what lets the clip be compressed into something whose
	// length the standard library could not work out.
	ms, ok := voice.WAVMillis(data)
	if !ok {
		return voice.Clip{}, fmt.Errorf(
			"the synthesizer did not write a WAV, so its length cannot be measured.\n" +
				"Ask it for RIFF/WAVE output — `pace: voice` needs the duration")
	}

	clip := voice.Clip{File: key + ".wav", MIME: "audio/wav", Ms: ms, Text: excerpt(text, 72)}
	if encoder == "" {
		return clip, nil
	}

	m4aPath := filepath.Join(dir, key+".m4a")
	enc := exec.Command(encoder, "-y", "-loglevel", "error", "-i", wavPath, "-c:a", "aac", "-b:a", "48k", m4aPath)
	if err := enc.Run(); err != nil {
		// A failed compression is not a failed recording: the WAV plays.
		return clip, nil
	}
	_ = os.Remove(wavPath)
	return voice.Clip{File: key + ".m4a", MIME: "audio/mp4", Ms: ms, Text: clip.Text}, nil
}

func readManifest(dir string) voice.Manifest {
	m := voice.Manifest{Version: voice.Version, Clips: map[string]voice.Clip{}}
	raw, err := os.ReadFile(filepath.Join(dir, voice.ManifestName))
	if err != nil {
		return m
	}
	// A corrupt manifest is rebuilt rather than reported: everything in it is
	// derived, and the clips it forgets are simply recorded again.
	var found voice.Manifest
	if err := json.Unmarshal(raw, &found); err != nil || found.Clips == nil {
		return m
	}
	return found
}

func writeManifest(dir string, m voice.Manifest) error {
	encoded, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, voice.ManifestName), append(encoded, '\n'), 0o644)
}

// prune forgets and deletes clips no line in the document asks for any more.
func prune(dir string, m voice.Manifest, lines []string) int {
	live := make(map[string]bool, len(lines))
	for _, line := range lines {
		live[voice.Key(line)] = true
	}

	stale := make([]string, 0, len(m.Clips))
	for key := range m.Clips {
		if !live[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale) // deterministic, so a run's output reads the same twice

	for _, key := range stale {
		_ = os.Remove(filepath.Join(dir, m.Clips[key].File))
		delete(m.Clips, key)
	}
	return len(stale)
}

// ---------------------------------------------------------------------------
// Narration in a recording
// ---------------------------------------------------------------------------

// cue is one line of narration, the moment in the video it starts at, and how
// long it goes on for.
type cue struct {
	path  string
	start int // ms from the beginning of the scenario
	ms    int // the clip's own length, as measured when it was recorded
}

// narration is the two halves of ffmpeg's audio wiring, which cannot be one
// slice: the `-i` inputs have to precede the codec flags that describe the
// output, and the filter graph has to come after every input it references.
type narration struct {
	inputs []string
	mix    []string
}

// clipsFor is the narration the loader read for one view's document, which is
// where the clip durations live — the timeline carries the bytes but not the
// lengths, since a renderer never needs them and the recorder does.
func clipsFor(b *loader.Bundle, viewID string) map[string]voice.Loaded {
	for _, u := range b.Units {
		if u.ViewID == viewID {
			return u.VoiceData
		}
	}
	return nil
}

// stepsOf finds the steps of one compiled scenario. pickScenario has already
// resolved the ids and reported anything missing, so a miss here is simply no
// narration rather than an error worth repeating.
func stepsOf(t *ir.Timeline, viewID, scenarioID string) []ir.Step {
	for _, v := range t.Views {
		if v.ID != viewID {
			continue
		}
		for _, sc := range v.Scenarios {
			if sc.ID == scenarioID {
				return sc.Steps
			}
		}
	}
	return nil
}

// writeCues decodes the clips the timeline carries into files ffmpeg can open.
//
// The bytes are already in hand as data URIs — the loader inlined them — so this
// unpacks what is loaded rather than reading the sidecar a second time. That
// keeps one answer to "which clip belongs to this step": the timeline's, the same
// one the page plays.
// The clip lengths come from clips, keyed as the sidecar keys them, because the
// timeline records which step says what but not how long saying it takes.
func writeCues(dir string, steps []ir.Step, clips map[string]voice.Loaded) ([]cue, error) {
	var cues []cue
	for i, st := range steps {
		if st.Audio == "" {
			continue
		}
		mime, payload, ok := splitDataURI(st.Audio)
		if !ok {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("narration for step %q is not decodable: %w", st.ID, err)
		}
		path := filepath.Join(dir, fmt.Sprintf("voice-%03d%s", i, extFor(mime)))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, err
		}
		cues = append(cues, cue{path: path, start: st.Start, ms: clips[voice.Key(st.Desc)].Ms})
	}
	return cues, nil
}

// narrationOverrun is how far past the end of the recording the narration would
// still be talking, in milliseconds.
//
// It answers the question the author actually has — "will my video cut the voice
// off?" — and it is worth asking because the two clocks are set independently: a
// step is as long as its animation needs, while its line is as long as it takes
// to say. A walkthrough written to be *watched* is routinely a fraction of the
// same walkthrough read *aloud*, so this is the common case rather than the edge
// one. `pace: voice` is what makes it zero.
func narrationOverrun(cues []cue, duration int) int {
	end := 0
	for _, c := range cues {
		if c.start+c.ms > end {
			end = c.start + c.ms
		}
	}
	if end > duration {
		return end - duration
	}
	return 0
}

// mixArgs lays the cues onto the recording's timeline.
//
// Each clip is delayed to its step's start and the delayed streams are mixed,
// rather than concatenated: concatenation would place clips back to back and so
// silently invent its own timing, while a delay puts every line exactly where the
// step it explains begins. `normalize=0` matters — amix divides by the number of
// inputs by default, so an eight-step walkthrough would come out eight times too
// quiet.
//
// `-shortest` makes the video the master. Narration longer than the animation is
// cut off, which is the right way round: the frames are what was recorded, and
// `pace: voice` is how an author asks for the animation to make room instead.
func mixArgs(cues []cue, format string) narration {
	if len(cues) == 0 {
		return narration{}
	}

	var out narration
	var filter strings.Builder
	labels := ""
	for i, c := range cues {
		out.inputs = append(out.inputs, "-i", c.path)
		// i+1 because input 0 is the frame sequence.
		fmt.Fprintf(&filter, "[%d:a]adelay=%d:all=1[n%d];", i+1, c.start, i)
		labels += fmt.Sprintf("[n%d]", i)
	}
	fmt.Fprintf(&filter, "%samix=inputs=%d:normalize=0[narration]", labels, len(cues))

	codec := "aac"
	if format == "webm" {
		codec = "libopus" // aac in a webm container plays in nothing
	}
	out.mix = []string{
		"-filter_complex", filter.String(),
		"-map", "0:v", "-map", "[narration]",
		"-c:a", codec, "-shortest",
	}
	return out
}

// splitDataURI takes `data:<mime>;base64,<payload>` apart.
func splitDataURI(s string) (mime, payload string, ok bool) {
	if !strings.HasPrefix(s, "data:") {
		return "", "", false
	}
	comma := strings.Index(s, ",")
	if comma < 0 {
		return "", "", false
	}
	head := s[len("data:"):comma]
	return strings.TrimSuffix(head, ";base64"), s[comma+1:], true
}

// extFor is the file extension ffmpeg should see for a clip's type. It only has
// to cover what `voice` writes, and an unknown type still gets a container guess
// from ffmpeg's own probing.
func extFor(mime string) string {
	switch mime {
	case "audio/wav":
		return ".wav"
	case "audio/mp4", "audio/aac":
		return ".m4a"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	default:
		return ".bin"
	}
}

// excerpt shortens a line for a message, on a word boundary where it can.
func excerpt(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if i := strings.LastIndex(cut, " "); i > max/2 {
		cut = cut[:i]
	}
	return cut + "…"
}

// shellQuote makes a path safe inside the single-quoted shell word it is
// substituted into.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
