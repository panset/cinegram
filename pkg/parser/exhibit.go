package parser

import (
	"github.com/tejaspanse/cinegram/pkg/ast"
)

// An exhibit is a document kept beside the stage: the manifest, config or log
// excerpt a diagram is about, in a drawer until a reader opens it. Like a view
// or a storyboard frame it names a path and never bytes — pkg/loader reads the
// file, and this package only records where it was said.
//
//	exhibit deploy "Deployment manifest" from "manifests/deployment.yaml"
//	exhibit svc    "Service manifest"    from "manifests/service.yaml" { for: svc }
//
// `for` ties the exhibit to a diagram element, which lights up while the
// exhibit is hovered or zoomed.

// parseExhibit reads `exhibit <id> ["<title>"] from "<path>" [{ attrs }]`.
func parseExhibit(s *scanner) *ast.Exhibit {
	kw := s.next() // "exhibit"
	x := &ast.Exhibit{At: kw.at}

	id := s.peek()
	if id.kind != tokIdent {
		s.bag.ErrorHintf(id.at, "write `exhibit <id> from \"file.yaml\"`",
			"expected an exhibit name but found %s", describe(id))
		s.skipToLineEnd()
		return nil
	}
	x.ID = s.next().text

	if t := s.peek(); t.kind == tokString {
		x.Title = s.next().text
	}

	if !s.atKeyword("from") {
		t := s.peek()
		s.bag.ErrorHintf(t.at, "write `exhibit "+x.ID+" from \"file.yaml\"`",
			"expected `from` but found %s", describe(t))
		s.skipToLineEnd()
		return nil
	}
	s.next() // "from"

	path := s.peek()
	if path.kind != tokString {
		s.bag.ErrorHintf(path.at, "the path must be quoted, e.g. `from \"manifests/deployment.yaml\"`",
			"expected a quoted file path but found %s", describe(path))
		s.skipToLineEnd()
		return nil
	}
	s.next()
	x.Path = path.text
	x.PathAt = path.at

	if s.at("{") {
		x.Attrs = parseAttrBlock(s)
		checkAttrs(x.Attrs, exhibitAttrs, "exhibit", s.bag)
	}
	return x
}
