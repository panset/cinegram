// Package trace reads a distributed trace and normalises it into spans with
// millisecond offsets from the start of the trace.
//
// It exists because of a coincidence worth exploiting: pkg/ir holds absolute
// integer milliseconds and no geometry, and a trace is absolute times and no
// geometry. The two are the same shape, so a trace can be lowered into a
// scenario without the compiler, the runtime or the emitters learning anything.
//
// Nothing here does I/O and nothing here knows what a diagram is. This package
// turns bytes into spans; `cinegram trace` decides which node a span belongs to.
package trace

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Span is one operation, with times rebased so the trace starts at zero.
//
// Start and End are milliseconds from the first span in the trace, because that
// is what a scenario needs: a timeline's origin is the beginning of the story,
// not 1970.
type Span struct {
	ID       string
	ParentID string
	Service  string
	Name     string
	Start    int // ms from the trace's first span
	End      int // ms from the trace's first span
	Failed   bool
	Message  string            // status message, when the span failed
	Attrs    map[string]string // flattened span attributes
}

// Dur is how long the span took.
func (s Span) Dur() int { return s.End - s.Start }

// Trace is every span of one trace, ordered by start time.
type Trace struct {
	ID    string
	Spans []Span

	// Roots are the spans with no parent inside this trace. A well-formed trace
	// has one; a trace assembled from partial exports can have several, and
	// dropping the extras would silently lose work.
	Roots []Span
}

// Dur is the wall-clock length of the whole trace.
func (t *Trace) Dur() int {
	end := 0
	for _, s := range t.Spans {
		if s.End > end {
			end = s.End
		}
	}
	return end
}

// Services is every distinct service in the trace, sorted, so a caller can
// report what it could not map without a second pass.
func (t *Trace) Services() []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range t.Spans {
		if s.Service != "" && !seen[s.Service] {
			seen[s.Service] = true
			out = append(out, s.Service)
		}
	}
	sort.Strings(out)
	return out
}

// Children returns the spans whose parent is id, in start order.
func (t *Trace) Children(id string) []Span {
	var out []Span
	for _, s := range t.Spans {
		if s.ParentID == id {
			out = append(out, s)
		}
	}
	return out
}

