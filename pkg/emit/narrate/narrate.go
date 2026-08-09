// Package narrate turns a compiled timeline into a walkthrough in prose.
//
// This is the output an agent reads instead of watching the animation. A
// timeline is precise but it is a list of intervals; an animation is legible
// but only to something with eyes. Narration is the third form: the same facts,
// ordered as a reader would meet them, with each track stated as a sentence.
//
// The package is pure over ir.Timeline — no parser, no loader, no filesystem —
// so anything that can compile a document can narrate one, and the Markdown and
// JSON forms are two renderings of one structure rather than two walks of the
// timeline that might disagree.
package narrate

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ir"
)

// Doc is a whole timeline, narrated.
type Doc struct {
	Title string `json:"title,omitempty"`
	Root  string `json:"root,omitempty"`
	Views []View `json:"views"`
}

// View is one diagram's walkthrough.
type View struct {
	ID           string        `json:"id"`
	Title        string        `json:"title,omitempty"`
	Diagram      string        `json:"diagram,omitempty"`
	Scenarios    []Scenario    `json:"scenarios,omitempty"`
	Interactions []Interaction `json:"interactions,omitempty"`
}

// Scenario is one walkthrough of one diagram.
type Scenario struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Duration int    `json:"duration"`
	Steps    []Step `json:"steps,omitempty"`

	// Standing is the state that outlives the step that set it, with the
	// window over which it holds.
	Standing []Event `json:"standing,omitempty"`
}

// Step is one beat.
type Step struct {
	ID     string  `json:"id"`
	Name   string  `json:"name,omitempty"`
	Desc   string  `json:"desc,omitempty"`
	Start  int     `json:"start"`
	End    int     `json:"end"`
	Events []Event `json:"events,omitempty"`
}

