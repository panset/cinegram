package parser

import (
	"testing"
	"time"
)

// TestParseTerminatesOnMalformedInput guards the property that matters most
// when a source is parsed while it is being typed: parsing always finishes.
//
// It regressed once. `skipToLineEnd` stops at a `}` rather than consuming it,
// so that a malformed action inside a block does not swallow the block's
// closer — but the top-level loop recovered the same way, and a stray `}` out
// there satisfied every stop condition. The loop reported it, advanced past
// nothing, and reported it again forever. `scenario "x" { … }` written
// brace-style produces exactly that: the attribute block ends early and its
// real closer is left stranded at top level.
//
// Each case is run under a deadline because the failure mode is a hang, not a
// wrong answer, and a hung test that only shows up as a timeout tells you far
// less than a named one.
func TestParseTerminatesOnMalformedInput(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "stray closing brace at top level",
			src:  "flowchart LR\n  a[A]\n  b[B]\n  a --> b\n}\n",
		},
		{
			name: "scenario written as a braced body",
			src: "flowchart LR\n  a[A]\n  b[B]\n  a --> b\n\n" +
				"scenario \"x\" {\n  step s \"s\" {\n    flow a -> b { dur: 1s }\n  }\n}\n",
		},
		{
			name: "action inside the scenario attribute block",
			src:  "flowchart LR\n  a[A]\n  b[B]\n  a --> b\n\nscenario \"x\" {\n  flow a -> b { dur: 1s }\n}\n",
		},
		{
			name: "unbalanced braces piling up",
			src:  "flowchart LR\n  a[A]\n\nscenario \"x\"\n\n  step s {\n  }\n}\n}\n}\n",
		},
		{
			name: "half-typed step",
			src:  "flowchart LR\n  a[A]\n\nscenario \"x\"\n\n  step s \"s\" {\n    flow a ->\n",
		},
		{
			name: "punctuation soup",
			src:  "flowchart LR\n  a[A]\n\n}{},:->\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				// The result is irrelevant: any diagnostics are fine, and so is
				// none. Only returning at all is under test.
				Parse("t.dgm", tc.src)
			}()

			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("Parse did not terminate; error recovery failed to consume a token")
			}
		})
	}
}
