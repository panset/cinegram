package html

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/ir"
)

func sample() *ir.Timeline {
	return &ir.Timeline{
		Version: ir.Version,
		Root:    "main",
		Views: []ir.View{{
			ID:      "main",
			Diagram: ir.Diagram{Type: "flowchart", Direction: "LR", Mermaid: "flowchart LR\n  a --> b\n"},
			Nodes:   []ir.Node{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}},
			Edges:   []ir.Edge{{ID: "e0", From: "a", To: "b"}},
			Scenarios: []ir.Scenario{{
				ID: "s0", Name: "demo", Duration: 600, Speed: 1,
				Steps: []ir.Step{{
					ID: "s", Name: "hop", Start: 0, End: 600,
					Tracks: []ir.Track{{Kind: ir.TrackFlow, Start: 0, End: 600, Edge: "e0"}},
				}},
			}},
		}},
	}
}

// TestPageIsSelfContained is the property the preview depends on: the page must
// work from the filesystem and inside a CSP-restricted VS Code webview, so it
// may not reference any external resource.
func TestPageIsSelfContained(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := string(page)

	external := regexp.MustCompile(`(?i)(src|href)\s*=\s*["']?(https?:)?//`)
	if m := external.FindString(got); m != "" {
		t.Errorf("page references an external resource: %q", m)
	}

	for _, want := range []string{"<!doctype html>", "globalThis[\"mermaid\"]", "window.Cinegram", "CINEGRAM_TIMELINE"} {
		if !strings.Contains(got, want) {
			t.Errorf("page is missing %q", want)
		}
	}
}

// TestPageClaimsTheDocument pins the other half of the arrangement that lets one
// stylesheet serve both hosts.
//
// runtime.css scopes everything that lays out a whole page — the body
// background, the full-viewport height — behind .dgm-standalone, so that a host
// which contributes the sheet into a document it already owns does not have its
// prose restyled to match the diagram. The emitted page is that host's opposite
// number: it must actually say so, or it loses its own layout.
func TestPageClaimsTheDocument(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<body class="dgm-standalone">`) {
		t.Error("the page must mark itself standalone, or runtime.css lays out nothing")
	}
	if !strings.Contains(string(runtimeCSS), ".dgm-standalone") {
		t.Error("runtime.css no longer scopes its page rules; the class the page sets now means nothing")
	}
	if strings.Contains(string(runtimeCSS), "\nbody {") {
		t.Error("runtime.css styles a bare body again, which would restyle any host that loads it")
	}
}

// TestPageIsSizedForAPhone pins the other half of the responsive contract.
//
// runtime.css carries the narrow-screen layout — the single column, the
// picture-in-picture storyboard, the phone-sized present mode — and none of it
// can ever fire without this tag: a mobile browser without it lays the page out
// at ~980px and scales the result down, so the media queries see a desktop
// width and the reader gets an unreadable miniature of the wide layout.
func TestPageIsSizedForAPhone(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<meta name="viewport"`) {
		t.Error("the page has no viewport meta; every narrow-screen rule in runtime.css is dead on a phone")
	}
}

