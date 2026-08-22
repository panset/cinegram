package trace

import (
	"fmt"
	"strings"
	"testing"
)

// otlp builds an OTLP/JSON document from a compact description, so the tests
// read as traces rather than as JSON.
type sp struct {
	id, parent, service, name string
	startNs, endNs            int64
	failed                    bool
}

func otlp(spans ...sp) []byte {
	byService := map[string][]sp{}
	var order []string
	for _, s := range spans {
		if _, seen := byService[s.service]; !seen {
			order = append(order, s.service)
		}
		byService[s.service] = append(byService[s.service], s)
	}
	var rs []string
	for _, svc := range order {
		var js []string
		for _, s := range byService[svc] {
			status := ""
			if s.failed {
				status = `,"status":{"code":2,"message":"boom"}`
			}
			js = append(js, fmt.Sprintf(
				`{"traceId":"abc","spanId":%q,"parentSpanId":%q,"name":%q,`+
					`"startTimeUnixNano":"%d","endTimeUnixNano":"%d"%s}`,
				s.id, s.parent, s.name, s.startNs, s.endNs, status))
		}
		rs = append(rs, fmt.Sprintf(
			`{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":%q}}]},`+
				`"scopeSpans":[{"spans":[%s]}]}`, svc, strings.Join(js, ",")))
	}
	return []byte(`{"resourceSpans":[` + strings.Join(rs, ",") + `]}`)
}

const ms = 1_000_000
const base = 1_700_000_000_000_000_000 // an arbitrary epoch nanosecond

// TestParseRebasesOntoTheTraceStart is the whole point of the package: a
// timeline's origin is when the story began, not 1970.
func TestParseRebasesOntoTheTraceStart(t *testing.T) {
	tr, err := Parse(otlp(
		sp{id: "1", service: "gw", name: "POST /checkout", startNs: base, endNs: base + 900*ms},
		sp{id: "2", parent: "1", service: "pay", name: "charge", startNs: base + 100*ms, endNs: base + 600*ms},
	))
	if err != nil {
		t.Fatal(err)
	}
	if tr.Spans[0].Start != 0 {
		t.Errorf("first span starts at %d, want 0", tr.Spans[0].Start)
	}
	if got := tr.Spans[1]; got.Start != 100 || got.Dur() != 500 {
		t.Errorf("child at %d for %dms, want 100 for 500", got.Start, got.Dur())
	}
	if tr.Dur() != 900 {
		t.Errorf("trace length = %d, want 900", tr.Dur())
	}
	if tr.ID != "abc" {
		t.Errorf("trace id = %q, want abc", tr.ID)
	}
}

// TestNanosecondPrecisionSurvives covers why the timestamps are json.Number: a
// 19-digit nanosecond epoch is past float64's exact-integer range, so decoding
// through a float would shift the whole trace.
func TestNanosecondPrecisionSurvives(t *testing.T) {
	tr, err := Parse(otlp(
		sp{id: "1", service: "gw", name: "root", startNs: base + 1, endNs: base + 1 + 250*ms},
		sp{id: "2", parent: "1", service: "db", name: "query", startNs: base + 1 + 7*ms, endNs: base + 1 + 200*ms},
	))
	if err != nil {
		t.Fatal(err)
	}
	if tr.Spans[1].Start != 7 {
		t.Errorf("offset = %dms, want 7 — nanosecond precision was lost", tr.Spans[1].Start)
	}
}

