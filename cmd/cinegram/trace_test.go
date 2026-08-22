package main

import (
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/parser"
	"github.com/tejaspanse/cinegram/pkg/trace"
)

// diagram is the picture the traces below are replayed over.
const diagram = `flowchart LR
  gateway[API Gateway]
  checkout[checkout]
  payments[payments]
  psp[Card Network]
  db[(orders-db)]

  gateway --> checkout
  checkout --> payments
  payments --> psp
  checkout --> db
`

func table(t *testing.T) *parser.Result {
	t.Helper()
	res, bag := parser.Parse("inline.dgm", diagram)
	if bag.HasErrors() {
		t.Fatalf("the test diagram does not parse:\n%s", bag)
	}
	return res
}

// span takes milliseconds for readability and stores the microseconds the type
// actually holds.
func span(id, parent, service, name string, startMs, endMs int, failed bool) trace.Span {
	return trace.Span{ID: id, ParentID: parent, Service: service, Name: name,
		StartUs: startMs * 1000, EndUs: endMs * 1000, Failed: failed,
		Attrs: map[string]string{}}
}

// TestServicesResolveByIDThenLabel is what keeps the common case flag-free:
// `service.name` is usually already what the node is called.
func TestServicesResolveByIDThenLabel(t *testing.T) {
	res := table(t)
	tr := &trace.Trace{Spans: []trace.Span{
		span("1", "", "checkout", "a", 0, 10, false),
		// Matches the *label* "Card Network" on node psp, not any node id.
		span("2", "1", "card-network", "b", 0, 10, false),
	}}

	got, err := resolveServices(tr, res.Symbols, "")
	if err != nil {
		t.Fatal(err)
	}
	if got["checkout"] != "checkout" {
		t.Errorf("checkout resolved to %q, want the node of the same id", got["checkout"])
	}
	if got["card-network"] != "psp" {
		t.Errorf("card-network resolved to %q, want psp via its label", got["card-network"])
	}
}

// TestUnmappedServiceIsAnError: animating the wrong box is worse than refusing,
// so an unmatched service stops the run and names both sides.
func TestUnmappedServiceIsAnError(t *testing.T) {
	res := table(t)
	tr := &trace.Trace{Spans: []trace.Span{span("1", "", "orders-postgres", "a", 0, 10, false)}}

	_, err := resolveServices(tr, res.Symbols, "")
	if err == nil {
		t.Fatal("an unmappable service was accepted")
	}
	for _, want := range []string{"orders-postgres", "--map", "db"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should mention %q: %v", want, err)
		}
	}
}

func TestMapOverridesAndIsValidated(t *testing.T) {
	res := table(t)
	tr := &trace.Trace{Spans: []trace.Span{span("1", "", "orders-postgres", "a", 0, 10, false)}}

	got, err := resolveServices(tr, res.Symbols, "orders-postgres=db")
	if err != nil {
		t.Fatal(err)
	}
	if got["orders-postgres"] != "db" {
		t.Errorf("mapped to %q, want db", got["orders-postgres"])
	}

	if _, err := resolveServices(tr, res.Symbols, "orders-postgres=nosuchnode"); err == nil {
		t.Error("--map to a node that does not exist was accepted")
	}
	if _, err := resolveServices(tr, res.Symbols, "garbled"); err == nil {
		t.Error("--map without an = was accepted")
	}
}

func TestNormaliseMeetsNamingConventions(t *testing.T) {
	for _, s := range []string{"payments-api", "paymentsApi", "Payments API", "payments_api"} {
		if got := normalise(s); got != "paymentsapi" {
			t.Errorf("normalise(%q) = %q, want paymentsapi", s, got)
		}
	}
}

// TestBandsKeepOverlappingPhasesTogether is the load-bearing rule. Steps run in
// sequence, so two phases that overlapped must share one step or their timings
// would be invented.
func TestBandsKeepOverlappingPhasesTogether(t *testing.T) {
	tr := &trace.Trace{Spans: []trace.Span{
		span("1", "", "gateway", "root", 0, 1000, false),
		span("2", "1", "checkout", "a", 40, 260, false),
		span("3", "1", "payments", "b", 200, 900, false), // overlaps a
		span("4", "1", "db", "c", 950, 990, false),       // after both
	}}
	tr.Roots = []trace.Span{tr.Spans[0]}

	bands := bandsOf(tr, tr.Phases())
	if len(bands) != 2 {
		t.Fatalf("got %d bands, want 2 — a and b overlap, c stands alone", len(bands))
	}
	if len(bands[0].phases) != 2 {
		t.Errorf("first band holds %d phases, want the two overlapping ones", len(bands[0].phases))
	}
	if bands[0].start != 40_000 || bands[0].end != 900_000 {
		t.Errorf("first band = [%d,%d]us, want [40000,900000]", bands[0].start, bands[0].end)
	}
	if bands[1].start != 950_000 {
		t.Errorf("second band starts at %dus, want 950000", bands[1].start)
	}
}

