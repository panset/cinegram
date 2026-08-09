package parser

import (
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/units"
)

// Attribute vocabularies.
//
// The tables are per construct and, for actions, per action kind. `ease` means
// something on a flow and nothing on a highlight, and saying so is the whole
// point: an attribute the compiler never reads is silent at runtime, so a
// `colour:` or `durr:` typo would otherwise animate exactly like a file that
// never mentioned it.
//
// Everything named here belongs to the animation language — no key is specific
// to a diagram type, so these tables stay valid when a second diagram parser
// arrives.
//
// An unknown key is a *warning*, so a file written against a newer binary still
// compiles against an older one. A known key carrying an unusable value is an
// *error*, because the author plainly meant it to do something.

var scenarioAttrs = map[string]string{
	"speed": "float", "loop": "bool", "autoplay": "bool", "style": "string",
}

var stepAttrs = map[string]string{
	"dur": "duration", "delay": "duration", "style": "string", "label": "string",
	"desc": "string",
}

var bindingAttrs = map[string]string{
	"label": "string", "style": "string",
}

// timingAttrs are the keys every action understands: where it sits inside its
// step, how long it lasts, and how it is captioned and styled.
var timingAttrs = map[string]string{
	"dur": "duration", "delay": "duration", "at": "duration",
	"label": "string", "style": "string",
}

// actionAttrs is the per-action vocabulary. Anything absent falls back to
// timingAttrs, so a new action kind is never accidentally attribute-free.
var actionAttrs = map[ast.ActionKind]map[string]string{
	ast.ActionFlow: extendAttrs(timingAttrs, map[string]string{
		"color": "string", "ease": "ease", "status": "status",
		"repeat": "count", "bidi": "bool", "msg": "count",
	}),
	ast.ActionHighlight: extendAttrs(timingAttrs, map[string]string{
		"color": "string",
	}),
	ast.ActionPulse: extendAttrs(timingAttrs, map[string]string{
		"color": "string", "repeat": "count",
	}),
	ast.ActionNote: extendAttrs(timingAttrs, map[string]string{
		"side": "side",
	}),
	ast.ActionDim:   timingAttrs,
	ast.ActionShow:  timingAttrs,
	ast.ActionHide:  timingAttrs,
	ast.ActionFocus: timingAttrs,
	ast.ActionWait:  timingAttrs,
	ast.ActionSeq:   timingAttrs,

	ast.ActionSet: extendAttrs(persistAttrs, map[string]string{
		"badge": "string", "state": "string",
	}),
	ast.ActionGauge: extendAttrs(persistAttrs, map[string]string{
		"label": "string", "value": "string",
	}),
	ast.ActionUnset: persistAttrs,
}

// persistAttrs are the keys a persistent action understands. Deliberately not
// built on timingAttrs: `dur` on a `set` reads as though it would expire, and
// it never does — the state lasts until something replaces it. Leaving the key
// out makes that a warning instead of a silent misunderstanding.
var persistAttrs = map[string]string{
	"delay": "duration", "at": "duration",
	"style": "string", "color": "string",
}

// easeNames are the progress curves a flow may travel with. The runtime remaps
// progress by name and falls back to linear, so an unrecognised name would
// animate as though it had not been written — worth an error rather than a
// shrug.
var easeNames = []string{"linear", "in", "out", "in-out"}

// statusNames are the outcomes a flow may declare.
//
// `style` stays open-ended — it is a CSS hook and a renderer may invent names
// for it. `status` is the opposite: the runtime draws a failure differently
// rather than merely recolouring it, so the set is closed and an unrecognised
// value is an error rather than a class nobody styles.
var statusNames = []string{"ok", "fail"}

// sideNames are the placements a note may ask for. It is a preference rather
// than a coordinate — the renderer still clamps the note into the stage and
// shoves it clear of its neighbours — but an unrecognised name would silently
// fall back to `above`, so the set is closed.
var sideNames = []string{"above", "below", "left", "right"}

// attrsFor returns the vocabulary an action kind accepts.
func attrsFor(kind ast.ActionKind) map[string]string {
	if a, ok := actionAttrs[kind]; ok {
		return a
	}
	return timingAttrs
}

func extendAttrs(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// checkAttrs reports unknown keys and unusable values against a vocabulary.
// what names the construct for the message ("flow", "step", "click").
func checkAttrs(attrs ast.Attrs, allowed map[string]string, what string, b *diag.Bag) {
	for _, k := range attrs.Keys() {
		v, _ := attrs.Get(k)
		typ, ok := allowed[k]
		if !ok {
			b.WarnHintf(v.At, suggestAttr(k, allowed), "unknown %s attribute %q", what, k)
			continue
		}
		var err error
		switch typ {
		case "duration":
			_, err = units.ParseMillis(v.Raw)
		case "float":
			_, err = units.ParseFloat(v.Raw)
		case "count":
			// A count selects or repeats — `msg: 2.7` truncating to message 2
			// would silently animate the wrong arrow, so fractions are errors.
			var f float64
			f, err = units.ParseFloat(v.Raw)
			if err == nil && (f < 1 || f != float64(int(f))) {
				b.ErrorHintf(v.At, "use a whole number, counting from 1",
					"attribute %q: %q is not a positive whole number", k, v.Raw)
			}
		case "bool":
			_, err = units.ParseBool(v.Raw)
		case "ease":
			if !oneOf(easeNames, v.Raw) {
				b.ErrorHintf(v.At, "valid easings are "+strings.Join(easeNames, ", "),
					"attribute %q: %q is not an easing", k, v.Raw)
			}
		case "status":
			if !oneOf(statusNames, v.Raw) {
				b.ErrorHintf(v.At, "valid statuses are "+strings.Join(statusNames, ", "),
					"attribute %q: %q is not a status", k, v.Raw)
			}
		case "side":
			if !oneOf(sideNames, v.Raw) {
				b.ErrorHintf(v.At, "valid sides are "+strings.Join(sideNames, ", "),
					"attribute %q: %q is not a side", k, v.Raw)
			}
		}
		if err != nil {
			b.Errorf(v.At, "attribute %q: %v", k, err)
		}
	}
}

// suggestAttr proposes a correction for an unknown attribute key.
//
// The distance bound is tighter than the one used for node names. Attribute
// vocabularies are small sets of short similar words, so the generous bound
// that works for user-chosen ids would confidently offer "label" for "wobble".
// Below the bound, listing the vocabulary is the more useful answer.
func suggestAttr(key string, allowed map[string]string) string {
	keys := make([]string, 0, len(allowed))
	for k := range allowed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	best, bestDist := "", 1<<30
	for _, c := range keys {
		if d := editDistance(strings.ToLower(key), c); d < bestDist {
			best, bestDist = c, d
		}
	}
	if best != "" && bestDist <= 2 && 2*bestDist < len(key) {
		return "did you mean " + best + "?"
	}
	if len(keys) == 0 {
		return "this construct takes no attributes"
	}
	return "known here: " + strings.Join(keys, ", ")
}

func oneOf(candidates []string, s string) bool {
	for _, c := range candidates {
		if c == s {
			return true
		}
	}
	return false
}