// TestScriptPayloadCannotEscape checks that diagram text containing a closing
// script tag is neutralised rather than breaking out of the script element.
func TestScriptPayloadCannotEscape(t *testing.T) {
	tl := sample()
	tl.Views[0].Diagram.Mermaid = "flowchart LR\n  a[\"</script><script>alert(1)</script>\"] --> b\n"

	page, err := Render(tl, Options{Title: "esc"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(page)

	// Exactly the four script elements we emit: the theme boot script, mermaid,
	// the runtime, the bootstrap that mounts the player.
	if n := strings.Count(got, "</script>"); n != 4 {
		t.Errorf("found %d closing script tags, want 4 — the payload escaped", n)
	}
	if strings.Contains(got, "<script>alert(1)") {
		t.Error("injected script survived unescaped")
	}
}

// TestTitleFallsBackToScenario keeps the browser tab useful when no title is
// passed in.
func TestTitleFallsBackToScenario(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "<title>demo</title>") {
		t.Error("expected the title to fall back to the first scenario name")
	}
}

// --- the page's theme control ---------------------------------------------

// TestThePageCarriesItsOwnThemeControl pins where dark and light are decided.
//
// The control is a property of the page, not of a diagram on it: it lives
// outside #cinegram because the player empties that element on every render —
// a theme switch itself causes one — and a control that disappeared when it
// was pressed would be no control at all. One per page, because two would be
// two views of one state, free to disagree.
func TestThePageCarriesItsOwnThemeControl(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := string(page)

	// The control as markup, not merely the marker: runtime.js is inlined into
	// this page and names the marker itself, three times, to find the control
	// and wire it.
	button := strings.TrimSpace(ThemeToggleHTML())
	if n := strings.Count(got, button); n != 1 {
		t.Fatalf("the page carries %d theme controls, want exactly 1", n)
	}
	toggle := strings.Index(got, button)
	player := strings.Index(got, `id="cinegram"`)
	if player < 0 {
		t.Fatal("the page has no player element at all")
	}
	if toggle > player {
		t.Error("the theme control is emitted after the player element, which means it is " +
			"inside or below it; the player empties its root on every render, so the " +
			"control has to sit outside #cinegram to survive being pressed")
	}
}

// TestTheThemeBootScriptBeatsTheFirstPaint is about a flash, so only the order
// says anything.
//
// runtime.js loads at the end of the body: by the time it could stamp the root
// element the browser has already painted a light page, and a reader who chose
// dark watches it flip. The three lines that read the preference therefore have
// to stand in the head, ahead of the player and ahead of every script.
func TestTheThemeBootScriptBeatsTheFirstPaint(t *testing.T) {
	page, err := Render(sample(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := string(page)

	boot := strings.Index(got, ThemeBootScript())
	if boot < 0 {
		t.Fatal("the page carries no theme boot script; a page opened with dark remembered " +
			"paints white first and then flips")
	}
	if head := strings.Index(got, "</head>"); boot > head {
		t.Error("the theme boot script stands outside the head, so the first paint has already " +
			"happened by the time it runs")
	}
	if player := strings.Index(got, `id="cinegram"`); boot > player {
		t.Error("the theme boot script comes after the player element; the page paints before it runs")
	}
	if runtime := strings.Index(got, "window.Cinegram"); runtime >= 0 && boot > runtime {
		t.Error("the theme boot script comes after runtime.js, which is exactly the ordering it " +
			"exists to avoid — the runtime loads at the end of the body")
	}
}

// TestASitedPageDoesNotDoubleTheControl is the other half of the "exactly one"
// rule, for the branch a site generator takes.
//
// A page rendered with site chrome has a header of its own, and pkg/sitegen
// puts the control there among the page's other actions. If Render emitted one
// too, every page of a generated site would show two switches for one page-wide
// state, and the one the reader did not press would sit there lying.
func TestASitedPageDoesNotDoubleTheControl(t *testing.T) {
	page, err := Render(sample(), Options{Nav: []byte("<nav class=\"dgm-site-nav\"></nav>\n")})
	if err != nil {
		t.Fatal(err)
	}
	got := string(page)

	if n := strings.Count(got, strings.TrimSpace(ThemeToggleHTML())); n != 0 {
		t.Errorf("a sited page emitted %d theme controls of its own; the site header owns "+
			"that control, and two of them is a page that can disagree with itself", n)
	}
	// The boot script is not the control, and a sited page needs it just as
	// much: it decides the palette of the first paint, which nothing else can.
	if !strings.Contains(got, ThemeBootScript()) {
		t.Error("a sited page carries no theme boot script, so it flashes the wrong palette " +
			"before the stylesheet's default is overridden")
	}
}

// TestTheRailDoesNotOwnTheTheme is the mechanical proof that the control moved.
//
// The rail's theme button was a page-scoped switch inside one diagram's tools:
// in a VS Code webview it overrode the editor, and two players on one page
// fought over the one attribute. Neither is fixed by moving the button — both
// are fixed by the player no longer being the thing that decides.
func TestTheRailDoesNotOwnTheTheme(t *testing.T) {
	js := string(runtimeJS)

	if strings.Contains(js, "themeBtn") {
		t.Error("runtime.js still builds a theme button in the rail; the theme is the page's, " +
			"and a page-scoped control inside a diagram's tools is the defect this replaced")
	}
	// The storage key belongs to the page chrome now, and to nothing else. One
	// mention is the single declaration the control reads it through; a second
	// is a second owner, which is how the key got out of step with the
	// attribute in the first place.
	if n := strings.Count(js, "'dgm.theme'"); n != 1 {
		t.Errorf("runtime.js mentions 'dgm.theme' %d times, want exactly 1 — the key has one "+
			"owner, the page's theme control, and the player reads the root element instead", n)
	}
	// And the player watches rather than decides.
	if !strings.Contains(js, "attributeFilter: ['data-theme']") {
		t.Error("runtime.js no longer observes data-theme; a player that does not watch the " +
			"attribute cannot follow page chrome, an editor or a site's palette toggle")
	}
}

// TestTheThemeControlIsLightAndDark pins the control's whole vocabulary.
//
// A press moves through three: the side you are not seeing, the side you were,
// and following the system again. Following is the state every reader arrives in
// — no data-theme at all, so runtime.css's prefers-color-scheme rules answer and
// an OS switch moves the page live — and it used to be unreachable once a side
// was pinned, which made clearing localStorage the only way back to it. A pinned
// side still means pinned: a reader who asked for dark meant dark, sunrise
// included. Letting go is now a press.
func TestTheThemeControlCyclesThroughSystem(t *testing.T) {
	js := string(runtimeJS)

	// Following the system is the *absence* of the attribute. A
	// `data-theme="system"` would match neither palette in the stylesheet and
	// leave the page with none, so the write has to be a removal.
	if !strings.Contains(js, "if (state === 'system') document.documentElement.removeAttribute('data-theme');") {
		t.Error("chooseTheme does not remove data-theme for 'system'; stamping the word would " +
			"match neither palette and leave the page unstyled")
	}
	if !strings.Contains(js, "'theme-system'") {
		t.Error("ICONS has no glyph for the system state, so the control cannot show which of " +
			"three it is in")
	}

	// The one thing that made a third state look dangerous, held to: both
	// readers collapse every value that is not a side to "follow the system", so
	// there is no second resolution of 'system' to disagree with the first. The
	// boot script must keep testing for the two sides only.
	if !strings.Contains(ThemeBootScript(), "if (t === 'light' || t === 'dark')") {
		t.Error("the boot script no longer tests for exactly the two sides; it and themeChoice " +
			"agree about 'system' only by both treating anything else as following the system")
	}
	if !strings.Contains(js, "return v === 'light' || v === 'dark' ? v : null;") {
		t.Error("themeChoice no longer reads a stored 'system' as null, so the attribute the " +
			"boot script removed and the glyph the control draws can now disagree")
	}

	// The cycle is derived from what is on screen, not a fixed list. A fixed
	// order would send the first press from "following" to whichever side came
	// first, which half the time is the side already showing — a button that
	// looks like it does nothing.
	if !strings.Contains(js, "chooseTheme(nextTheme());") {
		t.Error("the control's press does not go through nextTheme, so the cycle is no longer " +
			"computed from the palette on screen")
	}
	if !strings.Contains(js, "if (!pinned) return other;") {
		t.Error("nextTheme does not leave 'following' for the side that is not showing; the " +
			"first press would then pin the palette already on screen half the time")
	}
	if !strings.Contains(js, "return 'system';") {
		t.Error("nextTheme never returns to 'system', which is the whole point: a pinned side " +
			"was otherwise a dead end short of clearing localStorage")
	}
	if !strings.Contains(js, "return themeChoice() || (systemDark() ? 'dark' : 'light');") {
		t.Error("nothing resolves the effective theme from the stored choice and the system; " +
			"without it the press above has nothing truthful to flip")
	}

	// And the glyph has to stay honest while nothing is stored — the one case
	// with no data-theme mutation for anyone to observe. The stylesheet
	// re-resolves on an OS switch and says nothing about it, so the button
	// listens to the same media query itself, and stops trusting it the moment
	// a choice exists.
	if !strings.Contains(js, "function watchSystemTheme(btn) {") {
		t.Error("the theme control does not watch prefers-color-scheme; while nothing is stored " +
			"an OS switch repaints the page through the stylesheet alone, and the glyph is left " +
			"drawing the palette that has just gone")
	}
	if !strings.Contains(js, "if (themeChoice()) return;") {
		t.Error("the media-query listener has no stored-choice guard, so it would redraw the " +
			"glyph from a system the page stopped obeying at the first press")
	}
	if !strings.Contains(js, "watchSystemTheme(btn);") {
		t.Error("watchSystemTheme is never called on the button it is written for")
	}
}

// --- the rail thins out ----------------------------------------------------

// TestTheMapCarriesItsOwnFitControl pins where "fit the diagram" now lives.
//
// The minimap is on screen exactly while the stage shows less than the whole
// diagram — syncMap decides that from the transform alone — which is exactly
// when fitting the diagram means anything. Putting the control inside the map
// is what makes those two facts one fact: the button cannot be present and
// useless, and it needs no visibility rule of its own. Take it out of the map
// and it becomes what it was in the rail, a target that is dead most of the
// time.
func TestTheMapCarriesItsOwnFitControl(t *testing.T) {
	js, css := string(runtimeJS), string(runtimeCSS)

	if !strings.Contains(js, "dgm-map-fit") {
		t.Error("runtime.js builds no dgm-map-fit control; with the rail's Fit button gone " +
			"there is now no pointer-driven way back to the whole diagram at all")
	}
	if !strings.Contains(css, ".dgm-map-fit {") {
		t.Error("runtime.css does not style .dgm-map-fit, so the control is an unpositioned " +
			"button sitting on top of the thumbnail it is meant to sit in the corner of")
	}
	// The control is only as good as the box it inherits its visibility from,
	// and .dgm-map.is-off is what takes that box away.
	if !strings.Contains(css, ".dgm-map.is-off") {
		t.Error("runtime.css no longer hides the minimap when it is off; the fit control is " +
			"inside the map precisely so that one rule governs both")
	}
	// A focusable button inside an aria-hidden subtree is an error, not a
	// nuance: a reader can tab to it and hear nothing there. The map used to
	// carry the attribute whole; it may not any more, because it holds a
	// control now, so the attribute belongs on the decorative children.
	if strings.Contains(js, "this.map.setAttribute('aria-hidden'") {
		t.Error("the minimap is aria-hidden again, and it now contains a focusable button — " +
			"a reader can tab into a subtree that claims not to exist")
	}
	for _, part := range []string{
		"this.mapBody.setAttribute('aria-hidden', 'true')",
		"this.mapRect.setAttribute('aria-hidden', 'true')",
	} {
		if !strings.Contains(js, part) {
			t.Errorf("missing %q: the thumbnail and the view rectangle are decoration, and "+
				"moving aria-hidden onto them is what keeps the button reachable without "+
				"reading a picture of a picture aloud", part)
		}
	}
	if !strings.Contains(js, "this.map.setAttribute('aria-label'") {
		t.Error("the minimap has no accessible name; with aria-hidden off it is now announced, " +
			"and an unnamed group holding one button says nothing about what it is")
	}
	// The thumbnail is a clone of the live diagram, tabindex and all, and the
	// click handlers behind those tab stops are not cloned with them. Left
	// alone they are a second, dead copy of every clickable node in the
	// diagram, sitting inside the subtree that was just marked aria-hidden —
	// the same error the map's own attribute had to move to avoid.
	if !strings.Contains(js, "all[m].removeAttribute('tabindex')") {
		t.Error("the minimap clone keeps its tabindex attributes, so every clickable node in " +
			"the diagram is a tab stop twice, and the second one is inside an aria-hidden " +
			"subtree and does nothing when pressed")
	}
	// SHORTCUTS is both the key handler's documentation and the help overlay's
	// content. A gesture the runtime honours and the sheet does not mention is
	// a gesture nobody finds.
	if !strings.Contains(js, "['Minimap',") {
		t.Error("SHORTCUTS does not mention the minimap, so the help sheet never tells anyone " +
			"the map can be dragged or double-clicked to fit")
	}
}

// TestRestartIsForTheModesWithNoScrub pins why Restart is still in the rail.
//
// Everywhere else it is the fourth way to do one thing: Home does it, clicking
// step 1 does it, and dragging the scrub to zero does it while being the
// gesture a reader reaches for unprompted. Presenter mode and reel mode hide
// `.dgm-foot`, so in those two there is no scrub to drag and no keyboard in a
// presenter's hand — which is the whole argument for the button, and therefore
// the whole extent of it.
func TestRestartIsForTheModesWithNoScrub(t *testing.T) {
	js, css := string(runtimeJS), string(runtimeCSS)

	if !strings.Contains(js, "dgm-btn dgm-nofoot") {
		t.Error("the rail's Restart button no longer carries dgm-nofoot, so it stands in every " +
			"mode again — including the ones whose scrub already does the job")
	}
	if !strings.Contains(css, ".dgm-nofoot { display: none; }") {
		t.Error("runtime.css does not hide .dgm-nofoot by default; the class only means " +
			"anything if being absent is the resting state")
	}
	for _, mode := range []string{".dgm-present .dgm-nofoot", ".dgm-reel .dgm-nofoot"} {
		if !strings.Contains(css, mode) {
			t.Errorf("runtime.css has no %s rule; that is one of the two modes that hide "+
				".dgm-foot, and hiding the scrub without offering Restart leaves a "+
				"presenter with no pointer-driven way back to the start", mode)
		}
	}
	// The pairing is the point: dgm-authoring names what presenting takes
	// away, dgm-nofoot names what it puts back. They must never land on one
	// element, or a presenter rule and a presenter rule would fight at equal
	// weight over the same button.
	if strings.Contains(js, "dgm-authoring dgm-nofoot") || strings.Contains(js, "dgm-nofoot dgm-authoring") {
		t.Error("a control carries both dgm-authoring and dgm-nofoot; .dgm-present .dgm-authoring " +
			"and .dgm-present .dgm-nofoot are both (0,2,0), so which one wins would be " +
			"decided by source order rather than by anything anyone meant")
	}
}

// TestTheRailHasNoFitButton is the mechanical proof of the move.
//
// Fit in the rail did nothing until something had zoomed, duplicated the
// stage's double-click, and sat beside a minimap that appears at exactly the
// moment it becomes useful. Two controls for one signal, one of them dead most
// of the time.
func TestTheRailHasNoFitButton(t *testing.T) {
	js := string(runtimeJS)

	if strings.Contains(js, "zoomBtn") {
		t.Error("runtime.js still builds the rail's Fit button; it duplicates the minimap's " +
			"own fit control and is dead whenever the map is not showing")
	}
	// The rail should not have grown the control back under another name
	// either: the class is what ties the button's life to the map's.
	if strings.Contains(js, "dgm-btn dgm-map-fit") {
		t.Error("the fit control is being built as a rail button, which puts it back in the " +
			"column whose visibility rules have nothing to do with the zoom")
	}
}

// --- the rail down to three ------------------------------------------------

// TestTheRailCollapseIsGone is the mechanical proof of the phase.
//
// The rail carried a ⋯ toggle, an is-open class, a writer that kept the two in
// step and a matchMedia watcher (with a legacy addListener fallback) whose only
// job was to close the thing again when the window stopped being narrow. All of
// that existed because seven controls stacked over a phone-sized diagram is
// most of the diagram. The column is three at rest now, and three 42px squares
// stand at any width — so the state machine has nothing left to be a state
// machine about, and a collapse that cannot be reached is worse than none: it
// is a mode nobody can see, holding a class the stylesheet still honours.
func TestTheRailCollapseIsGone(t *testing.T) {
	js, css := string(runtimeJS), string(runtimeCSS)

	for _, gone := range []string{"railMore", "setRailOpen", "watchRailCollapse", "rail-more"} {
		if strings.Contains(js, gone) {
			t.Errorf("runtime.js still carries %q; the narrow-screen collapse is deleted, and "+
				"half a state machine is the one shape worse than all of it", gone)
		}
	}
	for _, gone := range []string{"rail-more", ".dgm-rail.is-open", "max-width: 520px"} {
		if strings.Contains(css, gone) {
			t.Errorf("runtime.css still carries %q; with no toggle to press, a rule that hides "+
				"the tools below a breakpoint hides them for good", gone)
		}
	}
	// The column was sized around the speed button's widest label. That button
	// is gone, so the width has to have moved with it — a comment and a number
	// that outlive their reason are how the next reader learns the wrong thing.
	//
	// Read out of the rule that owns it, not out of the whole sheet: "42px"
	// appears in prose and in other boxes, and an assertion any of those could
	// satisfy is an assertion about nothing.
	rail := ruleBody(t, css, ".dgm-rail {")
	if !strings.Contains(rail, "width: 42px;") {
		t.Errorf("the rail is not 42px wide; 56 was 42px of button sized to fit the string "+
			"\"0.25x\", and nothing in the column has a label any more. Rule reads:\n%s", rail)
	}
	if !strings.Contains(rail, "padding: 4px;") {
		t.Errorf("the rail's padding is not 4px; 32px of button inside 4px a side and a 1px "+
			"border is the 42px width above, and the two numbers are one number. Rule reads:\n%s", rail)
	}
	// The stage reserves a strip so the floating column never covers a
	// clickable node — a node with a `click … view` binding near the right edge
	// would otherwise not be reachable at all. The strip is the rail plus its
	// inset plus some breath, so it is wrong the moment the rail's width moves
	// without it.
	if stage := ruleBody(t, css, ".dgm-stage {"); !strings.Contains(stage, "padding-right: 62px;") {
		t.Errorf("the stage reserves the wrong strip for the rail; 42px of rail, 10px of "+
			"inset and 10px of breath is 62, and reserving the old 76 wastes 14px of every "+
			"diagram at every width. Rule reads:\n%s", stage)
	}
}

// TestASelectLooksLikeOneAndTheScenarioPickerSaysSo pins the affordance.
//
// `appearance: none` takes the platform's arrow off along with its styling, and
// for a long time nothing put one back: every <select> here rendered as a
// bordered box indistinguishable from a text input. On the scenario picker that
// was not cosmetic — the control is the only thing telling a reader the
// walkthrough on screen is one of several, and with no arrow it told them
// nothing. A diagram with three scenarios read as a diagram with one.
func TestASelectLooksLikeOneAndTheScenarioPickerSaysSo(t *testing.T) {
	js, css := string(runtimeJS), string(runtimeCSS)

	// The skin is shared with .dgm-btn; the arrow is not, and the two have to stay
	// in separate rules. Folding the chevron into the shared one put an arrow on
	// every button on the page, Present included.
	skin := ruleBody(t, css, ".dgm-btn,\n.dgm-select {")
	if !strings.Contains(skin, "appearance: none") {
		t.Error("the shared button/menu rule no longer sets appearance: none, so a <select> is " +
			"the platform's control again and the arrow below fights it")
	}
	if strings.Contains(skin, "linear-gradient(") {
		t.Error("the chevron is in the rule .dgm-btn shares, so every button on the page draws " +
			"a dropdown arrow it cannot open")
	}
	// The `background` shorthand resets background-image, which is how the arrow
	// would go missing again without anybody editing the arrow.
	if strings.Contains(skin, "background:") {
		t.Error("the shared rule uses the background shorthand, which resets the chevron's " +
			"background-image; set background-color instead")
	}

	// The arrow itself, asserted on its declarations rather than through ruleBody:
	// both rules' selectors contain the text ".dgm-select {", so an anchor on that
	// finds whichever comes first and cannot tell them apart.
	//
	// Two gradients rather than a background-image, so the arrow is drawn in
	// currentColor and one declaration serves both palettes.
	for _, decl := range []string{
		"linear-gradient(45deg, transparent 50%, currentColor 50%)",
		"linear-gradient(135deg, currentColor 50%, transparent 50%)",
		"padding-right: 26px",
	} {
		if !strings.Contains(css, decl) {
			t.Errorf("runtime.css is missing %q; without the arrow a select with "+
				"appearance: none is a text field to look at", decl)
		}
	}

	// Under the title at caption size, with a backstop rather than a cap: a
	// scenario name is a sentence, and capping it to fit beside the title
	// truncated the one thing the menu exists to show.
	picker := ruleBody(t, css, ".dgm-picker {")
	for _, want := range []string{"max-width", "text-overflow: ellipsis"} {
		if !strings.Contains(picker, want) {
			t.Errorf("the scenario picker does not set %s. Rule reads:\n%s", want, picker)
		}
	}
	// The row sizes both of its controls, and pins line-height as well as
	// font-size: a select and a button do not agree on their intrinsic height from
	// a font size alone, which is how Present came to stand taller than the picker
	// beside it.
	row := ruleBody(t, css, ".dgm-scenario .dgm-btn,\n.dgm-scenario .dgm-select {")
	for _, want := range []string{"font-size: 12px", "line-height: 16px", "padding-top", "padding-bottom"} {
		if !strings.Contains(row, want) {
			t.Errorf("the scenario row does not set %s, so its picker and its Present button "+
				"are sized apart. Rule reads:\n%s", want, row)
		}
	}
	// Vertical only: the picker's right-hand padding is the chevron's room.
	if strings.Contains(row, "padding:") {
		t.Error("the scenario row sets padding wholesale, which takes the chevron's room from " +
			"the picker and puts the arrow on top of the text")
	}
	if !strings.Contains(css, ".dgm-heading-row {") {
		t.Error("the heading is not two rows, so the scenario has nowhere to sit but beside " +
			"the title, where it has to be truncated to fit")
	}

	// And it says what it is, since the arrow alone does not name it.
	if !strings.Contains(js, "this.picker.setAttribute('aria-label', 'Scenario');") {
		t.Error("the scenario picker has no accessible name; a screen reader announces an " +
			"unnamed select, and the option text alone does not say what choosing does")
	}
	if !strings.Contains(js, "'Choose one of ' + scenarios.length + ' scenarios'") {
		t.Error("the picker's title does not say how many there are, which is the fact that " +
			"makes it worth opening")
	}
	// The picker and its caption hide together and nothing else goes with them.
	// Wrapping them is what keeps Present — which shares the row — on a page with
	// one scenario, and Present is the way out of presenter mode.
	if !strings.Contains(js, "this.pickerWrap.style.display = scenarios.length > 1 ? '' : 'none';") {
		t.Error("the picker and its caption do not hide together when there is one scenario, " +
			"so a document with one captions a control that is not on the page")
	}
	if !strings.Contains(js, "this.scenarioLine.appendChild(this.presentBtn);") {
		t.Error("Present is not in the scenario row; it sat at the far end of a bar as wide as " +
			"the page, hanging past the stage's own edge over the step list")
	}
	if strings.Contains(js, "this.pickerWrap.appendChild(this.presentBtn)") {
		t.Error("Present is inside the wrapper that hides with the picker, so a document with " +
			"one scenario has no way out of presenter mode")
	}
	// The heading is shown in inline mode, so Present has to be hidden by name
	// there — an embed is a diagram inside somebody's page and has no business
	// taking the whole screen.
	if !strings.Contains(css, ".dgm-inline .dgm-present-btn") {
		t.Error("inline mode does not hide Present, which now lives in the heading that inline " +
			"mode shows")
	}
}

// TestTheStepListReachesTheStage pins the two boxes being one height.
//
// .dgm-body sets `align-items: start`, which is right for the storyboard panel —
// content, sized by what it holds — and wrong for the step list, which is
// furniture beside the stage. A four-step walkthrough left the panel floating
// short of the diagram next to it, two boxes of different heights with nothing
// explaining the difference.
func TestTheStepListReachesTheStage(t *testing.T) {
	css := string(runtimeCSS)

	steps := ruleBody(t, css, ".dgm-steps {")
	if !strings.Contains(steps, "align-self: stretch") {
		t.Errorf("the step list does not stretch, so it stops short of the stage whenever the "+
			"walkthrough has few steps. Rule reads:\n%s", steps)
	}
	// And it still scrolls rather than growing the row, or a thirty-step
	// walkthrough would stretch the stage to match it instead.
	if !strings.Contains(steps, "overflow: auto") {
		t.Error("the step list no longer scrolls; stretching it without that lets a long " +
			"walkthrough set the height of the stage beside it")
	}
	body := ruleBody(t, css, ".dgm-body {")
	if !strings.Contains(body, "align-items: start") {
		t.Error(".dgm-body no longer starts its items, so the storyboard panel stretches too — " +
			"it is content, and its height is what it holds")
	}
}

// TestTransportSitsAtTheTimeline pins the cluster that moves the clock.
//
// Play used to stand in the top bar while the scrub it drives sat at the bottom,
// which asked a reader to operate one thing from two places. Previous and next
// step are the arrow keys made visible: onKey has read them from the beginning,
// and a reader who never went looking for a shortcut was dragging the scrub and
// guessing where the beats were.
func TestTransportSitsAtTheTimeline(t *testing.T) {
	js, css := string(runtimeJS), string(runtimeCSS)

	if !strings.Contains(js, "el('div', 'dgm-transport')") {
		t.Error("runtime.js builds no transport cluster, so Play is back to being a bar away " +
			"from the timeline it drives")
	}
	for _, label := range []string{"'Previous step'", "'Next step'"} {
		if !strings.Contains(js, label) {
			t.Errorf("the transport has no %s button; the arrow keys stay the only way to "+
				"walk the beats, which is the thing this cluster exists to surface", label)
		}
	}
	// The glyph is the whole control, so the label cannot be written as text:
	// textContent would delete the icon. syncPlay writes both together.
	if strings.Contains(js, "this.playBtn.textContent") {
		t.Error("something writes playBtn.textContent; the play control is an icon button now " +
			"and that would delete its glyph — see syncPlay")
	}
	if !strings.Contains(js, "Player.prototype.syncPlay") {
		t.Error("syncPlay is gone, so nothing keeps the play glyph and its accessible name in " +
			"step with whether the clock is running")
	}
	// Play left the bar, so the rule that used to restore it there restores
	// nothing, and the third rule that arbitrated between them has no parties.
	if strings.Contains(css, ".dgm-inline .dgm-controls > .dgm-play") {
		t.Error("runtime.css still restores a bar Play for inline mode; the control is in the " +
			"foot, which inline shows, so the rule and the conflict it caused are both dead")
	}
	if !strings.Contains(css, ".dgm-transport {") {
		t.Error("runtime.css does not style .dgm-transport, so the cluster is unspaced buttons " +
			"crowding the scrub")
	}
}

// TestSpeedLivesInTheTransport pins where playback speed went, and the invariant
// underneath every move it has made: a reader can always change the rate, and
// exactly one control writes the key that remembers it.
//
// It began as a button in the rail, cycling one way through five rates. It moved
// to the settings sheet, because the rail is the narrowest space on the page, a
// control whose label is its own value has to widen it, and dgm-authoring hid it
// from presenters. It now sits in the foot beside the transport it belongs to,
// where a horizontal row has the width the column did not and it is one click
// instead of three.
func TestSpeedLivesInTheTransport(t *testing.T) {
	js, css := string(runtimeJS), string(runtimeCSS)

	for _, gone := range []string{"speedBtn", "cycleSpeed"} {
		if strings.Contains(js, gone) {
			t.Errorf("runtime.js still carries %q; the rail button and its one-way cycle are "+
				"what the sheet's menu replaces, and two writers of dgm.speed is one too many", gone)
		}
	}
	if !strings.Contains(js, "el('select', 'dgm-select dgm-foot-speed')") {
		t.Error("the foot builds no speed menu; with the rail button gone there is then no " +
			"way at all for a reader to change the playback rate")
	}
	// One writer, because the rate is remembered per origin rather than per
	// diagram: a second menu would be a second view of one stored preference.
	if n := strings.Count(js, "self.setSpeed(parseFloat("); n != 1 {
		t.Errorf("runtime.js has %d speed menus, want exactly 1: the rate is a preference and "+
			"two controls for it are two things to keep honest", n)
	}
	// The ends are open because the material stopped being hand-authored
	// explainers: `cinegram trace` replays measured time, and a trace is 8ms or
	// four minutes. Coarseness was a property of cycling, and this is a select.
	for _, rate := range []string{"0.1", "10"} {
		if !strings.Contains(js, "var SPEED_PRESETS = [0.1, 0.25, 0.5, 1, 1.5, 2, 4, 10];") {
			t.Errorf("SPEED_PRESETS no longer reaches %sx, so a trace measured in milliseconds "+
				"or minutes cannot be watched at a sensible rate", rate)
			break
		}
	}
	// Additive on purpose: the original five are still there, because the chosen
	// rate is remembered and dropping one would move somebody's saved preference.
	for _, rate := range []string{"0.25", "0.5", "1.5", "2"} {
		if !strings.Contains(js, rate) {
			t.Errorf("SPEED_PRESETS dropped %sx; widening the range must not move a rate a "+
				"reader has already chosen and had remembered", rate)
		}
	}
	// The key is the whole point of moving the control rather than deleting it:
	// a rate chosen here is remembered, and adoptScenarioSpeed is what keeps an
	// authored `speed:` outranking it.
	if !strings.Contains(js, "prefSet('dgm.speed', String(v))") {
		t.Error("the speed menu no longer writes dgm.speed, so the choice is forgotten " +
			"on the next page — which is the one thing a preference has to do")
	}
	// The sheet is a dialog and stays one: Esc and the backdrop close it, and a
	// role that went missing would leave a screen reader reading a settings
	// panel as part of the page behind it.
	if !strings.Contains(js, "box.setAttribute('role', 'dialog')") {
		t.Error("the shortcuts sheet is no longer a dialog; it is an overlay a reader opens " +
			"deliberately, and a role that went missing would leave a screen reader reading " +
			"it as part of the page behind it")
	}
	// The property cycleSpeed had and a <select> does not: it stepped to the
	// next preset by value, so a scenario declaring `speed: 0.8` went somewhere
	// sensible instead of snapping to the slowest. A menu without the declared
	// rate on it would show its first option instead — 0.25x, and wrong.
	for _, part := range []string{
		"opt.setAttribute('data-dgm-declared', '1')",
		"sel.insertBefore(opt, at)",
	} {
		if !strings.Contains(js, part) {
			t.Errorf("syncSpeed is missing %q, so a scenario's own rate no longer joins the "+
				"menu: a scenario declaring speed 0.8 would read as 0.25x, which is not a "+
				"rounder answer but a different one", part)
		}
	}
	// The sheet's one row went with the control. Styling a class nothing builds
	// is dead weight in a sheet every page carries, so the rule left too.
	if strings.Contains(css, ".dgm-help-row {") {
		t.Error("runtime.css still styles .dgm-help-row, which nothing builds now that speed " +
			"lives in the foot")
	}
	if !strings.Contains(css, ".dgm-help-section {") {
		t.Error("runtime.css does not style .dgm-help-section; the sheet holds settings and " +
			"shortcuts now, and an unnamed half of it is a list of rates with no heading")
	}
}

// TestCineAnnouncesItsState pins the one accessible fact a toggle owes.
//
// Cine turns the auto-follow camera on. It said so by tinting itself with an
// is-on class — which a screen reader cannot see — while its accessible name
// stayed "Camera follows each step" whether the camera was following or not: a
// name that describes the button rather than its state. aria-pressed is the
// state, and it belongs beside the class in setFollow, which is the class's one
// writer: two writers is how a picture and an announcement come apart.
func TestCineAnnouncesItsState(t *testing.T) {
	js := string(runtimeJS)

	const attr = "this.cineBtn.setAttribute('aria-pressed', this.follow ? 'true' : 'false');"
	if n := strings.Count(js, attr); n != 2 {
		t.Errorf("runtime.js sets the Cine button's aria-pressed %d times, want exactly 2 — "+
			"buildRail states the resting value, setFollow keeps it, and nothing else may "+
			"write it or the attribute and the is-on class can disagree", n)
	}
	// Adjacency is the guarantee, not the count: apart, one of the two can be
	// moved or dropped without the other looking wrong. Judged on the code with
	// the comments taken out, so that rewording the paragraph between them —
	// which this file expects people to do — is not a test failure, and so that
	// a comment cannot pad the two lines far enough apart to slip through a
	// window measured in bytes.
	if !strings.Contains(codeOnly(js), "this.cineBtn.classList.toggle('is-on', this.follow);\n"+attr) {
		t.Error("aria-pressed is not written on the line after the is-on class in setFollow; " +
			"apart, they are two facts about the camera that can drift, and the one a " +
			"screen reader hears is the one nobody notices has drifted")
	}
}

// TestADeepLinkedTimeCannotLandPastTheEnd pins the clamp on both of the paths
// that turn a hash `t=` into a time. They used to disagree: applyHash went
// through seek, which clamps at both ends, while the read on load clamped only
// at zero — so `#t=30000` on an eighteen-second scenario, which is what
// `frame --at 30s` builds, put "30.0s / 18.0s" on the clock while the scrubber,
// bounded by its own max, sat at the end.
func TestADeepLinkedTimeCannotLandPastTheEnd(t *testing.T) {
	js := codeOnly(string(runtimeJS))

	if strings.Contains(js, "this.time = Math.max(0, ms);") {
		t.Error("a hash `t=` is clamped at zero but not at the scenario's duration, so a link " +
			"past the end leaves the clock reading a time the scrubber has already stopped " +
			"short of — and a scenario's own end is the only total the page ever shows")
	}
	// Twice, once per path, and by the same arithmetic: the guarantee is that
	// the same link lands in the same place whether it was loaded or navigated
	// to, which is a property of the two agreeing rather than of either alone.
	const clamp = "this.time = Math.min(max, Math.max(0, ms));"
	if n := strings.Count(js, clamp); n != 2 {
		t.Errorf("runtime.js clamps a `t=` into range %d times, want exactly 2 — seek for a "+
			"hashchange and the read in build() for a page loaded at a moment. Fewer means "+
			"one of the two paths accepts a time past the end again", n)
	}
}

// ruleBody returns the declarations of the first CSS rule opening with sel, so
// that a test about one box's numbers cannot be satisfied by the same numbers
// somewhere else in a two-thousand-line sheet.
//
// The match is anchored to the start of a line, which is what tells
// `.dgm-stage {` from `.dgm-inline .dgm-stage {` — and, since every nested rule
// in this sheet is indented, a top-level rule from the same selector inside an
// @supports or @media block.
func ruleBody(t *testing.T, css, sel string) string {
	t.Helper()
	i := strings.Index(css, "\n"+sel)
	if i < 0 {
		t.Fatalf("runtime.css has no top-level %s rule at all", sel)
	}
	rest := css[i+len(sel)+1:]
	j := strings.Index(rest, "}")
	if j < 0 {
		t.Fatalf("the %s rule is never closed", sel)
	}
	return rest[:j]
}

// codeOnly strips whole-line comments and indentation from a JavaScript source,
// leaving the statements. runtime.js explains itself at length and is meant to
// keep doing so, so a test about what two lines of code do next to each other
// has no business failing when the paragraph above them is rewritten.
func codeOnly(js string) string {
	var out []string
	for _, line := range strings.Split(js, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		out = append(out, t)
	}
	return strings.Join(out, "\n")
}
