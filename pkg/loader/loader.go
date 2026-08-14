// Package loader resolves a document and everything it references into one
// bundle.
//
// The parser is pure — content in, syntax tree out — so that it stays testable
// without a filesystem and usable from a webview or a WASM build that has no
// disk at all. Every path that a `view` declaration names is therefore resolved
// here instead, and reading is injected rather than assumed, which keeps that
// property intact one level up.
//
// A view's identity is its resolved path, not the alias a document happened to
// give it. Two files that reference the same diagram under different names
// produce one view, so a diamond costs nothing and a cycle terminates on its
// own.
package loader

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/parser"
	"github.com/tejaspanse/cinegram/pkg/source"
	"github.com/tejaspanse/cinegram/pkg/voice"
)

// ReadFileFunc reads a source file. os.ReadFile satisfies it.
type ReadFileFunc func(path string) ([]byte, error)

// Option adjusts what Load pulls in besides the documents themselves.
type Option func(*config)

type config struct{ voice bool }

// WithVoice inlines the narration sidecar beside each document, if it has one.
//
// Off by default, and the default is the important half. A recorded document's
// clips are megabytes — far more than the diagram, the timeline and the runtime
// together — so a page that embeds them has to be asked for. Leaving it off also
// keeps compilation a pure function of the checked-in sources: the demo site
// regenerates to the same bytes whether or not the person running it happens to
// have recorded narration locally, which is what keeps //site:site_test honest.
//
// A page built without this still narrates. The prose is in every step's Desc,
// and speaking it is something the browser can do unaided.
func WithVoice() Option { return func(c *config) { c.voice = true } }

// Unit is one parsed source file in a bundle.
type Unit struct {
	ViewID string // canonical id, derived from the path
	Title  string // from the declaration that pulled this file in, or ""
	Path   string // cleaned path on disk
	Result *parser.Result
	Bag    *diag.Bag // this file's diagnostics, labelled with its path

	// Views maps each local alias this document declared to the canonical
	// ViewID it resolved to. Compilation rewrites bindings through it.
	Views map[string]string

	// FrameData maps each storyboard `img` path, exactly as written, to a
	// self-contained `data:` URI. Inlining happens here for the same reason
	// view paths resolve here: the parser never touches a filesystem, and the
	// emitted page has to work from one anyway.
	FrameData map[string]string

	// VoiceData maps voice.Key(prose) to the recording of that prose, inlined
	// the same way and for the same reasons as FrameData.
	//
	// It is empty unless a sidecar directory sits beside this document, and that
	// absence is the opt-in: narration is only ever inlined into a page whose
	// author ran `cinegram voice`, so no ordinary document pays for audio it
	// never asked for.
	VoiceData map[string]voice.Loaded
}

// Bundle is every document reachable from an entry file.
type Bundle struct {
	Root  string // canonical ViewID of the entry document
	Units []*Unit
}

// Bags returns every unit's diagnostics, root first.
func (b *Bundle) Bags() []*diag.Bag {
	out := make([]*diag.Bag, 0, len(b.Units))
	for _, u := range b.Units {
		out = append(out, u.Bag)
	}
	return out
}

// HasErrors reports whether any unit failed to parse or validate.
func (b *Bundle) HasErrors() bool {
	for _, u := range b.Units {
		if u.Bag.HasErrors() {
			return true
		}
	}
	return false
}

// Load reads path and every document reachable from it, breadth-first.
//
// An unreadable reference is reported against the `view` declaration that named
// it and loading continues, so one run surfaces every broken path rather than
// stopping at the first. Only a failure to read the entry file itself is
// returned as an error.
func Load(path string, readFile ReadFileFunc, opts ...Option) (*Bundle, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	root := filepath.Clean(path)

	content, err := readFile(root)
	if err != nil {
		return nil, err
	}

	ids := newIDSet()
	byPath := map[string]*Unit{}

	rootUnit := parseUnit(root, ids.assign(root, root), "", content)
	byPath[root] = rootUnit

	bundle := &Bundle{Root: rootUnit.ViewID, Units: []*Unit{rootUnit}}

	// Breadth-first over the queue rather than recursion: the queue grows as
	// units are parsed, and dedup by path means a cycle simply finds its
	// target already loaded.
	for i := 0; i < len(bundle.Units); i++ {
		u := bundle.Units[i]
		loadFrames(u, readFile)
		if cfg.voice {
			loadVoice(u, readFile)
		}

		for _, decl := range u.Result.Document.Views {
			if decl.Path == "" || filepath.IsAbs(decl.Path) {
				continue // already reported by validation
			}
			target := filepath.Clean(filepath.Join(filepath.Dir(u.Path), decl.Path))

			if seen, ok := byPath[target]; ok {
				u.Views[decl.ID] = seen.ViewID
				continue
			}

			data, err := readFile(target)
			if err != nil {
				u.Bag.ErrorHintf(decl.PathAt, hintFor(err),
					"cannot read view %q from %s", decl.ID, decl.Path)
				continue
			}

			sub := parseUnit(target, ids.assign(target, root), decl.Title, data)
			byPath[target] = sub
			bundle.Units = append(bundle.Units, sub)
			u.Views[decl.ID] = sub.ViewID
		}
	}

	return bundle, nil
}

func parseUnit(path, viewID, title string, content []byte) *Unit {
	res, bag := parser.Parse(path, string(content))
	return &Unit{
		ViewID:    viewID,
		Title:     title,
		Path:      path,
		Result:    res,
		Bag:       bag,
		Views:     map[string]string{},
		FrameData: map[string]string{},
		VoiceData: map[string]voice.Loaded{},
	}
}

