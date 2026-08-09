package parser

import (
	"github.com/tejaspanse/cinegram/pkg/ast"
)

// A storyboard is the side-stage: what the human sees while the diagram
// animates. Like the rest of this half of the language it names things and
// never models them — a frame is a path and a caption, and turning that path
// into bytes is pkg/loader's job, not this package's.
//
//	storyboard "What the user sees" {
//	  frame idp_login { img: "frames/idp-login.svg", caption: "The IdP login form" }
//	  frame consent   { caption: "Caption-only frames are allowed" }
//	}
//
// Frame names are flat across every storyboard block in a document. There is
// one panel at runtime, so qualifying a name (`user.idp_login`) would buy
// nothing, and flat is how steps and views already work.

// parseStoryboard reads one `storyboard ["<title>"] { frame ... }` block.
func parseStoryboard(s *scanner) *ast.Storyboard {
	kw := s.next() // "storyboard"
	sb := &ast.Storyboard{StartPos: kw.at}

	if t := s.peek(); t.kind == tokString {
		sb.Title = s.next().text
	}
	if !s.expect("{") {
		s.skipToLineEnd()
		return sb
	}

	for {
		s.skipNewlines()
		t := s.peek()
		if t.kind == tokEOF {
			s.bag.ErrorHintf(t.at, "add the closing `}`", "unterminated storyboard block")
			return sb
		}
		if s.accept("}") {
			if len(sb.Frames) == 0 {
				s.bag.Warnf(kw.at, "storyboard block declares no frames")
			}
			return sb
		}
		if !s.atKeyword("frame") {
			s.bag.ErrorHintf(t.at, "every entry starts with `frame`",
				"expected `frame` but found %s", describe(t))
			s.skipToLineEnd()
			continue
		}
		if f := parseFrame(s); f != nil {
			sb.Frames = append(sb.Frames, f)
		}
	}
}

// parseFrame reads `frame <name> { img: "...", caption: "..." }`.
func parseFrame(s *scanner) *ast.Frame {
	kw := s.next() // "frame"

	name := s.peek()
	if name.kind != tokIdent {
		s.bag.ErrorHintf(name.at, `write it as: frame idp_login { img: "frames/idp-login.svg" }`,
			"expected a frame name but found %s", describe(name))
		s.skipToLineEnd()
		return nil
	}
	s.next()

	f := &ast.Frame{Name: name.text, At: kw.at}
	if !s.at("{") {
		t := s.peek()
		s.bag.ErrorHintf(t.at, `give the frame an image, a caption, or both`,
			"expected `{` after frame %q but found %s", f.Name, describe(t))
		s.skipToLineEnd()
		return nil
	}

	attrs := parseAttrBlock(s)
	checkAttrs(attrs, frameAttrs, "frame", s.bag)

	f.Img = attrs.String("img")
	if v, ok := attrs.Get("img"); ok {
		f.ImgAt = v.At
	}
	f.Caption = attrs.String("caption")

	// A frame with neither is not a thing anyone can look at, and a `scene`
	// naming it would blank the panel rather than fail loudly.
	if f.Img == "" && f.Caption == "" {
		s.bag.ErrorHintf(f.At, "give it an `img`, a `caption`, or both",
			"frame %q declares nothing to show", f.Name)
	}
	return f
}
