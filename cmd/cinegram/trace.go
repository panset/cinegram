package main

// Replaying a real trace over a diagram somebody drew.
//
// Every `dur:` in a hand-written scenario is a number an author chose because
// the animation felt right at that speed. This command replaces those numbers
// with measurements: spans become flows, span durations become durations, and
// failed spans become failed flows. The animation stops illustrating a mental
// model and starts showing what happened.
//
// **The diagram is not generated.** It is the author's picture and stays exactly
// as written — only a scenario is appended. That is deliberate: a service graph
// derived from a trace is a different, uglier drawing than the one a person
// makes to explain a system, and the point here is to animate *theirs*.

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/parser"
	"github.com/tejaspanse/cinegram/pkg/symbol"
	"github.com/tejaspanse/cinegram/pkg/trace"
)

func cmdTrace(args []string, stdout, stderr io.Writer) error {
	var over, mapping, name string
	var speed, scale float64
	input, output, err := parseArgsWith("trace", args, func(fs *flag.FlagSet) {
		fs.StringVar(&over, "over", "", "the .dgm whose diagram the trace is replayed over (required)")
		fs.StringVar(&mapping, "map", "", "service=node pairs, comma separated, for names that do not match")
		fs.StringVar(&name, "name", "", "name for the generated scenario")
		fs.Float64Var(&speed, "speed", 0, "playback rate for the scenario (default: chosen from the trace's length)")
		fs.Float64Var(&scale, "scale", 0, "stretch the timeline by this factor, for traces too fast to watch (default: chosen)")
	})
	if err != nil {
		return err
	}
	if over == "" {
		return fmt.Errorf("trace needs --over <file.dgm>: the diagram to replay the trace on")
	}
	over = resolvePath(over)

	raw, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	tr, err := trace.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", input, err)
	}

	source, err := os.ReadFile(over)
	if err != nil {
		return err
	}
	res, bag := parser.Parse(over, string(source))
	if bag.HasErrors() {
		return fmt.Errorf("%s does not parse; fix it before replaying a trace over it:\n%s", over, bag)
	}

	nodes, err := resolveServices(tr, res.Symbols, mapping)
	if err != nil {
		return err
	}

	if name == "" {
		name = scenarioName(tr)
	}
	scales := []float64{scale}
	if scale == 0 {
		scales = scaleLadder(suggestScale(tr.DurUs()))
	}

	sc, warnings := renderScenario(tr, nodes, res.Symbols, name, speed, scales)
	for _, w := range warnings {
		fmt.Fprintln(stderr, "cinegram: warning: "+w)
	}

	out := string(source)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += "\n" + sc

	if output == "" {
		output = over // in place: the scenario belongs with the diagram it describes
	}
	if err := os.WriteFile(output, []byte(out), 0o644); err != nil {
		return err
	}
	word := "scenario"
	if len(scales) > 1 {
		word = "scenarios"
	}
	fmt.Fprintf(stdout, "%s: added %d %s — %d spans over %s, %s\n",
		output, len(scales), word, len(tr.Spans), fmtMicros(tr.DurUs()), outcomeWord(tr.Failed()))
	if len(scales) > 1 || scales[0] != 1 {
		var rungs []string
		for _, f := range scales {
			rungs = append(rungs, scaleWord(f))
		}
		fmt.Fprintf(stdout, "  pick the pace in the page's scenario picker: %s\n", strings.Join(rungs, " · "))
		fmt.Fprintf(stdout, "  (the timeline is stretched; every desc keeps the real measurement)\n")
	}
	return nil
}

func outcomeWord(failed bool) string {
	if failed {
		return "with failures"
	}
	return "all ok"
}

// ---------------------------------------------------------------------------
// Mapping services onto nodes
// ---------------------------------------------------------------------------