// TestBandCoversASubtreeOutlivingItsParent: a child can outlast the parent's
// recorded end, and a band that stopped at the parent would clip it.
func TestBandCoversASubtreeOutlivingItsParent(t *testing.T) {
	tr := &trace.Trace{Spans: []trace.Span{
		span("1", "", "gateway", "root", 0, 1000, false),
		span("2", "1", "checkout", "a", 40, 100, false),
		span("3", "2", "payments", "slow child", 50, 800, false),
	}}
	tr.Roots = []trace.Span{tr.Spans[0]}

	bands := bandsOf(tr, tr.Phases())
	if len(bands) != 1 || bands[0].end != 800_000 {
		t.Errorf("band = %+v, want one ending at 800", bands)
	}
}

// TestCrossServiceSpanBecomesAFlow, and same-service work becomes a highlight.
func TestCrossServiceSpanBecomesAFlow(t *testing.T) {
	res := table(t)
	nodes := map[string]string{"checkout": "checkout", "payments": "payments"}
	byID := map[string]trace.Span{"1": span("1", "", "checkout", "parent", 0, 500, false)}

	line, warn := actionFor(span("2", "1", "payments", "charge", 100, 400, false), byID, nodes, res.Symbols, 0, 1)
	if warn != "" {
		t.Errorf("unexpected warning: %s", warn)
	}
	for _, want := range []string{"flow checkout -> payments", `label: "charge"`, "at: 100ms", "dur: 300ms"} {
		if !strings.Contains(line, want) {
			t.Errorf("%q missing from: %s", want, line)
		}
	}

	// Same service: work, not a call.
	same, _ := actionFor(span("3", "1", "checkout", "internal", 10, 20, false), byID, nodes, res.Symbols, 0, 1)
	if !strings.HasPrefix(same, "highlight checkout") {
		t.Errorf("same-service span became %q, want a highlight", same)
	}
}

// TestFailedSpanCarriesItsStatus, so a failed call draws as a failed flow.
func TestFailedSpanCarriesItsStatus(t *testing.T) {
	res := table(t)
	nodes := map[string]string{"checkout": "checkout", "payments": "payments"}
	byID := map[string]trace.Span{"1": span("1", "", "checkout", "parent", 0, 500, false)}

	line, _ := actionFor(span("2", "1", "payments", "charge", 0, 100, true), byID, nodes, res.Symbols, 0, 1)
	if !strings.Contains(line, "status: fail") {
		t.Errorf("a failed span produced %q, want status: fail", line)
	}
}

// TestUndrawnCallIsReportedNotInvented is one of the more valuable diagnostics
// here: it means the diagram and production disagree about the architecture.
func TestUndrawnCallIsReportedNotInvented(t *testing.T) {
	res := table(t)
	// The diagram draws checkout --> db, but not payments --> db.
	nodes := map[string]string{"payments": "payments", "orders-postgres": "db"}
	byID := map[string]trace.Span{"1": span("1", "", "payments", "parent", 0, 500, false)}

	line, warn := actionFor(span("2", "1", "orders-postgres", "SELECT", 10, 60, false), byID, nodes, res.Symbols, 0, 1)
	if warn == "" {
		t.Fatal("a call the diagram does not draw passed without a warning")
	}
	if !strings.Contains(warn, "does not draw") {
		t.Errorf("warning = %q", warn)
	}
	if !strings.HasPrefix(line, "highlight db") {
		t.Errorf("fell back to %q, want a highlight rather than an invented edge", line)
	}
}

// TestZeroLengthSpanStillDraws: a sub-millisecond span rounds to 0ms, and a
// zero-duration flow would animate nothing at all.
func TestZeroLengthSpanStillDraws(t *testing.T) {
	res := table(t)
	nodes := map[string]string{"checkout": "checkout", "payments": "payments"}
	byID := map[string]trace.Span{"1": span("1", "", "checkout", "parent", 0, 500, false)}

	line, _ := actionFor(span("2", "1", "payments", "instant", 5, 5, false), byID, nodes, res.Symbols, 0, 1)
	if strings.Contains(line, "dur: 0ms") {
		t.Errorf("a zero-length span produced %q, which draws nothing", line)
	}
}

func TestSuggestSpeedOnlyFastForwards(t *testing.T) {
	if s := suggestSpeed(180000); s <= 1 {
		t.Errorf("a three-minute animation got speed %v, want faster than real time", s)
	}
	if s := suggestSpeed(8000); s != 1 {
		t.Errorf("an 8s animation got speed %v, want real time", s)
	}
	// Slowing down is suggestScale's job: playback rate cannot go far enough and
	// the reader's own speed menu bottoms out at 0.25x.
	if s := suggestSpeed(300); s != 1 {
		t.Errorf("a short animation got speed %v; slowing down is the scale's job", s)
	}
}