// Event is one track, both as a sentence and as the fields it was built from.
//
// Both, deliberately. Text is what a reader (or a model reading prose) wants;
// the fields are what a program filtering for "every failing flow" wants, and
// re-parsing the sentence to get them back would be absurd.
type Event struct {
	Kind  string `json:"kind"`
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`

	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Target string `json:"target,omitempty"`
	Label  string `json:"label,omitempty"`
	Value  string `json:"value,omitempty"`
	Note   string `json:"note,omitempty"`
	Style  string `json:"style,omitempty"`
	Status string `json:"status,omitempty"`
}

// Interaction is one click binding.
type Interaction struct {
	Kind    string   `json:"kind"`
	Source  string   `json:"source"`
	Text    string   `json:"text"`
	View    string   `json:"view,omitempty"`
	Step    string   `json:"step,omitempty"`
	URL     string   `json:"url,omitempty"`
	Targets []string `json:"targets,omitempty"`
	Label   string   `json:"label,omitempty"`
}

// Build walks a timeline into its narrated form.
func Build(t *ir.Timeline, title string) Doc {
	doc := Doc{Title: title, Root: t.Root}
	for i := range t.Views {
		doc.Views = append(doc.Views, buildView(&t.Views[i]))
	}
	return doc
}

func buildView(v *ir.View) View {
	out := View{ID: v.ID, Title: v.Title, Diagram: v.Diagram.Type}

	// A scene track names a frame; what a reader wants told is what that frame
	// shows. The captions live on the view, so they are resolved once here
	// rather than looked up per event.
	captions := map[string]string{}
	if v.Storyboard != nil {
		for _, f := range v.Storyboard.Frames {
			captions[f.ID] = f.Caption
		}
	}

	for i := range v.Scenarios {
		out.Scenarios = append(out.Scenarios, buildScenario(&v.Scenarios[i], captions))
	}
	for _, b := range v.Bindings {
		out.Interactions = append(out.Interactions, buildInteraction(b))
	}
	return out
}

func buildScenario(sc *ir.Scenario, captions map[string]string) Scenario {
	out := Scenario{ID: sc.ID, Name: sc.Name, Duration: sc.Duration}

	for _, st := range sc.Steps {
		step := Step{ID: st.ID, Name: st.Name, Desc: st.Desc, Start: st.Start, End: st.End}
		for _, tr := range st.Tracks {
			step.Events = append(step.Events, buildEvent(tr, captions))
		}
		// Tracks are emitted in source order, which is not always time order
		// once `delay` and `seq` are involved. A walkthrough is read forwards,
		// so sort it that way — stably, so simultaneous actions keep the order
		// the author wrote them in.
		sort.SliceStable(step.Events, func(i, j int) bool {
			return step.Events[i].Start < step.Events[j].Start
		})
		out.Steps = append(out.Steps, step)
	}

	for _, tr := range sc.Persistent {
		out.Standing = append(out.Standing, buildEvent(tr, captions))
	}
	return out
}

func buildEvent(tr ir.Track, captions map[string]string) Event {
	e := Event{
		Kind: string(tr.Kind), Start: tr.Start, End: tr.End,
		From: tr.From, To: tr.To, Target: tr.Target,
		Label: tr.Label, Value: tr.Value, Note: tr.Text,
		Style: tr.Style, Status: tr.Status,
	}
	// A scene's caption is the only field a track cannot carry itself, and it
	// is the whole content of the sentence. Fold it into Value so the
	// structured half says what the panel showed too.
	if tr.Kind == ir.TrackScene && e.Value == "" {
		e.Value = captions[tr.Target]
	}
	e.Text = sentence(e)
	return e
}

// sentence states one event in plain English.
func sentence(e Event) string {
	switch e.Kind {
	case "flow":
		s := "**" + e.From + " → " + e.To + "**"
		if e.Label != "" {
			s += " carries " + strconv.Quote(e.Label)
		} else {
			s += " carries a message"
		}
		if e.Status == "fail" {
			s += ", which fails"
		}
		return s + " " + window(e)

	case "note":
		return "a note on **" + e.Target + "**: " + strconv.Quote(e.Note) + " " + window(e)

	case "highlight":
		return "**" + e.Target + "** is highlighted" + styleSuffix(e) + " " + window(e)
	case "dim":
		return "**" + e.Target + "** is dimmed " + window(e)
	case "pulse":
		return "**" + e.Target + "** pulses " + window(e)
	case "show":
		return "**" + e.Target + "** appears " + window(e)
	case "hide":
		return "**" + e.Target + "** is concealed " + window(e)
	case "focus":
		return "attention narrows to **" + e.Target + "** " + window(e)

	case "scene":
		// The caption, when there is one: it says what the viewer is looking
		// at, where the frame id only says which picture it is.
		what := e.Value
		if what == "" {
			what = e.Target
		}
		return "the storyboard shows *" + what + "* " + window(e)

	case "set":
		if e.Label != "" && e.Value != "" {
			return "**" + e.Target + "** is badged " + strconv.Quote(e.Label) +
				" and marked *" + e.Value + "* " + window(e)
		}
		if e.Label != "" {
			return "**" + e.Target + "** is badged " + strconv.Quote(e.Label) + " " + window(e)
		}
		return "**" + e.Target + "** is marked *" + e.Value + "* " + window(e)

	case "gauge":
		return "**" + e.Target + "** reads " + e.Label + " = " + strconv.Quote(e.Value) + " " + window(e)
	}
	return e.Kind + " on **" + e.Target + "** " + window(e)
}

func styleSuffix(e Event) string {
	if e.Style == "" {
		return ""
	}
	return " (" + e.Style + ")"
}

func window(e Event) string {
	if e.Start == e.End {
		return "(at " + secs(e.Start) + ")"
	}
	return "(" + secs(e.Start) + "–" + secs(e.End) + ")"
}

func secs(ms int) string {
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func buildInteraction(b ir.Binding) Interaction {
	out := Interaction{
		Kind: b.Kind, Source: b.Source, View: b.View, Step: b.Step,
		URL: b.URL, Targets: b.Targets, Label: b.Label,
	}
	click := "Clicking **" + b.Source + "** "
	switch b.Kind {
	case "view":
		out.Text = click + "opens the **" + b.View + "** view"
	case "step":
		out.Text = click + "jumps to step **" + b.Step + "**"
	case "url":
		out.Text = click + "opens <" + b.URL + "> in a new tab"
	case "reveal":
		out.Text = click + "reveals " + strings.Join(b.Targets, ", ")
	default:
		out.Text = click + "does something the narrator does not know about (" + b.Kind + ")"
	}
	if b.Label != "" {
		out.Text += " — labelled " + strconv.Quote(b.Label)
	}
	return out
}

// JSON renders the walkthrough as data.
func JSON(d Doc) ([]byte, error) {
	out, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Markdown renders the walkthrough as prose.
func Markdown(d Doc) []byte {
	var b strings.Builder

	title := d.Title
	if title == "" {
		title = "Walkthrough"
	}
	fmt.Fprintf(&b, "# %s\n", title)

	for _, v := range d.Views {
		name := v.Title
		if name == "" {
			name = v.ID
		}
		fmt.Fprintf(&b, "\n## %s\n", name)
		fmt.Fprintf(&b, "\n`%s`", v.ID)
		if v.Diagram != "" {
			fmt.Fprintf(&b, " · %s", v.Diagram)
		}
		if v.ID == d.Root {
			b.WriteString(" · opens here")
		}
		b.WriteString("\n")

		for _, sc := range v.Scenarios {
			writeScenario(&b, sc)
		}
		if len(v.Interactions) > 0 {
			b.WriteString("\n### Interactions\n\n")
			for _, in := range v.Interactions {
				fmt.Fprintf(&b, "- %s\n", in.Text)
			}
		}
	}
	return []byte(b.String())
}

func writeScenario(b *strings.Builder, sc Scenario) {
	name := sc.Name
	if name == "" {
		name = sc.ID
	}
	fmt.Fprintf(b, "\n### Scenario: %s\n", name)
	fmt.Fprintf(b, "\n`%s` · %s long\n", sc.ID, secs(sc.Duration))

	for i, st := range sc.Steps {
		label := st.Name
		if label == "" {
			label = st.ID
		}
		fmt.Fprintf(b, "\n#### %d. %s\n", i+1, label)
		fmt.Fprintf(b, "\n%s–%s\n", secs(st.Start), secs(st.End))
		if st.Desc != "" {
			fmt.Fprintf(b, "\n%s\n", st.Desc)
		}
		if len(st.Events) > 0 {
			b.WriteString("\n")
			for _, e := range st.Events {
				fmt.Fprintf(b, "- %s\n", e.Text)
			}
		}
	}

	if len(sc.Standing) > 0 {
		b.WriteString("\n#### Standing state\n\n")
		b.WriteString("State that outlives the step that set it, with the window it holds for.\n\n")
		for _, e := range sc.Standing {
			fmt.Fprintf(b, "- %s\n", e.Text)
		}
	}
}
