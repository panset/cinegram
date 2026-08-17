package site

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

// mem builds an fs.FS over an in-memory tree, paths relative to examples/.
func mem(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for p, content := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

const plainExample = `%% A tiny demo.
flowchart LR
  a[One] --> b[Two]

scenario "run" { speed: 1.0 }

  step go "Go" {
    flow a -> b { dur: 500ms }
  }
`

// A README with the shape the splitter cares about: the pitch above the first
// ##, a heading per guide page, a cross-reference between two of them, a link
// into the repository, and a shell block whose comment starts with ##.
func fakeReadme() []byte {
	var b strings.Builder
	b.WriteString("# Cinegram\n\nThe pitch, which the home page does better.\n\n")
	for i, p := range guidePages {
		for _, s := range p.Sections {
			fmt.Fprintf(&b, "## %s\n\nProse for %s.\n\n", s, s)
			if i == 0 {
				b.WriteString("```sh\ncinegram preview a.dgm\n## then open it\n```\n\n")
			}
		}
	}
	return []byte(b.String())
}

func build(t *testing.T, files map[string]string) map[string][]byte {
	t.Helper()
	out, _, err := Build(mem(files), fakeReadme())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return out
}

func page(t *testing.T, out map[string][]byte, rel string) string {
	t.Helper()
	content, ok := out[rel]
	if !ok {
		t.Fatalf("no %s in the output; have %v", rel, sortedKeys(out))
	}
	return string(content)
}

// --- the examples tour ----------------------------------------------------

func TestAnExamplePageMountsItsOwnTimeline(t *testing.T) {
	out := build(t, map[string]string{"01-basics/01-plain.dgm": plainExample})

	// The tour keeps the source's ordering prefix in the page name: Zensical
	// sorts filenames to build its navigation, and the tour has an order.
	md := page(t, out, "examples/01-basics/01-plain.md")
	if !strings.Contains(md, `data-cinegram="01-basics/01-plain"`) {
		t.Errorf("the page names no timeline:\n%s", md)
	}
	// The name in the div and the file on disk are the same string with
	// .json on the end; nothing at build time checks that at read time, so
	// it is checked here.
	name := "01-basics/01-plain"
	if _, ok := out[TimelineDir+"/"+name+".json"]; !ok {
		t.Errorf("no timeline written for %s; have %v", name, sortedKeys(out))
	}
	if !strings.Contains(md, "A tiny demo.") {
		t.Error("the page does not carry the example's blurb")
	}
	if !strings.Contains(md, "```dgm") {
		t.Error("the page does not show the source that produced it")
	}
}

func TestTheTimelineIsTheCompiledDocument(t *testing.T) {
	out := build(t, map[string]string{"plain.dgm": plainExample})

	var timeline struct {
		Version int    `json:"version"`
		Root    string `json:"root"`
		Views   []struct {
			ID string `json:"id"`
		} `json:"views"`
	}
	if err := json.Unmarshal(out[TimelineDir+"/plain.json"], &timeline); err != nil {
		t.Fatalf("the timeline is not valid JSON: %v", err)
	}
	if timeline.Version == 0 || timeline.Root == "" || len(timeline.Views) == 0 {
		t.Errorf("the timeline is empty: %+v", timeline)
	}
}

// The player finds its timeline relative to the loader's own URL, so depth
// costs it nothing. The playground link is an ordinary href and does not get
// that for free.
func TestThePlaygroundLinkClimbsOutOfNestedFolders(t *testing.T) {
	out := build(t, map[string]string{
		"top.dgm":           plainExample,
		"a/deep/nested.dgm": plainExample,
	})

	if md := page(t, out, "examples/top.md"); !strings.Contains(md, "(../playground/#doc=") {
		t.Errorf("a top-level example does not reach the playground one level up:\n%s", md)
	}
	if md := page(t, out, "examples/a/deep/nested.md"); !strings.Contains(md, "(../../../playground/#doc=") {
		t.Errorf("a nested example does not climb far enough:\n%s", md)
	}
}

func TestFolderIndexesListWhatIsInThem(t *testing.T) {
	out := build(t, map[string]string{
		"01-basics/01-plain.dgm": plainExample,
		"02-more/01-other.dgm":   plainExample,
	})

	root := page(t, out, "examples/index.md")
	for _, want := range []string{"(01-basics/index.md)", "(02-more/index.md)"} {
		if !strings.Contains(root, want) {
			t.Errorf("the tour index does not link %s:\n%s", want, root)
		}
	}
	inner := page(t, out, "examples/01-basics/index.md")
	if !strings.Contains(inner, "(01-plain.md)") {
		t.Errorf("a folder index does not link its own pages:\n%s", inner)
	}
	if !strings.Contains(inner, "A tiny demo.") {
		t.Error("a folder index does not carry blurbs")
	}
}

func TestPrevNextLinksResolveAcrossFolders(t *testing.T) {
	out := build(t, map[string]string{
		"01-basics/01-plain.dgm": plainExample,
		"02-more/01-other.dgm":   plainExample,
	})
	// From 01-basics/plain.md, the next page is one folder up and one down.
	if md := page(t, out, "examples/01-basics/01-plain.md"); !strings.Contains(md, "(../02-more/01-other.md)") {
		t.Errorf("next does not resolve across the folder edge:\n%s", md)
	}
	if md := page(t, out, "examples/02-more/01-other.md"); !strings.Contains(md, "(../01-basics/01-plain.md)") {
		t.Errorf("prev does not resolve across the folder edge:\n%s", md)
	}
}

func TestEveryGeneratedFileIsUnderASweptFolder(t *testing.T) {
	// The sweep only deletes inside site.Generated. Anything written outside
	// it would survive its source being removed, forever.
	out := build(t, map[string]string{"plain.dgm": plainExample})
	for _, rel := range sortedKeys(out) {
		swept := false
		for _, gen := range Generated {
			if strings.HasPrefix(rel, gen+"/") {
				swept = true
				break
			}
		}
		if !swept {
			t.Errorf("%s is generated but sits outside %v, so nothing will ever delete it", rel, Generated)
		}
	}
}

// --- the guide ------------------------------------------------------------

func TestTheGuideIsTheReadmeCutUp(t *testing.T) {
	out := build(t, map[string]string{"plain.dgm": plainExample})

	md := page(t, out, "guide/01-language.md")
	if !strings.Contains(md, "# The language\n") {
		t.Error("the page has no title of its own")
	}
	for _, s := range guidePages[0].Sections {
		if !strings.Contains(md, "## "+s+"\n") {
			t.Errorf("the language page is missing the %q section", s)
		}
	}
	// The pitch above the README's first ## is the home page's job.
	if strings.Contains(md, "The pitch") {
		t.Error("the guide carried the README's opening pitch onto a page")
	}
	// A ## inside a fenced block is a shell comment, not a heading.
	if strings.Contains(md, "# then open it\n") && !strings.Contains(md, "## then open it") {
		t.Error("a ## inside a code fence was treated as a heading")
	}
}

func TestAReadmeHeadingNobodyClaimsFailsTheBuild(t *testing.T) {
	readme := append(fakeReadme(), []byte("## Something new\n\nAdded last week.\n")...)
	_, _, err := Build(mem(map[string]string{"plain.dgm": plainExample}), readme)
	if err == nil {
		t.Fatal("Build accepted a README section that appears on no page")
	}
	if !strings.Contains(err.Error(), "Something new") {
		t.Errorf("the error does not name the orphaned section: %v", err)
	}
}

func TestAPageWantingAMissingSectionFailsTheBuild(t *testing.T) {
	// The mirror image: the README was reorganised out from under the table.
	readme := strings.Replace(string(fakeReadme()), "## Commands\n", "## CLI\n", 1)
	_, _, err := Build(mem(map[string]string{"plain.dgm": plainExample}), []byte(readme))
	if err == nil {
		t.Fatal("Build accepted a page whose section no longer exists")
	}
	if !strings.Contains(err.Error(), "Commands") {
		t.Errorf("the error does not name the missing section: %v", err)
	}
}

func TestLinksAreRewrittenForASite(t *testing.T) {
	// One of each case the README actually contains.
	readme := strings.Replace(string(fakeReadme()),
		"Prose for Commands.",
		"See [failure paths](#failure-paths), [the licence](LICENSE) and [upstream](https://example.com/x).",
		1)
	out, _, err := Build(mem(map[string]string{"plain.dgm": plainExample}), []byte(readme))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	md := string(out["guide/05-commands.md"])
	// "Failure paths" is on the storytelling page, so the anchor has to grow
	// a page in front of it.
	if !strings.Contains(md, "(02-storytelling.md#failure-paths)") {
		t.Errorf("a cross-page anchor was not repointed:\n%s", md)
	}
	// The site does not serve the repository.
	if !strings.Contains(md, "("+repoBlob+"LICENSE)") {
		t.Errorf("a repository path was not sent to GitHub:\n%s", md)
	}
	if !strings.Contains(md, "(https://example.com/x)") {
		t.Error("an absolute link was rewritten")
	}
}

func TestAnAnchorToNothingFailsTheBuild(t *testing.T) {
	readme := strings.Replace(string(fakeReadme()),
		"Prose for Commands.", "See [gone](#no-such-heading).", 1)
	_, _, err := Build(mem(map[string]string{"plain.dgm": plainExample}), []byte(readme))
	if err == nil {
		t.Fatal("Build accepted a link to an anchor no heading generates")
	}
	if !strings.Contains(err.Error(), "no-such-heading") {
		t.Errorf("the error does not name the dead anchor: %v", err)
	}
}
