package parser

import (
	"strings"

	"github.com/tejaspanse/diagramator/pkg/ast"
)

// linkMatch is a Mermaid link operator found in a line, including any label
// carried in its middle (`-- text -->`).
type linkMatch struct {
	start int // byte offset of the first character of the operator
	end   int // byte offset just past the operator
	link  ast.Link
}

// matchLink attempts to match a link operator at s[i:].
//
// Mermaid's disambiguation rule is subtle but consistent, and worth stating
// because it drives the whole function: an operator of exactly two dashes or
// two equals signs with no arrowhead is not a link, it is the *opening half* of
// a labelled link (`A-- text -->B`). Three or more, or any arrowhead, closes
// the operator immediately. The dotted family works the same way with `-.` as
// the opener and `.-`/`.->` as the closer.
func matchLink(s string, i int) (linkMatch, bool) {
	j := i
	bidir := false
	if j < len(s) && s[j] == '<' {
		if j+1 >= len(s) || (s[j+1] != '-' && s[j+1] != '=') {
			return linkMatch{}, false
		}
		bidir = true
		j++
	}

	var (
		style  ast.LinkStyle
		length int
		closed bool
	)

	switch {
	case s[j] == '-' && j+1 < len(s) && s[j+1] == '.':
		style = ast.LinkDotted
		j++ // the leading dash
		dots := 0
		for j < len(s) && s[j] == '.' {
			j++
			dots++
		}
		length = dots + 1
		if j < len(s) && s[j] == '-' {
			j++
			closed = true
		}

	case s[j] == '-' && j+1 < len(s) && s[j+1] == '-':
		style = ast.LinkNormal
		n := 0
		for j < len(s) && s[j] == '-' {
			j++
			n++
		}
		length = n
		closed = n >= 3

	case s[j] == '=' && j+1 < len(s) && s[j+1] == '=':
		style = ast.LinkThick
		n := 0
		for j < len(s) && s[j] == '=' {
			j++
			n++
		}
		length = n
		closed = n >= 3

	default:
		return linkMatch{}, false
	}

	head := ast.ArrowNone
	if j < len(s) {
		switch s[j] {
		case '>':
			head, j, closed = ast.ArrowNormal, j+1, true
		case 'o':
			head, j, closed = ast.ArrowCircle, j+1, true
		case 'x':
			head, j, closed = ast.ArrowCross, j+1, true
		}
	}

	link := ast.Link{
		Style:  style,
		Head:   head,
		Bidir:  bidir,
		Length: length,
	}

	if closed {
		link.Operator = s[i:j]
		return linkMatch{start: i, end: j, link: link}, true
	}

	// Unclosed: this is the opening half of a labelled link. Find the closer
	// and take everything between as the label.
	labelStart := j
	closeStart, closeEnd, closeHead, ok := findLinkCloser(s, j, style)
	if !ok {
		return linkMatch{}, false
	}
	link.Head = closeHead
	link.Label = strings.TrimSpace(s[labelStart:closeStart])
	link.Operator = s[i:closeEnd]
	return linkMatch{start: i, end: closeEnd, link: link}, true
}

// findLinkCloser scans forward from i for the closing half of a labelled link
// of the given style. It returns the offsets bounding the closer and the
// arrowhead it carries.
func findLinkCloser(s string, i int, style ast.LinkStyle) (start, end int, head ast.ArrowHead, ok bool) {
	for k := i; k < len(s); k++ {
		var j int
		switch style {
		case ast.LinkDotted:
			// Closer is `.-` or `.->`, possibly with extra dots.
			if s[k] != '.' {
				continue
			}
			j = k
			for j < len(s) && s[j] == '.' {
				j++
			}
			if j >= len(s) || s[j] != '-' {
				continue
			}
			j++

		case ast.LinkThick:
			if s[k] != '=' {
				continue
			}
			j = k
			for j < len(s) && s[j] == '=' {
				j++
			}
			if j-k < 2 {
				continue
			}

		default: // ast.LinkNormal
			if s[k] != '-' {
				continue
			}
			j = k
			for j < len(s) && s[j] == '-' {
				j++
			}
			if j-k < 2 {
				continue
			}
		}

		head = ast.ArrowNone
		if j < len(s) {
			switch s[j] {
			case '>':
				head, j = ast.ArrowNormal, j+1
			case 'o':
				head, j = ast.ArrowCircle, j+1
			case 'x':
				head, j = ast.ArrowCross, j+1
			}
		}
		return k, j, head, true
	}
	return 0, 0, ast.ArrowNone, false
}

// splitLinks divides a line into node segments separated by link operators.
// Bracket depth and quoting are tracked so that operators inside a node label,
// such as `a["x --> y"] --> b`, are not mistaken for links.
//
// The returned segs slice always has exactly len(links)+1 entries.
func splitLinks(s string) (segs []string, offs []int, links []linkMatch) {
	depth := 0
	inQuote := false
	start := 0

	for i := 0; i < len(s); {
		c := s[i]
		if inQuote {
			if c == '"' {
				inQuote = false
			}
			i++
			continue
		}
		switch c {
		case '"':
			inQuote = true
			i++
		case '[', '(', '{':
			depth++
			i++
		case ']', ')', '}':
			// Clamp: the asymmetric shape `id>text]` closes a bracket it
			// never opened, and must not push depth negative.
			if depth > 0 {
				depth--
			}
			i++
		default:
			if depth == 0 {
				if m, ok := matchLink(s, i); ok {
					segs = append(segs, s[start:m.start])
					offs = append(offs, start)
					// A `|label|` suffix overrides any mid-operator label.
					end := m.end
					if lbl, next, found := matchPipeLabel(s, end); found {
						m.link.Label = lbl
						end = next
					}
					m.end = end
					links = append(links, m)
					start = end
					i = end
					continue
				}
			}
			i++
		}
	}
	segs = append(segs, s[start:])
	offs = append(offs, start)
	return segs, offs, links
}

// matchPipeLabel reads a `|text|` edge label starting at or after i, skipping
// intervening spaces.
func matchPipeLabel(s string, i int) (label string, next int, ok bool) {
	j := i
	for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
		j++
	}
	if j >= len(s) || s[j] != '|' {
		return "", i, false
	}
	j++
	k := strings.IndexByte(s[j:], '|')
	if k < 0 {
		return "", i, false
	}
	return unquote(strings.TrimSpace(s[j : j+k])), j + k + 1, true
}