// TestSuggestScaleStretchesWhatCannotBeWatched covers the case playback rate
// cannot reach: an 8ms trace needs the timeline itself made longer, both to be
// visible and because whole milliseconds have no room for a 0.8ms span.
func TestSuggestScaleStretchesWhatCannotBeWatched(t *testing.T) {
	if got := suggestScale(8_000); got < 100 {
		t.Errorf("an 8ms trace got scale %v, want a large stretch", got)
	}
	if got := suggestScale(2_000_000); got != 1 {
		t.Errorf("a 2s trace got scale %v, want no stretch", got)
	}
	if got := suggestScale(0); got != 1 {
		t.Errorf("an empty trace got scale %v, want 1", got)
	}
}

// TestScaleLadderOffersRealTimeToo: the picker is the selector, and the real-time
// telling is what shows the reader why a stretch was needed.
func TestScaleLadderOffersRealTimeToo(t *testing.T) {
	rungs := scaleLadder(1000)
	if len(rungs) < 2 {
		t.Fatalf("ladder = %v, want several rungs", rungs)
	}
	if rungs[0] != 1000 {
		t.Errorf("ladder starts at %v, want the chosen scale first", rungs[0])
	}
	if rungs[len(rungs)-1] != 1 {
		t.Errorf("ladder ends at %v, want real time last", rungs[len(rungs)-1])
	}
	if got := scaleLadder(1); len(got) != 1 || got[0] != 1 {
		t.Errorf("a trace needing no stretch got %v, want just real time", got)
	}
}

func TestFmtMicrosKeepsSubMillisecondTruth(t *testing.T) {
	for in, want := range map[int]string{
		800: "0.8ms", 1700: "1.7ms", 8000: "8ms",
		220_000: "220ms", 1_910_000: "1.91s",
	} {
		if got := fmtMicros(in); got != want {
			t.Errorf("fmtMicros(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFmtMillis(t *testing.T) {
	for in, want := range map[int]string{0: "0ms", 999: "999ms", 1000: "1s", 2600: "2.6s", 2570: "2.57s"} {
		if got := fmtMillis(in); got != want {
			t.Errorf("fmtMillis(%d) = %q, want %q", in, got, want)
		}
	}
}

// TestOneLineKeepsGeneratedTextInsideAString: a .dgm string cannot span lines,
// and a stray quote would end it early.
func TestOneLineKeepsGeneratedTextInsideAString(t *testing.T) {
	got := oneLine("GET /a\n  said \"no\"")
	if strings.ContainsAny(got, "\n\"") {
		t.Errorf("oneLine left a newline or quote in %q", got)
	}
}

// TestRenderedScenarioParses is the end-to-end guard: whatever this command
// writes has to be something the parser accepts, or the feature emits garbage.
func TestRenderedScenarioParses(t *testing.T) {
	res := table(t)
	tr := &trace.Trace{ID: "abc", Spans: []trace.Span{
		span("1", "", "gateway", "POST /checkout", 0, 900, false),
		span("2", "1", "checkout", "checkout.create", 20, 880, false),
		span("3", "2", "payments", "charge", 100, 700, true),
		span("4", "3", "card-network", "POST /authorize", 120, 650, true),
		span("5", "2", "orders-postgres", "INSERT", 750, 850, false),
	}}
	tr.Roots = []trace.Span{tr.Spans[0]}

	nodes, err := resolveServices(tr, res.Symbols, "orders-postgres=db")
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := renderScenario(tr, nodes, res.Symbols, "replay", 1, []float64{1})

	full, bag := parser.Parse("gen.dgm", diagram+"\n"+sc)
	if bag.HasErrors() {
		t.Fatalf("the generated scenario does not parse:\n%s\n--- generated ---\n%s", bag, sc)
	}
	if len(full.Document.Scenarios) != 1 {
		t.Fatalf("got %d scenarios, want 1", len(full.Document.Scenarios))
	}
	if !strings.Contains(sc, "outcome: fail") {
		t.Error("a trace containing failures should produce outcome: fail")
	}
	if !strings.Contains(sc, "%% Generated by") {
		t.Error("the generated block should say it was generated")
	}
}

// TestScaleMsRounds is the single place microseconds become the whole
// milliseconds `ir` holds, so both the rounding and the stretch are pinned here.
func TestScaleMsRounds(t *testing.T) {
	cases := []struct {
		us    int
		scale float64
		want  int
	}{
		{1600, 1, 2},     // rounds up, not truncates
		{1400, 1, 1},     // and down
		{800, 1000, 800}, // 0.8ms stretched 1000x is 800ms
		{200, 1000, 200}, // a 0.2ms gap becomes visible
		{800, 1, 1},      // unstretched, it has nowhere to go — hence the scale
		{1_910_000, 1, 1910},
	}
	for _, c := range cases {
		if got := scaleMs(c.us, c.scale); got != c.want {
			t.Errorf("scaleMs(%dus, %v) = %d, want %d", c.us, c.scale, got, c.want)
		}
	}
}