func TestRootsAreSpansWhoseParentIsAbsent(t *testing.T) {
	// "9" is a parent that was never exported — a subtree on its own, which is
	// what a partial export looks like.
	tr, err := Parse(otlp(
		sp{id: "1", parent: "9", service: "gw", name: "orphan", startNs: base, endNs: base + 100*ms},
		sp{id: "2", parent: "1", service: "db", name: "query", startNs: base + 10*ms, endNs: base + 40*ms},
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Roots) != 1 || tr.Roots[0].ID != "1" {
		t.Errorf("roots = %+v, want just the orphan", tr.Roots)
	}
}

// TestPhasesGroupByTheRootsChildren: a request's children are the things that
// happened, and each is one step's worth of story.
func TestPhasesGroupByTheRootsChildren(t *testing.T) {
	tr, _ := Parse(otlp(
		sp{id: "1", service: "gw", name: "POST /checkout", startNs: base, endNs: base + 900*ms},
		sp{id: "2", parent: "1", service: "auth", name: "authorise", startNs: base + 10*ms, endNs: base + 200*ms},
		sp{id: "3", parent: "1", service: "pay", name: "charge", startNs: base + 210*ms, endNs: base + 800*ms},
		sp{id: "4", parent: "3", service: "db", name: "insert", startNs: base + 300*ms, endNs: base + 500*ms},
	))
	var names []string
	for _, p := range tr.Phases() {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "authorise,charge" {
		t.Errorf("phases = %v, want the root's two children in start order", names)
	}
}

// TestPhasesDescendThroughASoleChild: one phase for the whole trace narrates
// nothing, so a single-child root is stepped over.
func TestPhasesDescendThroughASoleChild(t *testing.T) {
	tr, _ := Parse(otlp(
		sp{id: "1", service: "lb", name: "ingress", startNs: base, endNs: base + 900*ms},
		sp{id: "2", parent: "1", service: "gw", name: "POST /checkout", startNs: base, endNs: base + 890*ms},
		sp{id: "3", parent: "2", service: "auth", name: "authorise", startNs: base + 10*ms, endNs: base + 200*ms},
		sp{id: "4", parent: "2", service: "pay", name: "charge", startNs: base + 210*ms, endNs: base + 800*ms},
	))
	var names []string
	for _, p := range tr.Phases() {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "authorise,charge" {
		t.Errorf("phases = %v, want the grandchildren, not the lone child", names)
	}
}

func TestSubtreeCarriesEverythingBeneath(t *testing.T) {
	tr, _ := Parse(otlp(
		sp{id: "1", service: "gw", name: "root", startNs: base, endNs: base + 900*ms},
		sp{id: "2", parent: "1", service: "pay", name: "charge", startNs: base + 100*ms, endNs: base + 800*ms},
		sp{id: "3", parent: "2", service: "db", name: "insert", startNs: base + 200*ms, endNs: base + 400*ms},
		sp{id: "4", parent: "3", service: "db", name: "fsync", startNs: base + 250*ms, endNs: base + 380*ms},
		sp{id: "5", parent: "1", service: "mail", name: "notify", startNs: base + 810*ms, endNs: base + 880*ms},
	))
	var ids []string
	for _, s := range tr.Subtree("2") {
		ids = append(ids, s.ID)
	}
	if strings.Join(ids, ",") != "2,3,4" {
		t.Errorf("subtree = %v, want the span and its descendants only", ids)
	}
}

func TestServicesAreSortedAndDeduped(t *testing.T) {
	tr, _ := Parse(otlp(
		sp{id: "1", service: "gw", name: "a", startNs: base, endNs: base + 10*ms},
		sp{id: "2", parent: "1", service: "pay", name: "b", startNs: base, endNs: base + 10*ms},
		sp{id: "3", parent: "1", service: "gw", name: "c", startNs: base, endNs: base + 10*ms},
	))
	if got := strings.Join(tr.Services(), ","); got != "gw,pay" {
		t.Errorf("services = %q, want gw,pay", got)
	}
}

func TestFailedReadsBothStatusSpellings(t *testing.T) {
	tr, _ := Parse(otlp(
		sp{id: "1", service: "gw", name: "ok", startNs: base, endNs: base + 10*ms},
		sp{id: "2", parent: "1", service: "pay", name: "bad", startNs: base, endNs: base + 10*ms, failed: true},
	))
	if !tr.Failed() {
		t.Error("a trace containing an error span should report Failed")
	}
	if tr.Spans[0].Failed {
		t.Error("the healthy span was marked failed")
	}

	// The enum's name, which some exporters send instead of the number.
	named := []byte(`{"resourceSpans":[{"resource":{"attributes":[]},"scopeSpans":[{"spans":[
      {"spanId":"1","name":"x","startTimeUnixNano":"1","endTimeUnixNano":"2",
       "status":{"code":"STATUS_CODE_ERROR"}}]}]}]}`)
	tr2, err := Parse(named)
	if err != nil {
		t.Fatal(err)
	}
	if !tr2.Spans[0].Failed {
		t.Error(`status code "STATUS_CODE_ERROR" was not read as a failure`)
	}
}

// TestBackwardsClockIsClamped: an exporter whose end precedes its start must not
// become a negative duration on a timeline.
func TestBackwardsClockIsClamped(t *testing.T) {
	tr, _ := Parse(otlp(
		sp{id: "1", service: "gw", name: "root", startNs: base + 500*ms, endNs: base + 100*ms},
	))
	if got := tr.Spans[0].Dur(); got != 0 {
		t.Errorf("duration = %d, want 0 rather than a negative span", got)
	}
}

func TestParseRejectsWhatIsNotATrace(t *testing.T) {
	for name, in := range map[string]string{
		"not json":   `{ nope`,
		"no spans":   `{"resourceSpans":[]}`,
		"empty file": ``,
	} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s was accepted as a trace", name)
		}
	}
}

// TestLegacyInstrumentationLibrarySpans keeps pre-1.0 exports working, since a
// trace that silently contains no spans is a confusing thing to debug.
func TestLegacyInstrumentationLibrarySpans(t *testing.T) {
	in := []byte(`{"resourceSpans":[{"resource":{"attributes":[
      {"key":"service.name","value":{"stringValue":"gw"}}]},
      "instrumentationLibrarySpans":[{"spans":[
      {"spanId":"1","name":"root","startTimeUnixNano":"1000000","endTimeUnixNano":"3000000"}]}]}]}`)
	tr, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Spans) != 1 || tr.Spans[0].Service != "gw" {
		t.Errorf("spans = %+v, want one span in gw", tr.Spans)
	}
}

func TestAttributesAreFlattenedToStrings(t *testing.T) {
	in := []byte(`{"resourceSpans":[{"resource":{"attributes":[]},"scopeSpans":[{"spans":[
      {"spanId":"1","name":"x","startTimeUnixNano":"0","endTimeUnixNano":"1000000",
       "attributes":[
         {"key":"http.status_code","value":{"intValue":"503"}},
         {"key":"db.system","value":{"stringValue":"postgres"}},
         {"key":"retry","value":{"boolValue":true}}]}]}]}]}`)
	tr, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	a := tr.Spans[0].Attrs
	if a["http.status_code"] != "503" || a["db.system"] != "postgres" || a["retry"] != "true" {
		t.Errorf("attrs = %v, want every value flattened to its string form", a)
	}
}