// resolveServices decides which diagram node each service is.
//
// Names are matched rather than demanded, because `service.name` is usually
// already the thing the node is called — and when it is not, `--map` says so.
// An unmatched service is a hard error listing both sides: guessing would
// animate the wrong box, which is worse than refusing.
func resolveServices(tr *trace.Trace, t *symbol.Table, mapping string) (map[string]string, error) {
	out := map[string]string{}

	explicit := map[string]string{}
	for _, pair := range strings.Split(mapping, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, fmt.Errorf("--map takes service=node pairs, got %q", pair)
		}
		explicit[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	// Two lookups, tried in order: the node's id, then its label. Both are
	// normalised, so "Vector Search" matches "vector-search" and "vectorsearch".
	byID := map[string]string{}
	byLabel := map[string]string{}
	var ids []string
	for _, n := range t.Nodes() {
		byID[normalise(n.ID)] = n.ID
		if n.Label != "" {
			byLabel[normalise(n.Label)] = n.ID
		}
		ids = append(ids, n.ID)
	}
	sort.Strings(ids)

	var unmapped []string
	for _, svc := range tr.Services() {
		if want, ok := explicit[svc]; ok {
			if _, known := t.Node(want); !known {
				return nil, fmt.Errorf("--map sends %q to %q, which is not a node in the diagram.\navailable nodes: %s",
					svc, want, strings.Join(ids, ", "))
			}
			out[svc] = want
			continue
		}
		if id, ok := byID[normalise(svc)]; ok {
			out[svc] = id
			continue
		}
		if id, ok := byLabel[normalise(svc)]; ok {
			out[svc] = id
			continue
		}
		unmapped = append(unmapped, svc)
	}

	if len(unmapped) > 0 {
		return nil, fmt.Errorf(
			"these services have no node in the diagram: %s\navailable nodes: %s\nname them with --map %s=<node>",
			strings.Join(unmapped, ", "), strings.Join(ids, ", "), unmapped[0])
	}
	return out, nil
}

// normalise reduces a name to its letters and digits, so that a service called
// `payments-api`, a node id `paymentsApi` and a label "Payments API" all meet.
func normalise(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Rendering the scenario
// ---------------------------------------------------------------------------

// band is a stretch of trace during which something was always in flight.
type band struct {
	start, end int
	phases     []trace.Span
}

// bandsOf groups phases that overlap in time.
//
// This is the one real design problem in the command. Steps run in *sequence* —
// each begins where the last ended — while a trace is a tree of spans that
// happily overlap. Serialising overlapping phases into consecutive steps would
// invent timings, which is precisely the thing this command exists to stop.
//
// So a step boundary is only cut where the trace is genuinely quiet: bands are
// maximal runs of overlapping phases, and the gap between two bands becomes the
// step's `delay`. Absolute times therefore survive exactly, and a step boundary
// carries a real meaning — nothing was in flight across it.
func bandsOf(tr *trace.Trace, phases []trace.Span) []band {
	var bands []band
	for _, p := range phases {
		// A phase lasts as long as its subtree, since a child may outlive its
		// parent's recorded end when a service returns before its work is done.
		start, end := p.StartUs, p.EndUs
		for _, s := range tr.Subtree(p.ID) {
			if s.StartUs < start {
				start = s.StartUs
			}
			if s.EndUs > end {
				end = s.EndUs
			}
		}
		if n := len(bands); n > 0 && start < bands[n-1].end {
			if end > bands[n-1].end {
				bands[n-1].end = end
			}
			bands[n-1].phases = append(bands[n-1].phases, p)
			continue
		}
		bands = append(bands, band{start: start, end: end, phases: []trace.Span{p}})
	}
	return bands
}

// renderScenario writes the .dgm text for the trace.
func renderScenario(tr *trace.Trace, nodes map[string]string, t *symbol.Table, name string, speed float64, scales []float64) (string, []string) {
	var b strings.Builder
	var warnings []string

	fmt.Fprintf(&b, "%%%% Generated by `cinegram trace` — every duration below is measured, not chosen.\n")
	fmt.Fprintf(&b, "%%%% -\n")
	if tr.ID != "" {
		fmt.Fprintf(&b, "%%%% trace %s · %d spans · %s wall clock\n", tr.ID, len(tr.Spans), fmtMicros(tr.DurUs()))
	}
	for _, r := range tr.Roots {
		fmt.Fprintf(&b, "%%%% root: %s (%s) took %s\n", r.Name, r.Service, fmtMicros(r.DurUs()))
	}
	if len(scales) > 1 || scales[0] != 1 {
		fmt.Fprintf(&b, "%%%% -\n")
		fmt.Fprintf(&b, "%%%% This trace is too fast to watch at its own speed, and the timeline is whole\n")
		fmt.Fprintf(&b, "%%%% milliseconds, so its briefest span (%s) would have nowhere to go. It is\n", fmtMicros(shortestUs(tr)))
		fmt.Fprintf(&b, "%%%% therefore told once per pace below — same spans, same ratios, different\n")
		fmt.Fprintf(&b, "%%%% stretch — and the scenario picker chooses between them. Every `desc`\n")
		fmt.Fprintf(&b, "%%%% states the real measurement whichever is showing.\n")
	}
	fmt.Fprintf(&b, "%%%% -\n")
	fmt.Fprintf(&b, "%%%% Step boundaries fall where the trace was quiet — nothing in flight. Spans\n")
	fmt.Fprintf(&b, "%%%% that overlapped stay in one step, placed by `at:`, so the concurrency is real.\n")

	phases := tr.Phases()
	bands := bandsOf(tr, phases)

	// Say what is missing. A span covering the whole trace — the root, and any
	// sole child beneath it — cannot be drawn without collapsing every step into
	// one, since a step sizes itself to its longest action. Naming those spans is
	// the honest alternative to dropping them silently.
	if skipped := undrawn(tr, bands); len(skipped) > 0 {
		fmt.Fprintf(&b, "%%%% -\n")
		fmt.Fprintf(&b, "%%%% Not drawn, because each spans the whole trace and an action that long\n")
		fmt.Fprintf(&b, "%%%% would collapse every step into one:\n")
		for _, s := range skipped {
			fmt.Fprintf(&b, "%%%%   %s (%s) %s, from %s\n",
				oneLine(s.Name), s.Service, fmtMicros(s.DurUs()), fmtMicros(s.StartUs))
		}
	}

	byID := map[string]trace.Span{}
	for _, s := range tr.Spans {
		byID[s.ID] = s
	}

	for _, scale := range scales {
		rate := speed
		if rate == 0 {
			rate = suggestSpeed(int(float64(tr.DurUs()) / 1000 * scale))
		}

		title := name
		if len(scales) > 1 {
			// Only a ladder needs its rungs distinguished; a single telling of a
			// trace slow enough to watch is just the trace.
			title = scenarioTitle(name, scale, tr)
		}
		fmt.Fprintf(&b, "\nscenario %q { speed: %s", title, trimFloat(rate))
		if tr.Failed() {
			b.WriteString(", outcome: fail")
		}
		b.WriteString(" }\n")

		prevEnd := 0
		for i, bd := range bands {
			var actions []string
			for _, p := range bd.phases {
				for _, s := range tr.Subtree(p.ID) {
					line, warn := actionFor(s, byID, nodes, t, bd.start, scale)
					if warn != "" && scale == scales[0] {
						// Warn once, not once per rung: the topology is the same
						// in every telling.
						warnings = append(warnings, warn)
					}
					if line != "" {
						actions = append(actions, line)
					}
				}
			}
			if len(actions) == 0 {
				continue
			}

			fmt.Fprintf(&b, "\n  step %s %q {\n", stepID(bd, i), stepTitle(bd))
			fmt.Fprintf(&b, "    desc: %q\n", stepDesc(tr, bd))
			if gap := scaleMs(bd.start-prevEnd, scale); gap > 0 {
				// Idle time is part of what happened; a request that waited 400ms
				// between phases should show 400ms of nothing. That includes the
				// lead-in before the first phase, which is the entry service doing
				// its own work — dropping it would shift every later timing.
				fmt.Fprintf(&b, "    delay: %dms\n", gap)
			}
			for _, a := range actions {
				fmt.Fprintf(&b, "    %s\n", a)
			}
			b.WriteString("  }\n")
			prevEnd = bd.end
		}
	}
	return b.String(), warnings
}

// scaleWord names a pace the way a reader would say it.
func scaleWord(scale float64) string {
	if scale == 1 {
		return "real time"
	}
	return trimFloat(scale) + "x slower"
}

// scenarioTitle keeps every rung distinct in the picker and states the real
// duration in each, so the pace is never mistaken for the measurement.
func scenarioTitle(base string, scale float64, tr *trace.Trace) string {
	if scale == 1 && len(base) > 0 {
		return fmt.Sprintf("%s · real time", base)
	}
	return fmt.Sprintf("%s · %s", base, scaleWord(scale))
}

// undrawn is every span no step will show, in start order.
//
// It exists so the generated file can admit its own omissions: a reader who
// counts nine spans in the trace and six flows on the diagram deserves to be
// told which three went where, rather than left to wonder.
func undrawn(tr *trace.Trace, bands []band) []trace.Span {
	shown := map[string]bool{}
	for _, bd := range bands {
		for _, p := range bd.phases {
			for _, s := range tr.Subtree(p.ID) {
				shown[s.ID] = true
			}
		}
	}
	var out []trace.Span
	for _, s := range tr.Spans {
		if !shown[s.ID] {
			out = append(out, s)
		}
	}
	return out
}

// actionFor turns one span into one line of scenario.
//
// A span that crossed a service boundary is a flow along the edge between the
// two nodes; a span that stayed inside one service is work, so it highlights
// that node. A crossing the diagram does not draw is reported rather than
// invented — that warning is one of the more useful things here, because it
// means the diagram and production disagree.
func actionFor(s trace.Span, byID map[string]trace.Span, nodes map[string]string, t *symbol.Table, origin int, scale float64) (string, string) {
	node, ok := nodes[s.Service]
	if !ok {
		return "", ""
	}
	at := scaleMs(s.StartUs-origin, scale)
	if at < 0 {
		at = 0
	}
	dur := scaleMs(s.DurUs(), scale)

	parent, hasParent := byID[s.ParentID]
	from := ""
	if hasParent {
		from = nodes[parent.Service]
	}

	// Same service, or no parent in this trace: this is work, not a call.
	if from == "" || from == node {
		return fmt.Sprintf("highlight %s { at: %dms, dur: %dms%s }",
			node, at, maxInt(dur, 1), styleFor(s)), ""
	}

	if _, _, found := t.FindEdge(from, node); !found {
		return fmt.Sprintf("highlight %s { at: %dms, dur: %dms%s }", node, at, maxInt(dur, 1), styleFor(s)),
			fmt.Sprintf("the trace shows %s calling %s, which the diagram does not draw — shown as work on %s instead",
				from, node, node)
	}

	return fmt.Sprintf("flow %s -> %s { label: %q, at: %dms, dur: %dms%s }",
		from, node, label(s), at, maxInt(dur, 1), statusFor(s)), ""
}

func styleFor(s trace.Span) string {
	if s.Failed {
		return ", style: fail"
	}
	return ", style: busy"
}

func statusFor(s trace.Span) string {
	if s.Failed {
		return ", status: fail"
	}
	return ""
}

// label is what the arrow says: the span's name, plus the HTTP status when the
// span carried one, because "GET /orders 503" is the whole story in one label.
func label(s trace.Span) string {
	name := s.Name
	if code := s.Attrs["http.status_code"]; code != "" {
		name += " · " + code
	}
	return oneLine(name)
}

func stepID(bd band, i int) string {
	base := normalise(bd.phases[0].Name)
	if base == "" {
		base = fmt.Sprintf("band%d", i+1)
	}
	// A step id has to be an identifier, and two bands can share a phase name.
	return fmt.Sprintf("t%d%s", i+1, base)
}

func stepTitle(bd band) string {
	names := make([]string, 0, len(bd.phases))
	for _, p := range bd.phases {
		names = append(names, p.Name)
	}
	if len(names) == 1 {
		return oneLine(names[0])
	}
	// Overlapping phases share a step, and saying so is more honest than
	// picking one of them to name it after.
	return oneLine(strings.Join(names, " ∥ "))
}

// stepDesc states what was measured and nothing more.
//
// Deliberately factual: this is generated prose, and generated prose that
// editorialises is how a diagram ends up asserting something nobody checked.
// Interpretation is a human's job — and with `retells`, it can be added as a
// second telling over exactly these timings.
func stepDesc(tr *trace.Trace, bd band) string {
	spans, failed := 0, 0
	slowest := trace.Span{}
	for _, p := range bd.phases {
		for _, s := range tr.Subtree(p.ID) {
			spans++
			if s.Failed {
				failed++
			}
			if s.DurUs() > slowest.DurUs() {
				slowest = s
			}
		}
	}

	total := tr.DurUs()
	share := 0
	if total > 0 {
		share = (bd.end - bd.start) * 100 / total
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s, %d%% of the trace.", fmtMicros(bd.end-bd.start), share)
	if spans > 1 {
		fmt.Fprintf(&sb, " %d spans", spans)
		if len(bd.phases) > 1 {
			fmt.Fprintf(&sb, " across %d overlapping calls", len(bd.phases))
		}
		fmt.Fprintf(&sb, ", slowest %s at %s.", oneLine(slowest.Name), fmtMicros(slowest.DurUs()))
	}
	if failed > 0 {
		fmt.Fprintf(&sb, " %d failed", failed)
		if msg := firstMessage(tr, bd); msg != "" {
			fmt.Fprintf(&sb, ": %s", oneLine(msg))
		} else {
			sb.WriteString(".")
		}
	}
	return sb.String()
}

func firstMessage(tr *trace.Trace, bd band) string {
	for _, p := range bd.phases {
		for _, s := range tr.Subtree(p.ID) {
			if s.Failed && s.Message != "" {
				return s.Message
			}
		}
	}
	return ""
}

func scenarioName(tr *trace.Trace) string {
	base := "trace"
	if len(tr.Roots) > 0 && tr.Roots[0].Name != "" {
		base = oneLine(tr.Roots[0].Name)
	}
	return fmt.Sprintf("%s (%s)", base, fmtMicros(tr.DurUs()))
}

// suggestScale stretches a trace that is too fast to watch.
//
// Playback rate cannot do this job, for three reasons. It does not go far enough
// — an 8ms trace at the runtime's slowest preset of 0.25x is 32ms of wall clock,
// which is two frames. The reader's own speed menu offers only 0.25x to 2x, so
// touching it throws away whatever the scenario asked for. And decisively, the
// timeline is integer milliseconds: a 0.8ms span has nowhere to go, and no
// playback rate recovers a number that was never written down.
//
// Stretching the timeline instead keeps every ratio exact — a call that took a
// third of the request still takes a third of the animation — and gives the
// rounding room to work in. The factor is announced in the file and every `desc`
// reports the real measurement, so nothing is quietly misrepresented.
//
// Factors are round numbers because "stretched 500x" is a sentence a reader can
// hold, where 437.5x is noise.
func suggestScale(durUs int) float64 {
	const target = 6000 // ms of animation we are aiming for
	durMs := float64(durUs) / 1000
	if durMs <= 0 || durMs >= 1500 {
		return 1 // long enough to watch; playback rate handles the rest
	}
	want := target / durMs
	for _, f := range niceScales {
		if f >= want {
			return f
		}
	}
	return 10000
}

// niceScales are the factors offered, ascending.
var niceScales = []float64{2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000}

// scaleLadder is the set of scales to emit, one scenario each.
//
// A fast trace gets a ladder rather than a single stretch, because the scenario
// picker is already a selector and the interesting comparison is between them:
// the slow-motion telling is where the reader sees what happened, and the real-
// time telling is where they see *why* it needed slowing down. Both are the same
// spans; only the stretch differs.
func scaleLadder(auto float64) []float64 {
	if auto <= 1 {
		return []float64{1}
	}
	out := []float64{auto}
	// One rung in the middle, then real time.
	for i := len(niceScales) - 1; i >= 0; i-- {
		if niceScales[i] < auto/4 {
			out = append(out, niceScales[i])
			break
		}
	}
	return append(out, 1)
}

// suggestSpeed keeps a long replay watchable: a four-minute batch job is not
// going to be watched in real time. It runs on the *scaled* length, since that is
// what the animation will actually be.
func suggestSpeed(durMs int) float64 {
	switch {
	case durMs <= 0:
		return 1
	case durMs > 120000:
		return 8
	case durMs > 30000:
		return 4
	case durMs > 12000:
		return 2
	}
	return 1
}

// scaleMs converts microseconds to the whole milliseconds `ir` holds, applying
// the stretch factor. Rounding rather than truncating: at scale 1 a 1.6ms span
// should read as 2ms, not 1ms.
func scaleMs(us int, scale float64) int {
	return int(float64(us)*scale/1000 + 0.5)
}

// shortestUs is the briefest span in the trace, which is what decides whether
// whole milliseconds have enough room.
func shortestUs(tr *trace.Trace) int {
	shortest := -1
	for _, s := range tr.Spans {
		if d := s.DurUs(); d > 0 && (shortest < 0 || d < shortest) {
			shortest = d
		}
	}
	if shortest < 0 {
		return 0
	}
	return shortest
}

// fmtMicros prints a measured duration at a precision that does not lie: a
// sub-millisecond span gets a decimal, because "0ms" is worse than useless.
func fmtMicros(us int) string {
	switch {
	case us < 10*1000:
		// Below 10ms a whole-millisecond figure is a lie of the same kind the
		// scaling exists to avoid: 1.7ms is not 2ms when the whole trace is 8ms.
		return trimFloat(float64(us)/1000) + "ms"
	case us < 1000*1000:
		return fmt.Sprintf("%dms", (us+500)/1000)
	}
	return trimFloat(float64(us)/1e6) + "s"
}

func fmtMillis(ms int) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	// TrimRight, not TrimSuffix: 1000ms formats as "1.00" and stripping one zero
	// would leave "1.0s".
	return trimFloat(float64(ms)/1000) + "s"
}

func trimFloat(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// fmtMillis is kept for the few places already counting whole milliseconds.

// oneLine keeps generated text inside a .dgm string, which cannot span lines.
func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, `"`, "'")), " ")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
