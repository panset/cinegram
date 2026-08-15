package sitegen

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/loader"
)

// encodePlaygroundDoc turns a bundle into the playground's share-link
// fragment: the same JSON the playground's own Share button produces —
// {v:1, entry, open, files:[{p,t,d}]} — raw-deflated and base64url'd. The
// formats have to match exactly (deflate-raw, unpadded base64url), because
// the other end is web/playground/playground.js's loadFromHash.
//
// Sources go in as text; storyboard images go in as their original bytes,
// recovered from the data: URIs the loader already built — the loader read
// them once, and reading them again would race an edit.
func encodePlaygroundDoc(b *loader.Bundle, read loader.ReadFileFunc) (string, error) {
	type file struct {
		P string `json:"p"`
		T string `json:"t"`
		D string `json:"d"`
	}
	doc := struct {
		V     int    `json:"v"`
		Entry string `json:"entry"`
		Open  string `json:"open"`
		Files []file `json:"files"`
	}{V: 1}

	seen := map[string]bool{}
	for i, u := range b.Units {
		if i == 0 {
			doc.Entry = u.Path
			doc.Open = u.Path
		}
		if !seen[u.Path] {
			seen[u.Path] = true
			source, err := read(u.Path)
			if err != nil {
				return "", err
			}
			doc.Files = append(doc.Files, file{P: u.Path, T: "text", D: string(source)})
		}
		// Frame paths are as written, relative to the unit — resolve them the
		// way the loader did so the playground's loader finds them the same
		// way from the same .dgm text. Sorted: FrameData is a map, and a
		// committed page whose bytes depended on iteration order would churn
		// on every sync.
		dir := path.Dir(u.Path)
		written := make([]string, 0, len(u.FrameData))
		for w := range u.FrameData {
			written = append(written, w)
		}
		sort.Strings(written)
		for _, w := range written {
			resolved := path.Clean(path.Join(dir, w))
			dataURI := u.FrameData[w]
			if seen[resolved] || dataURI == "" {
				continue
			}
			seen[resolved] = true
			_, b64, ok := strings.Cut(dataURI, ";base64,")
			if !ok {
				continue // an unreadable image compiled with no data; skip it
			}
			doc.Files = append(doc.Files, file{P: resolved, T: "b64", D: b64})
		}
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	w, err := flate.NewWriter(&out, flate.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(raw); err != nil {
		return "", fmt.Errorf("deflating playground doc: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(out.Bytes()), nil
}