// loadVoice inlines whatever narration sits beside this unit.
//
// The sidecar is found at a known path rather than by listing a directory, which
// keeps the loader's whole contact with the filesystem inside the readFile it was
// handed — a missing manifest is an ordinary read error and simply means this
// document has no recorded narration.
//
// Nothing here is fatal. A document narrates or it does not; a half-written
// sidecar should cost the author a warning and a page that still animates, not a
// build. The one thing worth an error is a manifest from a *newer* format, since
// carrying on would mean guessing at fields this binary does not know.
func loadVoice(u *Unit, readFile ReadFileFunc) {
	dir := voice.DirFor(u.Path)

	raw, err := readFile(filepath.Join(dir, voice.ManifestName))
	if err != nil {
		return // no sidecar: the common case, and not a problem
	}

	var m voice.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		u.Bag.WarnHintf(source.Pos{Line: 1, Col: 1}, "re-run `cinegram voice` on this file to rebuild it",
			"the narration manifest in %s is unreadable: %v", dir, err)
		return
	}
	if m.Version > voice.Version {
		u.Bag.ErrorHintf(source.Pos{Line: 1, Col: 1}, "upgrade cinegram, or delete "+dir+" and re-record it",
			"the narration in %s is version %d and this binary understands %d", dir, m.Version, voice.Version)
		return
	}

	for key, clip := range m.Clips {
		if clip.File == "" {
			continue
		}
		mime := clip.MIME
		if mime == "" {
			mime = voice.MIMEFor[strings.ToLower(filepath.Ext(clip.File))]
		}
		if mime == "" {
			u.Bag.WarnHintf(source.Pos{Line: 1, Col: 1}, "re-record it as .wav or .m4a",
				"narration clip %s has an audio type this cannot inline", clip.File)
			continue
		}

		data, err := readFile(filepath.Clean(filepath.Join(dir, clip.File)))
		if err != nil {
			u.Bag.WarnHintf(source.Pos{Line: 1, Col: 1}, "re-run `cinegram voice` on this file",
				"cannot read narration clip %s: %v", clip.File, err)
			continue
		}

		u.VoiceData[key] = voice.Loaded{
			Data: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
			Ms:   clip.Ms,
		}
	}
}

// frameMIME maps an image extension onto the type a data URI has to declare.
//
// The set is closed rather than guessed at: a browser shown `data:;base64,...`
// renders nothing at all, and a frame that silently fails to appear is exactly
// the failure a storyboard cannot afford. Anything else is an error naming what
// is supported.
var frameMIME = map[string]string{
	".svg":  "image/svg+xml",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// loadFrames inlines every storyboard image this unit declares.
//
// An unreadable image is reported against the `img` attribute and loading
// continues, the same way a broken `view` path is: one run should surface every
// missing frame rather than stopping at the first.
func loadFrames(u *Unit, readFile ReadFileFunc) {
	dir := filepath.Dir(u.Path)

	for _, sb := range u.Result.Document.Storyboards {
		for _, f := range sb.Frames {
			if f.Img == "" {
				continue // caption-only, which is allowed
			}
			if _, done := u.FrameData[f.Img]; done {
				continue // two frames may legitimately show the same picture
			}

			mime, ok := frameMIME[strings.ToLower(filepath.Ext(f.Img))]
			if !ok {
				u.Bag.ErrorHintf(f.ImgAt, "supported image types: "+strings.Join(frameTypes(), ", "),
					"frame %q has an image type this cannot inline: %q", f.Name, f.Img)
				continue
			}

			data, err := readFile(filepath.Clean(filepath.Join(dir, f.Img)))
			if err != nil {
				u.Bag.ErrorHintf(f.ImgAt, hintFor(err),
					"cannot read the image for frame %q from %s", f.Name, f.Img)
				continue
			}

			// base64 cannot produce a `<`, so the payload is safe inside the
			// <script> element the timeline is embedded in.
			u.FrameData[f.Img] = "data:" + mime + ";base64," +
				base64.StdEncoding.EncodeToString(data)
		}
	}
}

func frameTypes() []string {
	out := make([]string, 0, len(frameMIME))
	for ext := range frameMIME {
		out = append(out, ext)
	}
	sort.Strings(out)
	return out
}

func hintFor(err error) string {
	if errors.Is(err, fs.ErrNotExist) {
		return "check the path is relative to the file that declares the view"
	}
	return fmt.Sprintf("%v", err)
}

// idSet hands out canonical view ids, keeping them stable and unique because
// they end up in the page's location.hash.
type idSet struct {
	taken map[string]bool
}

func newIDSet() *idSet { return &idSet{taken: map[string]bool{}} }

// assign derives an id from path relative to the root document's directory, so
// a sibling file reads as "pod-a" rather than a full path. Collisions get a
// numeric suffix rather than an error: two files may legitimately share a name
// in different directories.
func (s *idSet) assign(path, root string) string {
	rel, err := filepath.Rel(filepath.Dir(root), path)
	if err != nil {
		rel = filepath.Base(path)
	}
	base := slug(strings.TrimSuffix(rel, filepath.Ext(rel)))
	if base == "" {
		base = "view"
	}

	id := base
	for n := 2; s.taken[id]; n++ {
		id = base + "-" + strconv.Itoa(n)
	}
	s.taken[id] = true
	return id
}

// slug reduces a relative path to something safe in a URL fragment.
func slug(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
