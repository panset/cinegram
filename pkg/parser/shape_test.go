package parser

import (
	"testing"

	"github.com/tejaspanse/diagramator/pkg/ast"
)

func TestParseNodeRef(t *testing.T) {
	tests := []struct {
		in      string
		id      string
		label   string
		shape   ast.Shape
		class   string
		declare bool
	}{
		{in: "svc", id: "svc", label: "svc", shape: ast.ShapeNone},
		{in: "  pod1  ", id: "pod1", label: "pod1", shape: ast.ShapeNone},
		{in: "client[External Client]", id: "client", label: "External Client", shape: ast.ShapeRect, declare: true},
		{in: "lb[(Cloud Load Balancer)]", id: "lb", label: "Cloud Load Balancer", shape: ast.ShapeCylinder, declare: true},
		{in: "a(round)", id: "a", label: "round", shape: ast.ShapeRound, declare: true},
		{in: "a([stadium])", id: "a", label: "stadium", shape: ast.ShapeStadium, declare: true},
		{in: "a[[subroutine]]", id: "a", label: "subroutine", shape: ast.ShapeSubroutine, declare: true},
		{in: "a((circle))", id: "a", label: "circle", shape: ast.ShapeCircle, declare: true},
		{in: "a(((double)))", id: "a", label: "double", shape: ast.ShapeDoubleCircle, declare: true},
		{in: "a{rhombus}", id: "a", label: "rhombus", shape: ast.ShapeRhombus, declare: true},
		{in: "a{{hexagon}}", id: "a", label: "hexagon", shape: ast.ShapeHexagon, declare: true},
		{in: `a[/parallelogram/]`, id: "a", label: "parallelogram", shape: ast.ShapeParallelogram, declare: true},
		{in: `a[\parallelogram alt\]`, id: "a", label: "parallelogram alt", shape: ast.ShapeParallelogramAlt, declare: true},
		{in: `a[/trapezoid\]`, id: "a", label: "trapezoid", shape: ast.ShapeTrapezoid, declare: true},
		{in: `a[\trapezoid alt/]`, id: "a", label: "trapezoid alt", shape: ast.ShapeTrapezoidAlt, declare: true},
		{in: "a>asymmetric]", id: "a", label: "asymmetric", shape: ast.ShapeAsymmetric, declare: true},

		// Quoting protects delimiters and colons inside labels.
		{in: `a["bracket ] inside"]`, id: "a", label: "bracket ] inside", shape: ast.ShapeRect, declare: true},
		{in: "ns[namespace: prod]", id: "ns", label: "namespace: prod", shape: ast.ShapeRect, declare: true},

		// A :::class suffix outside brackets is peeled off; a colon inside
		// a label is not.
		{in: "pod1:::busy", id: "pod1", label: "pod1", shape: ast.ShapeNone, class: "busy"},
		{in: "pod1[Pod A]:::busy", id: "pod1", label: "Pod A", shape: ast.ShapeRect, class: "busy", declare: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			ref, _, ok := parseNodeRef(tc.in)
			if !ok {
				t.Fatalf("parseNodeRef(%q) failed", tc.in)
			}
			if ref.ID != tc.id {
				t.Errorf("id = %q, want %q", ref.ID, tc.id)
			}
			if ref.Label != tc.label {
				t.Errorf("label = %q, want %q", ref.Label, tc.label)
			}
			if ref.Shape != tc.shape {
				t.Errorf("shape = %q, want %q", ref.Shape, tc.shape)
			}
			if ref.Class != tc.class {
				t.Errorf("class = %q, want %q", ref.Class, tc.class)
			}
			if ref.Declare != tc.declare {
				t.Errorf("declare = %v, want %v", ref.Declare, tc.declare)
			}
		})
	}
}

func TestParseNodeRefEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "\t"} {
		if _, _, ok := parseNodeRef(in); ok {
			t.Errorf("parseNodeRef(%q) succeeded, want failure", in)
		}
	}
}