// Subtree is span id and everything beneath it, in start order.
func (t *Trace) Subtree(id string) []Span {
	byParent := map[string][]Span{}
	for _, s := range t.Spans {
		byParent[s.ParentID] = append(byParent[s.ParentID], s)
	}
	var out []Span
	var walk func(string)
	walk = func(cur string) {
		for _, c := range byParent[cur] {
			out = append(out, c)
			walk(c.ID)
		}
	}
	for _, s := range t.Spans {
		if s.ID == id {
			out = append(out, s)
			break
		}
	}
	walk(id)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// Phases splits the trace into the groups a scenario should narrate as steps.
//
// A request's shape is usually one root with a handful of children — authorise,
// reserve stock, charge, notify — and each of those subtrees is one thing that
// happened, which is exactly what a step is. So the root's children become the
// phases and everything beneath a child travels with it.
//
// Two fallbacks keep this total. A root with a single child would produce one
// step for the whole trace, which narrates nothing, so its grandchildren are
// used instead. A trace with no usable tree at all degrades to one phase per
// span, which is still ordered and still correct — just less well grouped.
func (t *Trace) Phases() []Span {
	if len(t.Roots) == 0 {
		return t.Spans
	}
	var phases []Span
	for _, r := range t.Roots {
		kids := t.Children(r.ID)
		if len(kids) == 1 {
			if grandkids := t.Children(kids[0].ID); len(grandkids) > 0 {
				phases = append(phases, grandkids...)
				continue
			}
		}
		if len(kids) == 0 {
			phases = append(phases, r)
			continue
		}
		phases = append(phases, kids...)
	}
	sort.SliceStable(phases, func(i, j int) bool { return phases[i].Start < phases[j].Start })
	return phases
}

// Failed reports whether any span in the trace failed, which is what decides a
// scenario's `outcome`.
func (t *Trace) Failed() bool {
	for _, s := range t.Spans {
		if s.Failed {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// OTLP/JSON
// ---------------------------------------------------------------------------

// The subset of OTLP/JSON that matters here. Only the fields actually read are
// declared: an exporter is free to add more, and a decoder that insisted on
// knowing every field would break on the next version of the spec.
type otlpFile struct {
	ResourceSpans []struct {
		Resource struct {
			Attributes []otlpAttr `json:"attributes"`
		} `json:"resource"`
		ScopeSpans []struct {
			Spans []otlpSpan `json:"spans"`
		} `json:"scopeSpans"`
		// Pre-1.0 exporters wrote instrumentationLibrarySpans. Reading both
		// costs one field and saves a confusing "trace has no spans".
		InstrumentationLibrarySpans []struct {
			Spans []otlpSpan `json:"spans"`
		} `json:"instrumentationLibrarySpans"`
	} `json:"resourceSpans"`
}

type otlpSpan struct {
	TraceID      string      `json:"traceId"`
	SpanID       string      `json:"spanId"`
	ParentSpanID string      `json:"parentSpanId"`
	Name         string      `json:"name"`
	Start        json.Number `json:"startTimeUnixNano"`
	End          json.Number `json:"endTimeUnixNano"`
	Attributes   []otlpAttr  `json:"attributes"`
	Status       struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
}

type otlpAttr struct {
	Key   string `json:"key"`
	Value struct {
		StringValue *string      `json:"stringValue"`
		IntValue    *json.Number `json:"intValue"`
		DoubleValue *json.Number `json:"doubleValue"`
		BoolValue   *bool        `json:"boolValue"`
	} `json:"value"`
}

func (a otlpAttr) str() string {
	switch {
	case a.Value.StringValue != nil:
		return *a.Value.StringValue
	case a.Value.IntValue != nil:
		return a.Value.IntValue.String()
	case a.Value.DoubleValue != nil:
		return a.Value.DoubleValue.String()
	case a.Value.BoolValue != nil:
		return strconv.FormatBool(*a.Value.BoolValue)
	}
	return ""
}

// Parse reads OTLP/JSON and rebases every span onto the trace's own start.
//
// Times arrive as nanoseconds since the epoch, and as *strings* — JSON cannot
// hold a 19-digit integer exactly, so the OTLP spec sends them quoted and a
// float64 would lose the microseconds. json.Number keeps the digits and the
// division to milliseconds happens on int64.
func Parse(data []byte) (*Trace, error) {
	var f otlpFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("this is not OTLP/JSON: %w", err)
	}

	type raw struct {
		span    otlpSpan
		service string
	}
	var rows []raw
	for _, rs := range f.ResourceSpans {
		service := ""
		for _, a := range rs.Resource.Attributes {
			if a.Key == "service.name" {
				service = a.str()
			}
		}
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				rows = append(rows, raw{s, service})
			}
		}
		for _, ss := range rs.InstrumentationLibrarySpans {
			for _, s := range ss.Spans {
				rows = append(rows, raw{s, service})
			}
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no spans found: expected resourceSpans[].scopeSpans[].spans[]")
	}

	// The origin is the earliest start in the file, so the story begins when the
	// first thing happened rather than at an arbitrary root.
	origin := int64(-1)
	for _, r := range rows {
		if n, ok := nanos(r.span.Start); ok && (origin < 0 || n < origin) {
			origin = n
		}
	}
	if origin < 0 {
		return nil, fmt.Errorf("no span carried a readable startTimeUnixNano")
	}

	t := &Trace{}
	ids := map[string]bool{}
	for _, r := range rows {
		start, ok1 := nanos(r.span.Start)
		end, ok2 := nanos(r.span.End)
		if !ok1 || !ok2 {
			continue // a span with no clock cannot be placed on a timeline
		}
		if t.ID == "" {
			t.ID = r.span.TraceID
		}
		attrs := map[string]string{}
		for _, a := range r.span.Attributes {
			attrs[a.Key] = a.str()
		}
		s := Span{
			ID:       r.span.SpanID,
			ParentID: r.span.ParentSpanID,
			Service:  r.service,
			Name:     r.span.Name,
			Start:    int((start - origin) / 1e6),
			End:      int((end - origin) / 1e6),
			Failed:   failed(r.span.Status.Code),
			Message:  r.span.Status.Message,
			Attrs:    attrs,
		}
		// A clock that ran backwards is the exporter's problem, not something to
		// propagate into a timeline as a negative duration.
		if s.End < s.Start {
			s.End = s.Start
		}
		t.Spans = append(t.Spans, s)
		ids[s.ID] = true
	}
	if len(t.Spans) == 0 {
		return nil, fmt.Errorf("every span was missing a usable start or end time")
	}

	sort.SliceStable(t.Spans, func(i, j int) bool { return t.Spans[i].Start < t.Spans[j].Start })

	// A root is a span whose parent is not in this file — which covers both a
	// true root and a subtree exported on its own.
	for _, s := range t.Spans {
		if s.ParentID == "" || !ids[s.ParentID] {
			t.Roots = append(t.Roots, s)
		}
	}
	return t, nil
}

// nanos reads a quoted or bare integer nanosecond timestamp.
func nanos(n json.Number) (int64, bool) {
	if n.String() == "" {
		return 0, false
	}
	v, err := n.Int64()
	if err != nil {
		// Some exporters write nanoseconds as a float in scientific notation.
		f, ferr := n.Float64()
		if ferr != nil {
			return 0, false
		}
		return int64(f), true
	}
	return v, true
}

// failed reads OTLP's status code, which arrives as either the enum's number or
// its name depending on the exporter.
func failed(code any) bool {
	switch v := code.(type) {
	case float64:
		return int(v) == 2 // STATUS_CODE_ERROR
	case string:
		return strings.EqualFold(v, "STATUS_CODE_ERROR") || v == "2" || strings.EqualFold(v, "ERROR")
	}
	return false
}
